#!/usr/bin/env bash
set -euo pipefail

# Text styling
BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[0;33m"
NC="\033[0m"

echo -e "${BLUE}${BOLD}=========================================${NC}"
echo -e "${BLUE}${BOLD}        Starting minIoT Platform         ${NC}"
echo -e "${BLUE}${BOLD}=========================================${NC}"

# Check .env
if [ ! -f .env ]; then
    echo -e "${YELLOW}[!] .env not found. Copying from .env.example...${NC}"
    cp .env.example .env
fi

# Ensure docker is available
if ! command -v docker &> /dev/null; then
    echo -e "\033[0;31mError: Docker is not installed or not in PATH.\033[0m"
    exit 1
fi

# Spin up infrastructure services
echo -e "${GREEN}[+] Starting infrastructure (TimescaleDB, Redis, EMQX)...${NC}"
docker compose up -d timescaledb redis emqx

echo -e "\n${BLUE}[*] Waiting for infrastructure healthchecks...${NC}"
docker compose ps

echo -e "\n${GREEN}${BOLD}✔ Infrastructure started successfully!${NC}"
echo -e "-----------------------------------------"
echo -e "• PostgreSQL/TimescaleDB : localhost:5432 (db: miniot, user: postgres)"
echo -e "• Redis                  : localhost:6379"
echo -e "• EMQX MQTT TCP Broker   : localhost:1883"
echo -e "• EMQX MQTT WebSocket    : localhost:8083"
echo -e "• EMQX Web Dashboard     : http://localhost:18083 (admin / public)"
echo -e "-----------------------------------------"
echo -e "To start all apps with Docker, run:"
echo -e "  ${YELLOW}docker compose --profile apps up -d${NC}"
echo -e "To view logs, run:"
echo -e "  ${YELLOW}docker compose logs -f${NC}"
echo -e "To stop services, run:"
echo -e "  ${YELLOW}docker compose down${NC}"
echo -e "-----------------------------------------"
