#!/bin/bash

if [[ ../../contracts/container_executor.sol -nt ../../contracts/container_executor.json ]]; then
    exit 1
fi

if [[ ../../contracts/device_registry.sol -nt ../../contracts/device_registry.json ]]; then
    exit 1
fi

if [[ ../../contracts/directory.sol -nt ../../contracts/directory.json ]]; then
    exit 1
fi

if [[ ../../contracts/token_bank.sol -nt ../../contracts/token_bank.json ]]; then
    exit 1
fi

if [[ ../../contracts/whisper_directory.sol -nt ../../contracts/whisper_directory.json ]]; then
    exit 1
fi

if [[ ../../contracts/agreements.sol -nt ../../contracts/agreements.json ]]; then
    exit 1
fi

exit 0

