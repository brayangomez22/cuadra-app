# Cuadra: development commands. See CLAUDE.md.

BACKEND := backend
SQLC_VERSION := v1.31.1
OAPI_CODEGEN_VERSION := v2.8.0
OPENAPI_SPEC := $(CURDIR)/api/openapi.yaml

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

migrate: ## Apply migrations (goose up) as MIGRATION_DATABASE_URL and set app_user's password
	cd $(BACKEND) && go run ./cmd/migrate

sqlc: ## Regenerate sqlc code
	cd $(BACKEND) && go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

openapi: ## Regenerate code from api/openapi.yaml (one package per oapi-codegen.yaml)
	@cd $(BACKEND) && for cfg in $$(find . -name oapi-codegen.yaml -not -path './vendor/*' | sort); do \
		echo "oapi-codegen $$cfg"; \
		(cd "$$(dirname "$$cfg")" && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) -config oapi-codegen.yaml $(OPENAPI_SPEC)) || exit 1; \
	done

obs-up: ## Start the observability stack (Grafana on :3000)
	docker compose --profile observability up -d otel-collector prometheus loki tempo grafana

obs-down: ## Stop the observability stack
	docker compose --profile observability stop otel-collector prometheus loki tempo grafana
