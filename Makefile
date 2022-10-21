#srcDir=$(shell pwd)

all: clean deps build compress

wan: clean deps build-wan compress
lan: clean deps build-lan compress

clean:
	-rm -rf $(dir $(abspath $(lastword $(MAKEFILE_LIST))))target/

deps:
	go mod tidy

build: build-wan build-lan

build-wan:
	set GOOS=linux;GOARCH=amd64
	go build -ldflags "-s -w" -o target/xxqg_server cmd/wan/main.go

build-lan:
		set GOOS=windows;GOARCH=amd64
		go build -ldflags "-s -w -X main.VERSION=v1.0.1" -o target/xxqg_agent.exe cmd/lan/main.go

compress:
	-upx --lzma target/*