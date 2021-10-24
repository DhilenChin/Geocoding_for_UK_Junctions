# Junction-coordinates
Getting coordinates of junctions on UK motorways

## To run locally
Steps:
1. Download the [Great-Britain osm.pbf](http://download.geofabrik.de/europe/great-britain-latest.osm.pbf) file. Copy it into the repository directory locally.
2. Run the program.
3. Head to `http://localhost:8080/?road=M11&junc=10`. It should be showing the coordinates of M11 J10. 
4. Change the road and junction names to get the respective coordinates.
## To run locally, but calling Google Cloud
Steps:
1. If you have Google Cloud Account set up, in terminal run 
```
kubectl port-forward -n nugraph service/junction-coordinates 8080:http
```
2. Head to `http://localhost:8080/?road=M11&junc=10`. It should be showing the coordinates of M11 J10.
3. Change the road and junction names to get the respective coordinates.