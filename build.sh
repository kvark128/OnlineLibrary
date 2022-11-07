#!/bin/sh

case "$1" in
	32)
		ARCH=386
		MFLAG="-m32"
		;;
	64)
		ARCH=amd64
		MFLAG="-m64"
		;;
	*)
		echo "Usage: $0 (32|64)"
		exit 1
		;;
esac

BUILD_DIR="$PWD/.build/$ARCH"
INCLUDE_DIR="$BUILD_DIR/include"
LIB_DIR="$BUILD_DIR/lib"

EXTERNAL_DIR="$PWD/external"
SONIC_DIR="$EXTERNAL_DIR/sonic"
MINIMP3_DIR="$EXTERNAL_DIR/minimp3"

gcc $MFLAG -c -O2 -o $SONIC_DIR/sonic.o $SONIC_DIR/sonic.c
ar rcs $SONIC_DIR/libsonic.a $SONIC_DIR/sonic.o
install -D -p $SONIC_DIR/sonic.h $INCLUDE_DIR/sonic.h
install -D -p $SONIC_DIR/libsonic.a $LIB_DIR/libsonic.a
install -D -p $MINIMP3_DIR/minimp3.h $INCLUDE_DIR/minimp3.h
rm $SONIC_DIR/*.o $SONIC_DIR/*.a

export CGO_ENABLED=1
export CGO_CFLAGS=-I$INCLUDE_DIR
export CGO_LDFLAGS=-L$LIB_DIR
export GOOS=windows
export GOARCH=$ARCH

RSRC_FILE="rsrc.syso"
go run cmd/rsrc/rsrc.go -arch $ARCH -manifest OnlineLibrary.exe.manifest -o $RSRC_FILE
go build -tags walk_use_cgo -ldflags "-s -H=windowsgui"
rm $RSRC_FILE
