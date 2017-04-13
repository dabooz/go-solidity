# bootstrap make

default: directory.json agreements.json metering.json

directory.json: directory.sol
	~/code/ethbuilds/container/solc/solc --optimize --combined-json abi,bin directory.sol | jq -r '.' >directory.json

agreements.json: agreements.sol
	~/code/ethbuilds/container/solc/solc --optimize --combined-json abi,bin agreements.sol | jq -r '.' >agreements.json

metering.json: metering.sol
	~/code/ethbuilds/container/solc/solc --optimize --combined-json abi,bin metering.sol | jq -r '.' >metering.json

clean:
	-rm -f directory.json
	-rm -f agreements.json
	-rm -f metering.json
