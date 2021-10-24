package main

import (
	"github.com/qedus/osmpbf"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

type testData struct {
	testName            string
	nodeInfos           []nodeInfo
	expectedCoordinates []coordinates
}

func TestRoadJuncMapMaker(t *testing.T) {
	assert := assert.New(t)

	testCases := []testData{
		{"road with single junction", []nodeInfo{{coordinates{1.0, 2.0}, "J1", "M1"}}, []coordinates{{1.0, 2.0}}},
		{"road with multiple junctions", []nodeInfo{{coordinates{3.0, 4.0}, "J1", "M2"}, {coordinates{5.0, 6.0}, "J2", "M2"}}, []coordinates{{3.0, 4.0}, {5.0, 6.0}}},
		{"different road with same junction name", []nodeInfo{{coordinates{7.0, 8.0}, "J2", "M3"}, {coordinates{9.0, 10.0}, "J2", "M3"}}, []coordinates{{7.0, 8.0}, {9.0, 10.0}}},
	}

	for _, testdata := range testCases {
		idMap := make(map[int64]nodeInfo)
		for _, nodes := range testdata.nodeInfos {
			idMap[int64(len(idMap))] = nodes
			roadJunc := roadJuncMapMaker(idMap)
			for _, coor := range roadJunc.Road[nodes.road].Junc[nodes.junc].Coordinates {
				assert.Contains(testdata.expectedCoordinates, coor, testdata.testName)
			}
		}
	}
}

type osmpbfstruct struct {
	index        int
	nodesAndWays []interface{}
}

func generateTestStruct() *osmpbfstruct {
	return &osmpbfstruct{
		nodesAndWays: []interface{}{
			&osmpbf.Way{
				Tags: map[string]string{
					reftag:     "M100",
					highwaytag: motorwaytag,
				},
				NodeIDs: []int64{100, 200, 300},
			},
			&osmpbf.Node{
				ID:  100,
				Lat: 1.0,
				Lon: 2.0,
				Tags: map[string]string{
					reftag:     "10",
					highwaytag: motorwayjunctiontag,
				},
			},
			&osmpbf.Node{
				ID:  200,
				Lat: 3.0,
				Lon: 4.0,
				Tags: map[string]string{
					reftag:     "20",
					highwaytag: motorwayjunctiontag,
				},
			},
			&osmpbf.Node{
				ID:   400,
				Lat:  5.0,
				Lon:  6.0,
				Tags: nil,
			},
			&osmpbf.Way{
				Tags: map[string]string{
					reftag:     "M200",
					highwaytag: "tertiary",
				},
				NodeIDs: []int64{500},
			},
		},
	}
}

type idMaptest struct {
	testName     string
	iD           int64
	expectedCoor coordinates
	expectedJunc string
	expectedRoad string
	expectedBool bool
}

func (o *osmpbfstruct) getNextItem() (interface{}, error) {
	if o.index >= len(o.nodesAndWays) {
		return nil, io.EOF
	}
	item := o.nodesAndWays[o.index]
	o.index++
	return item, nil
}

func (o *osmpbfstruct) Decode() (interface{}, error) {
	return o.getNextItem()
}

func TestDecodePbf(t *testing.T) {
	assert := assert.New(t)
	o := generateTestStruct()
	idMap, err := decodePbf(o)
	if err != nil {
		return
	}
	testCases := []idMaptest{
		{"Test for details in Node 100", 100, coordinates{1.0, 2.0}, "10", "M100", true},
		{"Test for details in Node 200", 200, coordinates{3.0, 4.0}, "20", "M100", true},
		{"Test for details in Node 300", 300, coordinates{}, "", "M100", true},
		{"Test for non-existence Node 400", 400, coordinates{}, "", "", false},
		{"Test for non-existence Node 500", 500, coordinates{}, "", "", false},
	}

	for _, test := range testCases {
		prop, ok := idMap[test.iD]
		assert.Equal(test.expectedCoor, prop.coor, test.testName)
		assert.Equal(test.expectedJunc, prop.junc, test.testName)
		assert.Equal(test.expectedRoad, prop.road, test.testName)
		assert.Equal(test.expectedBool, ok, test.testName)
	}
}
