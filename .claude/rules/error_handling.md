# Error Handling

## Flow

```
Repo (raw error / nil) → Use case (wraps into AppError) → Handler (response.HandleError) → HTTP status + i18n
```

## AppError Constructors

**4xx — no cause:**

| Constructor | When |
|---|---|
| `BadRequest(key)` 400 | Invalid input, malformed JSON, FK violation |
| `Unauthorized(key)` 401 | Auth failed or missing |
| `Forbidden(key)` 403 | Authenticated but not allowed |
| `NotFound(key)` 404 | Resource doesn't exist |
| `Conflict(key)` 409 | Duplicate / already exists |
| `ValidationFailed(key)` 422 | Validation failure, domain entity errors |
| `TooManyRequests(key)` 429 | Rate limit or quota exceeded |

**5xx — always carry cause:**

| Constructor | When |
|---|---|
| `InternalServerError(key, err)` 500 | Unexpected system errors only |
| `ServiceUnavailable(key, err)` 503 | External service down, circuit breaker open |

Never use 500 as catch-all. FK violations → 400. Domain validation → 422.

## Granular Business Codes

Use `WithCode()` + `WithData()` for business-specific errors:

```go
// Feature not entitled (403 with granular code)
apperr.Forbidden("entitlement.not_entitled").
    WithCode("FEATURE_NOT_ENTITLED").
    WithData(map[string]any{
        "feature":      "ocr_scan",
        "current_plan": "free",
        "upgrade_cta":  "entitlement.upgrade_to_pro",
    })

// Quota exceeded (429 with granular code)
apperr.TooManyRequests("entitlement.quota_exceeded").
    WithCode("QUOTA_EXCEEDED").
    WithData(map[string]any{
        "feature":   "ocr_scan",
        "limit":     10,
        "used":      10,
        "remaining": 0,
        "resets_at": "2026-03-30T00:00:00Z",
    })
```

## Key: i18n key, not English

```go
// Bad
apperr.InternalServerError("failed to save", err)
// Good
apperr.NotFound("vocabulary.not_found")
```

## Repository Error Convention

Repositories return raw errors only, never AppError.

- **Not found** → `(nil, nil)`. Use case checks `if result == nil` → `apperr.NotFound(key)`.
- **Other errors** → raw error. Use case wraps → `apperr.InternalServerError(key, err)`.

## Domain Validation Errors

Domain uses `errors.New()` sentinels. Use cases map via mapper → `ValidationFailed(key)`.

## AppError Methods

- `Error()` → `"NOT_FOUND: vocabulary.not_found"` (debug/logs)
- `Message()` → `"vocabulary.not_found"` (i18n key for response)
- `Unwrap()` → cause (5xx only)
- `WithCode(code)` → override error code for granular business errors
- `WithData(map)` → attach structured details (rendered as `error.details` in response)
