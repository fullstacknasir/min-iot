#!/usr/bin/env bash
set -euo pipefail

# Text styling
BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
RED="\033[0;31m"
NC="\033[0m"

echo -e "${BLUE}${BOLD}====================================================${NC}"
echo -e "${BLUE}${BOLD}            minIoT Platform Docker Deploy           ${NC}"
echo -e "${BLUE}${BOLD}====================================================${NC}"

# Check .env
if [ ! -f .env ]; then
    echo -e "${YELLOW}[!] .env not found. Copying from .env.example...${NC}"
    cp .env.example .env
fi

# Ensure docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH.${NC}"
    exit 1
fi

MODE="all"
if [ "${1:-}" = "--infra" ] || [ "${1:-}" = "-i" ]; then
    MODE="infra"
fi

if [ "$MODE" = "infra" ]; then
    echo -e "${GREEN}[+] Starting infrastructure only (TimescaleDB, Redis, EMQX, pgAdmin 4)...${NC}"
    docker compose up -d timescaledb redis emqx pgadmin4
else
    echo -e "${GREEN}[+] Building and starting all minIoT services in Docker...${NC}"
    docker compose up -d --build
fi

echo -e "\n${BLUE}[*] Waiting for container health status...${NC}"
docker compose ps

echo -e "\n${GREEN}${BOLD}✔ minIoT Platform is up and running!${NC}"
echo -e "----------------------------------------------------"
if [ "$MODE" = "all" ]; then
    echo -e "${CYAN}${BOLD}Web Frontend Dashboard :${NC} http://localhost:4200"
    echo -e "${CYAN}${BOLD}minIoT Backend API      :${NC} http://localhost:8080 (Health: http://localhost:8080/health)"
fi
echo -e "${CYAN}${BOLD}PostgreSQL / Timescale  :${NC} localhost:5432 (Database: miniot, User: postgres)"
echo -e "${CYAN}${BOLD}pgAdmin 4 Web Console   :${NC} http://localhost:5050 (Auto-connected to TimescaleDB)"
echo -e "${CYAN}${BOLD}Redis Cache & State     :${NC} localhost:6379"
echo -e "${CYAN}${BOLD}EMQX MQTT TCP Broker    :${NC} localhost:1883"
echo -e "${CYAN}${BOLD}EMQX MQTT WebSocket     :${NC} localhost:8083"
echo -e "${CYAN}${BOLD}EMQX Admin Dashboard    :${NC} http://localhost:18083 (admin / public)"
echo -e "----------------------------------------------------"
echo -e "${YELLOW}${BOLD}Default Credentials:${NC}"
echo -e "• minIoT Admin User     : ${BOLD}admin@miniot.io${NC} / ${BOLD}AdminPassword123!${NC}"
echo -e "• pgAdmin 4 Web UI      : ${BOLD}admin@miniot.io${NC} / ${BOLD}admin${NC} (Desktop mode enabled)"
echo -e "• EMQX Dashboard        : ${BOLD}admin${NC} / ${BOLD}public${NC}"
echo -e "----------------------------------------------------"
echo -e "Useful management commands:"
echo -e "• View logs in real-time : ${YELLOW}docker compose logs -f${NC}"
echo -e "• View specific logs     : ${YELLOW}docker compose logs -f api web${NC}"
echo -e "• Stop all services      : ${YELLOW}docker compose down${NC}"
echo -e "• Reset database volumes : ${YELLOW}docker compose down -v${NC}"
echo -e "----------------------------------------------------"
