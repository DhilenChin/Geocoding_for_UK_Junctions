package main

import (
	"flag"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"io"
)

func main() {
	isLocal := flag.Bool("isLocal", true, "Running locally")
	address := flag.String("address", "localhost:8080", "address to serve on")
	readyAddress := flag.String("readyAddress", "localhost:8081", "address for status server")
	flag.Parse()
	isReady := atomic.NewBool(false)
	go setupStatusServer(*readyAddress, isReady)
	fileUrl := "http://download.geofabrik.de/europe/great-britain-latest.osm.pbf"
	fileName := "great-britain-latest.osm.pbf"
	var file io.ReadCloser
	var err error
	if *isLocal {
		file, err = readFile(fileName)
		if err != nil {
			log.Fatal().Err(err).Msg("readFile: could not read the file locally")
		}
	} else {
		file, err = readURL(fileUrl)
		if err != nil {
			log.Fatal().Err(err).Msg("readURL: could not read from URL")
		}
	}
	idMap, idMapErr := idMapMaker(file)
	if idMapErr != nil {
		log.Fatal().Err(idMapErr).Msg("idMapMaker: could not call function in main")
	}
	roadJunc := roadJuncMapMaker(idMap)
	roadJuncErr := roadJunc.serve(isReady, address)

	if roadJuncErr != nil {
		log.Fatal().Err(roadJuncErr).Msg("httpReq: could not call function in main")
	}
}
