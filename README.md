# minIoT — Scalable IoT Platform Monorepo

`minIoT` is an enterprise-grade IoT platform built for high-throughput sensor telemetry ingestion, storage in TimescaleDB hypertables, real-time caching in Redis, organization-scoped multi-tenancy, and visualization with Angular, TanStack Query, and Apache ECharts.

---

## 📁 Monorepo Folder Structure

```text
minIoT/
├── apps/
│   ├── api/                          # REST API service (Gin, Golang)
│   │   ├── cmd/
│   │   │   ├── api/main.go           # HTTP server entrypoint
│   │   │   └── seed/main.go          # Migration & seed script (one org, one admin)
│   │   ├── docs/
│   │   │   └── openapi.yaml          # OpenAPI 3.0.3 specification
│   │   ├── internal/
│   │   │   ├── config/               # Environment configuration (JWT, rate limits)
│   │   │   ├── database/             # TimescaleDB & Redis clients + migration runner
│   │   │   ├── handlers/             # Auth, User, Device, Telemetry, Docs, & EMQX handlers
│   │   │   ├── middleware/           # JWT Auth, RBAC, Structured Logging & Rate Limiters
│   │   │   ├── models/               # Multi-tenant DTOs, schemas & credentials
│   │   │   ├── repository/           # Org-scoped database & cache queries
│   │   │   └── security/             # Bcrypt password hashing & JWT/device tokens
│   │   ├── migrations/               # PostgreSQL migration SQL files
│   │   ├── test/                     # Integration tests (auth, devices, telemetry, users, RBAC)
│   │   ├── Dockerfile                # Multi-stage production container
│   │   └── go.mod
│   │
│   ├── ingestor/                     # MQTT Telemetry Ingestor (Golang)
│   │   ├── cmd/ingestor/main.go      # Ingestor service entrypoint
│   │   ├── internal/
│   │   │   ├── config/               # Ingestor configuration (TLS options, health port)
│   │   │   ├── health/               # HTTP health-check probe server (:8081/health)
│   │   │   ├── mqtt/                 # Paho MQTT subscriber client (TLS & TCP support)
│   │   │   ├── processor/            # JSON schema validation & idempotent storage
│   │   │   ├── storage/              # TimescaleDB hypertable writer & Redis cache
│   │   │   └── validator/            # Payload structural and data type validation
│   │   ├── Dockerfile                # Multi-stage production container
│   │   └── go.mod
│   │
│   └── web/                          # Frontend Dashboard (Angular + TypeScript, Not Standalone)
│       ├── src/
│       │   ├── app/
│       │   │   ├── core/             # Auth, Device, Telemetry & User services, models, guards, interceptors
│       │   │   ├── pages/            # Login, Device List, Device Detail, Admin Devices, Admin Users
│       │   │   ├── widgets/          # ThingsBoard Custom Widgets (Live SSE & Apache ECharts)
│       │   │   ├── shared/           # Moment.js timestamp pipe
│       │   │   ├── app.component.ts  # Root container with navigation & responsive layout
│       │   │   └── app.module.ts     # Root NgModule declaring all components & widgets
│       │   ├── index.html            # Shell with Material Icons & Inter font
│       │   ├── main.ts               # Bootstrapping via platformBrowserDynamic
│       │   └── styles.css            # Tailwind CSS imports
│       ├── Dockerfile                # Multi-stage Nginx container
│       ├── nginx.conf                # SPA routing configuration
│       └── package.json
│
├── infra/                            # Infrastructure definitions & configurations
│   ├── emqx/
│   │   ├── certs/                    # SSL / TLS certificates for MQTTS (port 8883)
│   │   ├── emqx.conf                 # Development EMQX configuration
│   │   └── emqx-production.conf      # Production TLS/SSL EMQX configuration
│   ├── postgres/
│   │   └── init/
│   │       └── 01-init.sql           # TimescaleDB extension, hypertables & seed data
│   └── redis/
│       └── redis.conf                # Redis 7 caching configuration
│
├── tools/                            # Development & testing utilities
│   ├── device-simulator/             # Multi-device Go simulator publishing fake telemetry
│   ├── device-verifier/              # MQTT authorization test verifying topic permissions
│   └── scripts/
│       ├── init-db.sh                # Database migration runner
│       └── test-device-permissions.sh# ACL & isolation verification script
│
├── .env.example                      # Environment template (includes TLS & security defaults)
├── docker-compose.yml                # TimescaleDB, Redis, EMQX & application profiles
├── go.work                           # Go multi-module workspace
├── Makefile                          # Developer workflow automation
├── start.sh                          # Executable root infrastructure launcher
└── README.md                         # Platform documentation
```

---

## 🏗️ Architecture & Component Overview

```
                                [IoT Devices / Sensors] 
                                           │
                     ┌─────────────────────┴─────────────────────┐
                     │ (Plain MQTT: 1883)                        │ (TLS Encrypted MQTTS: 8883)
                     ▼                                           ▼
               ┌───────────────────────────────────────────────────────┐
               │              EMQX 5 Enterprise Broker                 │
               │  • Per-device HTTP Auth (/api/v1/emqx/auth)           │
               │  • Topic-level ACLs (/api/v1/emqx/acl)                │
               │  • Connect / Disconnect Webhooks (/api/v1/emqx/events)│
               └──────────────────────────┬────────────────────────────┘
                                          │ Subscribed (devices/+/telemetry)
                                          ▼
                               ┌─────────────────────┐
                               │    apps/ingestor    │
                               │  (Golang Ingestor)  │
                               │  • JSON Schema Val  │
                               │  • Health (:8081)   │
                               └──────────┬──────────┘
                                          │
                         ┌────────────────┴────────────────┐
                         ▼                                 ▼
              ┌─────────────────────┐           ┌─────────────────────┐
              │ TimescaleDB (PG 16) │           │    Redis 7 Cache    │
              │ • 30-Day Retention  │           │ • Latest Snapshots  │
              │ • 1h Continuous Agg │           │ • Pub/Sub Live SSE  │
              │ • Hypertable Series │           │ • Rate Limit Keys   │
              └──────────┬──────────┘           └──────────┬──────────┘
                         │                                 │
                         └────────────────┬────────────────┘
                                          │ Scoped to Caller's Organization
                                          ▼
                               ┌─────────────────────┐
                               │      apps/api       │
                               │    (Gin Golang)     │
                               │ • JWT Auth Rotation │
                               │ • Rate Limiters     │
                               │ • Structured Slog   │
                               │ • Swagger UI /docs  │
                               └──────────┬──────────┘
                                          │ HTTP REST & SSE Streaming (:8080)
                                          ▼
                               ┌─────────────────────┐
                               │      apps/web       │
                               │  Angular + Tailwind │
                               │  • TanStack Query   │
                               │  • Apache ECharts   │
                               │  • ThingsBoard Wdgt │
                               └─────────────────────┘
```

1. **EMQX 5 Broker** (`:1883` TCP, `:8883` SSL/TLS, `:8083` WS, `:8084` WSS, `:18083` Dashboard):
   High-performance MQTT broker delegating client authentication and topic authorization dynamically to the API. Devices may publish exclusively to `devices/{deviceId}/telemetry` and subscribe to `devices/{deviceId}/commands`.
2. **PostgreSQL 16 with TimescaleDB** (`:5432`):
   Time-series hypertable (`telemetry`) with 30-day raw data retention policies and an hourly continuous aggregate (`telemetry_hourly`), alongside relational multi-tenant tables (`organizations`, `users`, `refresh_tokens`, `devices`, `device_credentials`).
3. **Redis 7** (`:6379`):
   In-memory key-value cache storing instantaneous device state snapshots, Redis Pub/Sub channels for live SSE telemetry broadcasting (`telemetry:device:{id}`), and sliding-window rate limit counters.
4. **`apps/api`** (`:8080`):
   Organization-scoped REST API providing user authentication, token refresh rotation, device CRUD, one-time credential provisioning, device enable/disable/revoke lifecycle, user management, telemetry history and time-bucket aggregations, real-time Server-Sent Events (SSE), and interactive Swagger UI docs (`GET /docs`).
5. **`apps/ingestor`** (`:8081` Health):
   Concurrent MQTT worker validating incoming payloads against JSON schema, rejecting malformed messages, persisting idempotent readings with device and server timestamps, updating Redis caches, and serving an HTTP health probe.
6. **`apps/web`** (`:4200`):
   Enterprise Angular (NgModule / Not Standalone) dashboard built with TailwindCSS utilities, TanStack Query, Apache ECharts, ThingsBoard-style custom widgets, Moment.js date formatting, and Angular Material icons.

---

## 🔐 Security, Multi-Tenancy & Reliability Architecture

### 1. No Secrets in Code
- All credentials (database passwords, Redis passwords, JWT signing keys, admin passwords) are loaded dynamically from environment variables using `godotenv`.
- In production (`APP_ENV=production`), the API logs a security alert if the fallback development secret is detected.
- Per-device MQTT credentials are generated as cryptographically secure 256-bit random tokens, displayed exactly once upon provisioning, and stored as one-way SHA-256 hashes.

### 2. Strict Input Validation Everywhere
- **API Request Validation**: Gin struct tags enforce email formatting, password length ($\ge 8$ chars), and required fields.
- **Query Parameter Sanitization**: Time ranges (`from`, `to`) strictly enforce RFC3339 timestamps and ordering (`from <= to`). Limit parameters are bounded ($\le 1000$) to prevent memory exhaustion DoS.
- **SQL Injection Defense**: Dynamic `time_bucket` intervals are validated using strict regex (`^[0-9]+\s*[a-zA-Z]+$`) and mapped to standard intervals (`1m` $\rightarrow$ `1 minute`, `1h` $\rightarrow$ `1 hour`), while queries use parameterized SQL placeholders (`$1, $2, ...`).
- **Telemetry Schema Validation**: Ingestor validates JSON syntax, verifies epoch/ISO timestamps, enforces numeric metrics, and blocks device ID spoofing (payload device ID must match the MQTT topic).

### 3. Authorization & Multi-Tenant Scoping
- **Organization Isolation**: Every database query for devices, users, credentials, and telemetry is filtered with `WHERE organization_id = $...`. Cross-tenant reads are rejected with `404 Not Found`.
- **Role-Based Access Control (RBAC)**:
  - `admin`: Full administrative access (user management, device management, credential provisioning, enable/disable/revoke).
  - `operator`: Fleet management (create, update, delete, provision, enable/disable/revoke) and direct telemetry ingestion.
  - `viewer`: Read-only access to organization devices, credential metadata, and historical telemetry.
- **MQTT Broker ACLs**: EMQX webhook ACL denies access unless client ID matches the device ID and topic adheres to `devices/{deviceId}/telemetry` (publish) or `devices/{deviceId}/commands` (subscribe). Revoked or disabled devices are rejected at authentication.

### 4. Sliding-Window Rate Limiting
- **Login Rate Limiter**: 5 attempts per minute per IP address, backed by Redis sliding window counters (returning `429 Too Many Requests` with `Retry-After`).
- **General API Rate Limiter**: 120 requests per minute per client IP for protected endpoints.

### 5. Production TLS / SSL for MQTT
- EMQX includes secure SSL listener `listeners.ssl.default` on port `8883` and `listeners.wss.default` on port `8084`.
- Pre-generated CA and server certificates reside in `infra/emqx/certs/` for immediate TLS enablement.
- `apps/ingestor` supports connecting via `ssl://` or `tls://` with configurable TLS certificate verification.

### 6. Structured Logging
- `apps/api` uses standard library `log/slog` in `StructuredLogger()` middleware, outputting JSON logs in production with timestamp, HTTP status, method, path, latency, client IP, user ID, and organization ID.
- `apps/ingestor` uses structured logging for telemetry processing, recording topic, payload size, duration, and rejection reasons.

### 7. Comprehensive Health Probes
- **API Service**: `GET /health` (`:8080/health`) validates TimescaleDB and Redis connectivity, returning 200 OK or 503 Service Unavailable if degraded.
- **Ingestor Service**: `GET /health` (`:8081/health`) validates TimescaleDB, Redis, and MQTT broker connectivity.
- **Docker Compose**: Container health checks monitor PostgreSQL (`pg_isready`), Redis (`redis-cli ping`), and EMQX (`emqx ctl status`).

---

## 📡 API Reference

Interactive OpenAPI documentation & Swagger UI is available at **`http://localhost:8080/docs`** (raw spec at `/docs/openapi.yaml`).

| Method | Endpoint | Required Role | Description |
|---|---|---|---|
| `GET` | `/health` | Public | System health check (PostgreSQL & Redis status) |
| `GET` | `/docs` | Public | Interactive Swagger UI API documentation |
| `POST` | `/api/v1/auth/register` | Public | Register new organization & admin, or join org |
| `POST` | `/api/v1/auth/login` | Public (Rate Limited)| Authenticate credentials; returns access & refresh token |
| `POST` | `/api/v1/auth/refresh` | Public | Rotate refresh token; returns new token pair |
| `POST` | `/api/v1/auth/logout` | Public | Revoke active refresh token session |
| `GET` | `/api/v1/devices` | `viewer`, `operator`, `admin` | List organization devices |
| `POST` | `/api/v1/devices` | `operator`, `admin` | Register new IoT device |
| `GET` | `/api/v1/devices/:id` | `viewer`, `operator`, `admin` | Get device details |
| `PUT` | `/api/v1/devices/:id` | `operator`, `admin` | Update device name, type, or metadata |
| `DELETE` | `/api/v1/devices/:id` | `operator`, `admin` | Delete device and credentials |
| `POST` | `/api/v1/devices/:id/credentials` | `operator`, `admin` | Provision device token (shown once) |
| `GET` | `/api/v1/devices/:id/credentials` | `viewer`, `operator`, `admin` | View credential metadata (secret is hidden) |
| `POST` | `/api/v1/devices/:id/enable` | `operator`, `admin` | Enable device & activate credentials |
| `POST` | `/api/v1/devices/:id/disable` | `operator`, `admin` | Disable device & suspend credentials |
| `POST` | `/api/v1/devices/:id/revoke` | `operator`, `admin` | Permanently revoke device credentials |
| `GET` | `/api/v1/devices/:id/telemetry/latest` | `viewer`, `operator`, `admin` | Instantaneous cached telemetry from Redis |
| `GET` | `/api/v1/devices/:id/telemetry/history` | `viewer`, `operator`, `admin` | Query TimescaleDB hypertable points |
| `GET` | `/api/v1/devices/:id/telemetry/aggregates` | `viewer`, `operator`, `admin` | Time-bucket downsampled values (avg/min/max/count) |
| `GET` | `/api/v1/devices/:id/telemetry/live` | `viewer`, `operator`, `admin` | Live Server-Sent Events (SSE) telemetry stream |
| `POST` | `/api/v1/telemetry` | `operator`, `admin` | Submit telemetry directly via HTTP |
| `GET` | `/api/v1/users` | `admin` | List all users in organization |
| `POST` | `/api/v1/users` | `admin` | Invite new user with specific role |
| `PATCH` | `/api/v1/users/:id/role` | `admin` | Change user role (`admin`, `operator`, `viewer`) |
| `DELETE` | `/api/v1/users/:id` | `admin` | Remove user from organization |

---

## 🚀 Quickstart & Commands

### 1. Launch Infrastructure
```bash
./start.sh
# Or directly:
docker compose up -d timescaledb redis emqx
```

### 2. Seed Default Organization & Admin
```bash
make seed
```
Default credentials:
- **Email**: `admin@miniot.io`
- **Password**: `AdminPassword123!`
- **Organization**: `Acme IoT Corporation` (`org-default-001`)

### 3. Run Monorepo Test Suite
```bash
make test
```
Executes all unit and integration tests across `/apps/api` (race detector enabled), `/apps/ingestor` (race detector enabled), and `/apps/web` (Vitest headless suite).

### 4. Local Development
```bash
# Terminal 1: Run Go REST API
make dev-api

# Terminal 2: Run Go MQTT Ingestor
make dev-ingestor

# Terminal 3: Run Angular Web Dashboard
make dev-web

# Terminal 4: Run IoT Sensor Simulator
make simulate                       # Default: 1 device, 3s interval
make simulate DEVICES=5 INTERVAL=2  # 5 simulated devices publishing every 2s
```

---

## ⚠️ Remaining Risks & Production Hardening Recommendations

While `minIoT` implements robust multi-tenancy, rate limiting, and input validation, the following considerations should be addressed before enterprise high-scale production deployments:

1. **Production TLS Certificates**:
   - The certificates in `infra/emqx/certs/` are self-signed. In public production environments, replace them with certificates signed by a trusted Certificate Authority (e.g., Let's Encrypt, AWS Certificate Manager, or internal PKI/Vault), and configure `EMQX_TLS_SKIP_VERIFY=false`.
2. **Secret Management**:
   - Production secrets (`JWT_SECRET`, database passwords) should be retrieved dynamically from a dedicated secrets manager (e.g. HashiCorp Vault, AWS Secrets Manager, or Kubernetes Secrets) rather than static `.env` files.
3. **Database High Availability & Scaling**:
   - The current TimescaleDB setup runs a single primary container. High-availability streaming replication (patroni/pgpool) and distributed hypertables should be provisioned for fleets exceeding tens of thousands of messages per second.
4. **Horizontal MQTT Ingestor Scaling**:
   - When deploying multiple instances of `apps/ingestor`, configure EMQX shared subscriptions (`$share/ingestor_group/devices/+/telemetry`) so that messages are load-balanced evenly across ingestor worker replicas.
5. **CORS Restrictions**:
   - `CORS_ALLOWED_ORIGINS` defaults to `http://localhost:4200,http://localhost:80`. In production, restrict this strictly to the fully qualified domain name (FQDN) of the deployed dashboard.
