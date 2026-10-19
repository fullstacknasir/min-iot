# minIoT Multi-Device Telemetry Simulator

The minIoT Device Simulator is a CLI utility that authenticates with the minIoT REST API, dynamically provisions simulated devices with per-device cryptographic tokens, and concurrently streams realistic temperature and humidity telemetry over MQTT into the EMQX broker.

---

## Features

- **Automated Login**: Authenticates against the minIoT API (`POST /api/v1/auth/login`) with organization admin credentials to obtain a short-lived JWT.
- **Dynamic Provisioning**: Automatically registers devices (`POST /api/v1/devices`) and provisions unique per-device access secrets (`POST /api/v1/devices/{id}/credentials`).
- **Concurrent Multi-Device Simulation**: Supports spinning up multiple independent MQTT publisher sessions concurrently using the `-devices` or `-n` CLI flag.
- **Strict Topic & Schema Compliance**: Publishes to `devices/{deviceId}/telemetry` with JSON conforming to the platform's JSON Schema validator.
- **Realistic Telemetry Generation**: Emits temperature and humidity metrics with baseline offsets per device and natural jitter.
- **Graceful Shutdown**: Traps `SIGINT`/`SIGTERM` to cleanly disconnect all MQTT client sessions.

---

## CLI Options & Flags

| Flag | Shorthand | Default | Description | Environment Fallback |
| :--- | :--- | :--- | :--- | :--- |
| `-devices` | `-n` | `1` | Number of concurrent simulated devices | — |
| `-interval` | — | `3` | Publishing interval in seconds | — |
| `-prefix` | — | `sim-sensor` | Device ID prefix (e.g. `sim-sensor-001`) | — |
| `-duration` | — | `0s` | Total run duration (e.g. `30s`, `5m`, or `0s` for indefinite) | — |
| `-api` | — | `http://localhost:8080/api/v1` | minIoT REST API base URL | `API_URL` |
| `-broker` | — | `tcp://localhost:1883` | EMQX MQTT broker TCP URI | `EMQX_BROKER` |
| `-email` | — | `admin@miniot.io` | Admin email for API authentication | `API_EMAIL` |
| `-password`| — | `AdminPassword123!` | Admin password for API authentication | `API_PASSWORD` |

---

## Usage Examples

### Prerequisites

Ensure the infrastructure and API are running:
```bash
./start.sh          # Starts TimescaleDB, Redis, EMQX
make seed           # Applies schema migrations and seeds default admin
make dev-api        # Runs API on http://localhost:8080
```

### 1. Run with Default Single Device (Indefinite)
```bash
cd tools/device-simulator
go run main.go
```
Or from the project root:
```bash
make simulate
```

### 2. Run Multiple Devices (e.g. 5 Devices)
```bash
cd tools/device-simulator
go run main.go -devices 5
```
Or via Makefile:
```bash
make simulate DEVICES=5
```

### 3. Customize Interval and Timed Execution
Simulate 10 devices publishing every 2 seconds for a total test duration of 60 seconds:
```bash
cd tools/device-simulator
go run main.go -n 10 -interval 2 -duration 60s
```
Or via Makefile:
```bash
make simulate DEVICES=10 INTERVAL=2
```

### 4. Custom Device Prefix & Credentials
```bash
cd tools/device-simulator
go run main.go -prefix warehouse-sensor -devices 3 -email admin@miniot.io -password AdminPassword123!
```

---

## Verification

When the simulator is running, you can monitor ingested data across the platform:

1. **PostgreSQL / TimescaleDB Hypertable**:
   ```bash
   docker exec miniot-timescaledb psql -U postgres -d miniot -c \
     "SELECT time, received_at, device_id, metric_name, metric_value FROM telemetry WHERE device_id LIKE 'sim-sensor%' ORDER BY time DESC LIMIT 10;"
   ```

2. **Redis Latest Telemetry Cache**:
   ```bash
   docker exec miniot-redis redis-cli get "device:sim-sensor-001:latest"
   ```

3. **EMQX Dashboard**:
   Open [http://localhost:18083](http://localhost:18083) (credentials: `admin` / `public`) to observe active client connections and publish throughput.
