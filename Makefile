.PHONY: build test lint vet clean run dist

BINARY := graphify-lens
BUILD_DIR := build

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/graphify-lens/

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test -v -race -count=1 ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

lint:
	gofmt -s -w .
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)/ dist/

dist:
	bash scripts/dist.sh

deps:
	go mod tidy
	go mod verify
