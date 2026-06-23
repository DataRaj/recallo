.PHONY: build run dev test tidy lint clean sqlc migrateup migratedown new_migration

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

## Generate sqlc models and queries
sqlc:
	@sqlc generate

## Create a new migration file (e.g., make new_migration name=add_users_table)
new_migration:
	@migrate create -ext sql -dir db/migrations -seq $(name)

## Apply all up migrations
migrateup:
	@migrate -path db/migrations -database "$$DATABASE_URL" -verbose up

## Revert all down migrations
migratedown:
	@migrate -path db/migrations -database "$$DATABASE_URL" -verbose down

## Remove compiled artifacts
clean:
	@rm -rf bin/
