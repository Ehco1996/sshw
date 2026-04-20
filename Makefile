BINARY := sshw
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.Build=$(VERSION)

.PHONY: build clean install lint test

build:
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/sshw

install:
	CGO_ENABLED=0 go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/sshw

clean:
	rm -f $(BINARY)

lint:
	golangci-lint run ./...

test:
	go test -count=1 ./...
