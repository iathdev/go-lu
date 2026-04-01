# Plan B — Database-driven Entitlement (Scale)

> **Mục tiêu:** Migrate từ Plan A (config) sang DB-driven mà **chỉ thay loader** — interface, middleware, response format, PG quota counter giữ nguyên.
>
> **Prerequisite:** Plan A đã ship — `Service` interface, middleware, response format, PG quota counter đều đang hoạt động.
> **Storage:** Postgres (7 bảng canonical, bao gồm `usage_counts` SoT + `usage_records` audit) + Redis (entitlement cache + quota read cache) + In-memory cache
> **Effort ước tính:** 1-2 tuần
> **Tham khảo:** [`research_feature_gating.md`](research_feature_gating.md) §4 (Deep-dive DB-driven)

---

## 1. Gì giữ nguyên từ Plan A (không sửa)

| Component | File | Tại sao không đổi |
|---|---|---|
| `Service` interface | `shared/entitlement/service.go` | Plan B implement cùng interface. Caller không biết data source đổi |
| `types.go` | `shared/entitlement/types.go` | `Plan`, `Entitlement`, `CheckResult` struct giữ nguyên |
| `errors.go` | `shared/entitlement/errors.go` | Error types + codes giữ nguyên |
| Middleware `CheckFeature()`, `CheckQuota()` | `shared/middleware/entitlement.go` | Middleware gọi `svc.Check()` — không biết svc là config hay DB |
| Response format (429/403 + details) | `shared/response/` | Mobile đã integrate. Không thay đổi |
| PG quota counter (`usage_counts`) | `INSERT ... ON CONFLICT DO UPDATE WHERE count < limit` | PG là SoT cho quota enforcement. Redis chỉ là read cache. Xem [`usage_counting.md`](usage_counting.md) |
| JWT `plan_slug` claim | `shared/middleware/auth.go` | JWT vẫn chỉ chứa plan_slug. Entitlement resolution từ cached DB data |
| `GET /api/entitlements/me` endpoint | handler | Gọi `svc.GetEntitlements()` — same interface |
| Route registration | `server/router.go` | `middleware.CheckQuota("ocr_scan")` giữ nguyên |
| Scope check trong use case | vocabulary use cases | `svc.Check(ctx, userID, "hsk_content")` giữ nguyên |

---

## 2. Gì thay đổi — Migration từ A → B

| Thay đổi | Chi tiết | Effort |
|---|---|---|
| **Thêm 5 DB tables** (features, plans, plan_entitlements, user_plans, user_entitlements) | Migration files. Seed data từ Plan A config. `usage_counts` + `usage_records` đã có từ Plan A | 0.5 ngày |
| **Thêm `DBService`** | Implement `Service` interface, đọc từ Postgres + cache | 2-3 ngày |
| **Thêm two-level cache** | In-memory (sync.Map, TTL 60s) + Redis hash | 1 ngày |
| **Thêm Admin API** | CRUD plans, features, entitlements. Protected by admin role | 1-2 ngày |
| **Thêm `user_entitlements`** | Per-user override cho promo, enterprise, A/B | 0.5 ngày |
| **Thêm reconciliation job** | Daily: compare `usage_counts.count` vs `COUNT(*)` from `usage_records` → alert drift | 0.5 ngày |
| **DI wiring** | Đổi `NewConfigService()` → `NewDBService()` trong DI container | 10 phút |
| **Xóa Plan A config** | Xóa `config.go` + `config_service.go` | 5 phút |

---

## 3. DB Schema

### 3.1 Migration — 7 tables

**Tại sao 7 bảng?**

| Bảng | Trả lời câu hỏi | Ví dụ | Tại sao tách riêng | Nếu bỏ thì sao |
|---|---|---|---|---|
| **`features`** | Hệ thống có những tài nguyên nào cần phân quyền? | `ocr_scan`, `ai_chat`, `hsk_content` | 1 feature reuse across nhiều plans. Thêm feature = INSERT 1 row, không sửa code | Thêm feature = INSERT vào mọi plan. Không có catalog trung tâm |
| **`plans`** | Hệ thống có những plan nào? | `free`, `pro`, `basic`, `trial_7day` | Plan là entity độc lập, có lifecycle (active/inactive), metadata riêng | Không tách được — đây là core entity |
| **`plan_entitlements`** | Plan X cho phép dùng feature Y như thế nào? | `free` + `ocr_scan` = quota 3/ngày | Bảng JOIN giữa plans và features. Thay quota = UPDATE 1 row. Thêm plan = INSERT N rows | Không tách được — đây là core relationship |
| **`user_plans`** | User X đang thuộc plan nào? | user-123 → `pro`, active, expires 2026-12-31 | 1 user có history nhiều plans. Cần track status, trial expiry, external_id (sync Prep) | Dùng `users.plan_slug` → mất history, không track trial, không sync Prep |
| **`user_entitlements`** | User X có ngoại lệ gì so với plan? | user-123 được promo unlimited OCR 7 ngày | Override thuộc user, không thuộc plan. Có `effective_to` (tự expire), `source` (audit) | Không làm được promo, enterprise deal, A/B test. Phải tạo plan riêng cho mỗi exception |
| **`usage_records`** | User X đã dùng feature Y bao nhiêu lần? | user-123 đã scan 2 lần hôm nay | Append-only audit trail cho analytics, abuse detection, billing proof. Async write, không trên hot path | Mất audit trail. Không chứng minh usage cho billing/dispute |

```sql
-- migration: 000004_create_entitlement_tables.up.sql

-- 1. features — danh mục tất cả tài nguyên cần phân quyền
--    Mỗi row = 1 feature (ocr_scan, ai_chat, hsk_content, ...)
--    Thêm feature mới = INSERT 1 row
CREATE TABLE features (
    id          UUID PRIMARY KEY,
    key         VARCHAR(80) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(20) NOT NULL,         -- 'boolean', 'numeric', 'metered'
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 2. plans — "hệ thống có những plan nào?"
--    Mỗi row = 1 plan (free, pro, basic, trial_7day, ...)
--    Thêm plan = INSERT 1 row + INSERT entitlements
CREATE TABLE plans (
    id          UUID PRIMARY KEY,
    slug        VARCHAR(50) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(20) NOT NULL,         -- 'free', 'paid', 'custom', 'trial'
    is_active   BOOLEAN DEFAULT TRUE,
    is_default  BOOLEAN DEFAULT FALSE,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 3. plan_entitlements — "plan X cho phép dùng feature Y như thế nào?"
--    JOIN table giữa plans và features
--    Thay đổi quota = UPDATE 1 row, không deploy
CREATE TABLE plan_entitlements (
    id              UUID PRIMARY KEY,
    plan_id         UUID NOT NULL,
    feature_id      UUID NOT NULL,
    value_boolean   BOOLEAN,
    value_numeric   BIGINT,
    value_json      JSONB,
    reset_period    VARCHAR(20),              -- 'day', 'month', 'billing_cycle'
    is_soft_limit   BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(plan_id, feature_id)
);

-- 4. user_plans — "user X đang thuộc plan nào?"
--    Track status, trial expiry, sync với Prep subscription (external_id)
--    1 user có thể có history nhiều user_plans
CREATE TABLE user_plans (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    plan_id         UUID NOT NULL,
    status          VARCHAR(20) NOT NULL,         -- 'active', 'trialing', 'canceled', 'past_due'
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    external_id     VARCHAR(255),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_user_plans_user_status ON user_plans(user_id, status);

-- 5. user_entitlements — "user X có ngoại lệ gì so với plan?"
--    Promo, enterprise deal, A/B test. Có effective_from/to (tự expire)
--    Không có bảng này → phải tạo plan riêng cho mỗi exception
CREATE TABLE user_entitlements (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    feature_id      UUID NOT NULL,
    value_boolean   BOOLEAN,
    value_numeric   BIGINT,
    value_json      JSONB,
    reset_period    VARCHAR(20),
    effective_from  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to    TIMESTAMPTZ,
    source          VARCHAR(50) NOT NULL,    -- 'promotional', 'custom_deal', 'ab_test', 'migration', ... (validate ở app layer)
    created_by      VARCHAR(255),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_user_entitlements_user ON user_entitlements(user_id);

-- 6. usage_counts — quota counter (Source of Truth)
--    PG atomic upsert cho enforcement. Redis chỉ là read cache.
--    Chi tiết xem usage_counting.md
CREATE TABLE usage_counts (
    user_id       UUID        NOT NULL,
    feature_key   VARCHAR(50) NOT NULL,
    period_key    VARCHAR(10) NOT NULL,
    count         INT         NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, feature_key, period_key)
);
CREATE INDEX idx_usage_counts_period ON usage_counts (period_key);

-- 7. usage_records — append-only audit trail
--    Async write, không trên hot path. Dùng cho analytics + abuse detection + billing proof
CREATE TABLE usage_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    feature_key     VARCHAR(50) NOT NULL,
    request_id      UUID NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_usage_records_request UNIQUE (user_id, feature_key, request_id)
);
CREATE INDEX idx_usage_records_user_feature ON usage_records(user_id, feature_key, recorded_at);
```

### 3.2 Seed data — migrate từ Plan A config

```sql
-- migration: 000005_seed_entitlement_data.up.sql

-- Features — key PHẢI match Plan A feature keys
INSERT INTO features (key, name, type) VALUES
    ('ocr_scan',       'OCR Scan',          'metered'),
    ('create_card',    'Create Card',       'metered'),
    ('pronunciation',  'Pronunciation',     'metered'),
    ('recall_writing', 'Recall Writing',    'metered'),
    ('ai_chat',        'AI Chat',           'boolean'),
    ('mastery_check',  'Mastery Check',     'boolean'),
    ('speed_writing',  'Speed Writing',     'boolean'),
    ('weakness_report','Weakness Report',   'boolean'),
    ('hsk_content',    'HSK Content',       'numeric'),
    ('flashcard_type', 'Flashcard Type',    'numeric'),
    ('grammar',        'Grammar',           'numeric');

-- Plans
INSERT INTO plans (slug, name, type, is_default) VALUES
    ('free', 'Free', 'free', TRUE),
    ('pro',  'Pro',  'paid', FALSE);

-- Plan entitlements — Free
INSERT INTO plan_entitlements (plan_id, feature_id, value_boolean, value_numeric, value_json, reset_period)
SELECT p.id, f.id, v.val_bool, v.val_num, v.val_json, v.reset
FROM plans p, features f,
(VALUES
    ('free', 'ocr_scan',       NULL,  3,    NULL,                                          'day'),
    ('free', 'create_card',    NULL,  20,   NULL,                                          'day'),
    ('free', 'pronunciation',  NULL,  3,    NULL,                                          'day'),
    ('free', 'recall_writing', NULL,  5,    NULL,                                          'day'),
    ('free', 'ai_chat',        FALSE, NULL, NULL,                                          NULL),
    ('free', 'mastery_check',  FALSE, NULL, NULL,                                          NULL),
    ('free', 'speed_writing',  FALSE, NULL, NULL,                                          NULL),
    ('free', 'weakness_report',FALSE, NULL, NULL,                                          NULL),
    ('free', 'hsk_content',    NULL,  NULL, '{"max_level": 3}'::jsonb,                     NULL),
    ('free', 'flashcard_type', NULL,  NULL, '{"allowed": ["text"]}'::jsonb,                NULL),
    ('free', 'grammar',        NULL,  NULL, '{"mode": "tips_only"}'::jsonb,                NULL)
) AS v(plan_slug, feature_key, val_bool, val_num, val_json, reset)
WHERE p.slug = v.plan_slug AND f.key = v.feature_key;

-- Plan entitlements — Pro
INSERT INTO plan_entitlements (plan_id, feature_id, value_boolean, value_numeric, value_json, reset_period)
SELECT p.id, f.id, v.val_bool, v.val_num, v.val_json, v.reset
FROM plans p, features f,
(VALUES
    ('pro', 'ocr_scan',       NULL,  -1,   NULL,                                          'day'),
    ('pro', 'create_card',    NULL,  -1,   NULL,                                          'day'),
    ('pro', 'pronunciation',  NULL,  -1,   NULL,                                          'day'),
    ('pro', 'recall_writing', NULL,  -1,   NULL,                                          'day'),
    ('pro', 'ai_chat',        TRUE,  NULL, NULL,                                          NULL),
    ('pro', 'mastery_check',  TRUE,  NULL, NULL,                                          NULL),
    ('pro', 'speed_writing',  TRUE,  NULL, NULL,                                          NULL),
    ('pro', 'weakness_report',TRUE,  NULL, NULL,                                          NULL),
    ('pro', 'hsk_content',    NULL,  NULL, '{"max_level": 9}'::jsonb,                     NULL),
    ('pro', 'flashcard_type', NULL,  NULL, '{"allowed": ["text","image","video"]}'::jsonb, NULL),
    ('pro', 'grammar',        NULL,  NULL, '{"mode": "full"}'::jsonb,                     NULL)
) AS v(plan_slug, feature_key, val_bool, val_num, val_json, reset)
WHERE p.slug = v.plan_slug AND f.key = v.feature_key;
```

---

## 4. DBService implementation

### 4.1 Entitlement resolution

```go
// internal/shared/entitlement/db_service.go

type DBService struct {
    db          *gorm.DB
    redisClient *redis.Client
    memCache    *sync.Map          // Level 1: in-memory
}

func NewDBService(db *gorm.DB, redisClient *redis.Client) Service {
    return &DBService{db: db, redisClient: redisClient, memCache: &sync.Map{}}
}
```

**Resolution algorithm (giống research §4.2):**

```
Check(ctx, userID, featureKey):
  1. Lấy plan_slug từ ctx (JWT claim — giữ nguyên từ Plan A)
  2. resolveEntitlement(plan_slug, featureKey):
     a. Check Level 1 cache (in-memory, TTL 60s)    → hit? return
     b. Check Level 2 cache (Redis hash)             → hit? return + populate L1
     c. DB query:
        i.  SELECT FROM user_entitlements WHERE user_id AND feature.key = featureKey
            AND (effective_to IS NULL OR effective_to > NOW())
        ii. If found → use override (highest priority)
        iii.SELECT FROM plan_entitlements JOIN plans JOIN features
            WHERE plans.slug = plan_slug AND features.key = resource
        iv. Populate L2 + L1 cache
  3. Nếu quota type → PG atomic reserve (giữ nguyên từ Plan A, xem `usage_counting.md`)
  4. Return CheckResult
```

### 4.2 So sánh với Plan A

| Bước | Plan A | Plan B |
|---|---|---|
| Lấy plan_slug | JWT claim → ctx | **Giữ nguyên** |
| Resolve entitlement | `plans["free"].Entitlements["ocr_scan"]` (Go map) | L1 cache → L2 Redis → DB query |
| Customer override | Không support | `user_entitlements` table (step 2c.i) |
| Quota check | PG atomic upsert (`usage_counts`) + Redis read cache | **Giữ nguyên** |
| Return CheckResult | **Giữ nguyên** | **Giữ nguyên** |

→ Chỉ step "Resolve entitlement" thay đổi. Tất cả trước và sau giữ nguyên.

---

## 5. Bài toán đi kèm & ví dụ áp dụng từng feature

### 5.1 Entitlement config cache

**Bài toán:** Plan B config nằm trong DB. Mỗi request cần resolve "free plan + ocr_scan → limit 3, period day" từ `plan_entitlements` + `user_entitlements`. Không cache = mỗi request 1 DB JOIN query (1-5ms).

**Đây KHÔNG phải usage count cache.** So sánh 2 loại cache trong hệ thống:

| | Config cache (section này) | Usage count cache ([`usage_counting.md`](usage_counting.md)) |
|---|---|---|
| Cache cái gì | `limit: 3, type: metered, period: day` | `count: 2` (đã dùng bao nhiêu) |
| Thay đổi khi | Admin update plan, user upgrade (vài lần/ngày) | Mỗi request có quota (INCR) |
| Áp dụng cho | `CheckFeature`, `CheckQuota` (lấy limit), scope check | Quota enforcement, UI remaining |

**Cache strategy: Cache-aside (lazy load)**

```
Resolve entitlement config:
  Redis GET → hit → return config
  Redis GET → miss → DB JOIN query → SET Redis (TTL 5 phút) → return config
```

- **Read:** Redis first → miss → DB → populate Redis
- **Write:** Không có write thường xuyên (config hiếm khi đổi)
- **Invalidation:** Admin update hoặc user upgrade → DEL Redis key

Tại sao cache-aside: data thay đổi rất hiếm, read cực nhiều → lazy load đủ tốt. Không cần write-through vì không có write path thường xuyên.

**Redis key design:**

```
Key:    ent:{user_id}              — Redis hash
Field:  {feature_key}              — VD: ocr_scan, ai_chat
Value:  JSON resolved entitlement  — VD: {"type":"metered","limit":3,"period":"day"}
TTL:    5 phút
```

Dùng Redis hash → 1 `HGETALL` lấy toàn bộ entitlements của user (cho `GET /api/entitlements/me`). Hoặc `HGET` lấy 1 feature (cho middleware check).

**Invalidation:**

| Trigger | Action | Stale window |
|---|---|---|
| Admin update `plan_entitlements` | `DEL ent:*` (scan + delete affected keys) | Max 5 phút (TTL) cho users không bị DEL |
| User upgrade/downgrade | `DEL ent:{user_id}` | 0 — next request resolve lại ngay |
| User entitlement override added | `DEL ent:{user_id}` | 0 |

**Ví dụ flow — `CheckQuota("ocr_scan")`:**

```
Request → middleware CheckQuota("ocr_scan")
│
├── 1. Resolve config (cache này)
│   Redis: HGET ent:user-123 ocr_scan
│   → HIT: {type: metered, limit: 3, period: day}
│   (hoặc MISS → DB JOIN → SET Redis → return)
│
├── 2. Enforce quota (usage_counting.md)
│   PG: INSERT usage_counts ... ON CONFLICT DO UPDATE WHERE count < 3
│   → count = 2 → ALLOW
│
└── Return CheckResult{Allowed: true, Used: 2, Limit: 3, Remaining: 1}
```

**Có cần thêm L1 in-memory cache không?**

| | Chỉ Redis | Redis + L1 in-memory (LRU) |
|---|---|---|
| Latency | ~0.1ms (1 network hop) | ~ns cho ~95% requests |
| Complexity | Đơn giản | Thêm LRU, TTL, invalidation cross-instance |
| Khi nào | **Giai đoạn đầu Plan B — đủ** | Traffic rất cao, 0.1ms thành bottleneck |

**Recommendation:** Bắt đầu chỉ Redis. Thêm L1 khi có performance evidence.

Nếu cần L1 sau này: dùng `github.com/hashicorp/golang-lru/v2`, max 50K entries (~25MB RAM), TTL 60s, per-instance (không share). Invalidation: user upgrade → explicit delete L1 + L2. Admin update → L1 tự expire theo TTL (max 60s stale, chấp nhận được).

### 5.2 Customer override — Promotional

**Bài toán:** Marketing muốn tạo "Trial Pro 7 ngày" cho segment users, hoặc enterprise deal cho school_X.

**Ví dụ: Trial Pro 7 ngày cho OCR scan**

```sql
INSERT INTO user_entitlements (user_id, feature_id, value_numeric, reset_period, effective_from, effective_to, source)
SELECT 'user-123', f.id, -1, 'day', NOW(), NOW() + INTERVAL '7 days', 'promotional'
FROM features f WHERE f.key = 'ocr_scan';
```

**Resolution:**
1. `Check(ctx, "user-123", "ocr_scan")`
2. Query `user_entitlements` → found: `limit: -1, effective_to: 7 ngày sau` → **override plan entitlement**
3. User Free nhưng có unlimited scan trong 7 ngày
4. Sau 7 ngày → `effective_to` expired → fallback về plan entitlement (limit: 3)

**Ví dụ: Enterprise school unlimited cards**

```sql
-- Tất cả user của school_X có unlimited create_card
INSERT INTO user_entitlements (user_id, feature_id, value_numeric, reset_period, effective_to, source, created_by)
SELECT u.id, f.id, -1, 'day', NULL, 'custom_deal', 'admin@prep.vn'
FROM users u, features f
WHERE u.email LIKE '%@schoolx.edu.vn' AND f.key = 'create_card';
```

### 5.3 Usage records — Async audit

Chi tiết storage options, counting strategy (Reserve → Execute → Confirm/Rollback), và Service interface update xem [`usage_counting.md`](usage_counting.md).

Plan B thêm so với Plan A: `idempotency_key` enforcement + daily reconciliation job (`usage_counts.count` vs `COUNT(*)` from `usage_records`).

**Ví dụ query analytics:**
```sql
-- Top 10 users by OCR scan usage last 7 days
SELECT u.email, COUNT(*) as scans
FROM usage_records ur
JOIN features f ON ur.feature_id = f.id
JOIN users u ON ur.user_id = u.id
WHERE f.key = 'ocr_scan' AND ur.recorded_at > NOW() - INTERVAL '7 days'
GROUP BY u.email ORDER BY scans DESC LIMIT 10;

-- Daily usage trend for all metered features
SELECT f.key, DATE(ur.recorded_at), COUNT(*)
FROM usage_records ur JOIN features f ON ur.feature_id = f.id
GROUP BY f.key, DATE(ur.recorded_at) ORDER BY 2 DESC;
```

### 5.4 Admin API — Runtime config

**Bài toán:** Product/Commercial team muốn thay đổi entitlements không cần deploy.

**Endpoints:**

```
# Plans
GET    /api/admin/plans                          → list all plans
POST   /api/admin/plans                          → create plan
PUT    /api/admin/plans/:slug                    → update plan

# Features
GET    /api/admin/features                       → list all features
POST   /api/admin/features                       → register new feature

# Plan entitlements
GET    /api/admin/plans/:slug/entitlements       → list entitlements for plan
PUT    /api/admin/plans/:slug/entitlements/:key  → update entitlement (VD: change limit)
POST   /api/admin/plans/:slug/entitlements       → add entitlement to plan

# Customer overrides
GET    /api/admin/users/:id/overrides            → list overrides for user
POST   /api/admin/users/:id/overrides            → add override (promo, custom deal)
DELETE /api/admin/users/:id/overrides/:id        → remove override
```

**Ví dụ: Commercial team thay OCR scan limit Free 3 → 5**

```
PUT /api/admin/plans/free/entitlements/ocr_scan
{ "value_numeric": 5 }
```

→ Update DB → Invalidate cache (clear `entitlements:*` for free users) → Mọi Free user tự động có limit 5. **Không deploy, không restart.**

**Ví dụ: Thêm plan "Basic" ($2.99)**

```
POST /api/admin/plans
{ "slug": "basic", "name": "Basic", "type": "paid" }

POST /api/admin/plans/basic/entitlements
{ "feature_key": "ocr_scan", "value_numeric": 10, "reset_period": "day" }

POST /api/admin/plans/basic/entitlements
{ "feature_key": "ai_chat", "value_boolean": true }

POST /api/admin/plans/basic/entitlements
{ "feature_key": "hsk_content", "value_json": {"max_level": 6} }
```

→ Plan "basic" sẵn sàng. User assigned plan "basic" → có 10 scan/ngày, AI Chat enabled, HSK 1-6. **Không sửa code.**

### 5.5 Thêm feature mới — Video Flashcard (Phase 2)

**Bước 1: Register feature (Admin API hoặc migration)**

```sql
INSERT INTO features (key, name, type) VALUES ('video_flashcard', 'Video Flashcard', 'boolean');
```

**Bước 2: Add entitlements per plan**

```sql
INSERT INTO plan_entitlements (plan_id, feature_id, value_boolean)
SELECT p.id, f.id, CASE p.slug WHEN 'free' THEN FALSE WHEN 'pro' THEN TRUE END
FROM plans p, features f WHERE f.key = 'video_flashcard';
```

**Bước 3: Add check point trong code (1 lần)**

```go
// Route registration
api.POST("/flashcards/video", middleware.CheckFeature(svc, "video_flashcard"), handler.CreateVideoFlashcard)
```

→ Thêm feature = 1 DB insert + 1 dòng route registration. Không sửa entitlement logic.

### 5.6 Upgrade/Downgrade (giữ nguyên từ Plan A)

Flow giống Plan A — JWT refresh → new plan_slug → entitlement resolution tự trả kết quả khác vì lookup plan mới.

Thêm trong Plan B: invalidate cache `entitlements:{user_id}` khi plan change → resolution lấy data mới ngay.

### 5.7 Fail-open/Fail-closed (giữ nguyên từ Plan A)

Logic giữ nguyên. Thêm trong Plan B:
- **Entitlement resolution**: DB down → fallback Redis cache (L2). Redis down → fallback in-memory (L1). Cả 3 down → fail-closed cho feature/scope
- **Quota enforcement**: PG down → fail-open (allow, config) hoặc fail-closed (reject 503). Redis down → không ảnh hưởng enforcement (PG là SoT)

---

## 6. File structure (delta từ Plan A)

```
internal/shared/entitlement/
├── types.go              # GIỮA NGUYÊN
├── service.go            # GIỮA NGUYÊN (interface)
├── errors.go             # GIỮA NGUYÊN
├── config.go             # ❌ XÓA
├── config_service.go     # ❌ XÓA
├── db_service.go         # ✅ MỚI — implement Service từ DB
├── cache.go              # ✅ MỚI — two-level cache (in-memory + Redis)
├── repository.go         # ✅ MỚI — DB queries (plans, entitlements, overrides)
└── admin_handler.go      # ✅ MỚI — Admin API handlers

internal/shared/middleware/
├── entitlement.go        # GIỮA NGUYÊN

migrations/
├── 000004_create_entitlement_tables.up.sql    # ✅ MỚI
├── 000004_create_entitlement_tables.down.sql  # ✅ MỚI
├── 000005_seed_entitlement_data.up.sql        # ✅ MỚI
└── 000005_seed_entitlement_data.down.sql      # ✅ MỚI
```

**DI wiring change (1 chỗ):**

```go
// internal/infrastructure/di/container.go

// Plan A:
// entitlementSvc := entitlement.NewConfigService(entitlement.DefaultPlans(), redisClient)

// Plan B:
entitlementSvc := entitlement.NewDBService(db, redisClient)
```

---

## 7. Checklist migration A → B

- [ ] Tạo migration 000004 + 000005
- [ ] Run migration (tạo tables + seed data từ Plan A config)
- [ ] Implement `DBService` (implement `Service` interface)
- [ ] Implement `cache.go` (two-level cache)
- [ ] Implement `repository.go` (DB queries)
- [ ] Đổi DI wiring: `NewConfigService` → `NewDBService`
- [ ] Xóa `config.go` + `config_service.go`
- [ ] Test: mọi middleware + use case check vẫn pass (interface không đổi)
- [ ] Implement Admin API handlers
- [ ] Implement async usage_records writer
- [ ] Test: thêm plan mới qua Admin API → user assigned plan mới → entitlements đúng
- [ ] Test: thêm customer override → override plan entitlement
- [ ] Test: cache invalidation khi admin thay đổi entitlement
