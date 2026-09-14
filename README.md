# Agentrix Platform

## Overview

Agentrix is a multi-agent competitive gaming and bot evaluation platform built with Go, PostgreSQL, and React. It orchestrates automated matches between participant code submissions, provides sandboxed execution with slot-isolated perception, manages tournaments and contests, tracks leaderboard rankings, and generates match replays viewable on the web canvas.

## Architecture

Agentrix follows a layered modular architecture adhering to architectural decisions DP-001, DP-002, and ATD-015:

```text
main.go
  ├─ config       → Environment configuration (Server, PostgreSQL Database, Artifacts, Queue)
  ├─ tracer       → Structured logging with Zap and correlation context
  ├─ connection   → PostgreSQL database (pgx), artifact storage, and match job queue
  ├─ repository   → Segregated database access layer (CRUD and queries for domain entities)
  ├─ service      → Domain business logic, validation, authentication, and orchestration
  ├─ executor     → Match execution, background worker pool, and isolated bot sandbox
  ├─ game         → Game engine registry, validation, and Arena Basica simulation
  ├─ replay       → Replay recorder, sealed frames, and client projection
  ├─ server       → Gorilla mux HTTP server with RBAC middleware & REST handlers
  └─ web          → React + Vite frontend dashboard and HTML5 canvas replay viewer
```

## Prerequisites (Ubuntu / Linux)

- Go 1.25 or higher
- PostgreSQL 14+
- Python 3.10+ (for bot script execution)
- Make build tool
- Node.js 18+ (for Web frontend)

### Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `MODE` | Execution mode (`dev`, `gcp`, `railway`) | `dev` |
| `PORT` | HTTP Server port | `8080` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_NAME` | PostgreSQL database name | `agentrix` |
| `ACCESS_SECRET` | JWT Access Token Secret | `agentrix-access-secret-key-change-in-prod` |
| `REFRESH_SECRET` | JWT Refresh Token Secret | `agentrix-refresh-secret-key-change-in-prod` |
| `SESSION_SECRET` | Cookie Session Secret | `agentrix-session-secret-key-change-in-prod` |
| `ARTIFACTS_DIR` | Directory for submission code & replays | `./artifacts` |

## Quick Start

### 1. Setup Go Dependencies
```bash
make agentrix-setup
```

### 2. Setup PostgreSQL Database
```bash
make db-setup
```

### 3. Run Backend Tests & Linter
```bash
make test
make lint
```

### 4. Build and Run Server
```bash
make build
make run
```

### 5. Frontend Web Dashboard (React + Vite)
```bash
# Install frontend dependencies
make web-install

# Start development server with API proxy
make web-dev

# Build production bundle
make web-build
```

## API Documentation

The OpenAPI 3.1 specifications are organized under `open-api/`:
- `open-api/openapi.yaml`: Master API specification
- Resources: `auth.yaml`, `contests.yaml`, `categories.yaml`, `participants.yaml`, `games.yaml`, `agents.yaml`, `submissions.yaml`, `matches.yaml`, `results.yaml`, `rankings.yaml`, `replays.yaml`, `permissions.yaml`

## Contracts and Games

- `contracts/`: JSON Schemas defining the agent protocol, game manifest, and replay serialization.
- `games/arena-basica/`: Canonical 2-4 bot battle game with manifest, engine, renderer, and example bots (`bot_hunter.py`, `bot_random.py`).
- `web/`: Modern React dashboard and HTML5 canvas replay viewer.
