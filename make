#!/bin/bash

set -e

export GOPATH=${HOME}/code/go/src/dji-joe:${HOME}/code/go
export GOOS=linux

arch=$(arch)
test $# -eq 1 && arch="$1"

case "${arch}" in
    arm)
        echo "Building for '${arch}'"
        outfile="bin/dji-joe-armv7l"
	GOARCH=arm GOARM=7 CGO_ENABLED=1 \
              CGO_LDFLAGS+="-g -O2 -L$(pwd)/misc/libs/arm -lpcap" \
              CC=arm-linux-gnueabi-gcc CXX=arm-linux-gnueabi-g++ \
              go build -o ${outfile} src/main/main.go
        ;;

    i386|x86)
        echo "Building for '${arch}'"
        outfile="bin/dji-joe-${arch}"
        CGO_ENABLED=1 \
                   CGO_LDFLAGS+="-g -O2 -L$(pwd)/misc/libs/x86 -lpcap" \
                   CC=clang \
                   GOARCH=i386 go build -o ${outfile} src/main/main.go
        ;;

    x86_64|*)
        echo "Building for '${arch}'"
        outfile="bin/dji-joe-${arch}"
	go build -o ${outfile} src/main/main.go
        ;;
esac

echo "Built as '${outfile}'"
