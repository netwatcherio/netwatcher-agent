#!/bin/bash

# Set the name of your application
APP_NAME="netwatcher-agent"

# Set the path of your main.go file
MAIN_PATH="./"

# Create a bin directory
mkdir -p bin

# Build for macOS (amd64 and arm64)
GOOS=darwin GOARCH=amd64 go build -o bin/${APP_NAME}-darwin-amd64 ${MAIN_PATH}
GOOS=darwin GOARCH=arm64 go build -o bin/${APP_NAME}-darwin-arm64 ${MAIN_PATH}

# Build for Linux (amd64 and arm64)
GOOS=linux GOARCH=amd64 go build -o bin/${APP_NAME}-linux-amd64 ${MAIN_PATH}
GOOS=linux GOARCH=arm64 go build -o bin/${APP_NAME}-linux-arm64 ${MAIN_PATH}

# Build for Windows (amd64 and 386)
GOOS=windows GOARCH=amd64 go build -o bin/${APP_NAME}-windows-amd64.exe ${MAIN_PATH}
GOOS=windows GOARCH=386 go build -o bin/${APP_NAME}-windows-386.exe ${MAIN_PATH}

# Create ZIP archives for each release
cd bin
for file in *; do
    if [ -f "$file" ]; then
        zip "${file}.zip" "$file"
        rm "$file"
    fi
done
cd ..

echo "Build complete. Release files are in the 'bin' directory."