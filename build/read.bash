#!/bin/bash


read -p "Type the shard size as the read interval: " a

 # We assume that there at most 11 shards, and it can be changed to be larger
 # In fact, even if there are fewer than 11 shards, say 5 shards, the shard number 11 works, for shard 6-11 will be ignored.
for i in {0..11}; 
do
    num=$((a * i + 1))
    file="./mytestnet40-5s/192.168.0.4/node${num}/node${num}-blocklogger-brief.txt"
    if [ -f "$file" ]; then
        # Use the executable file logger to calculate TPS from log files.
       ./logger $file
    else
        echo "File $file doesn't exist"
    fi
done