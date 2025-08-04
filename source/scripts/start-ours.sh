#!/bin/bash

target_folder=$1
start_time=$2

#start.sh

for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  
        (./ours --root="$subfolder" --start-time="$start_time" > "$subfolder"/.out 2>&1 &)
    fi
done