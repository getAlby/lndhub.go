.env:
	cp .env_example .env

build:
	CGO_ENABLED=0 go build -o lndhub ./cmd/server

test:
	go test -p 1 -v -covermode=atomic -coverprofile=coverage.out -cover -coverpkg=./... ./...
