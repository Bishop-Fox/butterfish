
gofiles := $(shell find . -name '*.go' -type f -not -path "./vendor/*")
buildtime := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
commit := $(shell bash ./bin/gitcommit.sh)
flags := -X main.BuildVersion=dev -X main.BuildArch=dev -X main.BuildCommit=${commit} -X main.BuildTimestamp=${buildtime} -X main.BuildOs=dev

all: build test

proto/butterfish.pb.go: proto/butterfish.proto
	cd proto && protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative butterfish.proto

bin/butterfish: proto/butterfish.pb.go $(gofiles) Makefile
	mkdir -p bin
	cd go && go build -ldflags "${flags}" -o ../bin/butterfish

clean:
	rm -f bin/butterfish proto/*.go

watch: Makefile
	find . -name "*.go" -o -name "*.proto" -o -name "*.h" -o -name "*.c" | grep -v flan | entr -c make

test: proto/butterfish.pb.go
	cd go && go test ./...

build: bin/butterfish

.PHONY: all clean watch test build

