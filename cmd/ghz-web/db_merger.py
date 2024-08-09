
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

```sudo docker run  -v $PWD:/work --entrypoint "/usr/local/bin/python3" rekocd/python-pandas:3.12.0 "/work/sqlite-latency-extractor_XLS.py"```

```code```



