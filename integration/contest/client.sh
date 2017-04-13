#!/bin/bash

# exec 3>&1 4>&2
# trap 'exec 2>&4 1>&3' 0 1 2 3
# exec 1>/root/log.out 2>&1

# init and create ethereum account
echo "Creating Ethereum account."
cd /root
rm -rf .ethereum .ethash
mkdir .ethereum # to avoid geth y/N question

# get a prebuilt DAG that ethereum needs for mining to avoid 7 mins of dynamic generation time.
echo "Move the DAG into place if there is one."
mkdir .ethash
cd .ethash
mv /tmp/full-R23-0000000000000000 . 2>/dev/null || :
touch full-R23-0000000000000000
cd ..

echo $PASSWD >passwd
geth --password passwd account new | perl -p -e 's/[{}]//g' | awk '{print $NF}' >accounts

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
    "gasLimit": "0xadc6c0"
}
EOF

# set network ID and port
NETWORKID=$((RANDOM * RANDOM))
ETHERBASE=$(cat accounts)

echo "Starting Ethereum."
geth init /root/genesis.json
# --verbosity 6
geth --lightkdf --fast --shh --nodiscover --networkid $NETWORKID --minerthreads 1 --mine --rpc --rpcapi "admin,db,eth,debug,miner,net,shh,txpool,personal,web3" >/tmp/geth.log 2>&1 &

echo "Waiting for miner to mine a block."
BALANCE=0
while ! perl -e "exit($BALANCE == 0)"
do
    sleep 5
    BALANCE=$(geth --exec 'eth.getBalance(eth.accounts[0])' attach)
done
echo $BALANCE

# Mining is running. The on-demand miner will shut it down and then look for pending transactions.
#echo "Starting on-demand miner."
#MS=$(geth --exec "miner.stop()" attach)
#./odminer.sh >/tmp/odminer.log 2>&1 &

echo "Unlocking account for bootstrap."
while ! geth --exec personal.unlockAccount\(\"$ETHERBASE\",\"$PASSWD\",0\) attach
do
    sleep 1
done

echo "Bootstrapping Horizon V2 smart contracts."
mtn-bootstrapv2 $ETHERBASE >/tmp/bootstrap.log 2>&1
BRC=$?
if [ "$BRC" -ne 0 ]; then
    echo "Bootstrap failed."
    echo "$BRC"
fi

DIRADDR=$(cat directory)

echo "Delay so the blockchain can catch up"
sleep 30

export CMTN_DIRECTORY_VERSION=999

echo "Bootstrapping Horizon V2 smart contracts again."
mtn-bootstrapv2 $ETHERBASE $DIRADDR >/tmp/bootstrap2.log 2>&1
BRC=$?
if [ "$BRC" -ne 0 ]; then
    echo "Bootstrap failed."
    echo "$BRC"
fi

echo "Delay so the blockchain can catch up"
sleep 30

unset CMTN_DIRECTORY_VERSION

echo "Running agreement protocol tests."
mtn-agreement_protocol_test $DIRADDR $ETHERBASE >/tmp/agreement_protocol_test.log 2>&1
DRC=$?
if [ "$DRC" -ne 0 ]; then
    echo "Agreement protocol tests failed."
    echo "$DRC"
fi

echo "Running metering tests."
mtn-metering_test $DIRADDR $ETHERBASE 30 >/tmp/metering_test.log 2>&1
MRC=$?
if [ "$MRC" -ne 0 ]; then
    echo "Metering tests failed."
    echo "$MRC"
fi

echo "Running directory tests."
mtn-directory_test $DIRADDR $ETHERBASE 30 >/tmp/directory_test.log 2>&1
DRC=$?
if [ "$DRC" -ne 0 ]; then
    echo "Directory tests failed."
    echo "$DRC"
fi

echo "all done"
while :
do
	sleep 300
done

