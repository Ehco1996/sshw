.PHONY: fmt test build tidy ensure release

fmt:
	golangci-lint run --fix

test:
	go test -count=1  -coverpkg=./internal -timeout=10s ./...

build:
	go build -o .dist/sshw cmd/sshw/main.go

tidy:
	cat go.mod | grep -v ' indirect' > direct.mod
	mv direct.mod go.mod
	rm go.sum || true
	go mod tidy

ensure: tidy
	go mod download


release:
	goreleaser build --skip-validate --rm-dist
