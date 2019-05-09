# mars-voting-chaincode

This chaincode is used to capture votes of users. The identity of the voter will be verified using the "identity" chaincode. This chaincode will be deployed on the "voting" channel by the voting authority.

## Install and Instantiate 

First ssh into the EC2 that's running the container. Then access to shell of the container of voting authority using this command: `docker exec -i -t container_id /bin/bash`. 

Then follow this steps to install and instantiate the chaincode:

1. Clone the chaincode repo using the command `cd /opt/gopath/src/github.com && git clone https://github.com/narayanprusty/mars-voting-chaincode.git`
2. Install using this command: `peer chaincode install -n voting -v v1.0 -p github.com/mars-voting-chaincode`
3. Command to instantiate the chaincode: `peer chaincode instantiate -n voting -v 1.0 -c '{"Args":[]}' -C voting`
