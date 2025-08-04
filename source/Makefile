
TARGET_DIR := "../build"


proto-gen:
	@ echo "为./proto中所有.proto文件生成.pb.go文件"
	@ find ./proto -type f -name "*.proto" -exec protoc --proto_path=. --go_out=. --go_opt=paths=source_relative {} \;
	@ echo "生成结束"
.PHONY: proto-gen


proto-delete:
	@ echo "删除./proto中所有.pb.go文件"
	@ find ./proto -type f -name "*.pb.go" -delete
	@ echo "删除结束"
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

build-rapid:
	@ echo "Building rapid..."
	@ cd rapid/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv rapid/main/main $(TARGET_DIR)/rapid
.PHONY: build-rapid

build-rapid-latancy:
	@ echo "Building rapid latency..."
	@ cd rapid/abci/minibank/main && go build
	@ echo "move to $(TARGET_DIR)"
	@ mv rapid/abci/minibank/main/main $(TARGET_DIR)/rapid-latency
.PHONY: build-rapid-latancy

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