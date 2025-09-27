#!/bin/bash
set -e

CURRENT_DIR=${1:-$(pwd)}
GO_BIN_PATH=$(go env GOPATH)/bin
PROTO_DIR="${CURRENT_DIR}/api/proto"

echo "================================"
echo "  Proto Generation Started"
echo "================================"

# Check if proto directory exists
if [ ! -d "$PROTO_DIR" ]; then
    echo "❌ Error: Proto directory not found: $PROTO_DIR"
    exit 1
fi

# Clean and create genproto directory
echo ">> Cleaning genproto directory..."
rm -rf "${CURRENT_DIR}/genproto"
mkdir -p "${CURRENT_DIR}/genproto"

# Find and process all proto directories
for folder in "${PROTO_DIR}"/*; do
    if [ -d "$folder" ]; then
        FOLDER_NAME=$(basename "$folder")
        echo ">> Processing: $FOLDER_NAME"
        
        OUTPUT_DIR="${CURRENT_DIR}/genproto/${FOLDER_NAME}"
        mkdir -p "$OUTPUT_DIR"
        
        # Check if proto files exist
        if ls "$folder"/*.proto 1> /dev/null 2>&1; then
            protoc --plugin=protoc-gen-go="${GO_BIN_PATH}/protoc-gen-go" \
                   --plugin=protoc-gen-go-grpc="${GO_BIN_PATH}/protoc-gen-go-grpc" \
                   -I="$folder" \
                   -I="$PROTO_DIR" \
                   --go_out="$OUTPUT_DIR" \
                   --go_opt=paths=source_relative \
                   --go-grpc_out="$OUTPUT_DIR" \
                   --go-grpc_opt=paths=source_relative \
                   "$folder"/*.proto
            
            echo "   ✓ $FOLDER_NAME generated successfully"
            
            # Remove omitempty
            find "$OUTPUT_DIR" -name "*.go" -type f -exec sed -i 's/,omitempty//g' {} \; 2>/dev/null || true
        else
            echo "   ⚠ No .proto files found in $folder"
        fi
    fi
done

echo "================================"
echo "✅ Proto generation completed!"
echo "================================"

# Show generated files
echo ">> Generated files:"
find "${CURRENT_DIR}/genproto" -type f -name "*.go" 2>/dev/null || echo "No files generated"