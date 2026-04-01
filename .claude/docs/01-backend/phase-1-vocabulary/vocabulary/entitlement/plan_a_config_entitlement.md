# Plan A — Config-driven Entitlement (MVP)

> **Mục tiêu:** Ship entitlement system nhanh nhất có thể, đồng thời đặt nền tảng (interface, middleware, response format) để chuyển sang Plan B (DB-driven) mà **không sửa code ngoài loader**.
>
> **Storage:** Go config (in-memory map, loaded from YAML hoặc Go constants)
> **Effort ước tính:** 2-3 ngày
> **Tham khảo:** [`research_feature_gating.md`](research_feature_gating.md) §3, §4

---

## 1. Shared Interface — Base chung cho cả Plan A & B

Phần này **không thay đổi** khi chuyển sang Plan B. Đây là contract giữa entitlement system và phần còn lại của app.

### 1.1 Core types

```go
// internal/shared/entitlement/types.go

type EntitlementType string

const (
    TypeQuota   EntitlementType = "quota"    // metered: limit per period
    TypeFeature EntitlementType = "feature"  // boolean: on/off
    TypeScope   EntitlementType = "scope"    // constraint: JSON config
)

type Entitlement struct {
    Type        EntitlementType
    Enabled     bool            // feature type
    Limit       int64           // quota type: -1 = unlimited
    Period      string          // quota type: "day", "month"
    Config      map[string]any  // scope type: {"max_level": 3, "allowed": ["text"]}
    IsSoftLimit bool            // quota: cho vượt nhưng cảnh báo
}

type Plan struct {
    Slug         string
    DisplayName  string
    Entitlements map[string]Entitlement  // key = feature key: "ocr_scan", "ai_chat"
}

// Kết quả check entitlement — chứa đủ info cho mobile hiển thị CTA
type CheckResult struct {
    Allowed     bool
    Feature     string
    Type        EntitlementType
    Limit       int64           // quota: limit hiện tại
    Used        int64           // quota: đã dùng
    Remaining   int64           // quota: còn lại
    ResetsAt    *time.Time      // quota: thời điểm reset
    Config      map[string]any  // scope: constraint config
    CurrentPlan string
    UpgradeCTA  string          // mobile dùng để hiển thị đúng CTA screen
}
```

### 1.2 Service interface

```go
// internal/shared/entitlement/service.go

type Service interface {
    // Check kiểm tra user có quyền truy cập feature không
    // Middleware gọi hàm này — KHÔNG biết plan nào, KHÔNG biết limit bao nhiêu
    Check(ctx context.Context, userID string, featureKey string) (*CheckResult, error)

    // IncrementUsage tăng counter cho metered feature (gọi sau khi request thành công)
    IncrementUsage(ctx context.Context, userID string, featureKey string) error

    // GetEntitlements trả toàn bộ entitlements của user (cho GET /api/entitlements/me)
    GetEntitlements(ctx context.Context, userID string) (map[string]*CheckResult, error)

    // GetPlanForUser trả plan slug của user
    GetPlanForUser(ctx context.Context, userID string) (string, error)
}
```

**Tại sao interface này là base:** Cả Plan A (config loader) và Plan B (DB loader) đều implement cùng interface. Middleware, handler, use case chỉ biết `Service` — không biết data đến từ đâu.

### 1.3 Middleware

```go
// internal/shared/middleware/entitlement.go

// CheckFeature — cho feature type (boolean on/off)
// VD: middleware.CheckFeature(entitlementSvc, "ai_chat")
func CheckFeature(svc entitlement.Service, featureKey string) gin.HandlerFunc

// CheckQuota — cho quota type (metered, có counter)
// VD: middleware.CheckQuota(entitlementSvc, "ocr_scan")
func CheckQuota(svc entitlement.Service, featureKey string) gin.HandlerFunc
```

### 1.4 Response format

Chuẩn hóa response khi bị chặn — mobile dựa vào `code` + `details` để hiển thị CTA.

| Loại | HTTP status | `code` | `details` chứa |
|---|---|---|---|
| Quota exceeded | 429 | `QUOTA_EXCEEDED` | feature, limit, used, remaining, resets_at, current_plan, upgrade_cta |
| Feature locked | 403 | `FEATURE_NOT_ENTITLED` | feature, current_plan, upgrade_cta |
| Scope exceeded | 403 | `SCOPE_EXCEEDED` | feature, constraint, requested, current_plan, upgrade_cta |

### 1.5 API endpoint

```
GET /api/entitlements/me → trả toàn bộ entitlements + usage hiện tại của user
```

Mobile gọi khi app open + khi cần hiển thị quota badge.

---

## 2. Plan A implementation — Config loader

### 2.1 Entitlement config

```go
// internal/shared/entitlement/config.go

var defaultPlans = map[string]Plan{
    "free": {
        Slug: "free", DisplayName: "Free",
        Entitlements: map[string]Entitlement{
            "ocr_scan":       {Type: TypeQuota, Limit: 3, Period: "day"},
            "create_card":    {Type: TypeQuota, Limit: 20, Period: "day"},
            "pronunciation":  {Type: TypeQuota, Limit: 3, Period: "day"},
            "recall_writing": {Type: TypeQuota, Limit: 5, Period: "day"},
            "ai_chat":        {Type: TypeFeature, Enabled: false},
            "mastery_check":  {Type: TypeFeature, Enabled: false},
            "speed_writing":  {Type: TypeFeature, Enabled: false},
            "weakness_report":{Type: TypeFeature, Enabled: false},
            "hsk_content":    {Type: TypeScope, Config: map[string]any{"max_level": 3}},
            "flashcard_type": {Type: TypeScope, Config: map[string]any{"allowed": []string{"text"}}},
            "grammar":        {Type: TypeScope, Config: map[string]any{"mode": "tips_only"}},
        },
    },
    "pro": {
        Slug: "pro", DisplayName: "Pro",
        Entitlements: map[string]Entitlement{
            "ocr_scan":       {Type: TypeQuota, Limit: -1, Period: "day"},
            "create_card":    {Type: TypeQuota, Limit: -1, Period: "day"},
            "pronunciation":  {Type: TypeQuota, Limit: -1, Period: "day"},
            "recall_writing": {Type: TypeQuota, Limit: -1, Period: "day"},
            "ai_chat":        {Type: TypeFeature, Enabled: true},
            "mastery_check":  {Type: TypeFeature, Enabled: true},
            "speed_writing":  {Type: TypeFeature, Enabled: true},
            "weakness_report":{Type: TypeFeature, Enabled: true},
            "hsk_content":    {Type: TypeScope, Config: map[string]any{"max_level": 9}},
            "flashcard_type": {Type: TypeScope, Config: map[string]any{"allowed": []string{"text", "image", "video"}}},
            "grammar":        {Type: TypeScope, Config: map[string]any{"mode": "full"}},
        },
    },
}
```

### 2.2 Config service implementation

```go
// internal/shared/entitlement/config_service.go

type ConfigService struct {
    plans       map[string]Plan      // loaded at startup
    db          *gorm.DB             // quota counter (usage_counts SoT)
    redisClient *redis.Client        // quota read cache
}

func NewConfigService(plans map[string]Plan, db *gorm.DB, redisClient *redis.Client) Service {
    return &ConfigService{plans: plans, db: db, redisClient: redisClient}
}
```

**Lookup flow:**
1. Lấy `plan_slug` từ context (auth middleware đã set từ JWT)
2. Lookup `plans[plan_slug]` → lấy `Entitlement` cho feature key
3. Nếu quota type → PG atomic upsert `usage_counts` (`INSERT ... ON CONFLICT DO UPDATE WHERE count < limit`) → trả count mới. Update Redis read cache
4. Trả `CheckResult`

### 2.3 Plan source — JWT claim

```
Login → Prep /me → subscription status → map sang plan_slug → upsert users.plan_slug → JWT claim { plan: "free" }
Auth middleware → extract plan từ JWT → set ctx "user_plan"
```

**Cần thay đổi:**
- `auth/domain/user.go`: thêm `PlanSlug string`
- `auth/adapter/repository/model/user_model.go`: thêm `PlanSlug` column
- Migration: `ALTER TABLE users ADD COLUMN plan_slug VARCHAR(50) DEFAULT 'free'`
- `auth/adapter/security/jwt_service.go`: embed `plan` claim
- `shared/middleware/auth.go`: extract `plan` → set context

---

## 3. Bài toán đi kèm & ví dụ áp dụng từng feature

### 3.1 Quota — Usage counting

**Bài toán:** Đếm usage per-user per-feature per-day, atomic, shared across instances.

**Giải pháp:** PG (SoT, atomic upsert) + Redis (read cache) + PG audit (async). Chi tiết storage options, counting strategy, và Service interface update xem [`usage_counting.md`](usage_counting.md).

**Ví dụ áp dụng:**

| Resource | Free limit | Flow khi user gọi API |
|---|---|---|
| `ocr_scan` | 3/day | `POST /api/vocabularies/ocr-scan` → middleware `CheckQuota("ocr_scan")` → PG Reserve (atomic upsert WHERE count < 3) → count=2 → allow → action success → async INSERT usage_records |
| `create_card` | 20/day | `POST /api/vocabularies` → middleware `CheckQuota("create_card")` → PG Reserve → RowsAffected=0 (count=20, WHERE count < 20 fail) → 429 |
| `pronunciation` | 3/day | `POST /api/pronunciation/check` → middleware `CheckQuota("pronunciation")` → PG Reserve |
| `recall_writing` | 5/day | `POST /api/learning/stroke-recall` → middleware `CheckQuota("recall_writing")` → PG Reserve |

**Edge cases:**
- PG down → **fail-open** (allow request, log warning, config). Tại sao: tốn thêm vài cent tốt hơn block user hợp lệ
- Redis down → không ảnh hưởng enforcement (PG là SoT). Read path chậm hơn nhưng đúng
- User Pro (limit = -1) → middleware skip Reserve, return allowed ngay
- UTC midnight reset → period_key mới tự động (`"2026-03-21"`), old rows giữ cho analytics
- Action fail sau Reserve → **PG rollback** `UPDATE count = GREATEST(count - 1, 0)` + DEL Redis cache (xem `usage_counting.md` §3.5)

### 3.2 Feature lock — Boolean check

**Bài toán:** Feature có được phép dùng hay không, dựa trên plan.

**Giải pháp:** Lookup entitlement config → check `Enabled` boolean. Không cần Redis.

**Ví dụ áp dụng:**

| Resource | Free | Pro | Route registration |
|---|---|---|---|
| `ai_chat` | disabled | enabled | `api.POST("/ai-chat", middleware.CheckFeature(svc, "ai_chat"), handler.AIChat)` |
| `mastery_check` | disabled | enabled | `api.POST("/mastery-check", middleware.CheckFeature(svc, "mastery_check"), handler.MasteryCheck)` |
| `speed_writing` | disabled | enabled | `api.POST("/stroke-speed", middleware.CheckFeature(svc, "speed_writing"), handler.SpeedWriting)` |
| `weakness_report` | disabled | enabled | `api.GET("/pronunciation/report", middleware.CheckFeature(svc, "weakness_report"), handler.WeaknessReport)` |

**Flow:** Request → auth middleware (set plan) → `CheckFeature("ai_chat")` → lookup `plans["free"].Entitlements["ai_chat"].Enabled` → `false` → 403 `FEATURE_NOT_ENTITLED` + `upgrade_cta: "upgrade_ai_chat"`.

**Edge cases:**
- Config missing feature key → **fail-closed** (deny). Log error "unknown feature"
- Plan unknown → fallback to "free" plan

### 3.3 Scope — Domain-specific constraint

**Bài toán:** Feature available nhưng giới hạn phạm vi (VD: HSK level ≤ 3).

**Giải pháp:** Use case layer check constraint từ entitlement config. **Không dùng middleware** vì scope logic domain-specific.

**Ví dụ áp dụng:**

| Resource | Free constraint | Pro constraint | Check ở đâu |
|---|---|---|---|
| `hsk_content` | `max_level: 3` | `max_level: 9` | `VocabularyQuery.ListByHSKLevel()` — check level param vs `Config["max_level"]` |
| `flashcard_type` | `allowed: ["text"]` | `allowed: ["text","image","video"]` | `VocabularyCommand.CreateVocabulary()` — check flashcard type vs `Config["allowed"]` |
| `grammar` | `mode: "tips_only"` | `mode: "full"` | `GrammarQuery.GetGrammarForVocab()` — return tips only vs full context |

**Flow (HSK content):**
```go
// internal/vocabulary/application/usecase/vocabulary_query.go
func (uc *VocabularyQuery) ListByHSKLevel(ctx context.Context, level int, ...) {
    result, _ := uc.entitlementSvc.Check(ctx, userID, "hsk_content")
    maxLevel := result.Config["max_level"].(int)
    if level > maxLevel {
        return nil, apperror.New(CodeForbidden, "HSK level %d requires upgrade", level)
        // Response: 403 SCOPE_EXCEEDED + constraint + upgrade_cta
    }
    // ... proceed query
}
```

### 3.4 Upgrade/Downgrade

**Bài toán:** User thay đổi plan mid-session.

| Scenario | Xử lý trong Plan A |
|---|---|
| **Upgrade** | Mobile force refresh token → new JWT `plan: "pro"` → middleware đọc plan mới → tất cả checks pass. Quota counter Redis vẫn giữ count cũ, nhưng limit = -1 → always pass |
| **Downgrade** | Next token refresh → JWT `plan: "free"` → features lock ngay. Đang ở giữa session → cho hoàn thành request hiện tại (graceful: middleware chỉ check trước request mới) |
| **Data đã tạo từ Pro** | Không xóa. User vẫn thấy HSK 4+ cards đã có. Chỉ không tạo mới |

### 3.5 Timezone

**Bài toán:** "3 scan/ngày" — ngày nào?

**Giải pháp Plan A:** UTC midnight. Redis key dùng UTC date. Mobile hiển thị reset time theo local timezone.

### 3.6 Fail-open/Fail-closed

| Loại | Khi Redis/config lỗi | Lý do |
|---|---|---|
| Quota (PG down) | Fail-open (allow) | Worst case vài cent extra. Redis down → không ảnh hưởng enforcement |
| Feature | Fail-closed (deny, default "free") | Revenue protection |
| Scope | Fail-closed (default lowest scope) | Revenue protection |

---

## 4. Chuẩn bị cho Plan B — Gì cần làm đúng từ đầu

Những quyết định trong Plan A mà **ảnh hưởng trực tiếp** đến việc migrate sang Plan B:

| Quyết định Plan A | Tại sao quan trọng cho Plan B |
|---|---|
| **Interface `Service`** | Plan B chỉ cần implement interface này với DB loader. Middleware, handler, use case không đổi |
| **Feature key naming** (`ocr_scan`, `ai_chat`) | Plan B dùng làm `features.key` trong DB. Đổi key = migration phức tạp. Chọn tên stable từ đầu |
| **3 entitlement types** (quota/feature/scope) | Plan B map sang `features.type` column. Thêm type = thêm column |
| **PG `usage_counts` table + Redis read cache** | Plan B giữ nguyên PG SoT + Redis cache. Schema và flow không đổi |
| **Response format (429/403 + details)** | Mobile đã integrate. Plan B trả cùng format |
| **JWT chỉ chứa `plan_slug`** | Plan B resolve entitlements từ DB, không từ JWT. JWT vẫn chỉ chứa plan_slug |

**Gì KHÔNG cần làm trong Plan A:**
- ❌ Admin API
- ❌ `user_entitlements` (per-user override)
- ✅ `usage_records` table (async audit log) — moved to MVP scope, xem [`usage_counting.md`](usage_counting.md)
- ❌ Two-level cache (in-memory + Redis)
- ❌ Webhook subscription change listener

---

## 5. File structure

```
internal/shared/
├── entitlement/
│   ├── types.go              # Plan, Entitlement, CheckResult (shared)
│   ├── service.go            # Service interface (shared)
│   ├── config.go             # defaultPlans Go map (Plan A only → replaced by DB in Plan B)
│   ├── config_service.go     # ConfigService implements Service (Plan A only → replaced in Plan B)
│   └── errors.go             # QuotaExceeded, FeatureNotEntitled, ScopeExceeded
├── middleware/
│   ├── entitlement.go        # CheckFeature(), CheckQuota() middleware (shared)
│   └── ... (existing)
└── response/
    └── ... (existing — add entitlement error formatting)
```

**Khi migrate sang Plan B:** xóa `config.go` + `config_service.go`, thêm `db_service.go`. Mọi file khác giữ nguyên.
