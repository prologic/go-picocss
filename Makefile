.PHONY: dev build install image release test clean

CGO_ENABLED=0
COMMIT=$(shell git rev-parse --short HEAD)

all: dev

dev: build
	@./go-picocss -d

build: generate
	@go build -tags "netgo static_build" -installsuffix netgo \
		-ldflags "-w -X $(shell go list)/.GitCommit=$(COMMIT)" \
		.

generate:
	@rice embed-go

install: build
	@go install

image:
	@docker build -t prologic/go-picocss .

release:
	@./tools/release.sh

test:
	@go test -v -cover -race .

clean:
	@git clean -f -d -X
