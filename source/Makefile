
TARGET_DIR := "../build"


proto-gen:
	@ echo "Generate .pb.go files for all .proto files in ./proto..."
	@ find ./proto -type f -name "*.proto" -exec protoc --proto_path=. --go_out=. --go_opt=paths=source_relative {} \;
	@ echo "Generation has finished!"
.PHONY: proto-gen


proto-delete:
	@ echo "Remove all .pb.go files in ./proto..."
	@ find ./proto -type f -name "*.pb.go" -delete
	@ echo "Removal has finished"
.PHONY: proto-delete

build-urd:
	@ echo "Building urd..."
	@ cd urd/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv urd/main/main $(TARGET_DIR)/urd
.PHONY: build-urd

build-urd-latancy:
	@ echo "Building urd latency..."
	@ cd urd/abci/minibank/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv urd/abci/minibank/main/main $(TARGET_DIR)/urd-latency
.PHONY: build-urd-latancy


build-block-logger:
	@ echo "Building block-logger..."
	@ cd logger/blocklogger/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv logger/blocklogger/main/main $(TARGET_DIR)/logger
.PHONY: build-block-logger


build-pyramid:
	@ echo "Building pyramid..."
	@ cd pyramid/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv pyramid/main/main $(TARGET_DIR)/pyramid
.PHONY: build-pyramid

build-pyramid-latancy:
	@ echo "Building pyramid latency..."
	@ cd pyramid/abci/minibank/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv pyramid/abci/minibank/main/main $(TARGET_DIR)/pyramid-latency
.PHONY: build-pyramid-latancy

build-store:
	@ echo "Building store..."
	@ cd utils/store/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv utils/store/main/main $(TARGET_DIR)/store
.PHONY: build-store


build-all: 
	make build-block-logger
	make build-store
	make build-urd
	make build-urd-latancy
.PHONY: build-all