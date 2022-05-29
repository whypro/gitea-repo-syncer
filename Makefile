APP=gitea-repo-syncer

all: fmt vendor build

build:
	go build -o output/bin/${APP} cmd/main.go

run:
	. ./env.sh && ./output/bin/${APP}
.PHONY: run

fmt:
	go fmt ./cmd/... ./pkg/...
.PHONY: fmt

vendor:
	go mod tidy
	go mod vendor
.PHONY: vendor

docker-fmt:
	docker run -it -v ${PWD}:/go/src/github.com/whypro/${APP} -w /go/src/github.com/whypro/${APP} golang:1.18 go fmt ./cmd/... ./pkg/...

docker-build:
	docker run -it -v ${PWD}:/go/src/github.com/whypro/${APP} -w /go/src/github.com/whypro/${APP} golang:1.18 go build -o output/bin/${APP} cmd/main.go
