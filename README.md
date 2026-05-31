# Aegis AI-Powered API Firewall

Aegis is a language-agnostic AI API Firewall that sits in front of backend applications to inspect, score, filter, and protect incoming requests.

## Architecture
- **Data Plane (Proxy):** Go
- **AI Engine:** Python
- **Control Plane API:** Go
- **Frontend Dashboard:** React + TypeScript
- **Infrastructure:** PostgreSQL, Redis, NATS

## Prerequisites
- Docker & Docker Compose
- Go 1.21+
- Python 3.11+
- Node.js 18+

## Quick Start (Local Development)

1. **Start the Infrastructure**
   ```bash
   make up
   ```
   This will start PostgreSQL (5432), Redis (6379), and NATS (4222) in the background.

2. **Check Status**
   ```bash
   docker ps
   ```
   Verify that `aegis_postgres`, `aegis_redis`, and `aegis_nats` are running and healthy.

3. **Stop the Infrastructure**
   ```bash
   make down
   ```

4. **Clean Volumes (WARNING: Wipes local DB/Cache data)**
   ```bash
   make clean
   ```

## Development

- Run `make test` to run all Go tests across the workspace.
- Run `make mod-tidy` to tidy Go modules.
