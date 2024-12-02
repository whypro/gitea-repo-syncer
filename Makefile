APP=gitea-repo-syncer

all: fmt vendor build

build:
	go build -o output/bin/${APP} .

run:
	. ./env.sh && ./output/bin/${APP} > log
.PHONY: run

fmt:
	go fmt .
.PHONY: fmt

vendor:
	go mod tidy
	go mod download
.PHONY: vendor

docker-fmt:
	docker run -it -v ${PWD}:/go/src/github.com/whypro/${APP} -w /go/src/github.com/whypro/${APP} golang:1.22 go fmt .

docker-build:
	docker run -it -v ${PWD}:/go/src/github.com/whypro/${APP} -w /go/src/github.com/whypro/${APP} golang:1.22 go build -o output/bin/${APP} .
