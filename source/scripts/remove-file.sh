#!/bin/bash

target_folder=$1


# remove.sh

for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  # 确保子文件夹是一个目录
        nodeName=$(basename $subfolder)
        rm -r $subfolder/database
        rm $subfolder/$nodeName-blocklogger-brief.txt
        rm $subfolder/$nodeName-blocklogger.txt
        mkdir $subfolder/database
    fi
done