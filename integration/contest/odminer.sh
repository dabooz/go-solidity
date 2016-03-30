#!/bin/bash

echo "starting on-demand mining"

echo "shutting down the miner to wait for transactions"
DOWN=$(geth --exec "miner.stop()" attach)

# wait for a pending transaction to arrive before starting the miner up again
FAILSAFE=0
while :
do
    PT=$(geth --exec "web3.eth.getBlock('pending').transactions" attach)
    # echo $PT
    if [[ ("$PT" != "[]") || ("$FAILSAFE" -eq 24) ]]
    then
        echo "starting miner"
        DO_IT=$(geth --exec "miner.start(1);" attach)
        #echo "started miner"
        while [[ ("$PT" != "[]") || ("$FAILSAFE" -eq 24) ]]
        do
            #echo "blocking for block"
            WB=$(geth --exec "admin.sleepBlocks(1);" attach)
            echo "checking for more transactions"
            FAILSAFE=0
            PT=$(geth --exec "web3.eth.getBlock('pending').transactions" attach)
            # echo $PT
        done
        echo "stopping miner"
        THE_END=$(geth --exec "miner.stop()" attach)
    fi
    sleep 5
    ((FAILSAFE++))
done


