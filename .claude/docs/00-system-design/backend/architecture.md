# Kiến trúc Dự án

## 1. Cấu trúc Thư mục

```
myapp/
├── cmd/
│   └── api/
│       └── main.go                    # Entry point, khởi tạo DI container
│
├── internal/
│   ├── auth/                          # Auth module (SSO via Prep platform)
│   │   ├── domain/                    # Entities (User, PrepUser)
│   │   ├── application/
│   │   │   ├── port/
│   │   │   │   ├── inbound.go         # AuthUseCasePort
│   │   │   │   └── outbound.go        # UserRepositoryPort, PrepUserServicePort
│   │   │   ├── dto/                   # Request/Response DTOs
│   │   │   └── usecase/               # AuthUseCase
│   │   ├── adapter/
│   │   │   ├── handler/               # HTTP handlers (Gin)
│   │   │   ├── repository/            # Postgres (User)
│   │   │   └── service/               # PrepUserService (SSO token validation, cached)
│   │   └── module.go                  # Module wiring + RegisterRoutes
│   │
│   ├── vocabulary/                    # Vocabulary module
│   │   ├── domain/                    # Entities (Vocabulary, Folder, Topic, GrammarPoint, Language, Category, ProficiencyLevel)
│   │   ├── application/
│   │   │   ├── port/
│   │   │   │   ├── inbound.go         # VocabularyCommand/QueryPort, FolderCommand/QueryPort, TopicQueryPort, GrammarPointQueryPort, LanguageQueryPort, CategoryQueryPort, ProficiencyLevelQueryPort, ImportCommandPort
│   │   │   │   └── outbound.go        # VocabularyRepositoryPort, FolderRepositoryPort, TopicRepositoryPort, etc.
│   │   │   ├── dto/                   # DTOs for vocabulary, folder, topic, grammar_point, language, category, ocr, bulk_import
│   │   │   ├── mapper/               # Entity ↔ DTO mappers (vocabulary, folder, language, topic)
│   │   │   └── usecase/               # CQRS: vocabulary_command/query, folder_command/query, topic_query, grammar_point_query, language_query, category_query, proficiency_level_query, import_command
│   │   ├── adapter/
│   │   │   ├── handler/               # HTTP handlers
│   │   │   ├── repository/            # Postgres repositories + model/ subdirectory
│   │   │   └── service/               # OCRAdapter (bridges OCR module)
│   │   └── module.go
│   │
│   ├── ocr/                           # OCR module
│   │   ├── application/
│   │   │   ├── port/
│   │   │   │   ├── inbound.go         # OCRCommandPort
│   │   │   │   └── outbound.go        # OCRServicePort
│   │   │   ├── dto/
│   │   │   ├── mapper/               # Pronunciation mapper
│   │   │   └── usecase/               # OCRCommand (retry with fallback providers)
│   │   ├── adapter/
│   │   │   └── service/               # BaiduOCR, GoogleVision, PaddleOCR, TesseractOCR
│   │   └── module.go
│   │
│   ├── learning/                      # Learning module (chưa review — DB designed, chưa implement)
│   │
│   ├── shared/                        # Shared kernel
│   │   ├── common/                    # Device detection, IP extraction, JSONB helpers
│   │   ├── error/                     # AppError (codes: NOT_FOUND, BAD_REQUEST, etc.)
│   │   ├── logger/                    # Logger interface + Field constructors
│   │   ├── ctxlog/                    # Context-aware log fields (request_id, trace_id)
│   │   ├── i18n/                      # Translation engine (en, vi)
│   │   ├── middleware/                # Auth, CORS, i18n, Logger, RateLimit, Recovery, RequestID, Security, Timeout
│   │   ├── response/                  # APIResponse helpers (Success, SuccessList, SuccessNoContent, HandleError, ValidationError, etc.)
│   │   └── dto/                       # PaginationRequest/PaginatedResponse
│   │
│   ├── server/                        # HTTP server + router
│   │   ├── router.go                  # Route registration + health check + Swagger static files
│   │   └── server.go
│   │
│   └── infrastructure/                # Cross-cutting infrastructure
│       ├── di/                        # Container (NewApp), persistence init, observability init, OCR init
│       ├── config/                    # Viper config (auth, db, redis, log, circuitbreaker, observability, ratelimit, ocr, server)
│       ├── database/                  # GORM postgres connection + custom GORM logger
│       ├── cache/                     # Generic Cache[T] — memory (ristretto), redis, multilevel, metrics, loader
│       ├── circuitbreaker/            # gobreaker wrapper + registry + config
│       ├── logging/                   # Zap adapter (console, daily file, async OTLP)
│       ├── redis/                     # Redis client init
│       ├── sentry/                    # Sentry error tracking
│       └── tracing/                   # OpenTelemetry OTLP tracer
│
├── migrations/                        # SQL migration files (golang-migrate)
│
├── resources/
│   └── i18n/                          # Translation files (en, vi)
│
├── go.mod
├── go.sum
├── Makefile
└── CLAUDE.md
```

```
HTTP Request
    ↓
[Server] router.go → module.RegisterRoutes()
    ↓
[Middleware] SecurityHeaders → CORS → RequestID → OTEL → RequestLogger → Language → Recovery
    ↓                                     (global chain)
[Route Group] Public: + RateLimit    |    Protected: + Auth
    ↓
[Adapter] handler/
    ↓  calls interface
[Application] port/inbound.go (input port)
    ↓  implemented by
[Application] usecase/
    ↓  calls interface
[Application] port/outbound.go (output port)
    ↓  implemented by
[Adapter] repository/ | service/
    ↓
Database / Redis / External Services
```

## 2. Các Lớp (Layers)

### Domain Layer (`<module>/domain/`)
Đây là lớp trong cùng, chứa các quy tắc nghiệp vụ cốt lõi.
- **Entities**: Các đối tượng có định danh (Identity): `User`, `PrepUser`, `Vocabulary`, `Folder`, `Topic`, `GrammarPoint`, `Language`, `Category`, `ProficiencyLevel`.
- **Domain Errors**: Entity-level validation errors (ví dụ: `ErrWordRequired`, `ErrFolderNameRequired`).
- **Typed IDs**: Mỗi entity có value object ID riêng (`VocabularyID`, `FolderID`, `LanguageID`, etc.).
- **UUID v7**: Tất cả entity IDs dùng `uuid.Must(uuid.NewV7())` — time-ordered, tốt cho DB indexing.
- **Đặc điểm**: Không phụ thuộc vào bất kỳ lớp nào khác bên ngoài. Không import framework, ORM, hay crypto libraries.

### Application Layer (`<module>/application/`)
Lớp này điều phối các hoạt động của ứng dụng.
- **Inbound Ports (`port/inbound.go`)**: Interfaces cho handlers gọi usecases. CQRS split: Command ports (write) và Query ports (read).
- **Outbound Ports (`port/outbound.go`)**: Interfaces cho usecases gọi repositories và services.
- **Use Cases (`usecase/`)**: Triển khai các Inbound Ports. Vocabulary module dùng CQRS (tách command/query files riêng).
- **DTOs (`dto/`)**: Data Transfer Objects với Gin binding tags cho validation (`required`, `email`, `min`, `max`).
- **Mappers (`mapper/`)**: Entity ↔ DTO conversion (vocabulary module).
- **Đặc điểm**: Chỉ phụ thuộc vào Domain Layer.

### Adapter Layer (`<module>/adapter/`)
Chứa các implementations cụ thể để kết nối Core với thế giới bên ngoài.
- **Handler (Driving)**: Nhận request từ bên ngoài. Bind JSON/Query → validate → gọi usecase qua inbound port. Validation errors trả field-level details qua `ValidationError()`.
- **Repository (Driven)**: GORM repositories implement outbound ports. Entity ↔ Model tách biệt với `toEntity()`/`fromEntity()`. DB models in `repository/model/`.
- **Service (Driven)**: External service adapters — PrepUserService (SSO), OCR providers, cross-module adapters (OCRAdapter).
- **Đặc điểm**: Phụ thuộc vào Application Layer (implement các Ports).

### Infrastructure Layer (`internal/infrastructure/`)
Cung cấp các công cụ và cấu hình để chạy ứng dụng.
- **Config**: Load biến môi trường qua Viper từ `.env`. Tách file per concern (auth, database, redis, server, ratelimit, ocr, etc.).
- **Database**: GORM postgres connection + custom GORM logger (slow query detection >200ms).
- **Cache**: Generic `Cache[T]` — memory (ristretto), Redis, multilevel. Includes metrics và singleflight loader.
- **DI**: `container.go` → `initPersistence()` → `initOCR()` → module factories. Manual constructor injection.
- **Observability**: Zap structured logging + OTEL tracing + Sentry error tracking.
- **Resilience**: Circuit breaker (gobreaker) registry cho external service calls. In-memory per-process, không distributed.

### Deployment & Scaling
- Dự án sẽ **horizontal scaling** với **Kubernetes replicas** và **load balancer**.
- **Rate limiter**: Redis-backed token bucket (Lua script) — distributed, works correctly across instances.
- **Circuit breaker**: In-memory per-instance (gobreaker) — mỗi instance tự bảo vệ, không cần share state.
- **Cache**: Configurable via `CACHE_LEVEL` — L1 (memory only), L2 (Redis only), multi (L1 + L2).
- **Stateful components**: Khi thêm component in-memory mới, luôn cân nhắc multi-instance scenario.

### Shared Kernel (`internal/shared/`)
Code dùng chung giữa các modules.
- **AppError**: Error codes layered: entity errors → `AppError` (usecase) → HTTP status + i18n key (handler).
- **Response**: `Success()`, `SuccessList()` (pagination), `SuccessNoContent()`, `ValidationError()` (field-level validation errors), `HandleError()` (AppError mapping).
- **Middleware**: SecurityHeaders, CORS, RequestID (UUID v7), OTEL, RequestLogger, Language (i18n detection), Recovery (panic → Sentry), Auth (JWT via Prep SSO), RateLimit (Redis token bucket), Timeout.
- **Common**: Device detection, IP extraction, JSONB helpers.

---

## 3. Vòng đời của một API Request (Request Lifecycle)

**Ví dụ: Tạo từ vựng mới (Create Vocabulary)**

1.  **Client Request**:
    - Client gửi HTTP POST request tới `/api/v1/vocabularies` với JSON body.
    - Request đi kèm Header `Authorization: Bearer <prep_token>`.

2.  **Infrastructure (Server)**:
    - `http.Server` nhận request.
    - Request đi qua **Router** (`gin.Engine`) và middleware chain.

3.  **Global Middleware Chain**:
    - `SecurityHeadersMiddleware` — set security headers.
    - `CORSMiddleware` — handle CORS.
    - `RequestIDMiddleware` — gán UUID v7 request ID, propagate qua context.
    - `otelgin.Middleware` — OpenTelemetry tracing (nếu OTLP configured).
    - `RequestLoggerMiddleware` — log request/response (skip sensitive paths).
    - `LanguageMiddleware` — detect ngôn ngữ từ query/header.
    - `RecoveryMiddleware` — catch panic, report Sentry.

4.  **Route Group Middleware**:
    - Protected routes: `AuthMiddleware` — validate Prep SSO token, upsert user, set `user_id` vào context.

5.  **Adapter Layer (Handler)**:
    - **Handler** (`VocabularyHandler.CreateVocabulary`) nhận request.
    - **Binding/Validation**: `ShouldBindJSON(&req)` validate DTO.
    - Nếu dữ liệu sai → `ValidationError(c, err)` trả field-level details.
    - Nếu dữ liệu đúng → Gọi xuống Application Layer thông qua Inbound Port.

6.  **Application Layer (Use Case)**:
    - **Use Case** (`VocabularyCommand.CreateVocabulary`) nhận DTO.
    - **Business Logic**:
        - Chuyển đổi DTO sang Domain Entity (validate, generate UUID v7).
        - Entity validate trả entity errors → usecase map sang `AppError`.
    - Gọi xuống Persistence Layer thông qua Outbound Port `VocabularyRepositoryPort`.

7.  **Adapter Layer (Repository)**:
    - **Repository** (`VocabularyRepository`) nhận Entity.
    - **Mapping**: `fromEntity()` chuyển Domain Entity sang DB Model.
    - **Database Execution**: GORM INSERT vào PostgreSQL.
    - **Timestamp sync**: GORM-managed CreatedAt/UpdatedAt sync back vào entity.
    - Trả về kết quả cho Use Case.

8.  **Application Layer (Use Case) - Trả về**:
    - Nhận kết quả từ Repository.
    - Nếu thành công → Chuyển đổi Entity sang Response DTO (via mapper).
    - Trả DTO về cho Handler.

9.  **Adapter Layer (Handler) - Response**:
    - Handler nhận DTO từ Use Case.
    - `response.Success(c, 201, res)` — serialize thành JSON, translate message qua i18n.
    - Gửi HTTP Response về Client.

### Sơ đồ luồng dữ liệu

```
Client (HTTP)
   │
   ▼
[Infrastructure] HTTP Server / Router
   │
   ▼
[Middleware] SecurityHeaders → CORS → RequestID → OTEL → Logger → Language → Recovery
   │                                (global)
   ▼
[Route Group] Public: + RateLimit    |    Protected: + Auth
   │
   ▼
[Adapter] Handler (ShouldBindJSON → ValidationError if fail)
   │         (DTO)
   ▼
[Application] Use Case (Business Logic, Error mapping)
   │         (Entity)
   ▼
[Adapter] Repository (Entity ↔ Model mapping)
   │         (DB Model)
   ▼
[Infrastructure] Database (PostgreSQL) / Redis / External Services (Prep SSO, OCR providers)
```
