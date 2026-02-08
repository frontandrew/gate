.PHONY: help install migrate-up migrate-down migrate-create docker-up docker-down docker-logs docker-rebuild run build test test-coverage lint clean seed dev

.DEFAULT_GOAL := help

# Database configuration
DB_URL := postgresql://gate_user:gate_password@localhost:5432/gate_db?sslmode=disable

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install Go dependencies
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed successfully!"

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@migrate -path migrations -database "$(DB_URL)" up
	@echo "Migrations applied successfully!"

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@migrate -path migrations -database "$(DB_URL)" down
	@echo "Migrations rolled back successfully!"

migrate-create: ## Create new migration (use: make migrate-create name=migration_name)
ifndef name
	@echo "Error: Please specify migration name"
	@echo "Usage: make migrate-create name=<migration_name>"
	@exit 1
endif
	@migrate create -ext sql -dir migrations -seq $(name)
	@echo "Migration files created successfully!"

docker-up: ## Start all Docker services
	@echo "Starting Docker services..."
	@docker-compose -f docker/docker-compose.yml up -d
	@echo "Docker services started successfully!"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"

docker-down: ## Stop all Docker services
	@echo "Stopping Docker services..."
	@docker-compose -f docker/docker-compose.yml down
	@echo "Docker services stopped successfully!"

docker-logs: ## Show Docker logs
	@docker-compose -f docker/docker-compose.yml logs -f

docker-rebuild: ## Rebuild and restart Docker services
	@echo "Rebuilding Docker services..."
	@docker-compose -f docker/docker-compose.yml up -d --build
	@echo "Docker services rebuilt successfully!"

run: ## Run the API server
	@echo "Starting API server..."
	@go run cmd/api/main.go

build: ## Build the application binary
	@echo "Building application..."
	@mkdir -p bin
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api
	@echo "Build completed: bin/api"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "Tests completed!"

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run --timeout=5m
	@echo "Linting completed!"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean completed!"

seed: ## Seed database with test data
	@echo "Seeding database..."
	@psql "$(DB_URL)" -f scripts/seed.sql
	@echo "Database seeded successfully!"

dev: docker-up ## Start development environment
	@echo "Development environment ready!"
	@echo "Run 'make run' to start the API server"
