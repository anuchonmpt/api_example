.PHONY: help deps install-tools generate build run dev worker test test-coverage clean fmt vet lint migrate-up migrate-down migrate-status migrate-validate migrate-dry-run migrate-check sqlc-generate sqlc-vet docker-up docker-down

APP_NAME := api-example
ENV ?= local
GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'

deps: ## Download and tidy Go dependencies
	@go mod download
	@go mod tidy

install-tools: ## Install development tools
	@go install github.com/air-verse/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

generate: sqlc-generate ## Generate all generated code

build: ## Build server, worker, and migration binaries
	@mkdir -p bin
	@go build -o bin/server ./cmd/server
	@go build -o bin/worker ./cmd/worker
	@go build -o bin/migration ./cmd/migration

run: ## Run the HTTP server
	@go run ./cmd/server

dev: ## Run the HTTP server with Air
	@air

worker: ## Run the document worker
	@go run ./cmd/worker

test: ## Run all tests
	@go test -race -cover ./...

test-coverage: ## Generate an HTML coverage report
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Remove starter build and test artifacts
	@rm -rf ./bin ./tmp ./data ./coverage.out ./coverage.html

fmt: ## Format Go source
	@gofmt -w .

vet: ## Run go vet
	@go vet ./...

lint: ## Run golangci-lint
	@$(GOLANGCI_LINT) run

migrate-up: ## Apply pending migrations
	@go run ./cmd/migration $(ENV) up

migrate-down: ## Roll back migrations; use steps=N
	@go run ./cmd/migration $(ENV) down $(or $(steps),1)

migrate-status: ## Show migration status
	@go run ./cmd/migration $(ENV) status

migrate-validate: ## Validate the migration registry
	@go run ./cmd/migration validate

migrate-dry-run: ## Show pending migrations
	@go run ./cmd/migration $(ENV) dry-run

migrate-check: ## Check database and migration state
	@go run ./cmd/migration $(ENV) check

sqlc-generate: ## Generate pgx code from SQL
	@sqlc generate

sqlc-vet: ## Vet SQLC queries
	@sqlc vet

docker-up: ## Start PostgreSQL and Redis
	@docker compose up -d postgres redis

docker-down: ## Stop local services
	@docker compose down

.DEFAULT_GOAL := help
