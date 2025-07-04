@echo off
mkdir bin 2>nul

echo 编译 macOS (Intel)...
set GOOS=darwin
set GOARCH=amd64
go build -o bin\emcm-macos-intel

echo 编译 macOS (Apple Silicon)...
set GOOS=darwin
set GOARCH=arm64
go build -o bin\emcm-macos-m1

echo 编译 Linux (64-bit)...
set GOOS=linux
set GOARCH=amd64
go build -o bin\emcm-linux

echo 编译 Linux (ARM64)...
set GOOS=linux
set GOARCH=arm64
go build -o bin\emcm-linux-arm

echo 编译 Windows (64-bit)...
set GOOS=windows
set GOARCH=amd64
go build -o bin\emcm-windows.exe

echo 编译 Windows (ARM64)...
set GOOS=windows
set GOARCH=arm64
go build -o bin\emcm-windows-arm.exe

echo 编译完成！可在 bin 目录找到所有平台的可执行文件
pause