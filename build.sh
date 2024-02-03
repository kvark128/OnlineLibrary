#!/bin/sh

ARCH=`go env GOARCH`

case "$ARCH" in
	amd64)
		CC=x86_64-w64-mingw32-gcc
		CFLAGS="-Werror -O2"
		AR=x86_64-w64-mingw32-ar
		;;
	386)
		CC=i686-w64-mingw32-gcc
		CFLAGS="-Werror -O2"
		AR=i686-w64-mingw32-ar
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

mkdir -p $BUILD_DIR
mkdir -p $INCLUDE_DIR
mkdir -p $LIB_DIR

$CC $CFLAGS -c -o $BUILD_DIR/sonic.o $SONIC_DIR/sonic.c
$AR rcs $LIB_DIR/libsonic.a $BUILD_DIR/sonic.o
install -D -p $SONIC_DIR/sonic.h $INCLUDE_DIR/sonic.h
install -D -p $MINIMP3_DIR/minimp3.h $INCLUDE_DIR/minimp3.h

go run cmd/rsrc/rsrc.go -arch $ARCH -manifest OnlineLibrary.exe.manifest -o "rsrc_windows_$ARCH.syso"
env GOOS=windows CGO_ENABLED=1 CC=$CC CGO_CFLAGS="-I$INCLUDE_DIR $CFLAGS" CGO_LDFLAGS="-L$LIB_DIR -static" \
	go build -tags walk_use_cgo -ldflags "-s -H=windowsgui"
