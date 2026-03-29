# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API using **Modular Monolith** with **Hexagonal Architecture (Ports and Adapters)** per module. Built with Gin, GORM, and PostgreSQL. Module name: `learning-go`. Requires **Go 1.25+**.

Deployment target: **horizontal scaling** with Kubernetes replicas and load balancer. Do not assume single instance — stateful in-memory components (rate limiter, cache) must be evaluated for multi-instance correctness.

## Commands

```bash
make docker-up          # Start PostgreSQL + Redis via Docker Compose
make docker-down        # Stop Docker Compose services
make run                # Run the API server (go run cmd/api/main.go)
make build              # Build binary to bin/api
make migrate-up         # Run all pending migrations (requires golang-migrate CLI)
make migrate-down       # Roll back 1 migration (use make migrate-down-N for N steps)
make migrate-reset      # Drop all tables
go test ./...           # Run all tests
go test ./internal/auth/domain/...         # Run tests for a specific package
go test -run TestName ./internal/...       # Run a single test by name
```

Requires a `.env` file (copy from `.env.example`). The Makefile reads `.env` for DB connection vars. Migration commands require the [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI.

## Architecture

**Request flow:** HTTP Router -> Module.RegisterRoutes() -> Handler -> Input Port (interface) -> Use Case -> Output Port (interface) -> Repository -> Database

### Module Structure

Each module follows the same internal layout:

```
internal/<module>/
├── docs/                # Module-specific documentation (requirement, API contract, plan)
├── domain/              # Entities and domain errors. Zero external dependencies.
├── application/
│   ├── port/            # Input (driving) and output (driven) port interfaces
│   ├── dto/             # Data transfer objects for the module
│   └── usecase/         # Use case implementations
├── adapter/
│   ├── handler/         # HTTP handlers (Gin)
│   ├── repository/      # Repository implementations (Postgres, Redis)
│   └── security/        # Module-specific security (e.g., JWT in auth)
└── module.go            # Module wiring + RegisterRoutes(public, protected)
```

### Modules

- **`internal/auth/`** — Auth subdomain (SSO login via Prep platform, profile). Uses Prep User Service for SSO (`adapter/service/`). Currently the only implemented module.
- **`internal/vocabulary/`** — *(planned)* Vocabulary subdomain (CRUD vocabularies, folders, topics, grammar points). Will use CQRS pattern.
- **`internal/ocr/`** — *(planned)* OCR subdomain.
- **`internal/shared/`** — Shared kernel: AppError, logger, i18n (en/vi/zh/th/id), middleware, unified response formatting, shared DTOs.
- **`internal/server/`** — HTTP server, router, middleware chain registration.
- **`internal/infrastructure/`** — DI container, config (Viper), database, Redis, cache (multi-level), circuit breaker (gobreaker), logging (Zap), Sentry, OpenTelemetry tracing.

### Key Patterns

- **Module boundary**: Module A must NOT import internal packages of Module B. Cross-module communication goes through exported ports/interfaces.
- **Module registration**: Each module exposes `NewModule(deps...) *Module` and `RegisterRoutes(public, protected *gin.RouterGroup)`. DI container creates modules; router calls RegisterRoutes.
- **Domain entities vs DB models**: Domain entities in `<module>/domain/`, DB models in `<module>/adapter/repository/`. Repositories map between them via `toEntity()`/`fromEntity()`.
- **CQRS**: New feature modules should split use cases into `*Command` and `*Query` types with corresponding port interfaces (planned for vocabulary module).
- **Circuit breaker**: gobreaker v2 in `infrastructure/circuitbreaker/` with `BreakerRegistry`. In-memory per-process (not distributed). Converts `ErrOpenState`/`ErrTooManyRequests` to `apperr.ServiceUnavailable()`. Only `nil` and `ErrNotFound` count as success. Configurable via `CB_*` env vars.
- **Error handling**: `AppError` in `shared/error/` carries a typed `Code`. `response.HandleError(c, err)` maps code to HTTP status. All error messages are i18n keys translated at response layer.
- **i18n**: Language detected from `lang` query param > `X-Lang` header > `Accept-Language` header. Translation files in `resources/i18n/<lang>/<domain>.json`. Falls back to English, then raw key.
- **Middleware chain**: SecurityHeaders -> CORS -> RequestID -> OTEL -> RequestLogger -> Language -> Recovery. Rate limiting on public routes. JWT auth on `/api/*` routes.
- **DI**: Manual constructor injection in `infrastructure/di/`. No framework. Returns cleanup function for graceful shutdown.
- **Testing**: Colocated `*_test.go`. Table-driven tests with `t.Run()`. No mock framework — tests focus on domain logic.
- **Migrations**: SQL files in `migrations/` named `NNNNNN_description.{up,down}.sql` (6-digit zero-padded prefix).

### Adding a New Module

1. Create `internal/<module>/` with the standard layout (domain, application/{port,dto,usecase}, adapter/{handler,repository}, module.go)
2. In `module.go`: wire internal dependencies in `NewModule()`, register routes in `RegisterRoutes()`
3. In `infrastructure/di/container.go`: create the module and pass to `server.NewRouter()`
4. In `server/router.go`: add module parameter and call `module.RegisterRoutes(public, api)`

### Cache

Generic `Cache[T]` interface in `infrastructure/cache/` with three modes configured via `CACHE_LEVEL` env var:
- **`L1`** — In-memory only (ristretto). Fast but per-process, not shared across K8s replicas.
- **`L2`** — Redis only. Shared across instances.
- **`multi`** — Two-level (L1 + L2). L1 as hot cache, L2 as backing store.

### Rate Limiting

Redis-backed token bucket (`shared/middleware/ratelimit.go`) using Lua scripts for atomic operations. Distributed — works correctly across multiple K8s replicas.

## API Routes

Currently implemented:
- `GET /api/me` — Protected, returns authenticated user profile
- `GET /health` — Health check

Planned (not yet implemented): register, login, refresh, logout, vocabulary CRUD, folders.

## Documentation

Design docs in `.claude/docs/`

## Rules

- **Coding Style**: See [`.claude/rules/coding_style.md`](.claude/rules/coding_style.md)
- **Error Handling**: See [`.claude/rules/error_handling.md`](.claude/rules/error_handling.md)
- **API Response Contract**: See [`.claude/rules/api_response.md`](.claude/rules/api_response.md)
- **Planning Rules**: See [`.claude/rules/planning_rules.md`](.claude/rules/planning_rules.md)
