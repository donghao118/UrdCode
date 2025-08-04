#### To start our model:

1. Generate shard config file in ***cross_shard_config.json***. Please write all the available IPs in the `ip_in_use` field and assign a proportion to each of them. For example, if you want to allocate two cores to each node, you can assign a proportion of 2 to the four-core server with the IP address 192.168.200.11. Please fill in the configuration information of each shard in the `shards` field, including: `chain_id`, the name of the shard; `peer_num`, the number of shard nodes; `related_shards`, all related shards of this shard; `is_ishard`, whether the shard is an I shard; `key_range`, the primary key range of the shard.

**<span style="color:brown;">We have provided a batch of pre-written JSON files in the scripts folder, each containing different numbers of shards, numbers of shard nodes, and w parameters. For example, `11*40*0.3.json` indicates that this is a configuration file for 11 shards, where each shard contains 40 nodes and w = 0.3.</span>**

2. Generate configuration files for each node by `./ours --method=generate --config=../scripts/11*40*0.3.json --root=./mytestnet`. You can replace the `config` field with any other JSON configuration file, and replace the `root` field with any output folder path you want (if the folder itself does not exist, a folder will be created).

3. After generating all files, you can find some directionaries named by IP addresses in your root directionary. Distribute these folders to their corresponding servers, and you can specify any location.

4. You can use such sh file to start all nodes in the server. The first input parameter `target_folder` represents the path of the directory named with the IP address that you deploy on this server, and the second input parameter `start_time` indicates the time you want the experiment to start. All nodes will not start until this time point, which makes it convenient for you to start all nodes. To start all nodes at 18:00, you can use the command `bash start.sh./192.168.200.11 18:00`.

```
#!/bin/bash
# This is start.sh
target_folder=$1
start_time=$2

for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  
        (./ours --root="$subfolder" --start-time="$start_time" > "$subfolder"/.out 2>&1 &)
    fi
done
```

5. You should wait for seconds until the experiment finishes. You can use command `killall ours` to stop nodes in this server.

6. In the experiment, nodes will record log files through blocklogger and store them in a file path like `./192.168.200.11/node1/node1-blocklogger-brief.txt`. You can download these files and obtain the throughput and average abort rate for each shard by using the command `./logger./192.168.200.11/node1/node1-blocklogger-brief.txt`. In the test load, we assign the same number of cross-shard transactions to each shard, so the abort rate of cross-shard transactions can be calculated by averaging. For the abort rate of all transactions, it is calculated by weighted summation using the proportion of cross-shard transactions.

7. To calculate the latency, you can use `./ours-latency ./192.168.200.11/node1/ b1`, where the first parameter is the root directionary of node, and the second parameter is the shard which the node belongs to.

8. You maynot wish to deploy all files whenever starting an experinment. You can use `bash remove.sh` to remove database and log files. The first input parameter `target_folder` represents the path of the directory named with the IP address that you deploy on this server.

```
#!/bin/bash

# remove.sh

target_folder=$1

for subfolder in "$target_folder"/*; do
    if [ -d "$subfolder" ]; then  # 确保子文件夹是一个目录
        nodeName=$(basename $subfolder)
        rm -r $subfolder/database
        rm $subfolder/$nodeName-blocklogger-brief.txt
        rm $subfolder/$nodeName-blocklogger.txt
        mkdir $subfolder/database
    fi
done
```

9. For different experiments, you can modify `ours/main/start.go`. Specifically, the `transferSize` parameter specifies the step size, the `mustLen` parameter specifies the number of bytes of the transaction, the `accountNumTotal` parameter specifies the total number of virtual accounts in the system, the `commit_rates` specifies the total load of the system, the `cross_shard_rates` specifies the proportion of cross-shard transactions in the system, and the `txTotals` specifies the size of the experimental set. Then you should use `make build-all` to reconstruct the executable files.


#### To start pyramid

Pyramid and Uranus use the same configuration file; the only difference is that the startup command of pyramid is stored in `pyramid/main/start.go`, its executable program is named `pyramid`, and it uses `pyramid-latency` to calculate the latency. You can start pyramid using the same method as Uranus, and test the throughput, latency, and abort rate.

#### To start rapid

We implemented the transfer mechanism of Rapid based on Pyramid. You can use JSON files like `./scripts/rapid11.json` to generate configuration files for RapidChain.

The executable file is named `rapid`, and `rapid-latency` is used to calculate the latency. You can modify the system of RapidChain by `rapid/main/start.go` as well.














