# Cuadra: development commands. See CLAUDE.md.

BACKEND := backend

# Load local environment variables (not committed) if present.
-include .env
export

.DEFAULT_GOAL := help
.PHONY: help up down run test test-int lint check migrate sqlc openapi obs-up obs-down

help: ## List available commands
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-10s %s\n", $$1, $$2}'

up: ## Start Postgres (docker compose)
	docker compose up -d --wait postgres

down: ## Stop local containers
	docker compose --profile observability down

run: ## Run the API on :8080
	cd $(BACKEND) && go run ./cmd/api

test: ## Unit tests (fast, no Docker)
	cd $(BACKEND) && go test -race -count=1 ./...

test-int: ## Integration tests (requires Docker)
	cd $(BACKEND) && go test -race -count=1 -tags=integration ./...

lint: ## gofmt + go vet + golangci-lint
	@cd $(BACKEND) && unformatted="$$(gofmt -l .)" && if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	cd $(BACKEND) && go vet ./...
	cd $(BACKEND) && golangci-lint run ./...

check: lint test test-int ## Must pass before a task is done

migrate: ## Apply migrations (goose up)
	@echo "migrate: pendiente (T03)"; exit 1

sqlc: ## Regenerate sqlc code
	@echo "sqlc: pendiente (T03)"; exit 1

openapi: ## Regenerate code from api/openapi.yaml
	@echo "openapi: pendiente (T04)"; exit 1

obs-up: ## Start the observability stack (Grafana on :3000)
	docker compose --profile observability up -d otel-collector prometheus loki tempo grafana

obs-down: ## Stop the observability stack
	docker compose --profile observability stop otel-collector prometheus loki tempo grafana
