.DEFAULT_GOAL := help

SHELL         := /bin/sh
CLIENT_DIR    := client
SERVER_DIR    := server
AI_DIR        := ai-service
COMPOSE       ?= docker compose
NPM           ?= npm
GO            ?= go
GOFMT         ?= gofmt
PYTHON        ?= python3
PIP           ?= pip3

# ──────────────────────────────────────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@printf "NDHU Second-Hand Book Store — available targets\n\n"
	@printf "  install           Install client and AI service dependencies\n"
	@printf "  format            Format client (Prettier) and server (gofmt)\n"
	@printf "  check             Run lint, typecheck, go vet, and all tests\n"
	@printf "  build             Build client production bundle\n\n"
	@printf "  stack-up          Start full Docker Compose stack\n"
	@printf "  stack-down        Stop and remove containers\n"
	@printf "  stack-logs        Follow container logs\n\n"
	@printf "  server-db-up      Start only the PostgreSQL container\n"
	@printf "  server-migrate    Apply pending SQL migrations\n"
	@printf "  server-test       Run Go tests with coverage report\n"
	@printf "  server-dev        Apply migrations then start Go API\n"
	@printf "  server-format     Format Go sources with gofmt\n\n"
	@printf "  client-install    Install client npm dependencies\n"
	@printf "  client-dev        Start Next.js development server\n"
	@printf "  client-build      Build Next.js for production\n"
	@printf "  client-format     Format client sources with Prettier\n\n"
	@printf "  ai-install        Install AI service dependencies\n"
	@printf "  ai-dev            Start AI FastAPI service locally\n\n"
	@printf "  generate-api      Regenerate typed API client from OpenAPI spec\n"
	@printf "  seed              Seed staging database with demo data\n"

# ──────────────────────────────────────────────────────────────────────────────
# Top-level targets
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: install format check build

install: client-install ai-install

format: client-format server-format

check: server-vet server-test client-lint client-typecheck

build: client-build

# ──────────────────────────────────────────────────────────────────────────────
# Docker Compose
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: stack-up stack-down stack-logs

stack-up:
	$(COMPOSE) up -d --build

stack-down:
	$(COMPOSE) down

stack-logs:
	$(COMPOSE) logs -f

# ──────────────────────────────────────────────────────────────────────────────
# Server
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: server-db-up server-migrate server-test server-dev server-format server-vet

server-db-up:
	$(COMPOSE) up -d postgres

server-migrate:
	cd $(SERVER_DIR) && $(GO) run ./cmd/migrate

server-test:
	cd $(SERVER_DIR) && $(GO) test -race -coverprofile=coverage.out ./pkg/... ./cmd/middleware/... && \
		$(GO) tool cover -func=coverage.out

server-dev: server-migrate
	cd $(SERVER_DIR) && $(GO) run ./main.go

server-format:
	$(GOFMT) -w $(SERVER_DIR)

server-vet:
	cd $(SERVER_DIR) && $(GO) vet ./...

# ──────────────────────────────────────────────────────────────────────────────
# Client
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: client-install client-dev client-build client-format client-lint client-typecheck

client-install:
	cd $(CLIENT_DIR) && $(NPM) ci

client-dev:
	cd $(CLIENT_DIR) && $(NPM) run dev

client-build:
	cd $(CLIENT_DIR) && $(NPM) run build

client-format:
	cd $(CLIENT_DIR) && $(NPM) run format

client-lint:
	cd $(CLIENT_DIR) && $(NPM) run lint

client-typecheck:
	cd $(CLIENT_DIR) && $(NPM) run typecheck

# ──────────────────────────────────────────────────────────────────────────────
# AI service
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: ai-install ai-dev

ai-install:
	cd $(AI_DIR) && $(PIP) install -r requirements.txt

ai-dev:
	cd $(AI_DIR) && uvicorn main:app --reload --host 0.0.0.0 --port 8000

# ──────────────────────────────────────────────────────────────────────────────
# API code generation
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: generate-api

generate-api:
	cd $(SERVER_DIR) && swag init -g main.go --output docs
	cd $(CLIENT_DIR) && $(NPM) run generate:openapi:json
	cd $(CLIENT_DIR) && $(NPM) run generate:openapi:yaml
	cd $(CLIENT_DIR) && $(NPM) run generate:api

# ──────────────────────────────────────────────────────────────────────────────
# Seed
# ──────────────────────────────────────────────────────────────────────────────
.PHONY: seed

seed:
	cd $(SERVER_DIR) && $(GO) run ./cmd/seed
