## Project setup
`git clone https://github.boschdevcloud.com/SAI1RNG/ghz-custom.git`


### To start the database

`cd ghz-custom/cmd/ghz-web`  
`chmod +x ./ghz-web`
`sudo ./ghz-web`

### To make grpc calls  

`cd ghz-custom/cmd/ghz`  
`chmod +x ./ghz `

#### for publish call  
`./ghz --insecure --config=config.json`  


#### for subscribe call  
`./ghz --insecure --config=config_subscribe.json`  
