# Makefile for OnlineLibrary
#
# Copyright (C) 2024 Alexander Linkov <kvark128@yandex.ru>

ARCH = amd64
CC = x86_64-w64-mingw32-gcc
CFLAGS = -Werror -Wno-unused-result -O2
MANIFEST_FILE = OnlineLibrary.exe.manifest
SYSO_FILE = rsrc_windows_$(ARCH).syso

ifeq ($(ARCH), 386)
CC = i686-w64-mingw32-gcc
endif

BUILD_DIR = .build/$(ARCH)
INCLUDE_DIR = $(BUILD_DIR)/include
LIB_DIR = $(BUILD_DIR)/lib
SONIC_DIR = external/sonic
MINIMP3_DIR = external/minimp3
VPATH = $(SONIC_DIR) $(MINIMP3_DIR) $(BUILD_DIR) $(INCLUDE_DIR) $(LIB_DIR)

.SILENT: main
main: libs headers $(SYSO_FILE)
	env GOOS=windows GOARCH=$(ARCH) CGO_ENABLED=1 CC=$(CC) CGO_CFLAGS="-I$(shell pwd)/$(INCLUDE_DIR) $(CFLAGS)" CGO_LDFLAGS="-L$(LIB_DIR)" \
	go build -tags walk_use_cgo -ldflags "-s -H=windowsgui"

.SILENT: clean
clean:
	rm -r -f ./.build
	rm -f *.exe *.syso

$(SYSO_FILE): $(MANIFEST_FILE) $(wildcard cmd/rsrc/*.go) $(wildcard internal/config/*.go)
	go run cmd/rsrc/rsrc.go -arch $(ARCH) -manifest $(MANIFEST_FILE) -o $@

$(LIB_DIR)/libsonic.a: sonic.o
	mkdir -p $(LIB_DIR)
	ar rcs $@ $^

$(BUILD_DIR)/sonic.o: sonic.c sonic.h
	mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -c -o $@ $<

$(INCLUDE_DIR)/sonic.h:
	install -D -p $(SONIC_DIR)/sonic.h $@

$(INCLUDE_DIR)/minimp3.h:
	install -D -p $(MINIMP3_DIR)/minimp3.h $@

headers: $(INCLUDE_DIR)/sonic.h $(INCLUDE_DIR)/minimp3.h
libs: libsonic.a
