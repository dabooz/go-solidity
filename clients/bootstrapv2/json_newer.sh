#!/bin/bash

if [[ ../../contracts/directory.sol -nt ../../contracts/directory.json ]]; then
    exit 1
fi

if [[ ../../contracts/agreements.sol -nt ../../contracts/agreements.json ]]; then
    exit 1
fi

exit 0

