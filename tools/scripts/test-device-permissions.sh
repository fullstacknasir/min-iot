#!/usr/bin/env bash
set -euo pipefail

BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
RED="\033[0;31m"
NC="\033[0m"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo -e "${BLUE}${BOLD}==> Testing Device Authentication & Topic Authorization Permissions...${NC}"

API_PID=""
if ! curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${BLUE}[*] Starting API service in background...${NC}"
    (cd "$ROOT_DIR/apps/api" && go run cmd/api/main.go > /tmp/miniot-api-test.log 2>&1) &
    API_PID=$!
    
    for i in {1..25}; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            echo -e "${GREEN}[*] API service is online.${NC}"
            break
        fi
        sleep 0.5
    done
fi

cleanup() {
    if [ -n "$API_PID" ]; then
        echo -e "${BLUE}[*] Stopping background API service (PID $API_PID)...${NC}"
        kill "$API_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

echo -e "${BLUE}[*] Running device permission verification...${NC}"
cd "$ROOT_DIR/tools/device-verifier"
go run main.go

echo -e "${GREEN}${BOLD}✔ Device permission proofs completed successfully!${NC}"
