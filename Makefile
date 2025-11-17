.PHONY: help build run test clean docker-up docker-down migrate

# Variables
BINARY_NAME=crypto-bot
ADMIN_BINARY=crypto-admin-api
MAIN_PATH=cmd/bot/main.go
ADMIN_PATH=cmd/api/main.go
GO=go
GOFLAGS=-v

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: ./$(BINARY_NAME)"

run: ## Run the application
	@echo "Running application..."
	$(GO) run $(MAIN_PATH)

dev: docker-up run ## Start docker services and run app in dev mode

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

docker-up: ## Start Docker containers (PostgreSQL, Redis)
	@echo "Starting Docker containers..."
	docker-compose -f docker/docker-compose.yml up -d
	@echo "✅ Docker containers started"
	@echo "Waiting for PostgreSQL..."
	@sleep 3

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	docker-compose -f docker/docker-compose.yml down
	@echo "✅ Docker containers stopped"

docker-logs: ## Show Docker logs
	docker-compose -f docker/docker-compose.yml logs -f

db-reset: ## Reset database (drop and recreate)
	@echo "⚠️  Resetting database..."
	@docker-compose -f docker/docker-compose.yml exec postgres psql -U postgres -c "DROP DATABASE IF EXISTS crypto_bot_dev;"
	@docker-compose -f docker/docker-compose.yml exec postgres psql -U postgres -c "CREATE DATABASE crypto_bot_dev;"
	@echo "✅ Database reset complete"

install-deps: ## Install Go dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependencies installed"

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Code formatted"

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...
	@echo "✅ Linting complete"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "✅ Vet complete"

check: fmt vet test ## Run all checks (format, vet, test)

prod-build: ## Build for production
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-w -s" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Production build complete"

# Development helpers
.env: ## Create .env file from example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env file. Please fill in your credentials."; \
	else \
		echo "⚠️  .env file already exists"; \
	fi

setup: .env install-deps docker-up ## Initial setup (env, deps, docker)
	@echo "✅ Setup complete! Run 'make run' to start the bot"

# Scraper commands
scrape-now: ## Run scrapers immediately (for testing)
	@echo "Running scrapers..."
	@$(GO) run $(MAIN_PATH) --scrape-once
	@echo "✅ Scraping complete"

# Database helpers
db-shell: ## Open PostgreSQL shell
	docker-compose -f docker/docker-compose.yml exec postgres psql -U postgres -d crypto_bot_dev

db-backup: ## Backup database
	@echo "Creating database backup..."
	@mkdir -p backups
	@docker-compose -f docker/docker-compose.yml exec -T postgres pg_dump -U postgres crypto_bot_dev > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup created in backups/"

db-restore: ## Restore database from latest backup
	@echo "Restoring database..."
	@docker-compose -f docker/docker-compose.yml exec -T postgres psql -U postgres crypto_bot_dev < $(shell ls -t backups/*.sql | head -1)
	@echo "✅ Database restored"

# Monitoring
logs: ## Show application logs (when running in background)
	@tail -f logs/app.log

stats: ## Show bot statistics
	@echo "Bot statistics:"
	@echo "TODO: Implement stats gathering"

# Admin API
admin-build: ## Build Admin API
	@echo "Building Admin API..."
	$(GO) build $(GOFLAGS) -o $(ADMIN_BINARY) $(ADMIN_PATH)
	@echo "✅ Admin API build complete: ./$(ADMIN_BINARY)"

admin-run: ## Run Admin API
	@echo "Starting Admin API..."
	$(GO) run $(ADMIN_PATH)

admin-dev: docker-up admin-run ## Start docker and run Admin API in dev mode

admin-test: ## Test Admin API endpoints
	@echo "Testing Admin API..."
	@curl -s http://localhost:8080/api/v1/health | jq || echo "Admin API not running"
	@curl -s http://localhost:8080/api/v1/ping | jq

# All-in-one commands
restart: docker-down docker-up run ## Restart everything

fresh: clean docker-down docker-up build run ## Fresh start (clean, rebuild, restart)
