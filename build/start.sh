#!/bin/bash
#start.sh
killall pyramid
sleep   5
target_folder=./mytestnet-scale380p/192.168.0.4
start_time=15:34
enable_pipeline=false
bash remove-file.sh $target_folder
for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  
        # (./urd --root="$subfolder" --start-time="$start_time" --enable-pipeline=$enable_pipeline > "$subfolder"/.out 2>&1 &)
        (./pyramid --root="$subfolder" --start-time="$start_time" > "$subfolder"/.out 2>&1 &)
    fi
done

echo sleep

sleep 330
bash read.bash   $target_folder


killall pyramid