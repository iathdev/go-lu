# Danh sách Công nghệ và Thư viện (Tech Stack)

Tài liệu này liệt kê các công nghệ, công cụ và thư viện quan trọng được sử dụng trong dự án.

## 1. Công nghệ Cốt lõi (Core Technologies)

| Công nghệ | Phiên bản | Mô tả |
| :--- | :--- | :--- |
| **Go (Golang)** | 1.25+ | Ngôn ngữ lập trình chính, hiệu năng cao, concurrency tốt. |
| **PostgreSQL** | 15+ | Hệ quản trị cơ sở dữ liệu quan hệ (RDBMS) chính. |
| **Redis** | 7+ | In-memory data store cho token cache, rate limiting, L2 cache. |
| **Docker** | Latest | Containerization platform để đóng gói và chạy ứng dụng. |
| **Docker Compose** | Latest | Công cụ định nghĩa và chạy multi-container Docker applications. |

## 2. Thư viện Go Quan trọng (Key Go Packages)

### Web Framework
- **[Gin Gonic](https://github.com/gin-gonic/gin)** (`github.com/gin-gonic/gin`):
  - Web framework high-performance cho Go.
  - HTTP routing, middleware chain, JSON binding/validation.
  - Validation powered by `go-playground/validator/v10`.

### Database & ORM
- **[GORM](https://gorm.io/)** (`gorm.io/gorm`):
  - ORM phổ biến nhất cho Go.
  - Entity ↔ Model mapping, soft delete (`gorm.DeletedAt`), auto timestamps.
- **[GORM Postgres Driver](https://github.com/go-gorm/postgres)** (`gorm.io/driver/postgres`):
  - Driver kết nối PostgreSQL.
- **[Redis Go](https://github.com/redis/go-redis)** (`github.com/redis/go-redis/v9`):
  - Redis client cho Prep SSO token cache, distributed rate limiting (Lua scripts), L2 cache.

### Caching
- **[Ristretto](https://github.com/dgraph-io/ristretto)** — In-memory cache (L1). Dùng trong generic `Cache[T]` system.
- Generic `Cache[T]` interface với 3 mode: L1 (memory), L2 (Redis), multi (L1 + L2). Includes metrics tracking và singleflight loader.

### Configuration
- **[Viper](https://github.com/spf13/viper)** (`github.com/spf13/viper`):
  - Configuration management — load từ `.env` file.

### Authentication
- **Prep SSO**: Auth qua Prep platform — validate token via external API, upsert user locally.
- Token cache dùng generic `Cache[T]` (ristretto/Redis) với circuit breaker.
- Không dùng local JWT/bcrypt — authentication hoàn toàn delegate cho Prep platform.

### Observability
- **[Uber Zap](https://github.com/uber-go/zap)** (`go.uber.org/zap`):
  - Structured logging (JSON format). Multiple channels: console, daily file rotation, async OTLP.
- **[OpenTelemetry](https://opentelemetry.io/)** (`go.opentelemetry.io/otel`):
  - Distributed tracing via OTLP gRPC exporter.
  - Auto-inject trace_id/span_id vào logs.
- **[Sentry Go](https://github.com/getsentry/sentry-go)** (`github.com/getsentry/sentry-go`):
  - Error tracking và panic capture.
- **[Lumberjack](https://github.com/natefinch/lumberjack)** (`gopkg.in/natefinch/lumberjack.v2`):
  - Log file rotation (configurable max size, backups, age).

### Resilience
- **[GoBreaker](https://github.com/sony/gobreaker)** (`github.com/sony/gobreaker`):
  - Circuit breaker pattern cho external service calls (Prep SSO, OCR providers).
  - Registry-based: mỗi service có config riêng.

### Utilities
- **[Google UUID](https://github.com/google/uuid)** (`github.com/google/uuid` v1.6.0):
  - UUID v7 (time-ordered) cho entity IDs và request IDs.
  - Tốt cho DB indexing (B-tree friendly), chứa timestamp.
- **[golang-migrate](https://github.com/golang-migrate/migrate)**:
  - Database migration CLI tool.

## 3. Kiến trúc (Architecture)

- **CQRS**: Command/Query split cho Vocabulary module. Planned cho Learning module.
- **Ports split**: `inbound.go` (driving — handlers gọi usecases) và `outbound.go` (driven — usecases gọi repositories/services).
- **Dependency Injection**: Manual constructor injection qua DI container. Constructors trả về port interfaces.
- **i18n**: 2 ngôn ngữ hiện tại (en, vi). Architecture hỗ trợ mở rộng thêm (th, zh, id).
- **Cross-module communication**: Via exported port interfaces. Ví dụ: Vocabulary module dùng OCR module qua `OCRScannerPort` adapter.

## 4. Modules hiện tại

| Module | Trạng thái | Mô tả |
| :--- | :--- | :--- |
| **auth** | ✅ Implemented | SSO login via Prep platform, user profile |
| **vocabulary** | ✅ Implemented | CRUD vocabularies, folders, topics, grammar points, categories, proficiency levels, OCR scan, bulk import |
| **ocr** | ✅ Implemented | Multi-provider OCR (Baidu, Google Vision, PaddleOCR, Tesseract) with retry/fallback |
| **learning** | ❌ Not started | Learning progress, SM-2 scoring, learning modes (DB designed, chưa implement) |

## 5. Công cụ Phát triển (Development Tools)

- **Makefile**: Tự động hóa (run, build, docker-up, migrate).
- **Postman / cURL**: Test API.
- **Docker Compose**: Local development (PostgreSQL, Redis).
- **Swagger UI**: API docs served at `/docs` (static HTML + OpenAPI YAML).
