# Distributed Object Storage API

A production-grade, distributed file storage API built with Go, implementing Clean Architecture. This project provides a robust system for managing chunked file uploads directly to MinIO (S3-compatible) and handles event-driven background processing with NATS.

## Key Features

- **Direct-to-Storage Uploads**: API generates pre-signed PUT URLs for clients to upload file chunks directly to MinIO, reducing server memory usage and bandwidth.
- **Secure Downloads**: Generates time-limited pre-signed GET URLs for authenticated users to access files.
- **Multipart Management**: Tracks the status of individual file chunks in PostgreSQL and manages the upload lifecycle.
- **Event-Driven Processing**: NATS message queues trigger background workers for tasks like chunk assembly after uploads complete.
- **Clean Architecture**: Strong separation between Handler, Service, and Repository layers using PostgreSQL (`sqlc` + `pgx`).

## Tech Stack
- **Go 1.22+**, **Echo v4** (HTTP Framework)
- **PostgreSQL 16** (`pgx/v5`, `sqlc`)
- **MinIO** (S3 Storage)
- **NATS** (Message Queue)
- **Uber Zap** (Logging), **JWT** (Auth)


## How to Run

1. **Start Infrastructure**:
   ```bash
   docker-compose up -d
   ```

2. **Setup Config**:
   ```bash
   cp .env.example .env
   ```

3. **Run Migrations**:
   ```bash
   migrate -path migrations -database "postgres://objectstorage:objectstorage@localhost:5432/objectstorage?sslmode=disable" up
   ```

4. **Start the API Server**:
   ```bash
   go run cmd/main.go
   ```
