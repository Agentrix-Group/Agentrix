# Agentrix Backend Server

## Overview

Agentrix is a multi-agent competitive gaming and bot evaluation platform built with Go and MySQL. It orchestrates automated matches between participant code submissions, provides sandboxed execution, manages contests, tracks leaderboard rankings, and generates match replays viewable on the web.

## Architecture

Agentrix follows a layered modular architecture inspired by clean architecture and the Capibara reference design:

```text
main.go
  ├─ config       → Environment configuration (Server, Database, Artifacts, Queue)
  ├─ tracer       → Structured logging with Zap and correlation context
  ├─ connection   → MySQL database, artifact storage, and match job queue
  ├─ repository   → Database access layer (CRUD and queries for all domain entities)
  ├─ service      → Domain business logic, validation, authentication, and orchestration
  ├─ executor     → Match execution, background worker pool, and bot sandbox
  ├─ game         → Game engine registry, validation, and Arena Basica simulation
  ├─ replay       → Replay recorder and client projection
  └─ server       → Gorilla mux HTTP server with RBAC middleware & REST handlers
```

## Prerequisites (Ubuntu / Linux)

- Go 1.25 or higher
- MySQL 8.0+ / MariaDB
- Make build tool
- Node.js 18+ (for Web frontend)

### Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `MODE` | Execution mode (`dev`, `gcp`, `railway`) | `dev` |
| `PORT` | HTTP Server port | `8080` |
| `DB_USER` | MySQL user | `root` |
| `DB_PASSWORD` | MySQL password | `` |
| `DB_HOST` | MySQL host | `localhost` |
| `DB_PORT` | MySQL port | `3306` |
| `DB_NAME` | MySQL database name | `agentrix` |
| `ACCESS_SECRET` | JWT Access Token Secret | `agentrix-access-secret-key-change-in-prod` |
| `REFRESH_SECRET` | JWT Refresh Token Secret | `agentrix-refresh-secret-key-change-in-prod` |
| `SESSION_SECRET` | Gorilla Cookie Store Secret | `agentrix-session-secret-key-change-in-prod` |
| `ARTIFACTS_DIR` | Directory for submission code & replays | `./artifacts` |

## Quick Start

### 1. Setup Dependencies
```bash
make agentrix-setup
```

### 2. Setup Database
```bash
make db-setup
```

### 3. Run Tests
```bash
make test
```

### 4. Build and Run Server
```bash
make build
make run
```

## API Documentation

The OpenAPI 3.1 specifications are organized under `open-api/`:
- `open-api/openapi.yaml`: Master API specification
- Resources: `auth.yaml`, `contests.yaml`, `categories.yaml`, `participants.yaml`, `games.yaml`, `agents.yaml`, `submissions.yaml`, `matches.yaml`, `results.yaml`, `rankings.yaml`, `replays.yaml`, `permissions.yaml`

## Contracts and Games

- `contracts/`: JSON Schemas defining the agent protocol, game manifest, and replay serialization.
- `games/arena-basica/`: Canonical 2-4 bot battle game with manifest, engine, renderer, and example bots.
- `web/`: Web dashboard and replay viewer.
