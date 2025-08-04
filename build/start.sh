#!/bin/bash

target_folder=./mytestnet/192.168.0.4
start_time=17:36

#start.sh

for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  
        (./urd --root="$subfolder" --start-time="$start_time" > "$subfolder"/.out 2>&1 &)
    fi
done