BINARY=gapk

build:
	go build -o $(BINARY) ./cmd/gapk

run:
	go run ./cmd/gapk help

test:
	go test ./...

clean:
	rm -rf $(BINARY) dist/ build/
