#protoc --proto_path=./proto --go_out=./proto --go_opt=paths=source_relative ./proto/ours/abci/*.proto

#protoc --proto_path=./proto --go_out=./proto --go_opt=paths=source_relative ./proto/pyramid/abci/*.proto

find ./proto -type f -name "*.proto" -exec protoc --proto_path=./proto --go_out=./proto --go_opt=paths=source_relative {} \;