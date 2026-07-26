COMPOSE ?= docker compose
WEB_DIR := apps/web
API_DIR := apps/api

.DEFAULT_GOAL := help

.PHONY: help setup up down reset restart logs ps config build test test-web test-api format-check

help: ## Show the available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Forma development commands:\n\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Create .env from the safe local template when it is missing
	@test -f .env || cp .env.example .env

up: setup ## Build and start the local stack
	$(COMPOSE) up --build -d

down: ## Stop the local stack without deleting resume data
	$(COMPOSE) down

reset: ## Stop the stack and permanently delete its database volume
	$(COMPOSE) down --volumes --remove-orphans

restart: ## Restart application containers
	$(COMPOSE) restart api web

logs: ## Follow API and web logs
	$(COMPOSE) logs --follow --tail=200 api web

ps: ## Show container and health status
	$(COMPOSE) ps

config: ## Validate and render the Compose configuration
	$(COMPOSE) config --quiet

build: ## Build production frontend and backend binaries
	npm --prefix $(WEB_DIR) ci
	npm --prefix $(WEB_DIR) run build
	@mkdir -p bin
	cd $(API_DIR) && go build -o ../../bin/forma-api ./cmd/api

test: test-web test-api ## Run all local tests

test-web: ## Install locked web dependencies and run frontend tests
	npm --prefix $(WEB_DIR) ci
	npm --prefix $(WEB_DIR) run test --if-present
	npm --prefix $(WEB_DIR) run build
	npm --prefix $(WEB_DIR) run test:sites --if-present

test-api: ## Run Go unit and integration tests that need no external services
	cd $(API_DIR) && go test ./...

format-check: ## Fail when committed Go source is not gofmt-formatted
	@test -z "$$(find $(API_DIR) -type f -name '*.go' -exec gofmt -l {} +)"
