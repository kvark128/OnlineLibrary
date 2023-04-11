#!/bin/sh

CC=gcc
ARCH=`go env GOARCH`

case "$ARCH" in
	amd64)
		CFLAGS="-Werror -m64 -O2"
		;;
	386)
		CFLAGS="-Werror -m32 -O2"
		;;
	*)
		echo "Arch $ARCH not supported"
		exit 1
		;;
esac

BUILD_DIR="$PWD/.build/$ARCH"
INCLUDE_DIR="$BUILD_DIR/include"
LIB_DIR="$BUILD_DIR/lib"

EXTERNAL_DIR="$PWD/external"
SONIC_DIR="$EXTERNAL_DIR/sonic"
MINIMP3_DIR="$EXTERNAL_DIR/minimp3"

$CC $CFLAGS -c -o $SONIC_DIR/sonic.o $SONIC_DIR/sonic.c
ar rcs $SONIC_DIR/libsonic.a $SONIC_DIR/sonic.o
install -D -p $SONIC_DIR/sonic.h $INCLUDE_DIR/sonic.h
install -D -p $SONIC_DIR/libsonic.a $LIB_DIR/libsonic.a
install -D -p $MINIMP3_DIR/minimp3.h $INCLUDE_DIR/minimp3.h
rm $SONIC_DIR/*.o $SONIC_DIR/*.a

export CGO_ENABLED=1
export CGO_CFLAGS="-I$INCLUDE_DIR $CFLAGS"
export CGO_LDFLAGS=-L$LIB_DIR
export GOOS=windows

RSRC_FILE="rsrc.syso"
go run cmd/rsrc/rsrc.go -arch $ARCH -manifest OnlineLibrary.exe.manifest -o $RSRC_FILE
go build -tags walk_use_cgo -ldflags "-s -H=windowsgui"
rm -f $RSRC_FILE
