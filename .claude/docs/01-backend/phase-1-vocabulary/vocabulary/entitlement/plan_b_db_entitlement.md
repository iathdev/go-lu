# Plan B — Database-driven Entitlement (Scale)

> **Mục tiêu:** Migrate từ Plan A (config) sang DB-driven mà **chỉ thay loader** — interface, middleware, response format, Redis quota counter giữ nguyên.
>
> **Prerequisite:** Plan A đã ship — `Service` interface, middleware, response format, Redis counter đều đang hoạt động.
> **Storage:** Postgres (6 bảng canonical) + Redis (quota counter + entitlement cache) + In-memory cache
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
| Redis quota counter | Key format `quota:{uid}:{feature_key}:{date}` | Hot path vẫn dùng Redis. DB chỉ thay source of truth cho config |
| JWT `plan_slug` claim | `shared/middleware/auth.go` | JWT vẫn chỉ chứa plan_slug. Entitlement resolution từ cached DB data |
| `GET /api/entitlements/me` endpoint | handler | Gọi `svc.GetEntitlements()` — same interface |
| Route registration | `server/router.go` | `middleware.CheckQuota("ocr_scan")` giữ nguyên |
| Scope check trong use case | vocabulary use cases | `svc.Check(ctx, userID, "hsk_content")` giữ nguyên |

---

## 2. Gì thay đổi — Migration từ A → B

| Thay đổi | Chi tiết | Effort |
|---|---|---|
| **Thêm 6 DB tables** | Migration files. Seed data từ Plan A config | 0.5 ngày |
| **Thêm `DBService`** | Implement `Service` interface, đọc từ Postgres + cache | 2-3 ngày |
| **Thêm two-level cache** | In-memory (sync.Map, TTL 60s) + Redis hash | 1 ngày |
| **Thêm Admin API** | CRUD plans, features, entitlements. Protected by admin role | 1-2 ngày |
| **Thêm `user_entitlements`** | Per-user override cho promo, enterprise, A/B | 0.5 ngày |
| **Thêm `usage_records`** | Async audit log (không trên hot path) | 0.5 ngày |
| **DI wiring** | Đổi `NewConfigService()` → `NewDBService()` trong DI container | 10 phút |
| **Xóa Plan A config** | Xóa `config.go` + `config_service.go` | 5 phút |

---

## 3. DB Schema

### 3.1 Migration — 6 tables

**Tại sao 6 bảng?**

| Bảng | Trả lời câu hỏi | Ví dụ | Tại sao tách riêng | Nếu bỏ thì sao |
|---|---|---|---|---|
| **`features`** | Hệ thống có những tài nguyên nào cần phân quyền? | `ocr_scan`, `ai_chat`, `hsk_content` | 1 feature reuse across nhiều plans. Thêm feature = INSERT 1 row, không sửa code | Thêm feature = INSERT vào mọi plan. Không có catalog trung tâm |
| **`plans`** | Hệ thống có những plan nào? | `free`, `pro`, `basic`, `trial_7day` | Plan là entity độc lập, có lifecycle (active/inactive), metadata riêng | Không tách được — đây là core entity |
| **`plan_entitlements`** | Plan X cho phép dùng feature Y như thế nào? | `free` + `ocr_scan` = quota 3/ngày | Bảng JOIN giữa plans và features. Thay quota = UPDATE 1 row. Thêm plan = INSERT N rows | Không tách được — đây là core relationship |
| **`user_plans`** | User X đang thuộc plan nào? | user-123 → `pro`, active, expires 2026-12-31 | 1 user có history nhiều plans. Cần track status, trial expiry, external_id (sync Prep) | Dùng `users.plan_slug` → mất history, không track trial, không sync Prep |
| **`user_entitlements`** | User X có ngoại lệ gì so với plan? | user-123 được promo unlimited OCR 7 ngày | Override thuộc user, không thuộc plan. Có `effective_to` (tự expire), `source` (audit) | Không làm được promo, enterprise deal, A/B test. Phải tạo plan riêng cho mỗi exception |
| **`usage_records`** | User X đã dùng feature Y bao nhiêu lần? | user-123 đã scan 2 lần hôm nay | Redis counter = hot path nhưng volatile. Bảng này = durable audit trail cho analytics, abuse detection. Async write | Mất audit trail. Redis flush = mất data. Chấp nhận cho MVP nhưng không cho scale |

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

-- 6. usage_records — "user X đã dùng feature Y bao nhiêu lần?"
--    Durable audit trail (Redis counter là hot path nhưng volatile)
--    Async write, không trên hot path. Dùng cho analytics + abuse detection
CREATE TABLE usage_records (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    feature_id      UUID NOT NULL,
    quantity        BIGINT NOT NULL DEFAULT 1,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    idempotency_key VARCHAR(255) UNIQUE
);
CREATE INDEX idx_usage_records_user_feature ON usage_records(user_id, feature_id, recorded_at);
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
  3. Nếu quota type → Redis INCR (giữ nguyên từ Plan A)
  4. Return CheckResult
```

### 4.2 So sánh với Plan A

| Bước | Plan A | Plan B |
|---|---|---|
| Lấy plan_slug | JWT claim → ctx | **Giữ nguyên** |
| Resolve entitlement | `plans["free"].Entitlements["ocr_scan"]` (Go map) | L1 cache → L2 Redis → DB query |
| Customer override | Không support | `user_entitlements` table (step 2c.i) |
| Quota check | Redis INCR | **Giữ nguyên** |
| Return CheckResult | **Giữ nguyên** | **Giữ nguyên** |

→ Chỉ step "Resolve entitlement" thay đổi. Tất cả trước và sau giữ nguyên.

---

## 5. Bài toán đi kèm & ví dụ áp dụng từng feature

### 5.1 Two-level cache

**Bài toán:** DB query mỗi request → 1-5ms latency, DB overload. Cần cache nhưng phải invalidate khi plan/entitlement thay đổi.

**Giải pháp — 2 tầng cache, mỗi tầng giải quyết 1 vấn đề khác nhau:**

**Tại sao L1 ở đầu (không phải L2)?**

Entitlement config **gần như không thay đổi** — chỉ thay khi admin update plan hoặc user upgrade (vài lần/ngày). Nhưng **mỗi request đều phải check** (hàng trăm nghìn lần/ngày). Read cực nhiều, write cực ít → in-memory cache ở đầu là tối ưu nhất.

Nếu đặt L2 (Redis) ở đầu → mỗi request vẫn tốn 1 network hop (~0.1ms) dù data không đổi. L1 ở đầu = **0 network cho ~95% requests**.

**Level 1: In-memory (LRU cache)**

| | Chi tiết |
|---|---|
| **Giải quyết gì** | Tránh network call hoàn toàn. Data nằm ngay trong RAM của Go process → đọc nhanh như đọc biến |
| **Lưu vào đâu** | LRU cache trong RAM (VD: `github.com/hashicorp/golang-lru/v2`). Không dùng `sync.Map` vì sync.Map không có eviction — lưu nhiều user sẽ ăn RAM không giới hạn. LRU tự evict entry ít dùng nhất khi đầy |
| **Lưu cái gì** | Resolved entitlement của 1 user cho 1 feature. VD: user-123 + ocr_scan → `{type: metered, limit: 3, period: day}`. Đây là kết quả đã resolve xong (đã check user override → plan entitlement → merge) — không phải raw DB row |
| **Lưu khi nào** | Sau khi resolve xong từ L2 hoặc DB. Mỗi lần L1 miss → resolve → lưu kết quả vào L1 để lần sau không cần resolve lại |
| **Max size** | 50K entries (config via env `ENTITLEMENT_L1_MAX_SIZE`). ~50K × 500 bytes = **~25MB RAM**. Đủ cho ~4.5K concurrent users × 11 features. Vượt → LRU evict entry ít dùng nhất |
| **Key** | `{user_id}:{feature_key}` — VD: `user-123:ocr_scan` |
| **Value** | Go struct `Entitlement` (zero serialization — đọc trực tiếp từ memory) |
| **TTL** | 60s (boolean/numeric), 10s (metered — cần near-real-time vì quota thay đổi liên tục) |
| **Scope** | Per-instance (instance A và instance B có L1 cache riêng, không share) |
| **Mất khi** | (1) **TTL expire** — 60s/10s hết → entry tự biến mất → next request miss L1 → xuống L2. (2) **LRU eviction** — cache đầy 50K entries → entry ít truy cập nhất bị xóa nhường chỗ. (3) **Explicit delete** — user upgrade/downgrade → code chủ động xóa entries của user đó để force resolve lại. (4) **Instance restart** — RAM mất hết → cold start → mọi request đều miss L1 cho đến khi warm up |
| **Latency** | ~nanoseconds |

**Lưu nhiều có sao không?** Có LRU max size nên không lo OOM. Khi vượt 50K entries → entry ít dùng nhất bị evict → next request cho user đó miss L1 → xuống L2 Redis → resolve lại. Không mất data, chỉ chậm hơn 1 lần.

**Level 2: Redis hash**

| | Chi tiết |
|---|---|
| **Giải quyết gì** | Chia sẻ resolved entitlement giữa nhiều Go instances. Instance A resolve rồi → instance B đọc từ Redis, không cần query DB lại |
| **Lưu cái gì** | Cùng data như L1, nhưng serialized thành JSON. VD: `{"type":"metered","limit":3,"period":"day"}` |
| **Lưu khi nào** | Sau khi resolve từ DB. Mỗi lần L2 miss → query DB → lưu kết quả vào L2 (và L1) |
| **Key** | `entitlements:{user_id}` — Redis hash, mỗi field = 1 feature key. VD: `HSET entitlements:user-123 ocr_scan '{"type":"metered","limit":3}'` |
| **Value** | JSON string (cần serialize khi write, deserialize khi read) |
| **TTL** | 5 phút (dài hơn L1 vì mục đích là giữ cache cross-instance + qua restart) |
| **Scope** | Shared across tất cả instances (cùng Redis cluster) |
| **Mất khi** | Redis restart, TTL expire, explicit DEL khi invalidation |
| **Latency** | ~0.1ms (1 network hop tới Redis) |

**Tại sao không chỉ dùng 1 tầng?**

| Chỉ L1 (in-memory) | Chỉ L2 (Redis) |
|---|---|
| Mỗi instance cache riêng → cùng user query DB N lần (N = số instances). Instance restart = cold start toàn bộ | 0.1ms/request × 500K req/ngày = vẫn ok, nhưng thêm network hop không cần thiết cho data ít thay đổi (entitlement config thay đổi hiếm khi, chỉ khi admin update hoặc user upgrade) |

→ L1 chặn ~95% requests (trong 60s window). L2 chặn ~4% (cross-instance + sau L1 expire). DB chỉ bị hit ~1% (cold start, sau invalidation).

**Invalidation:**
- Admin thay đổi plan_entitlements → clear L2 `DEL entitlements:*` → L1 tự expire theo TTL (max 60s stale, chấp nhận được)
- User upgrade/downgrade → clear L2 `DEL entitlements:{user_id}` + clear L1 `memCache.Delete("ent:{user_id}:*")`
- User entitlement override added → clear L2 + L1 cho user đó

**Ví dụ `ocr_scan` (quota) — First request (cold):**

```
Check(ctx, "user-123", "ocr_scan")
│
▼
L1 in-memory cache: "ent:user-123:ocr_scan"
│ MISS (first request)
▼
L2 Redis: HGET entitlements:user-123 ocr_scan
│ MISS (chưa có)
▼
L3 DB: SELECT plan_entitlements
│      JOIN user_plans ON user_id = "user-123" AND status = "active"
│      JOIN features ON key = "ocr_scan"
│      → {type: metered, limit: 3, period: day}
│
├──► Set L2: HSET entitlements:user-123 ocr_scan '{"type":"metered","limit":3,"period":"day"}'
│         TTL 5 phút
├──► Set L1: memCache.Store("ent:user-123:ocr_scan", entitlement)
│         TTL 60s
▼
Quota check: Redis INCR quota:user-123:ocr_scan:2026-03-20
│ count = 2
│ 2 ≤ 3 → ALLOW
▼
Return CheckResult{Allowed: true, Used: 2, Remaining: 1, ...}
```

**Next request (within 60s — warm):**

```
Check(ctx, "user-123", "ocr_scan")
│
▼
L1 in-memory cache: "ent:user-123:ocr_scan"
│ HIT → {type: metered, limit: 3, period: day}
│ (skip L2 Redis, skip L3 DB)
▼
Quota check: Redis INCR quota:user-123:ocr_scan:2026-03-20
│ count = 3
│ 3 ≤ 3 → ALLOW
▼
Return CheckResult{Allowed: true, Used: 3, Remaining: 0, ...}
```

**Request khi hết quota:**

```
Check(ctx, "user-123", "ocr_scan")
│
▼
L1 HIT → {type: metered, limit: 3}
▼
Quota check: Redis INCR quota:user-123:ocr_scan:2026-03-20
│ count = 4
│ 4 > 3 → DENY
│ Redis DECR (rollback counter vì request bị chặn)
▼
Return CheckResult{Allowed: false, Used: 3, Remaining: 0, ResetsAt: "2026-03-21T00:00:00Z"}
→ HTTP 429 QUOTA_EXCEEDED + upgrade_cta
```

**Invalidation khi user upgrade:**

```
User upgrade free → pro
│
▼
Update user_plans: status = "active", plan = "pro"
│
├──► Clear L2: DEL entitlements:user-123
├──► Clear L1: memCache.Delete("ent:user-123:*")
▼
Next request → L1 MISS → L2 MISS → DB: resolve "pro" entitlements
→ {type: metered, limit: -1} → unlimited → ALLOW (skip quota check)
```

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

**Bài toán:** Redis counter là source of truth cho hot path nhưng volatile. Cần durable audit trail cho analytics, billing reconciliation, abuse detection.

**Giải pháp:** Async write `usage_records` sau khi request thành công. Không trên hot path.

```go
// Trong middleware, SAU khi handler trả success:
go func() {
    svc.RecordUsage(ctx, userID, featureKey, idempotencyKey)
    // INSERT INTO usage_records (user_id, feature_id, quantity, idempotency_key) ...
}()
```

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
- DB down → fallback Redis cache (L2). Redis down → fallback in-memory (L1)
- Cả 3 down → fail-closed cho feature/scope, fail-open cho quota

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
