# Project setup using docker images

' Idea: Different docker conatiners is used for different programs and each of the containrs are run in different timezone. '

ghz-web: 

```
docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/ghz -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name ghz-web-container golang:1.23 bash

docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/ghz -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name ghz-web-container likhithst/ghz-web-node
```

ghz as publisher: 

```
docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/ghz -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name ghz-publisher-container golang:1.23 bash

docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/ghz -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name ghz-publisher-container likhithst/ghz-publisher-node bash

./ghz --insecure --config=config.json -O json | http POST localhost:80/api/ingest
```

ghz as subscriber: 

```
docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/ghz -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name ghz-subscriber-container golang:1.23 bash

docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/ghz -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name ghz-subscriber-container likhithst/ghz-subscriber-node bash

export FICTITIOUS_DELAY=0.250

export LONG_DISTANCE_CONNECTION_DELAY=0.10

export PROCESS_DELAY=0.001

./ghz --insecure --config=config_subscribe.json -O json | http POST localhost:80/api/ingest
```

Kuksa-databroker:
```
docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/kuksa-databroker-stats -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name kuksa-databroker-container rust:1.82 bash

docker run --net host -e TZ=Asia/Kolkata -v $(pwd):/kuksa-databroker-stats -v /etc/timezone:/etc/timezone:ro -v /etc/localtime:/etc/localtime:ro -it --rm --name kuksa-databroker-container likhithst/kuksa-databroker-node bash

# run the program
 cargo run --bin databroker --features stats -- --address 127.0.0.1 --metadata ./data/vss-core/vss_release_4.0.json --insecure

 LD_PRELOAD=/libfaketime/src/libfaketime.so.1 FAKETIME="+1y i2,0" ./target/debug/databroker --address 127.0.0.1 --metadata ./data/vss-core/vss_release_4.0.json --insecure

 # or run the binary directly which is available in 'target' folder
```

# Libfaketime Installation

```sh
git clone https://github.com/wolfcw/libfaketime.git
cd libfaketime
git checkout ba9ed5b2898f234cfcefbe5c694b7d89dcec4334
make
make install
```

```sh
cd ghz/cmd/ghz-web
LD_PRELOAD=/libfaketime/src/libfaketime.so.1 FAKETIME="@2000-01-01 11:12:13" ./ghz-web
```

Run the 

# Changes to ghz

added local stats package 

maintain same go version between all main program and the local stats package

```sh
 go mod edit -replace google.golang.org/grpc=/mnt/c/Users/tolik/Documents/Research_Project/grpc-go
```

# to generate the excel use the devcontainer in ghz-web folder and run 
```sh
python sqlite-latency-extractor-dev.py
```


# Save the containers with the added packages using ```docker commit```

```sh
# save kuksa-databroker-container
docker commit kuksa-databroker-container kuksa-databroker-node
```
```sh
# save ghz-subscriber-container
docker commit ghz-subscriber-container ghz-subscriber-node
```
```sh
# save ghz-publisher-container
docker commit ghz-publisher-container ghz-publisher-node
```
```sh
# save ghz-web-container
docker commit ghz-web-container ghz-web-node
```

