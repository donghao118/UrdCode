#  Urd: A Pipelined Blockchain Sharding Protocol with Cooperative Cross-Shard Verification

## Introduction

This repository contains the source code of the Urd prototype system and implementations of other consensus protocols for comparison.

## Folder structure

```sh
├── build # the output binaries file
└── source # source code of system
```

## Prerequisites

The code is based on golang and the test is running on Linux. There are a few dependencies to run the code. The major libraries are listed as follows:

* rocksdb
* proto
* golang

### Install rocksdb (need for both test and build)

1. preparing the environment.

   ```bash
   sudo apt install build-essential
   sudo apt-get install libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev libgflags-dev
   ```

2. build rocksdb lib from source.

   ```bash
   wget https://github.com/facebook/rocksdb/archive/v6.28.2.zip
   
   unzip v6.28.2.zip && cd rocksdb-6.28.2
   
   make shared_lib
   make static_lib
   ```

3. install lib to system path.

   ```bash
   sudo make install-shared
   sudo make install-static
   
   echo "/usr/local/lib" | sudo tee /etc/ld.so.conf.d/rocksdb-x86_64.conf
   sudo ldconfig -v
   ```

4. verify whether rocksdb is installed

   ```bash
   # You can use source/rocks_test.cpp or the executable file source/rocks_test to verify whether rocksdb is installed successfully.
   g++ -std=c++11 -o rocks_test rocks_test.cpp -lrocksdb  -lpthread -ldl
   ./rocks_test
   # There will be 'get bar success!!' showed on your terminal if nothing is wrong.
   ```

### Install golang (need for build)

Follow the offical manual [https://go.dev/doc/install](https://go.dev/doc/install).

### Install proto-go (necessary only if you want to modify the proto files)

1. `protoc` is already provided and placed in the `source/proto/install` directory. Move it to your "gobin" directory.
2. In order to compile the code, you need to install `protoc-gen-go` and `protoc-gen-go-grpc`.

    ```sh
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest 
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    ```
3. For more details of how to use proto-go, refer to the [Protobuf Installation and Usage Guide](./source/proto/README.md).
## How to Build

Install all prerequisites, then enter the `source` folder and run `make build-all`, all the binaries will be palced in the `build` folder.

## How to Start

For protocol details and the way to deploy Urd, refer to the [Startup Guidelines](./source/start.md)
