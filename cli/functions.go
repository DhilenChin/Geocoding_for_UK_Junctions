package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qedus/osmpbf"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"io"
	"net/http"
	"os"
	"runtime"
)

type nodeInfo struct {
	//Struct to store info on individual node
	coor       coordinates
	junc, road string
}

type coordinates struct {
	Lat, Lon float64
}

type road struct {
	Road map[string]junc
}

func (roadJunc road) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	//2 arguments needed for the query
	road := request.URL.Query().Get("road")
	junc := request.URL.Query().Get("junc")
	if road == "" || junc == "" {
		http.Error(writer, "Keys are in this format: road=M1&junc=1", http.StatusBadRequest)
		return
	}

	juncstruct, ok := roadJunc.Road[road]
	if !ok {
		http.Error(writer, "Road not in database", http.StatusNotFound)
		return
	}
	coor, ok := juncstruct.Junc[junc]
	if !ok {
		http.Error(writer, "Road in database, but junction not in database", http.StatusNotFound)
		return
	}
	coorJson, err := json.Marshal(coor)
	if err != nil {
		log.Error().Err(err).Msg("httrReq: could not marshal the struct")
		http.Error(writer, "Could not marshal the strcut into a json", http.StatusUnprocessableEntity)
		return
	}
	_, err = writer.Write(coorJson)
	if err != nil {
		http.Error(writer, "Could not write json into json into server", http.StatusInternalServerError)
		log.Error().Err(err).Msg("httrReq: could write json onto server")
	}

}

func (roadJunc road) serve(ptsReady *atomic.Bool, address *string) error {
	http.Handle("/", roadJunc)

	log.Info().Msgf("Serving on HTTP port: %s", *address)
	ptsReady.Store(true)
	err := http.ListenAndServe(*address, nil)
	if err == http.ErrServerClosed {
		log.Error().Err(err).Msg("server was closed")
		return nil
	}
	if err != nil {
		return fmt.Errorf("httpReq: could not initialise server %w", err)
	}
	return nil
}

type junc struct {
	Junc map[string]listcoor
}

type listcoor struct {
	Coordinates []coordinates
}

const (
	reftag              = "ref"
	highwaytag          = "highway"
	motorwaytag         = "motorway"
	motorwaylinktag     = "motorway_link"
	motorwayjunctiontag = "motorway_junction"
	trunktag            = "trunk"
	trunklinktag        = "trunk_link"
	primarytag          = "primary"
	primarylinktag      = "primary_link"
)

func isMotorwayTrunkOrPrimary(tags map[string]string) bool {
	switch tags[highwaytag] {
	case motorwaytag, motorwaylinktag, trunktag, trunklinktag, primarytag, primarylinktag:
		return true
	default:
		return false
	}
}

func readFile(fileName string) (io.ReadCloser, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("readFile: could not open file name: %w", err)
	}
	return f, nil
}

func readURL(url string) (io.ReadCloser, error) {
	log.Info().Msg("Reading content from URL")
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not read data from url: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status error: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func deferFunc(inputFile io.Closer) {
	err := inputFile.Close()
	if err != nil {
		log.Error().Err(err).Msg("readFile: could not close file")
	}
}

func idMapMaker(inputFile io.ReadCloser) (map[int64]nodeInfo, error) {
	//making a map of map[node_id] -> nodeInfo
	defer deferFunc(inputFile)

	d := osmpbf.NewDecoder(inputFile)
	// use more memory from the start, it is faster
	d.SetBufferSize(osmpbf.MaxBlobSize)

	// start decoding with several goroutines, it is faster
	err := d.Start(runtime.GOMAXPROCS(-1))
	if err != nil {
		return nil, fmt.Errorf("idMapMaker: not able to start decoding the osm.pbf: %w", err)
	}

	idMap, err := decodePbf(d)
	if err != nil {
		return nil, fmt.Errorf("idMapMaker: not able to decoded file: %w", err)
	}
	return idMap, err
}

type decoder interface {
	Decode() (interface{}, error)
}

func decodePbf(decode decoder) (map[int64]nodeInfo, error) {
	// initialise map of node id -> node information (eg coordinates, road, junction)
	idMap := make(map[int64]nodeInfo)

	for {
		v, err := decode.Decode()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("idMapMaker: Could not read the osm.pbf: %w", err)
		}

		switch v := v.(type) {
		//mapping id to coordinates and junctions
		case *osmpbf.Node:
			// add node to map
			if v.Tags[highwaytag] != motorwayjunctiontag {
				continue
			}
			ref, okjunc := v.Tags[reftag]
			if !okjunc {
				continue
			}
			nodeProp := idMap[v.ID]
			nodeProp.coor.Lat = v.Lat
			nodeProp.coor.Lon = v.Lon
			nodeProp.junc = ref
			idMap[v.ID] = nodeProp

		case *osmpbf.Way:
			if !isMotorwayTrunkOrPrimary(v.Tags) {
				continue
			}
			ref, okroad := v.Tags[reftag]
			if !okroad {
				continue
			}
			for _, iD := range v.NodeIDs {
				nodeProp := idMap[iD]
				nodeProp.road = ref
				idMap[iD] = nodeProp
			}

		case *osmpbf.Relation:
			continue
		}
	}
	log.Info().Msg("Finish reading and sorting the content")
	return idMap, nil
}

func roadJuncMapMaker(idMap map[int64]nodeInfo) road {
	// initialise map[road]map[junction]coordinates
	roadJunc := make(map[string]junc)

	// iterate the nodes and put the nodes with both reference and junction info in the map
	for _, nodeProp := range idMap {
		empty := nodeProp.road == "" || nodeProp.junc == ""
		if empty {
			continue
		}
		juncCoor, ok := roadJunc[nodeProp.road]
		if !ok {
			juncCoor.Junc = make(map[string]listcoor)
		}
		coordinates := append(juncCoor.Junc[nodeProp.junc].Coordinates, nodeProp.coor)
		juncCoor.Junc[nodeProp.junc] = listcoor{coordinates}
		roadJunc[nodeProp.road] = juncCoor

	}

	return road{roadJunc}
}

func setupStatusServer(statusAddress string, isReady *atomic.Bool) {
	//Server that decides that the service is only ready after finish reading the osm.pbf content
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if !isReady.Load() {
			http.Error(w, "The service is not ready yet.", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	err := http.ListenAndServe(statusAddress, nil)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("Could not initiate the status server")
	}
}
