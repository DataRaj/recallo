.PHONY: build run dev test tidy lint clean

## Build the production binary into bin/
build:
	@go build -o bin/recallo ./cmd/api/...

## Run the already-built binary
run:
	@./bin/recallo

## Build then immediately run (hot workflow)
dev:
	@go build -o bin/recallo ./cmd/api/... && ./bin/recallo

## Run all tests with verbose output
test:
	@go test -v ./...

## Tidy and verify module dependencies
tidy:
	@go mod tidy
	@go mod verify

## Run go vet + staticcheck linters
lint:
	@go vet ./...

## Remove compiled artifacts
clean:
	@rm -rf bin/
