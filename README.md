# Metrici LPR Simulator

## Command line arguments
|ENV Variable |  Required  |  Description  |
|:--:|:--:|:--:|
| CONFIG | yes | Configuration file for this simulated Metrici LPR server. |

Example:
```bash
cd ~/go/src/github.com/matt-doug-davidson/metrici-lpr-simulator

go build metrici-lpr-simulator.go
CONFIG=~/SimulatorData/MetriciLPR/Cary/nc.yaml ./metrici-lpr-simulator

CONFIG=~/SimulatorData/MetriciLPR/Cary/nc.yaml go run metrici-lpr-simulator.go
```
## Docker
### Build
```bash
cd ~/go/src/github.com/matt-doug-davidson/metrici-lpr-simulator

docker build -t mddofapex/metrici-lpr-simulator:v0.02 .
```
### Run
The program running in the container expects three files in the /data directory as follows:
- a configuration file
- car_image
- plate_image

These files are placed in a host directory that is mapped via the volume option (-v or --volume) in the docker run command.

The configuration file is specified per application. Its name is passed to the container via the environmental variable CONFIG.

Example:
```bash
docker run --env CONFIG="nc.yaml" -v ~/SimulatorData/MetriciLPR/Cary:/data  mddofapex/metrici-lpr-simulator:v0.0.2
```
In the example command above,the files, nc.yaml, plate_image and car_image, are copied to the ~/SimulatorData/MetriciLPR/Cary directory. They are accessed by the container program in the /data directory.

### Push
```bash
docker push mddofapex/metrici-lpr-simulator:v0.0.2
```

## YAML Configuration File


| Setting     | Type   | Required  | Description |
|:------------|:-------|:----------|:------------|
| target-location  | string      | True | The location we are simulating the Metrici is located.|
| connector-host| string | True | The hostname of the Connector to which messsages are sent |
| car-image-file| string| False | The path to the car image file. This is applicable only when the simulator is run from command line (not in docker) |
|plate-image-file| string | False |The path to the plate image file. This is applicable only when the simulator is run from command line (not in docker) |
| debug | bool | True | The debug flag |
| cameras | Camera Configuration | True | An array of camera configurations (see below).

### Camera Configuration
Each camera in the cameras will have its unique configuration.
| Setting     | Type   | Required  | Description |
|:------------|:-------|:----------|:------------|
| id | int | Yes | The unique identifier for the camera. |
| direction | int | Yes | The direction the cars are flowing with respect to the camera. 1 - toward 2 - away, 3 - undetermined. |
| rate | float | Yes | The rate expresses the average vehicles per hour. The rate sets the average interval between vehicles. |
|rate-variance| float | Yes | The variance expressed as a percentage affects the vehicle report rate. This gives a bit a variation to the reports while averaging to the rate. |
|authkey| string | Yes | The authorization key for this camera. Authorization is disabled on the Connector but add this just in-case it is needed in the future |


The average interval between vehicles is 3600/rate.

The application uses go routines to perform multi-processing.

Example:
```yaml
target-location: Europe/Bucharest
connector-host: 10.52.16.76
# Define for running from CLI. Not needed for docker.
car-image-path: /home/allied/mapper123.json
plate-image-path: /home/allied/mapper123.json
cameras:
-
      id: 1
      direction: 1
      rate: 3600
      rate-variance: 10
      authkey: Test123
-
      id: 2
      direction: 2
      rate: 1200
      rate-variance: 5
      authkey: Test123
-
      id: 3
      direction: 1
      rate: 1800
      rate-variance: 15
      authkey: Test123
-
      id: 4
      direction: 2
      rate: 3600
      rate-variance: 13
      authkey: Test123
debug: false
