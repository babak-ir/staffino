# Staffino

Staffino is a Go backend. The first module is identity: a phone number can request a one-time code, verify it, and load the user that was created.

## Technology

| Piece | Used for |
|---|---|
| Go 1.27 | Service code |
| ConnectRPC | HTTP API. Browsers call it with JSON |
| Protocol Buffers | API contract in `proto/staffino/identity/v1` |
| Buf | Lint and Go code generation from the proto files |
| PostgreSQL 16 | Users and OTP challenges |
| pgx | PostgreSQL driver and connection pool |
| sqlc | Go query code generated from SQL |
| goose | Schema migrations in `migrations/identity` |
| Docker Compose | Local PostgreSQL |
| `log/slog` | Application logs |

The identity service does not talk to the database directly. Handlers call the service, and the service calls repository interfaces. `internal/identity/memory` is the in-memory implementation used by tests. `internal/identity/postgres` is the PostgreSQL implementation used by the server.

## Run locally

Start the database and apply the migration:

```bash
docker compose up -d postgres
goose -dir migrations/identity postgres \
  "postgres://staffino:staffino@localhost:5432/staffino?sslmode=disable" up
```

Start the API:

```bash
export DATABASE_URL="postgres://staffino:staffino@localhost:5432/staffino?sslmode=disable"
export OTP_PEPPER="local-dev-pepper"
go run ./cmd/core-api
```

The process listens on `:8080`. `GET /healthz` returns `ok`. The one-time code is printed in the server log. It expires after 2 minutes and can be used once.

```bash
curl -s localhost:8080/staffino.identity.v1.IdentityService/RequestOTP \
  -H 'Content-Type: application/json' \
  -H 'Connect-Protocol-Version: 1' \
  -d '{"phone":"+989121234567"}'
```

## Tests

```bash
go test ./internal/identity/
```
