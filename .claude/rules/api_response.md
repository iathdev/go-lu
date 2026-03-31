# API Response Contract

All endpoints MUST use the unified response envelope defined below. No exceptions.

## Envelope

```
{
  "success": bool,
  "data":    <T>,              // success only, omitted on error
  "error":   <ErrorObject>,    // error only, omitted on success
  "meta":    <Meta>            // always present
}
```

- `data` and `error` are **mutually exclusive** — present on their respective case, **omitted** (not null) on the other.

## Success

```go
// Single resource
response.Success(c, http.StatusOK, data)
response.Success(c, http.StatusCreated, data)

// Paginated list — pagination goes into meta, data is the items array
response.SuccessList(c, items, response.PaginationMeta{Total: t, Page: p, PageSize: ps, TotalPages: tp})

// No body (DELETE, side-effect actions)
response.SuccessNoContent(c)
```

## Errors

### From use cases (via AppError)

```go
response.HandleError(c, err)   // maps AppError → {success:false, error:{code,message,details?}, meta}
```

### Direct (middleware / router, no AppError)

```go
response.Unauthorized(c, "auth.unauthorized")
response.NotFound(c, "common.route_not_found")
response.BadRequest(c, "common.bad_request")
response.TooManyRequests(c, "common.too_many_requests")
response.ValidationError(c, bindingErr)   // 422 with field-level details
```

## Error Object

```json
{
  "code":    "NOT_FOUND",
  "message": "Không tìm thấy từ vựng",
  "details": {}
}
```

- `code` — machine-readable, SCREAMING_SNAKE_CASE
- `message` — human-readable, i18n-translated
- `details` — omitted for simple errors; shape depends on `code`

Error codes and constructors are defined in [`error_handling.md`](error_handling.md).

## Meta

Always present. Pagination fields appear **only** for list endpoints.

| Field | Type | Presence | Description |
|---|---|---|---|
| `request_id` | UUID v7 | always | From `X-Request-ID` or auto-generated |
| `timestamp` | RFC 3339 | always | Server UTC time |
| `total` | int | list only | Total items |
| `page` | int | list only | Current page |
| `page_size` | int | list only | Items per page |
| `total_pages` | int | list only | Total pages |
