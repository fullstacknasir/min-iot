.PHONY: help up down logs infra apps dev-api dev-ingestor dev-web simulate seed test test-permissions clean

help:
	@echo "minIoT Monorepo Command Center"
	@echo ""
	@echo "Docker Deployment Commands:"
	@echo "  make up                 Deploy the whole minIoT in Docker (PostgreSQL, Redis, EMQX, pgAdmin4, API, Ingestor, Web)"
	@echo "  make down               Stop and remove all Docker containers"
	@echo "  make infra              Start only infrastructure services in Docker (TimescaleDB, Redis, EMQX, pgAdmin4)"
	@echo "  make logs               Follow real-time Docker container logs"
	@echo ""
	@echo "Local Development Commands:"
	@echo "  make dev-api            Run Go API locally with live reload/go run"
	@echo "  make dev-ingestor       Run Go Ingestor locally"
	@echo "  make dev-web            Run Angular web frontend locally"
	@echo "  make simulate           Run IoT device simulator to publish sample telemetry"
	@echo "  make seed               Run database migrations and seed default org & admin"
	@echo "  make test               Run all unit & integration tests with race detector"
	@echo "  make test-permissions   Run live MQTT script proving topic isolation and auth"
	@echo "  make clean              Clean docker volumes and temporary files"

up:
	./start.sh

infra:
	./start.sh --infra

apps:
	./start.sh

down:
	docker compose down

logs:
	docker compose logs -f

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
