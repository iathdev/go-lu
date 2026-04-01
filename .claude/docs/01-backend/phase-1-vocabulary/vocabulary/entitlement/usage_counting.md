# Usage Counting — Đếm số lần sử dụng cho quota

> Bài toán: User Free được scan 3 lần/ngày. Đếm ở đâu? Đếm khi nào? Mất count thì sao?

---

## 1. Storage options

| Option | Latency | Durable | Multi-pod | Analytics | Plan A | Plan B |
|---|---|---|---|---|---|---|
| **A. PostgreSQL only** | 1-5ms | Yes | Yes | Native SQL | Được nhưng thiếu read cache | Được |
| **B. Redis only** | ~0.1ms | No | Yes | Không | Không (mất data = mất enforcement) | Không |
| **C. In-memory** | ~ns | No | **No** | Không | **Loại** (K8s multi-pod) | **Loại** |
| **D. PG (SoT) + Redis (cache)** | 1-5ms write, ~0.1ms read | Yes | Yes | PG side | **Recommended** | **Recommended** |

### Option A: PostgreSQL only

```sql
INSERT INTO usage_counts (user_id, feature_key, period_key, count)
VALUES ($1, $2, $3, 1)
ON CONFLICT (user_id, feature_key, period_key)
DO UPDATE SET count = usage_counts.count + 1
WHERE usage_counts.count < $4
RETURNING count
```

- **Ưu:** Durable, atomic, source of truth rõ ràng, analytics sẵn
- **Nhược:** Mỗi quota check = 1 DB round-trip (~1-5ms). Không có read cache cho UI hiển thị remaining quota
- **Plan:** Hoạt động đúng nhưng thiếu fast read path. Phù hợp nếu chưa có Redis

### Option B: Redis only

```
INCR quota:{user_id}:{feature_key}:{utc_date}   TTL 48h
```

- **Ưu:** ~0.1ms, INCR atomic, key per-day tự reset
- **Nhược:** Volatile — restart = mất count = user được free usage. Không analytics. Không chứng minh usage
- **Plan:** **Loại.** OCR scan tốn tiền thật (external API). Mất count = trả tiền cho usage không bill được

### Option C: In-memory

- **Loại.** K8s horizontal scaling → counter không share across pods. Limit thực tế = `limit × số pods`

### Option D: PG (SoT) + Redis (cache) — Recommended

```
Write path:  PG atomic upsert (1-5ms, durable, source of truth)
Read path:   Redis GET (~0.1ms) → miss → PG SELECT → populate Redis
```

- **Ưu:** PG crash = có WAL recovery. Redis crash = rebuild từ PG. Audit trail tự nhiên. Analytics trên PG
- **Nhược:** Write path chậm hơn Redis-only (~1-5ms vs ~0.1ms). Chấp nhận được cho vocabulary app
- **Plan:** **Recommended cho cả Plan A & B.** Single source of truth. Không drift. Không mất data

**Tại sao PG là SoT, không phải Redis?**

| | Redis as SoT | PG as SoT (chọn) |
|---|---|---|
| Redis crash | **Mất count → user free usage** | Cache miss → read PG. Không mất data |
| Redis restart | Counter reset về 0 | Rebuild cache từ PG |
| Redis eviction | Counter biến mất | Cache miss → PG fallback |
| Billing proof | Drift với PG, không chứng minh | PG là single source, luôn chính xác |
| Industry practice | Không ai dùng cho billing | Stripe, AWS, Cloudflare đều dùng durable store |

> *"Never make a volatile store the source of truth for data that has financial or contractual implications."*

---

## 2. Counting strategy

| Pattern | Mô tả | Vấn đề |
|---|---|---|
| **Post-action count** | Action → INSERT usage → check count after | Race: 2 request cùng action → cả 2 qua trước khi count cập nhật |
| **Atomic reserve in PG** | PG upsert WITH count < limit → action → rollback nếu fail | **Recommended.** Atomic ở DB level, không race |

### Recommended: PG Atomic Reserve → Execute → Confirm/Rollback

```
1. Reserve    PG atomic upsert: INCR count WHERE count < limit → trả count mới
              Nếu không có row returned → over limit → deny
              Update Redis cache với count mới
2. Execute    Handler chạy business logic
3. Confirm    Success → giữ count + INSERT usage_record (audit trail)
   Rollback   Fail → PG UPDATE count = count - 1 + invalidate Redis cache
```

---

## 3. Cache strategy

Option D dùng **hybrid 3 strategy** trong 1 flow:

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Cache Strategy Map                          │
├──────────────┬──────────────────┬────────────────────────────────────┤
│  Operation   │  Strategy        │  Flow                             │
├──────────────┼──────────────────┼────────────────────────────────────┤
│  Reserve     │  Write-through   │  PG upsert → SET Redis            │
│  (write)     │                  │  Cache luôn fresh sau write        │
├──────────────┼──────────────────┼────────────────────────────────────┤
│  Rollback    │  Invalidation    │  PG count-1 → DEL Redis           │
│  (write)     │                  │  Xóa cache, không update           │
│              │                  │  Next read sẽ lazy load từ PG     │
├──────────────┼──────────────────┼────────────────────────────────────┤
│  GetUsage    │  Cache-aside     │  Redis GET → miss → PG SELECT     │
│  (read)      │  (lazy load)     │  → SET Redis. Chỉ populate khi    │
│              │                  │  có read miss                      │
├──────────────┼──────────────────┼────────────────────────────────────┤
│  Audit       │  Write-behind    │  Async goroutine batch INSERT      │
│  (write)     │  (async)         │  Non-blocking, best-effort         │
└──────────────┴──────────────────┴────────────────────────────────────┘
```

**Tại sao mỗi operation dùng strategy khác nhau?**

| Operation | Tại sao strategy này |
|---|---|
| **Reserve → Write-through** | Sau reserve, count đã thay đổi. Nếu không SET Redis ngay → next GetUsage đọc cache cũ → UI hiển thị sai remaining. Write-through đảm bảo cache **luôn đúng sau write** |
| **Rollback → Invalidation** | DEL đơn giản hơn SET (không cần tính count mới). Rollback hiếm khi xảy ra → next read sẽ lazy load từ PG. Tránh bug: SET sai count nếu có concurrent rollback |
| **GetUsage → Cache-aside** | Read-only path cho UI (hiển thị "2/3 scans remaining"). Không cần eager load tất cả user — chỉ populate cache khi user thực sự mở app. Tiết kiệm Redis memory |
| **Audit → Write-behind** | Audit record không ảnh hưởng enforcement. Async giảm ~1-2ms latency trên hot path. Batch 50 records/INSERT giảm DB round-trips |

**Tại sao không dùng 1 strategy cho tất cả?**

| Nếu dùng 1 strategy cho tất cả | Vấn đề |
|---|---|
| Write-through cho tất cả | Audit write-through = thêm ~1-2ms trên hot path cho data không cần sync |
| Cache-aside cho tất cả | Reserve xong nhưng cache chưa update → UI hiển thị remaining sai cho đến khi GetUsage miss + reload |
| Write-behind cho tất cả | Reserve async = không đảm bảo PG đã commit trước khi cho request qua → race condition |

---

## 4. Triển khai Option D — Chi tiết

### 3.1 Tổng quan kiến trúc

```
                         ┌─────────────────────────────────────────────────┐
                         │              Gin Middleware                      │
                         │           CheckQuota("ocr_scan")                │
                         └──────────┬──────────────────────┬───────────────┘
                                    │                      │
                          ┌─────────▼─────────┐            │
                          │  1. RESERVE (PG)   │            │
                          │  Atomic upsert     │            │
                          │  count < limit     │            │
                          │  + Redis SET cache │            │
                          └─────────┬──────────┘            │
                                    │                      │
                           ┌────────▼────────┐             │
                    ┌──NO──┤   Allowed?       ├──YES──┐    │
                    │      └─────────────────┘        │    │
                    ▼                                  ▼    │
            ┌──────────────┐                  ┌────────────▼────────────┐
            │ No row       │                  │  2. EXECUTE             │
            │ returned     │                  │  c.Next() → Handler    │
            │              │                  │  → Use case → Repo     │
            │ Return 429   │                  └────────────┬────────────┘
            │ QUOTA_EXCEEDED│                              │
            └──────────────┘                     ┌─────────▼─────────┐
                                          ┌──NO──┤  Status < 400?    ├──YES──┐
                                          │      └───────────────────┘       │
                                          ▼                                  ▼
                                ┌──────────────────┐              ┌─────────────────────┐
                                │  3a. ROLLBACK     │              │  3b. CONFIRM         │
                                │  PG: count - 1    │              │  Giữ PG count        │
                                │  Redis: DEL cache │              │  Redis cache đã đúng │
                                └──────────────────┘              │  + async INSERT      │
                                                                  │    usage_record      │
                                                                  └─────────────────────┘
                                                                          │
                                                                  ┌───────▼───────┐
                                                                  │  goroutine    │
                                                                  │  INSERT INTO  │
                                                                  │ usage_records │
                                                                  │  (audit log)  │
                                                                  └───────────────┘
```

### 3.2 Database schema

#### usage_counts — counter table (SoT)

```sql
CREATE TABLE usage_counts (
    user_id       UUID        NOT NULL,
    feature_key   VARCHAR(50) NOT NULL,
    period_key    VARCHAR(10) NOT NULL,  -- "2026-04-01" (daily) hoặc "2026-04" (monthly)
    count         INT         NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, feature_key, period_key)
);

CREATE INDEX idx_usage_counts_period ON usage_counts (period_key);
```

| Column | Mô tả |
|---|---|
| `user_id` | Isolate counter per user |
| `feature_key` | `ocr_scan`, `vocab_create`, ... |
| `period_key` | Daily: `"2026-04-01"`, Monthly: `"2026-04"`. Key mới mỗi period → auto reset logic |
| `count` | Current usage count. **Source of truth** |

#### usage_records — audit log (append-only)

```sql
CREATE TABLE usage_records (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    feature_key     VARCHAR(50) NOT NULL,
    request_id      UUID        NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_usage_records_request UNIQUE (user_id, feature_key, request_id)
);

CREATE INDEX idx_usage_records_user_feature ON usage_records (user_id, feature_key, recorded_at);
```

**Tại sao 2 bảng?**
- `usage_counts`: fast atomic counter cho enforcement (1 row per user per feature per period)
- `usage_records`: append-only audit trail cho analytics/billing/abuse detection. Unique trên `request_id` → idempotent

### 3.3 PG atomic reserve

```sql
-- Reserve: atomic increment with limit check
-- Returns new count if allowed, no row if over limit
INSERT INTO usage_counts (user_id, feature_key, period_key, count, updated_at)
VALUES ($1, $2, $3, 1, NOW())
ON CONFLICT (user_id, feature_key, period_key)
DO UPDATE SET
    count = usage_counts.count + 1,
    updated_at = NOW()
WHERE usage_counts.count < $4   -- limit
RETURNING count
```

**Tại sao atomic?**
- `ON CONFLICT DO UPDATE ... WHERE count < limit` là **single statement** → PostgreSQL đảm bảo serializable cho row-level lock
- 2 concurrent requests trên cùng user: PG sẽ serialize — request A lock row, increment, release → request B lock row, thấy count mới, check limit
- Không cần explicit transaction hay SELECT FOR UPDATE

**Unlimited user (limit = -1):**

```go
func (repo *UsageRepository) Reserve(ctx context.Context, userID, featureKey, periodKey string, limit int) (int, bool, error) {
    // Pro user → skip counting
    if limit < 0 {
        return 0, true, nil
    }

    var count int
    result := repo.db.WithContext(ctx).Raw(`
        INSERT INTO usage_counts (user_id, feature_key, period_key, count, updated_at)
        VALUES (?, ?, ?, 1, NOW())
        ON CONFLICT (user_id, feature_key, period_key)
        DO UPDATE SET count = usage_counts.count + 1, updated_at = NOW()
        WHERE usage_counts.count < ?
        RETURNING count
    `, userID, featureKey, periodKey, limit).Scan(&count)

    if result.RowsAffected == 0 {
        return 0, false, nil // over limit
    }
    if result.Error != nil {
        return 0, false, result.Error
    }
    return count, true, nil
}
```

### 3.4 Redis cache design

Redis chỉ là **read cache** — không tham gia enforcement decision.

```
Key:    quota:{user_id}:{feature_key}:{period_key}
Value:  "{count}:{limit}"    VD: "2:3" (used 2 of 3)
TTL:    48h (tự cleanup, đủ buffer qua ngày)
```

```go
// Sau khi PG reserve thành công → update cache
func (repo *UsageCacheRepository) SetCount(ctx context.Context, userID, featureKey, periodKey string, count, limit int) error {
    key := fmt.Sprintf("quota:%s:%s:%s", userID, featureKey, periodKey)
    val := fmt.Sprintf("%d:%d", count, limit)
    return repo.redis.Set(ctx, key, val, 48*time.Hour).Err()
}

// Read path: UI hiển thị remaining quota
func (repo *UsageCacheRepository) GetCount(ctx context.Context, userID, featureKey, periodKey string) (count int, limit int, found bool, err error) {
    key := fmt.Sprintf("quota:%s:%s:%s", userID, featureKey, periodKey)
    val, err := repo.redis.Get(ctx, key).Result()
    if errors.Is(err, redis.Nil) {
        return 0, 0, false, nil // cache miss → caller reads PG
    }
    if err != nil {
        return 0, 0, false, err
    }
    fmt.Sscanf(val, "%d:%d", &count, &limit)
    return count, limit, true, nil
}
```

**Cache invalidation:**
- Reserve thành công → `SET` với count mới
- Rollback → `DEL` (force next read từ PG, tránh stale cache)
- Redis down → skip cache update, không ảnh hưởng enforcement

### 3.5 Rollback flow

```
Khi nào cần rollback?
│
├── Action fail (handler trả 4xx/5xx)
│   VD: OCR service timeout, DB error, validation fail
│   → PG: UPDATE count = count - 1
│   → Redis: DEL cache key
│   → User không mất quota cho request lỗi
│
├── Middleware abort trước handler
│   VD: middleware khác reject (auth fail sau quota check)
│   → PG: UPDATE count = count - 1
│   → Redis: DEL cache key
│
└── Panic trong handler
    → Recovery middleware catch → status 500 → rollback
```

```go
func (repo *UsageRepository) Rollback(ctx context.Context, userID, featureKey, periodKey string) error {
    return repo.db.WithContext(ctx).Exec(`
        UPDATE usage_counts
        SET count = GREATEST(count - 1, 0), updated_at = NOW()
        WHERE user_id = ? AND feature_key = ? AND period_key = ?
    `, userID, featureKey, periodKey).Error
}
```

**`GREATEST(count - 1, 0)`** — defensive: tránh count âm nếu có edge case bất thường.

### 3.6 Async audit trail write

Audit trail (`usage_records`) vẫn async vì:
- Không ảnh hưởng enforcement (enforcement dựa trên `usage_counts` đã commit)
- Giảm latency hot path (~1-2ms cho INSERT thêm)
- Mất 1 audit record không ảnh hưởng quota accuracy

```
                  ┌──────────────┐
  Confirm ──────► │  Channel     │ ──────► goroutine worker pool
  (non-blocking)  │  buffered    │         INSERT usage_records
                  │  size: 1000  │
                  └──────────────┘
                         │
                    Full? Drop + log warning
                    (audit là best-effort)
```

```go
type UsageWriter struct {
    ch      chan UsageRecord
    db      *gorm.DB
    wg      sync.WaitGroup
}

func NewUsageWriter(db *gorm.DB, workers int, bufSize int) *UsageWriter {
    writer := &UsageWriter{
        ch: make(chan UsageRecord, bufSize),
        db: db,
    }
    for i := 0; i < workers; i++ {
        writer.wg.Add(1)
        go writer.worker()
    }
    return writer
}

func (writer *UsageWriter) Enqueue(record UsageRecord) {
    select {
    case writer.ch <- record:
        // queued
    default:
        logger.Warn("[ENTITLEMENT] usage audit buffer full, dropping record")
    }
}

func (writer *UsageWriter) worker() {
    defer writer.wg.Done()
    batch := make([]UsageRecord, 0, 50)
    ticker := time.NewTicker(1 * time.Second)

    for {
        select {
        case record, ok := <-writer.ch:
            if !ok { // channel closed → shutdown
                writer.flush(batch)
                return
            }
            batch = append(batch, record)
            if len(batch) >= 50 {
                writer.flush(batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                writer.flush(batch)
                batch = batch[:0]
            }
        }
    }
}

func (writer *UsageWriter) flush(batch []UsageRecord) {
    if len(batch) == 0 {
        return
    }
    // Batch INSERT — 1 query cho 50 records
    // ON CONFLICT DO NOTHING — idempotent via request_id unique constraint
    writer.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&batch)
}

func (writer *UsageWriter) Close() {
    close(writer.ch)
    writer.wg.Wait()
}
```

### 3.7 Middleware pseudo-code

```go
func CheckQuota(svc Service, featureKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := auth.GetUserID(c)
        periodKey := time.Now().UTC().Format("2006-01-02") // daily

        result, err := svc.Reserve(ctx, userID, featureKey, periodKey)
        if err != nil {
            // PG down → fail-closed: reject 503
            logger.Error(ctx, "[ENTITLEMENT] PG unreachable, fail-closed", zap.Error(err))
            response.HandleError(c, apperr.ServiceUnavailable("common.service_unavailable", err))
            c.Abort()
            return
        }

        if !result.Allowed {
            response.QuotaExceeded(c, result)
            c.Abort()
            return
        }

        // Update Redis cache (best-effort, ignore error)
        _ = svc.UpdateCache(ctx, userID, featureKey, periodKey, result.Used, result.Limit)

        c.Next()

        if c.Writer.Status() >= 400 {
            svc.RollbackUsage(ctx, userID, featureKey, periodKey)
        } else {
            svc.RecordUsageAsync(ctx, userID, featureKey, c.GetString("request_id"))
        }
    }
}
```

### 3.8 Khó khăn và cách xử lý

| Khó khăn | Ảnh hưởng | Giải pháp |
|---|---|---|
| **PG down** | Không reserve được | **Fail-closed**: reject 503. Quota liên quan tiền (OCR scan cost) → không cho qua khi không enforce được |
| **PG slow (> 10ms)** | Hot path chậm | Monitoring + alert. PG atomic upsert trên PK lookup = 1-5ms bình thường. Nếu chậm → check connection pool, index, vacuum |
| **Redis down** | Cache miss → mọi read đi PG | Không ảnh hưởng enforcement. Read path chậm hơn nhưng đúng. PG handle được. Nếu traffic cao gây cache stampede → cân nhắc thêm singleflight cho read path |
| **Redis stale** | UI hiển thị sai remaining | Tự heal: next reserve → SET cache mới. Hoặc TTL 48h tự expire |
| **Rollback fail** | Count tăng vĩnh viễn cho request lỗi | Retry rollback 1 lần. Nếu vẫn fail → log error + metric. Worst case: user mất 1 quota slot, next period reset |
| **Concurrent reserve cùng user** | 2 request đồng thời | PG row-level lock serialize tự động. Request A lock → increment → release → Request B lock → thấy count mới → check limit. Không race |
| **Midnight UTC period rollover** | 23:59:59 reserve key ngày cũ, 00:00:01 key ngày mới | OK — mỗi period_key riêng. Không race |
| **Audit record mất** | Channel full hoặc PG write fail | Acceptable — audit là best-effort. `usage_counts` (SoT) không bị ảnh hưởng. Plan B thêm reconciliation |
| **Goroutine leak (writer)** | Shutdown không drain | `UsageWriter.Close()`: close channel → workers drain → `wg.Wait()` |
| **Count âm** | Rollback khi count đã = 0 (edge case: PG restart giữa reserve và rollback) | `GREATEST(count - 1, 0)` trong SQL |

### 3.9 Cấu hình

```env
# Quota enforcement
QUOTA_FAIL_CLOSED=true          # fail-closed: reject 503 khi PG down

# Usage writer (async audit trail)
USAGE_WRITER_WORKERS=5          # số goroutine worker
USAGE_WRITER_BUFFER=1000        # channel buffer size
USAGE_WRITER_BATCH_SIZE=50      # records per INSERT
USAGE_WRITER_FLUSH_INTERVAL=1s  # flush batch dù chưa đủ size

# Redis cache
QUOTA_CACHE_TTL=172800          # 48h, seconds
```

---

## 5. Plan A vs Plan B

| | Plan A (MVP) | Plan B (Scale) |
|---|---|---|
| **Source of truth** | `usage_counts` table (PG) | Giữ nguyên |
| **Read cache** | Redis — SET sau reserve, DEL on rollback | Giữ nguyên, thêm cache warming on login |
| **Counting** | PG atomic reserve → Execute → Confirm/Rollback | Giữ nguyên |
| **Audit trail** | Async `usage_records` via worker pool (best-effort) | Giữ nguyên, thêm idempotency_key column |
| **Idempotency** | `request_id` unique constraint trên `usage_records` | Thêm idempotency check trước reserve (Redis SET NX) |
| **Reconciliation** | Chưa cần — PG là SoT, không drift | Daily job: compare `COUNT(*)` from `usage_records` vs `usage_counts.count` → alert nếu mismatch |
| **Analytics** | Basic: `SELECT feature_key, COUNT(*) FROM usage_records` | Full: abuse detection, billing, usage trends |
| **Fail mode** | Fail-closed (reject 503) | Giữ nguyên fail-closed |

---

## 6. Service interface

```go
type UsageService interface {
    // Reserve atomic increment trong PG. Trả CheckResult với Allowed, Used, Limit, Remaining
    // Sau khi PG commit → update Redis cache (best-effort)
    Reserve(ctx context.Context, userID string, featureKey string, periodKey string) (*CheckResult, error)

    // RollbackUsage decrement counter trong PG khi action fail sau Reserve
    // + invalidate Redis cache
    RollbackUsage(ctx context.Context, userID string, featureKey string, periodKey string) error

    // RecordUsageAsync enqueue audit record vào worker pool (non-blocking)
    RecordUsageAsync(ctx context.Context, userID string, featureKey string, requestID string)

    // UpdateCache set Redis cache sau reserve thành công (best-effort)
    UpdateCache(ctx context.Context, userID string, featureKey string, periodKey string, used int, limit int) error

    // GetUsage đọc current count. Redis cache → miss → PG.
    // Dùng cho UI hiển thị remaining quota, không dùng cho enforcement
    GetUsage(ctx context.Context, userID string, featureKey string, periodKey string) (*CheckResult, error)
}

type CheckResult struct {
    Allowed   bool
    Used      int
    Limit     int    // -1 = unlimited
    Remaining int
    Feature   string
    ResetsAt  string // ISO 8601, next period start
}
```

`Reserve()` dùng cho quota enforcement (hot path). `GetUsage()` dùng cho read-only display.

---

## 7. So sánh với thiết kế cũ (Redis SoT)

| Aspect | Cũ: Redis SoT + PG audit | Mới: PG SoT + Redis cache |
|---|---|---|
| **Write path** | Redis Lua script (~0.1ms) | PG atomic upsert (~1-5ms) |
| **Read path** | Redis GET (~0.1ms) | Redis GET (~0.1ms), miss → PG |
| **Data loss on Redis crash** | **Mất counter → user free usage** | Cache miss → rebuild từ PG. Không mất data |
| **Data loss on PG crash** | Mất audit trail (best-effort) | WAL recovery. Counter + audit đều durable |
| **Complexity** | Lua script + async PG writer + reconciliation | PG upsert + Redis SET. Đơn giản hơn |
| **Drift** | Redis count ≠ PG count (cần reconciliation) | Không drift — PG là single source |
| **Latency tradeoff** | Write nhanh hơn ~1-5ms | Write chậm hơn ~1-5ms. Chấp nhận được cho vocab app |
| **Industry alignment** | Không standard | Stripe, AWS, Cloudflare pattern |
