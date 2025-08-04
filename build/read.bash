#!/bin/bash

# 读取用户输入的数字a
read -p "请输入间隔数字a: " a

# 循环遍历并操作相应文件
for i in {0..9};  # 这里假设最多到9，可以根据实际情况修改范围
do
    num=$((a * i + 1))
    file="./mytestnet/192.168.0.4/node${num}/node${num}-blocklogger-brief.txt"
    if [ -f "$file" ]; then
       ./logger $file
    else
        echo "文件 $file 不存在"
    fi
done