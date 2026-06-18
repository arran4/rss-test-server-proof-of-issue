.PHONY: build test run clean

BINARY_NAME=rss-test-server

build:
	go build -o $(BINARY_NAME) rss-test-server.go

test:
	go test -v ./...

run: build
	./$(BINARY_NAME)

clean:
	go clean
	rm -f $(BINARY_NAME)
