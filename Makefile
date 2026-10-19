.PHONY: help up down logs infra apps dev-api dev-ingestor dev-web simulate seed test test-permissions clean

help:
	@echo "minIoT Monorepo Command Center"
	@echo ""
	@echo "Infrastructure commands:"
	@echo "  make up                 Start infrastructure (TimescaleDB, Redis, EMQX)"
	@echo "  make down               Stop and remove containers"
	@echo "  make infra              Alias for make up"
	@echo "  make logs               Follow docker logs for infrastructure"
	@echo "  make apps               Start all services (infra + api + ingestor + web) via Docker"
	@echo ""
	@echo "Local development commands:"
	@echo "  make dev-api            Run Go API locally with live reload/go run"
	@echo "  make dev-ingestor       Run Go Ingestor locally"
	@echo "  make dev-web            Run Angular web frontend locally"
	@echo "  make simulate           Run IoT device simulator to publish sample telemetry"
	@echo "  make seed               Run database migrations and seed default org & admin"
	@echo "  make test               Run all unit & integration tests with race detector"
	@echo "  make test-permissions   Run live MQTT script proving topic isolation and auth"
	@echo "  make clean              Clean docker volumes and temporary files"

up: infra

infra:
	./start.sh

down:
	docker compose down

logs:
	docker compose logs -f

apps:
	docker compose --profile apps up -d --build

dev-api:
	cd apps/api && go run cmd/api/main.go

dev-ingestor:
	cd apps/ingestor && go run cmd/ingestor/main.go

dev-web:
	cd apps/web && npm start

DEVICES ?= 1
INTERVAL ?= 3

simulate:
	cd tools/device-simulator && go run main.go -devices $(DEVICES) -interval $(INTERVAL)

seed:
	cd apps/api && go run cmd/seed/main.go

test:
	cd apps/api && go test -v -race ./...
	cd apps/ingestor && go test -v -race ./...
	cd apps/web && npx ng test --watch=false

test-permissions:
	./tools/scripts/test-device-permissions.sh

clean:
	docker compose down -v
