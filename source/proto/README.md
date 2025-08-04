# Protobuf Installation and Usage Guide (Linux)

## Installation Steps

1. **Install protoc compiler**
   - Copy the `protoc` executable from `source/proto/install` folder to your user bin directory
   - Set execution permissions: `chmod a+x protoc`
   - Verify installation: `protoc --version` (successful installation will display version number)

   *For Windows users*:
   - Visit [Protobuf releases page](https://github.com/protocolbuffers/protobuf/releases/tag/v26.1) to download appropriate version
   - Extract and locate the executable in the bin folder, then add to PATH environment variable
   - Current employed version is v26.1

2. **Install Go plugins**
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

## Project Usage

* **Generate Go proto files**: `make proto-gen` (execute from project source directory)

* **Clean generated files**(all files will be deleted while generating new files): `make proto-delete`

## Proto File Writing Specifications

When generating proto files, the directory structure should match the package hierarchy. Take source/proto/urd/types/block.proto as a example:

```text
source/
├── proto/
│   ├── urd/
│   │   ├── types/
│   │   │   ├── block.proto
```


1. go_package option
    * **Format**: `emulator/proto/relative_file_path`
    * **Example**: For source/proto/urd/types/block.proto, use `emulator/proto/urd/types`

2. package declaration
    * Use the file's relative path as namespace to avoid bugs 
    * **Example**: For source/proto/urd/types/block.proto, declare as package `urd.types`;

3. import statements
    * Import other proto files using relative paths. 
    * **Example**: If you want to import source/proto/urd/types/block.proto in a new file, use `import "urd/types/block.proto"`;
    * Message types in the same namespace (same folder) can be used directly
    * Cross-namespace message types must be referenced with full namespace
    * **Example**: The data structure `Block` from above file should be referenced as `urd.types.Block`
