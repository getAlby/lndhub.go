.env:
	cp .env_example .env
build:
	CGO_ENABLED=0 go build -o lndhub main.go
