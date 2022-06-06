@echo off
go run cmd/rsrc/rsrc.go -arch 386 -manifest OnlineLibrary.exe.manifest -o rsrc.syso
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=1
go build -tags walk_use_cgo -ldflags "-s -H=windowsgui"
del rsrc.syso
