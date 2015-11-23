#!/bin/bash

exec 3>&1 4>&2
trap 'exec 2>&4 1>&3' 0 1 2 3
exec 1>/root/log.out 2>&1

# init and create ethereum account
echo "Creating Ethereum account."
cd /root
rm -rf .ethereum .ethash
mkdir .ethereum # to avoid geth y/N question

echo $PASSWD >passwd
geth-bcn --password passwd account new | perl -p -e 's/[{}]//g' | awk '{print $NF}' >accounts

echo "Setting up genesis block."
# create genesis block
cd /root
cat >genesis.json <<EOF
{
    "nonce": "0x0000000000000042",
    "difficulty": "0x000000100",
    "alloc": {},
    "mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "coinbase": "0x0000000000000000000000000000000000000000",
    "timestamp": "0x00",
    "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "gasLimit": "0x5dc6c0"
}
EOF

# set network ID and port
NETWORKID=$((RANDOM * RANDOM))
ETHERBASE=$(cat accounts)

echo "Starting Ethereum."
geth-bcn --fast --shh --verbosity 4 --nodiscover --networkid $NETWORKID --minerthreads 1 --mine --rpc --rpcapi "admin,db,eth,debug,miner,net,shh,txpool,personal,web3" --genesis /root/genesis.json >/tmp/geth.log 2>&1 &

echo "Waiting for miner to mine a block."
BALANCE=0
while ! perl -e "exit($BALANCE == 0)"
do
    sleep 5
    BALANCE=$(geth-bcn --exec 'eth.getBalance(eth.accounts[0])' attach)
done
echo $BALANCE

echo "Unlocking account for bootstrap."
while ! geth-bcn --exec personal.unlockAccount\(\"$ETHERBASE\",\"$PASSWD\"\) attach
do
    sleep 1
done

echo "Bootstrapping MTN smart contracts."
mtn-bootstrap $ETHERBASE >/tmp/bootstrap.log 2>&1
BRC=$?
if [ "$BRC" -ne 0 ]; then
    echo "Bootstrap failed."
    echo "$BRC"
fi

DIRADDR=$(cat directory)

echo "Running directory tests."
mtn-directory_test $DIRADDR $ETHERBASE 30 >/tmp/directory_test.log 2>&1
DRC=$?
if [ "$DRC" -ne 0 ]; then
    echo "Directory tests failed."
    echo "$DRC"
fi

echo "Starting Exchange REST Server."
mtn-gorest $DIRADDR $ETHERBASE >/tmp/restapi.log 2>&1 &

sleep 5

echo "Starting Device simulator."
WHISPERD=$(curl -sL http://localhost:8545 -X POST --data '{"jsonrpc":"2.0","method":"shh_newIdentity","params":[],"id":1}' | jq -r '.result')

echo $WHISPERD

mtn-device_owner $DIRADDR $ETHERBASE $WHISPERD >/tmp/device_owner.log 2>&1 &

echo "Starting Glensung simulator."
WHISPERP=$(curl -sL http://localhost:8545 -X POST --data '{"jsonrpc":"2.0","method":"shh_newIdentity","params":[],"id":1}' | jq -r '.result')

echo $WHISPERP

mtn-rest_container_provider $WHISPERP $ETHERBASE 30 >/tmp/glensung.log 2>&1 &

echo "all done"
while :
do
	sleep 300
done

