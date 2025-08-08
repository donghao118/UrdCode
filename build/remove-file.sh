#!/bin/bash

target_folder=$1


# remove.sh

for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  
        nodeName=$(basename $subfolder)
        # echo "remove $subfolder" 
        rm -r $subfolder/database
        rm $subfolder/$nodeName-blocklogger-brief.txt
        rm $subfolder/$nodeName-blocklogger.txt
        mkdir $subfolder/database
    fi
done