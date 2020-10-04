@echo off
rsrc -arch amd64 -manifest OnlineLibrary.exe.manifest -o rsrc.syso
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1
go build -tags walk_use_cgo -ldflags "-H=windowsgui"
del rsrc.syso
