.PHONY: help test check-migrations migrate migrate-down migrate-local migrate-local-down build push pull logs ps require-dockerhub-user require-clean-worktree

GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

ifeq ($(shell command -v docker-compose >/dev/null 2>&1; echo $$?),0)
	DOCKER_COMPOSE = docker-compose
else
	DOCKER_COMPOSE = docker compose
endif

ifneq (,$(wildcard .env))
	include .env
	export
endif

COMPOSE_PROJECT_NAME ?= shorter
service ?=
GO_CACHE_DIR ?= /tmp/shorter-go-cache
GO_MOD_CACHE_DIR ?= /tmp/shorter-go-mod-cache
DOCKERHUB_USER ?= $(shell [ -f .env ] && sed -n 's/^DOCKERHUB_USER=//p' .env | tail -n 1 | tr -d '\r')
IMAGE_TAG ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo local)
BACKEND_IMAGE ?= $(DOCKERHUB_USER)/shorter-backend
BOT_IMAGE ?= $(DOCKERHUB_USER)/shorter-bot
MIGRATE_IMAGE ?= $(DOCKERHUB_USER)/shorter-migrate
BUILD_SERVICES := backend bot migrate
RUN_SERVICES := backend bot

help:
	@echo "$(GREEN)Shorter commands$(NC)"
	@echo "  make test                 - check migrations and run Go tests"
	@echo "  make check-migrations     - validate SQL migration up/down pairs"
	@echo "  make migrate              - apply migrations through Docker Compose"
	@echo "  make migrate-local        - apply migrations from host using .env"
	@echo "  make build                - build Docker images"
	@echo "  make push                 - push Docker images with current tag and latest"
	@echo "  make pull                 - server deploy from current git commit; run git pull separately first"
	@echo "  make logs service=bot"
	@echo "  make ps"

test: check-migrations
	GOCACHE="$(GO_CACHE_DIR)" GOMODCACHE="$(GO_MOD_CACHE_DIR)" go test -v ./...

check-migrations:
	@./scripts/check_migrations.sh

migrate:
	COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) run --rm migrate up

migrate-down:
	COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) run --rm migrate down

migrate-local:
	go run ./cmd/migrate up

migrate-local-down:
	go run ./cmd/migrate down

build: check-migrations
	@echo "$(GREEN)Building Docker images with tag $(IMAGE_TAG)...$(NC)"
	DOCKERHUB_USER="$(DOCKERHUB_USER)" IMAGE_TAG="$(IMAGE_TAG)" $(DOCKER_COMPOSE) build $(BUILD_SERVICES)

push: require-clean-worktree require-dockerhub-user build
	@set -e; \
	for image in "$(BACKEND_IMAGE)" "$(BOT_IMAGE)" "$(MIGRATE_IMAGE)"; do \
		echo "$(YELLOW)Pushing $$image:$(IMAGE_TAG) and $$image:latest$(NC)"; \
		docker push "$$image:$(IMAGE_TAG)"; \
		docker tag "$$image:$(IMAGE_TAG)" "$$image:latest"; \
		docker push "$$image:latest"; \
	done

pull: require-dockerhub-user
	@set -e; \
	TAG="$$(git rev-parse --short HEAD)"; \
	echo "$(YELLOW)Deploying tag $$TAG...$(NC)"; \
	DOCKERHUB_USER="$(DOCKERHUB_USER)" IMAGE_TAG="$$TAG" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) pull $(BUILD_SERVICES); \
	DOCKERHUB_USER="$(DOCKERHUB_USER)" IMAGE_TAG="$$TAG" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) run --rm migrate up; \
	DOCKERHUB_USER="$(DOCKERHUB_USER)" IMAGE_TAG="$$TAG" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) up -d --no-build $(RUN_SERVICES); \
	DOCKERHUB_USER="$(DOCKERHUB_USER)" IMAGE_TAG="$$TAG" COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) ps

logs:
	@if [ -n "$(service)" ]; then \
		COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) logs -f "$(service)"; \
	else \
		COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) logs -f; \
	fi

ps:
	@COMPOSE_PROJECT_NAME="$(COMPOSE_PROJECT_NAME)" $(DOCKER_COMPOSE) ps

require-dockerhub-user:
	@if [ -z "$(DOCKERHUB_USER)" ]; then \
		echo "$(RED)Set DOCKERHUB_USER in .env or pass DOCKERHUB_USER=<name>$(NC)"; \
		exit 1; \
	fi

require-clean-worktree:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "$(RED)Commit or stash local changes before make push; Docker image tag is based on git HEAD.$(NC)"; \
		exit 1; \
	fi
