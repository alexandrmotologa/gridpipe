.PHONY: all build test clean run-studio

all: build

build:
	go build -o bin/gridpipe.exe ./cmd/gridpipe

test:
	go test -v -race ./...

clean:
	rm -rf bin/ dist/ web/dist

run-studio: build
	./bin/gridpipe.exe studio --port 7575
