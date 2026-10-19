#!/usr/bin/env bash
set -euo pipefail

echo "==> Initializing minIoT TimescaleDB schema and seed data..."
docker compose exec -T timescaledb psql -U postgres -d miniot < infra/postgres/init/01-init.sql
echo "==> Database initialization complete."
