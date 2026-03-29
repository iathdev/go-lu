# Coding Style

## Naming

- **No single-character receivers.** Use meaningful names: `profile`, `repo`, `handler` — not `p`, `r`, `h`.
- **No abbreviations** except: `ctx`, `err`, `db`, `cfg`, `id`, `req`, `res`, `tx`.

## Logging

All log messages MUST include module prefix: `[AUTH]`, `[VOCABULARY]`, `[OCR]`, `[SERVER]`.

```go
logger.Warn(ctx, "[VOCABULARY] error fetching topics", zap.Error(err))
```

### When to log

- **4xx errors**: Do NOT log. Just return `AppError`.
- **5xx errors**: Log when the context is important for debugging (external service failures, unexpected states). Not every 5xx needs a log — simple DB errors are already visible in GORM logger.
- **External service calls**: Log failures with extra context (status code, endpoint) before returning error.
- **Infrastructure/startup**: Log lifecycle events (`Info`) and init failures (`Warn`).
