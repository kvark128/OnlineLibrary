#!/bin/sh

ARCH=386
BUILD_DIR="$PWD/.build/$ARCH"
INCLUDE_DIR="$BUILD_DIR/include"
LIB_DIR="$BUILD_DIR/lib"

EXTERNAL_DIR="$PWD/external"
SONIC_DIR="$EXTERNAL_DIR/sonic"
MINIMP3_DIR="$EXTERNAL_DIR/minimp3"

gcc -m32 -c -O2 -o $SONIC_DIR/sonic.o $SONIC_DIR/sonic.c
ar rcs $SONIC_DIR/libsonic.a $SONIC_DIR/sonic.o
install -D -p $SONIC_DIR/sonic.h $INCLUDE_DIR/sonic.h
install -D -p $SONIC_DIR/libsonic.a $LIB_DIR/libsonic.a
install -D -p $MINIMP3_DIR/minimp3.h $INCLUDE_DIR/minimp3.h

export CGO_ENABLED=1
export CGO_CFLAGS=-I$INCLUDE_DIR
export CGO_LDFLAGS=-L$LIB_DIR
export GOOS=windows
export GOARCH=$ARCH

go run cmd/rsrc/rsrc.go -arch $ARCH -manifest OnlineLibrary.exe.manifest -o rsrc.syso
go build -tags walk_use_cgo -ldflags "-s -H=windowsgui"
