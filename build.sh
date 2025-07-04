#!/bin/bash

mkdir -p bin

echo "编译 macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -o bin/emcm-macos-intel

echo "编译 macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -o bin/emcm-macos-m1

echo "编译 Linux (64-bit)..."
GOOS=linux GOARCH=amd64 go build -o bin/emcm-linux

echo "编译 Linux (ARM64)..."
GOOS=linux GOARCH=arm64 go build -o bin/emcm-linux-arm

echo "编译 Windows (64-bit)..."
GOOS=windows GOARCH=amd64 go build -o bin/emcm-windows.exe

echo "编译 Windows (ARM64)..."
GOOS=windows GOARCH=arm64 go build -o bin/emcm-windows-arm.exe

echo "编译完成！可在 bin 目录找到所有平台的可执行文件"