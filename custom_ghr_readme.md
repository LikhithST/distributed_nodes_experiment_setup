# GHZ Custom

A repository for managing and making gRPC calls using the `ghz` tool.

## Table of Contents

- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
- [Starting the Database](#starting-the-database)
- [Making gRPC Calls](#making-grpc-calls)
  - [Publish Call](#publish-call)
  - [Subscribe Call](#subscribe-call)
- [Sample Config-files Structure](#sample-config-files-structure)
## Getting Started

### Prerequisites

Before you begin, ensure you have met the following requirements:

- You have `git` installed.
- You have `chmod` and `sudo` permissions.

### Installation

To clone the repository, run:

```sh
git clone https://github.boschdevcloud.com/SAI1RNG/ghz-custom.git -b feature/main
```

## Starting the Database

```sh
cd ghz-custom/cmd/ghz-web
chmod +x ./ghz-web
sudo ./ghz-web
```
## Making gRPC Calls
```sh
cd ghz-custom/cmd/ghz
chmod +x ./ghz
```
### Publish Call
```sh
./ghz --insecure --config=config.json
```

### Subscribe Call
```sh
./ghz --insecure --config=config_subscribe.json json
```

### Publish Call (with Database)
```sh
./ghz --insecure --config=config.json -O json | http POST localhost:80/api/ingest
```

### Subscribe Call (with Database)
```sh
./ghz --insecure --config=config_subscribe.json json | http POST localhost:80/api/ingest
```

## Sample Config-files Structure
<div>
  <div style="float: left; width: 45%;">
    <ol>
      <li><strong>timeout</strong>: The amount of time the ghz is required to subscribe to kuksa-databroker <code>{{.RequestNumber}}</code> placeholder.</li>
      <li><strong>request_id</strong>: This acts as a identifier to differntiate between multiple tests conducted. Example: <code>50sub300pub</code> indicates an experiment with 50 subscribers and 300 publish calls, request_id can be anything helping identify between tests.</li>
      <li><strong>description</strong>: Provides additional information about the request. Uses the <code>{{.RequestNumber}}</code> placeholder.</li>
      <li><strong>fields</strong>: An array specifying data fields to be kept track of, such as <code>["2", "12"]</code>, here having <b>"12"</b> is a must.</li>
    </ol>
  </div>
  <div style="float: right; width: 45%;">
    <img src="Images/publish.JPG" alt="Publish Config" width="100%">
    <br>
    <img src="Images/subscribe.JPG" alt="Subscribe Config" width="100%">
  </div>
  <div style="clear: both;"></div>
</div>

Note: The combination of `request_id` and `description` forms a unique id for identifying a particular publish call in a given test.

##  building ghz for arm64 architecture

use below docker command and inject the custom-ghz repo as a volume inside the golang docker container

 ```docker run -it -v $PWD:/go/work golang:1.20-alpine ```
 
 traverse to the cmd/ghz folder and execute below command
 
 ```go build . ```

## Commands used for testing on L4S testbed
### structure of L4S testbed
![image](https://media.github.boschdevcloud.com/user/2955/files/265fee04-9fa0-4986-91ee-2b3efdcce498)

### nuc2rng (acting as a vehicle computer)

#### Starting ghz-web to collect monitoring data at (/home/nuc2rng/kuksa/likhith-kuksa-l4s/ghz-custom/cmd/ghz-web)

 ```sudo ./ghz-web -config config.yaml ```
 
 config.yaml content
 
 ```
 server:
  port: 9999
  ```
  #### starting KUKSA databroker
  
  starting KUKSA at location: /home/nuc2rng/kuksa/likhith-kuksa-l4s/kuksa-databroker  

```docker compose up kuksa-databroker  ```

### pi2rng acting as a publisher

starting ghz in location : /home/pi2rng/kuksa/ghz-custom/cmd/ghz

```./ghz --insecure --config=config.json -O json | http POST 192.168.10.20:9999/api/ingest```


### pi3rng acting as a subscriber

starting ghz in location : /home/pi3rng/kuksa/ghz-custom/cmd/ghz

```./ghz --insecure --config=config_subscribe.json -O json | http POST 192.168.10.20:9999/api/ingest```

### generation of latency report from the ghz.db (containing monitoring info) in the nuc2rng folder

#### merging dbs from both pub and sub machines

execute following shell script from /home/nuc2rng/kuksa/likhith-kuksa-l4s/ghz-custom/cmd/ghz-web folder  : /home/nuc2rng/kuksa/likhith-kuksa-l4s/ghz-custom/cmd/ghz-web/copy-ghz-dbs-from-clients.sh
```
scp pi3rng@192.168.10.33:///home/pi3rng/kuksa/ghz-custom/cmd/ghz-web/data/ghz.db ./data/ghz-subscriber-info.db
scp pi2rng@192.168.10.22:///home/pi2rng/kuksa/ghz-custom/cmd/ghz-web/data/ghz.db ./data/ghz-publisher-info.db
```

execute below command to do merging

```sudo docker run -v $PWD:/work --entrypoint "/usr/local/bin/python3" rekocd/python-pandas:3.12.0 /work/data/db_merger.py /work/data/ghz-subscriber-info.db /work/data/ghz-publisher-info.db /work/data/ghz.db```

execute below command to do data extraction

```sudo docker run  -v $PWD:/work --entrypoint "/usr/local/bin/python3" rekocd/python-pandas:3.12.0 "/work/sqlite-latency-extractor_XLS.py"```



## copy report locally

```scp nuc2rng@10.163.13.248:///home/nuc2rng/kuksa/likhith-kuksa-l4s/ghz-custom/cmd/ghz-web/latency_and_mean_stats.xlsx .  ```


