# VoidCrew Makefile

BINARY_NAME=voidcrew

.PHONY: all build run test clean

all: build

build:
	go build -o $(BINARY_NAME) main.go

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)
