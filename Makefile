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
	@docker run --rm -v $(CURDIR)/migrations:/migrations --network gate-network migrate/migrate \
		-path=/migrations -database "postgresql://gate_user:gate_password@gate-postgres:5432/gate_db?sslmode=disable" up
	@echo "Migrations applied successfully!"

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@docker run --rm -v $(CURDIR)/migrations:/migrations --network gate-network migrate/migrate \
		-path=/migrations -database "postgresql://gate_user:gate_password@gate-postgres:5432/gate_db?sslmode=disable" down -all
	@echo "Migrations rolled back successfully!"

migrate-create: ## Create new migration (use: make migrate-create name=migration_name)
ifndef name
	@echo "Error: Please specify migration name"
	@echo "Usage: make migrate-create name=<migration_name>"
	@exit 1
endif
	@docker run --rm -v $(CURDIR)/migrations:/migrations migrate/migrate \
		create -ext sql -dir /migrations -seq $(name)
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

run: ## Run the API server (Docker)
	@echo "Building and starting API server..."
	@docker-compose -f docker/docker-compose.yml up --build api

run-d: ## Run the API server in background (Docker)
	@echo "Building and starting API server..."
	@docker-compose -f docker/docker-compose.yml up -d --build api
	@echo "API server started at http://localhost:8080"

build: ## Build the application image
	@echo "Building application..."
	@docker-compose -f docker/docker-compose.yml build api
	@echo "Build completed!"

test: ## Run tests
	@echo "Running tests..."
	@docker run --rm -v $(CURDIR):/app -w /app golang:1.22-alpine go test -v -race ./...
	@echo "Tests completed!"

lint: ## Run linter
	@echo "Running linter..."
	@docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:latest golangci-lint run --timeout=5m
	@echo "Linting completed!"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean completed!"

seed: ## Seed database with test data
	@echo "Seeding database..."
	@docker exec -i gate-postgres psql -U gate_user -d gate_db < scripts/seed.sql
	@echo "Database seeded successfully!"

dev: docker-up ## Start development environment
	@echo "Development environment ready!"
	@echo "Run 'make run' to start the API server"
