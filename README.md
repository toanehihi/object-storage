# ObjectStorage

Distributed File Storage API built with Go, Echo v4, PostgreSQL, MinIO, and NATS.

## Architecture

```
Client → API Server (Echo v4) → PostgreSQL + MinIO + NATS
                                                    ↓
                                             Worker (NATS consumer)
```

## Quick Start

```bash
# Start infrastructure
make docker-up

# Run database migrations
make migrate-up

# Start server
make run
```

## Project Structure

```
cmd/                   # Application entry point
  main.go

config/                # Environment-based configuration
  config.go

internal/              # Private application code
  handler/             # HTTP transport layer (Echo handlers)
  service/             # Business logic / use cases
  repository/          # Data access layer (interfaces + PostgreSQL)
  middleware/          # HTTP middleware (JWT auth)
  token/               # JWT token manager
  model/               # Domain entities (User, File, FileChunk)

sqlc/                  # Generated database queries (sqlc)
  query/               # SQL query definitions

pkg/                   # Shared infrastructure clients
  postgres/            # PostgreSQL connection pool
  minio/               # MinIO / S3 client
  nats/                # NATS messaging client

migrations/            # SQL migration files
```

## Auth Endpoints

| Method | Endpoint               | Auth     | Description              |
|--------|------------------------|----------|--------------------------|
| POST   | `/api/v1/auth/register`| No       | Register new user        |
| POST   | `/api/v1/auth/login`   | No       | Login with credentials   |
| POST   | `/api/v1/auth/refresh` | No       | Refresh token pair       |
| GET    | `/api/v1/auth/me`      | Bearer   | Get current user profile |
| GET    | `/health`              | No       | Health check             |

## Configuration

Copy `.env.example` to `.env` and adjust values:

```bash
cp .env.example .env
```

See [.env.example](.env.example) for all available environment variables.

## Tech Stack

- **Router**: [Echo v4](https://echo.labstack.com/)
- **Database**: PostgreSQL via [pgx/v5](https://github.com/jackc/pgx)
- **Query Gen**: [sqlc](https://sqlc.dev/)
- **Object Storage**: MinIO / S3
- **Messaging**: NATS
- **Auth**: JWT via [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- **Logging**: [zap](https://github.com/uber-go/zap)
- **Config**: [caarlos0/env](https://github.com/caarlos0/env)

## API Documentation

See [CONTRACT.md](CONTRACT.md) for the full API contract.
