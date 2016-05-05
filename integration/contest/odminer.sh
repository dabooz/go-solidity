#!/bin/bash

echo "starting on-demand mining"

echo "shutting down the miner to wait for transactions"
DOWN=$(geth --exec "miner.stop()" attach)

# wait for a pending transaction to arrive before starting the miner up again
FAILSAFE=0
while :
do
    PT1=$(geth --exec "web3.eth.getBlock('pending').transactions" attach)
    PT2=$(geth --exec "web3.eth.pendingTransactions" attach)

    if [[ ("$PT1" != "[]") || ("$PT2" != "null") || ("$FAILSAFE" -eq 24) ]]
    then
        echo "starting miner"
        DO_IT=$(geth --exec "miner.start(1);" attach)
        echo "started miner"
        while [[ ("$PT1" != "[]") || ("$PT2" != "null") || ("$FAILSAFE" -eq 24) ]]
        do
            #echo "blocking for 1 block"
            WB=$(geth --exec "admin.sleepBlocks(3);" attach)
            echo "checking for more transactions"
            FAILSAFE=0
            PT1=$(geth --exec "web3.eth.getBlock('pending').transactions" attach)
            PT2=$(geth --exec "web3.eth.pendingTransactions" attach)
        done
        echo "stopping miner"
        THE_END=$(geth --exec "miner.stop()" attach)
    fi
    sleep 5
    ((FAILSAFE++))
done
