# Entitlement System Research — Plan-based Resource Access Control

> Research phục vụ thiết kế hệ thống quản lý quyền truy cập tài nguyên theo plan (C13 trong `technical_challenges.md`).
>
> **Framing:** Đây không chỉ là "feature gating" (chặn/mở Free/Pro). Đây là **Entitlement System** — mỗi plan định nghĩa user được quyền truy cập resource nào, quota bao nhiêu, scope đến đâu. Khi thêm plan mới, thêm resource mới, thay đổi quota → **thay config/data, không sửa code.**

---

## 1. Yêu cầu

### 1.1 Yêu cầu hiện tại từ PRD (Free/Pro)

Nguồn: `docs/requirement.md` §13.1 + `internal/vocabulary/docs/requirement.md` §6.1

| Feature | Free | Pro (Prep HSK subscribers) |
|---|---|---|
| HSK Wordlists | HSK 1-3 | HSK 1-9 |
| Scan/day | Max 3 ảnh | Unlimited |
| Cards/day | Max 20 | Unlimited |
| Flashcard type | Text only | Text + Images (Phase 2: + Video) |
| Stroke & Recall | Guided xem only + Recall 5 từ/ngày | Full Guided + unlimited Recall + Speed Writing (Phase 2) |
| Pronunciation | Trial 3 từ/ngày | Unlimited + Weakness Report |
| Learning Modes | Discover + Recall + Review | All 7 modes (incl. Chat, Mastery) |
| Grammar | Tips giới hạn | Full context + Phase 2 module |
| AI Chat | Không | Unlimited |
| Ads | Non-intrusive | Ad-free |

> ⚠️ PRD §20 Open Question #7: _"Monetization model final?"_ — trạng thái **Open**, chờ Nhi + Tuyến + Sponsors approve trước Sprint 1 (Mar 31, 2026).

### 1.2 Yêu cầu scale — tại sao không chỉ là Free/Pro

Bảng trên là **snapshot hiện tại**. Nhưng hệ thống cần support:

| Scenario tương lai | Ví dụ | Tại sao xảy ra |
|---|---|---|
| **Thêm plan mới** | Basic ($2.99) giữa Free và Pro. Enterprise cho trường học (bulk license) | Monetization model chưa finalize. Commercial team có thể thêm tier bất cứ lúc nào |
| **Thêm resource mới** | Video Flashcard (Phase 2), Translation Practice (HSK 3.0), Handwriting AI evaluation | Mỗi sprint có thể ship feature mới cần entitlement |
| **Thay đổi quota** | Free scan 3→5/ngày (A/B test), Pro pronunciation 100→50/ngày (cost control) | Product/Commercial team điều chỉnh liên tục dựa trên data |
| **Promotional plan** | "Trial Pro 7 ngày", "Student discount", "Prep HSK bundle" | Marketing campaigns |
| **Per-resource pricing** | Mua riêng OCR addon, AI Chat addon (model Pleco) | À la carte model nếu subscription không convert đủ |
| **Multi-app entitlement** | Prep HSK subscriber unlock Pro ở cả Vocab app lẫn Grammar app | Prep ecosystem mở rộng sang nhiều learning utilities |

→ Nếu hardcode `if tier == "pro"` → mỗi thay đổi trên = sửa code, deploy, test. Cần **data-driven entitlement** để thay đổi = update config/DB.

### 1.3 Mô hình khái niệm

```
Plan ──── 1:N ──── Entitlement ──── N:1 ──── Resource
 │                      │
 │                      ├── type: "quota"    → limit + period
 │                      ├── type: "feature"  → enabled: true/false
 │                      └── type: "scope"    → constraint (VD: hsk_level <= 3)
 │
 User ── has ── Plan (từ Prep subscription)
```

**3 loại entitlement:**

| Loại | Ý nghĩa | Ví dụ | Enforcement |
|---|---|---|---|
| **Quota** | Resource X được dùng tối đa Y lần trong period Z | OCR Scan: 3/ngày, Cards: 20/ngày | Counter (Redis) + check trước mỗi request |
| **Feature** | Resource X có được phép dùng hay không | AI Chat: enabled/disabled | Check boolean trước khi vào handler |
| **Scope** | Resource X được dùng nhưng giới hạn phạm vi | HSK content: level ≤ 3, Grammar: tips_only | Check constraint trong use case layer |

### 1.4 Conversion triggers (ảnh hưởng UX khi bị chặn)

Từ PRD §13.2:

| Trigger | Behaviour |
|---|---|
| Soft paywall tại Stroke Practice | Free xem animation nhưng không viết → CTA "Unlock Stroke Practice" |
| HSK Level Gate | Hoàn thành HSK 3 → "Ready for HSK 4? Upgrade." |
| AI Chat preview | Free 1 session/week → "Want more? Go Pro." |
| Memory Score ceiling | Free pathway rất chậm → "Reach Mastered faster with Pro." |

→ Backend cần trả đủ info cho mobile hiển thị CTA: không chỉ reject mà còn trả lý do + remaining quota + upgrade CTA type.

---

## 2. Hiện trạng codebase

### 2.1 Đã có

| Component | File | Mô tả |
|---|---|---|
| **Rate limiter (IP-based)** | `shared/middleware/ratelimit.go` | Token bucket, in-memory, key = IP. Chỉ apply cho public routes (5 req/sec, burst 10) |
| **Redis** | `infrastructure/redis/` | Đã có client. Chỉ dùng cho refresh token store |
| **Auth middleware** | `shared/middleware/auth.go` | Extract JWT → set context: `user_id`, `email`, `prep_user_id`. **Không có plan info** |
| **User domain** | `auth/domain/user.go` | Chỉ: ID, PrepUserID, Email, Name. **Không có plan field** |

### 2.2 Chưa có

- Plan/entitlement model
- Per-user rate limiting
- Entitlement check middleware
- Integration với Prep subscription system
- Quota tracking

---

## 3. Các lựa chọn thiết kế

### 3.1 Entitlement storage — Ở đâu định nghĩa plan + entitlements?

| Option | Mô tả | Ưu điểm | Nhược điểm | Phù hợp khi |
|---|---|---|---|---|
| **A. Config file (YAML)** | Load YAML khi app start → in-memory. Thêm plan = thêm block YAML, restart | Đơn giản, git-controlled, code review | Thay đổi cần deploy. Không per-user override | MVP, 2-5 plans, thay đổi ít |
| **B. Database** | Plans + entitlements trong Postgres. Admin API/UI thay đổi runtime | Runtime-configurable, per-user override, queryable | Cần cache (DB trên hot path), cần build admin UI | > 3 plans, promo, A/B, enterprise deals |
| **C. Remote config (LaunchDarkly/Firebase)** | SaaS service quản lý config | A/B testing built-in, no deploy | External dependency, cost, vendor lock-in | A/B heavy, budget cho SaaS |
| **D. Go constants** | Embedded Go map/struct, compile-time type check | Type-safe, IDE autocomplete, build fail nếu sai field | Thay đổi = sửa code + deploy. Không tách config/code | Team nhỏ, ưu tiên correctness |
| **E. YAML + DB override** | YAML = baseline (git). DB = override cho A/B, promo, enterprise | Stable base + flexible exceptions. `expires_at` cho promo | 2 sources of truth, debug phức tạp | Stable base + cần exception flexibility |
| **F. Consul/etcd** | Distributed KV, native watch/subscribe hot-reload | Real-time update, multi-service consistent | Thêm infra dependency, learning curve | Multi-service ecosystem |

**Migration path:** A (MVP) → E (Growth) → B (Scale). Code check entitlements không đổi — chỉ đổi loader.

### 3.2 Plan source of truth — User thuộc plan nào?

#### Option A: Sync plan vào JWT claim (Recommend)

```
Prep Platform ──(login/refresh)──> Go Backend ──> users.plan_slug column (Postgres)
                                                      │
                                          JWT claim: { "plan": "free" }
                                          Auth middleware set context
```

- Login: gọi Prep `/me` → response kèm subscription status → map sang plan slug → upsert `users.plan_slug`
- JWT: embed `plan: "free"|"pro"|...` → middleware đọc từ token, zero latency
- Refresh token: re-check Prep subscription → update plan nếu thay đổi
- Webhook (Phase 2): Prep push event upgrade/downgrade → update real-time

**Ưu điểm:** Zero latency. Không dependency Prep per-request. Works khi Prep down.

**Nhược điểm:** Stale tối đa 15 phút (access token TTL). User upgrade → phải refresh token.

**Tại sao chấp nhận:** Upgrade là event hiếm (1 lần/user). Mobile trigger force refresh ngay sau purchase. Worst case 15 phút Free → refresh → Pro.

#### Option B: Query Prep realtime

**Loại.** 100-300ms/request. Prep down = app down. 500K extra calls/ngày.

#### Option C: Redis cache (TTL 5 phút)

**Không cần.** JWT claim đã đủ — simpler, faster, more resilient.

### 3.3 Quota enforcement — Đếm usage thế nào?

#### Option A: Redis counter (Recommend)

```
Key:   quota:{user_id}:{feature_key}:{date}    VD: quota:abc-123:ocr_scan:2026-03-20
Value: integer counter
TTL:   48h
```

- `INCR key` → so sánh với entitlement limit → allow/deny
- Atomic (Redis INCR). Key mới mỗi ngày → auto reset

**Ưu điểm:** ~0.1ms, atomic, đã có Redis, horizontal scale.

**Nhược điểm:** Redis down = không đếm được (fail-open/closed decision). Không persist long-term.

#### Option B: Postgres counter

**Không recommend hot path.** 1-5ms/write, 500K extra writes/ngày. Dùng cho audit log (async), không cho synchronous check.

#### Option C: In-memory

**Loại.** Mất khi restart, không share across instances.

### 3.4 Entitlement check — Enforce ở đâu?

#### Option A: Middleware + Use case layer (Recommend)

```
Request → Auth middleware (set plan context)
        → Entitlement middleware (check quota/feature by feature key)
        → Handler
        → Use case (check scope constraints)
```

| Loại entitlement | Enforce layer | Tại sao |
|---|---|---|
| **Quota** | Middleware | Generic — mọi quota check giống nhau (INCR + compare). Middleware nhận `resource` name từ route config |
| **Feature** | Middleware | Generic — check `enabled` boolean. Middleware nhận `resource` name từ route config |
| **Scope** | Use case layer | Domain-specific — "hsk_level ≤ 3" chỉ có nghĩa trong vocabulary context. Không thể generic middleware |

Route registration (data-driven, không hardcode plan name):

```go
// Thay vì: middleware.RequirePro()
// Dùng:    middleware.CheckEntitlement("ai_chat")
api.POST("/ai-chat", middleware.CheckEntitlement("ai_chat"), handler.AIChat)
api.POST("/ocr-scan", middleware.CheckQuota("ocr_scan"), handler.OCRScan)
```

Middleware nhận feature key → lookup entitlement config cho plan hiện tại → check. **Không biết plan nào, không biết limit bao nhiêu — chỉ biết feature key.**

#### Option B: Centralized entitlement service

Tất cả check đi qua 1 service: `entitlementService.Check(ctx, userID, featureKey) → allowed/denied`

**Ưu điểm:** Single point of enforcement. Dễ audit.

**Nhược điểm:** Mọi request phải gọi service → latency. Single point of failure.

**Không cần tách service riêng cho MVP.** Entitlement logic nằm trong shared package, gọi trực tiếp. Tách service khi cần cross-app entitlement (multi-app Prep ecosystem).

---

## 4. Deep-dive Database-driven — Best practices từ industry

> Tổng hợp từ Stripe Entitlements, Stigg, Schematic, OpenMeter (open-source Go), LaunchDarkly.

### 4.1 Canonical DB schema

Consensus pattern từ mọi platform — 6 bảng:

```sql
-- 1. Feature catalog — mô tả resource/capability trong hệ thống
CREATE TABLE features (
    id          UUID PRIMARY KEY,
    key         VARCHAR(80) UNIQUE NOT NULL,  -- stable lookup key: 'ocr_scan', 'ai_chat'
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(20) NOT NULL,         -- 'boolean' | 'numeric' | 'metered' | 'enum'
    metadata    JSONB
);

-- 2. Plans
CREATE TABLE plans (
    id          UUID PRIMARY KEY,
    slug        VARCHAR(50) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(20) NOT NULL,         -- 'free' | 'paid' | 'custom' | 'trial'
    is_active   BOOLEAN DEFAULT TRUE,
    metadata    JSONB
);

-- 3. Plan entitlements — plan X có quyền gì với feature Y
CREATE TABLE plan_entitlements (
    id              UUID PRIMARY KEY,
    plan_id         UUID,
    feature_id      UUID,
    value_boolean   BOOLEAN,                  -- boolean type: enabled?
    value_numeric   BIGINT,                   -- numeric/metered: limit (-1 = unlimited)
    value_json      JSONB,                    -- scope/complex config: {"max_level":3,"allowed":["text"]}
    reset_period    VARCHAR(20),              -- 'day' | 'month' | 'billing_cycle' | NULL (no reset)
    is_soft_limit   BOOLEAN DEFAULT FALSE,    -- soft limit: cho vượt nhưng cảnh báo
    UNIQUE(plan_id, feature_id)
);

-- 4. Customer user_plans
CREATE TABLE user_plans (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    plan_id         UUID,
    status          VARCHAR(20) NOT NULL,     -- 'active' | 'trialing' | 'canceled' | 'past_due'
    started_at      TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ,
    external_id     VARCHAR(255)              -- Prep subscription ID
);

-- 5. Customer-specific overrides — promotional, enterprise, A/B test
CREATE TABLE user_entitlements (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    feature_id      UUID,
    value_boolean   BOOLEAN,
    value_numeric   BIGINT,
    value_json      JSONB,
    reset_period    VARCHAR(20),
    effective_from  TIMESTAMPTZ NOT NULL,
    effective_to    TIMESTAMPTZ,              -- NULL = permanent
    source          VARCHAR(50) NOT NULL,     -- validate ở app layer, không CHECK constraint
    created_by      VARCHAR(255)
);

-- 6. Usage tracking — cho metered features (quota)
CREATE TABLE usage_records (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    feature_id      UUID,
    quantity        BIGINT NOT NULL DEFAULT 1,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    idempotency_key VARCHAR(255) UNIQUE       -- chống double-counting
);
```

**Tại sao 6 bảng thay vì 2 (plans + entitlements)?**

| Khác biệt | 2 bảng đơn giản | 6 bảng canonical |
|---|---|---|
| Feature catalog | Feature name embedded trong entitlements | `features` table riêng — reuse across plans, queryable |
| Feature types | JSONB catch-all | Typed columns (`value_boolean`, `value_numeric`, `value_json`) — validate dễ hơn |
| Per-user override | Không | `user_entitlements` — promo, enterprise, A/B test |
| Usage tracking | Redis counter only | `usage_records` — durable audit trail + Redis counter cho hot path |
| Reset policy | Hardcode | `reset_period` column — data-driven |
| Soft limit | Không | `is_soft_limit` — cho vượt nhưng cảnh báo (Stigg pattern) |

### 4.2 Entitlement resolution algorithm

Mọi platform đều converge về cùng precedence:

```
1. user_entitlements (per-user override)  → highest priority
2. plan_entitlements (via active subscription) → default
3. System default                              → deny (fail-closed)
```

Cho metered features: `hasAccess = (usage < limit) OR isSoftLimit`

### 4.3 3 loại access control — taxonomy chuẩn

| Type | Reset? | Ví dụ | Column |
|---|---|---|---|
| **Boolean** (flag) | Không | AI Chat enabled, SSO | `value_boolean` |
| **Numeric** (limit/config) | Không | Max folders, max HSK level | `value_numeric` / `value_json` |
| **Metered** (quota) | Có (`reset_period`) | OCR scan/ngày, cards/ngày | `value_numeric` + `reset_period` + `usage_records` |

### 4.4 Caching — two-level strategy

```
Request
  │
  ▼
Level 1: In-memory (microseconds)
  │ Resolved entitlement set per user. TTL 30-60s. Invalidate on event.
  │
  ▼ (miss)
Level 2: Redis (milliseconds)
  │ Hash: entitlements:{user_id} → {feature_key: value}. Invalidate via pub/sub.
  │
  ▼ (miss)
Level 3: Database (last resort)
  │ JOIN user_plans + plan_entitlements + user_entitlements. Cold start only.
```

**Invalidation triggers:** subscription change, plan entitlements modified, customer override added/removed, usage threshold crossed.

**Pitfalls:**
- Cache stampede → single-flight/mutex
- Stale sau plan change → webhook-driven invalidation, không chỉ TTL
- Metered features → Redis counter (hot path) + `usage_records` (async audit)

### 4.5 Plan change — best practices

| Aspect | Best practice | Lý do |
|---|---|---|
| **Upgrade** | Immediate — entitlements mở ngay | User vừa trả tiền → phải có value ngay |
| **Downgrade** | Configurable: end of billing cycle hoặc immediate + grace period | Tránh UX tệ |
| **Data retention** | **Không bao giờ xóa data** khi downgrade | Trust violation, legal issue |
| **Usage carryover** | Configurable: reset, rollover with cap, preserve | `rollover_max` (OpenMeter pattern) |
| **Trials** | Model như plan riêng có `expires_at`, không phải boolean flag | Track trial conversion |

### 4.6 Reference implementations

| Platform | Model | Relevance |
|---|---|---|
| **Stripe Entitlements** | Feature → Product → Active Entitlement. Boolean only. Persist locally + webhook sync | Pattern cho Prep subscription sync |
| **Stigg** | Plan → Entitlement → Feature + Add-on + Promotional override. 4 feature types, credit system, sidecar cache | Reference model cho schema design |
| **OpenMeter** (Go, open-source) | Feature → Entitlement → Grant → Balance. Postgres + ClickHouse + Kafka. Grant priority, rollover, soft limits | Closest Go implementation |
| **Schematic** | Plan → Feature → Flag ("Smart Flags"). Flags auto-evaluate entitlement state | Pattern cho flag-based enforcement |
| **LaunchDarkly** | Segment → Flag → Variation. Feature flags as entitlements | Risk: connection fail → single fallback |

### 4.7 Key pitfalls

| Pitfall | Hậu quả | Giải pháp |
|---|---|---|
| Hardcode plan name (`if plan == "pro"`) | Pricing change = code change | Lookup by feature key, không by plan |
| JWT chứa entitlements | Stale đến khi re-auth | JWT chỉ chứa `plan_slug`. Resolution từ cached DB |
| Check DB mỗi request | 1-5ms/req, overload | Two-level cache |
| Không idempotency usage tracking | Double-counting | `idempotency_key` trong `usage_records` |
| Hardcode reset policy | Đổi period = refactor | `reset_period` là data column |
| Entitlements = authorization | Mix plan logic với role logic | Layer riêng: Entitlements ≠ RBAC |

---

## 5. So sánh các app/platform tương tự

### 5.1 Entitlement patterns trong industry

| Platform | Model | Entitlement architecture | Scale |
|---|---|---|---|
| **Stripe Billing** | Subscription + metered usage | `Entitlement` API: plan → list of features. `Usage Records` cho metered billing. Customer portal cho self-serve upgrade | Triệu merchants, hàng trăm plan variants |
| **AWS IAM** | Policy-based | JSON policies attached to users/roles. Each policy = list of `{resource, action, effect}`. Evaluated at request time | Hàng triệu resources, hàng nghìn action types |
| **Spotify** | Free + Premium | Server-side feature flags per plan. Real-time entitlement check (gRPC). Catalog access controlled per-region per-plan | 600M+ users, 2 plans nhưng hàng trăm feature flags |
| **Duolingo** | Free + Super | Hearts system (quota), lesson gating (feature lock). Server-side config — thay đổi quota qua experiment framework | 100M+ users, heavy A/B testing trên entitlements |
| **LaunchDarkly** | Feature flag SaaS | Entitlements = feature flags with targeting rules. Per-user, per-plan, per-segment. SDK evaluates locally | 100K+ feature flags across customers |

### 5.2 Education apps cụ thể

| App | Tier model | Entitlement cơ chế | Nhận xét |
|---|---|---|---|
| **Duolingo** | Free + Super | Hearts = quota (5/ngày, restore 1/5h). Feature lock: Super features. Experiment-driven — quotas thay đổi liên tục qua A/B | Gây controversy (user ghét hearts) nhưng conversion rate cao |
| **Quizlet** | Free + Plus | Monthly quota (8 sets/tháng Free). Feature lock: AI, images, offline | Chuyển từ unlimited Free → quota Free 2024 — backlash lớn nhưng revenue tăng |
| **HelloChinese** | Free + VIP | Content lock: ~30% lessons Free. Feature lock: AI, pronunciation | Không quota — lock toàn bộ advanced content |
| **SuperChinese** | Free + Pro | Content lock: HSK 1-2 Free, 3+ Pro. Feature lock: AI tutor | Tương tự Prep model |
| **Pleco** | Free + add-ons | À la carte: mua riêng OCR ($9.99), handwriting ($9.99), audio ($9.99) | Per-resource pricing — mỗi resource = 1 entitlement purchase |

### 5.3 Nhận xét

- **Quota (daily limit):** Chỉ Duolingo và Quizlet dùng. Prep cần vì user **tạo content** (scan, cards) → resource cost per action. Content lock không đủ.
- **Best practices:**
  - Trả **remaining quota + reset time** (Quizlet): `{ remaining: 0, resets_at: "..." }`
  - CTA **ngay tại điểm chặn** (Duolingo, HelloChinese) — không redirect pricing page
  - **Graceful degradation** gần limit: warning "2 scans remaining" trước khi hết
- **Trend:** Mọi platform đều hướng tới **data-driven entitlements** — plan config tách khỏi code, thay đổi qua admin tool hoặc experiment framework.

---

## 6. Bài toán "per day" — Timezone handling

### 6.1 Vấn đề

Quota "3/ngày" — "ngày" theo timezone nào?

| Option | Ưu điểm | Nhược điểm |
|---|---|---|
| **UTC midnight** | Đơn giản. Consistent server-side | User VN (UTC+7): reset 7:00 sáng. Confusing |
| **User timezone midnight** | UX tự nhiên | Cần lưu timezone. Edge case: đổi timezone → game quota |
| **Rolling 24h window** | Không cần timezone. Fair | UX confusing: "Khi nào được scan lại?" |

### 6.2 Recommend: UTC midnight + display offset

- **Server:** Key dùng UTC date: `quota:{user_id}:ocr_scan:2026-03-20`
- **Client:** Hiển thị reset theo local timezone: "Quota resets at 7:00 AM"
- **Tại sao:** Complexity thấp, user VN reset 7:00 sáng — acceptable. Phase 2 migrate user timezone nếu cần.

### 6.3 Edge cases

| Case | Xử lý |
|---|---|
| User đổi timezone (travel) | UTC-based → không ảnh hưởng |
| User spoof timezone | Server dùng UTC → không bypass |
| Midnight UTC race | Redis INCR atomic → safe |

---

## 7. Bài toán upgrade/downgrade mid-session

### 7.1 Upgrade (plan thấp → plan cao)

| Scenario | Xử lý |
|---|---|
| User mua plan mới | Mobile → Prep purchase → success → force refresh token → new JWT có `plan: "pro"` → entitlements unlock ngay |
| Đã hết quota hôm nay → upgrade | Middleware check plan entitlement: new plan limit = unlimited → bypass counter. **Unlock ngay** |
| Đang ở giữa session → upgrade | Session tiếp tục. Features mới xuất hiện sau token refresh |

### 7.2 Downgrade (plan cao → plan thấp)

| Scenario | Xử lý |
|---|---|
| Subscription hết hạn | Prep webhook (Phase 2) hoặc next token refresh → `plan: "free"`. Locked features bị chặn ngay |
| Đang ở giữa locked feature session | Cho phép hoàn thành session hiện tại (graceful). Request tiếp theo bị chặn. **Tại sao:** cắt giữa chừng = UX tệ |
| Đã có data tạo từ plan cao (HSK 4+ cards) | **Không xóa.** User giữ data đã tạo, vẫn review được. Chỉ không tạo mới từ locked scope. **Tại sao:** xóa user data = trust violation |
| Memory Score đã tính Max_Points cao | Normalize score theo plan mới. Hiển thị message: "Your progress is preserved" |

---

## 8. Fail-open vs Fail-closed

Khi Redis down hoặc entitlement check fail:

| Loại | Khi lỗi | Tại sao |
|---|---|---|
| **Quota** | **Fail-open** — cho request đi qua | Worst case: Free user scan 5 thay vì 3 → tốn thêm $0.003. Tốt hơn block user hợp lệ |
| **Feature** | **Fail-closed** — block, mặc định plan thấp nhất | Fail-open = cho Free dùng paid feature → revenue leak |
| **Scope** | **Fail-closed** — mặc định scope thấp nhất | Tương tự feature |

**Monitoring:** Log mỗi lần fail-open. Alert nếu > 5%/5 phút. Daily reconciliation: actual usage vs quota.

---

## 9. Response format khi bị chặn

### 9.1 Quota exceeded (429)

```json
{
  "success": false,
  "error": {
    "code": "QUOTA_EXCEEDED",
    "message": "Daily scan limit reached",
    "details": {
      "feature": "ocr_scan",
      "entitlement_type": "quota",
      "limit": 3,
      "used": 3,
      "remaining": 0,
      "period": "day",
      "resets_at": "2026-03-21T00:00:00Z",
      "current_plan": "free",
      "upgrade_cta": "upgrade_scan_limit"
    }
  }
}
```

### 9.2 Feature not entitled (403)

```json
{
  "success": false,
  "error": {
    "code": "FEATURE_NOT_ENTITLED",
    "message": "AI Chat is not available on your current plan",
    "details": {
      "feature": "ai_chat",
      "entitlement_type": "feature",
      "current_plan": "free",
      "required_plan": "pro",
      "upgrade_cta": "upgrade_ai_chat"
    }
  }
}
```

### 9.3 Scope exceeded (403)

```json
{
  "success": false,
  "error": {
    "code": "SCOPE_EXCEEDED",
    "message": "HSK level 4+ content is not available on your current plan",
    "details": {
      "feature": "hsk_content",
      "entitlement_type": "scope",
      "constraint": { "max_level": 3 },
      "requested": { "level": 4 },
      "current_plan": "free",
      "upgrade_cta": "upgrade_hsk_level"
    }
  }
}
```

### 9.4 Quota info API (cho mobile hiển thị remaining)

```
GET /api/entitlements/me
```

```json
{
  "success": true,
  "data": {
    "plan": "free",
    "entitlements": {
      "ocr_scan": { "type": "quota", "limit": 3, "used": 1, "remaining": 2, "resets_at": "2026-03-21T00:00:00Z" },
      "create_card": { "type": "quota", "limit": 20, "used": 7, "remaining": 13, "resets_at": "2026-03-21T00:00:00Z" },
      "ai_chat": { "type": "feature", "enabled": false },
      "hsk_content": { "type": "scope", "max_level": 3 }
    }
  }
}
```

Mobile gọi API này khi app open + khi cần hiển thị quota badge (VD: "2 scans left").

---

## 10. Entitlement map tổng hợp (current PRD)

| Resource | Type | Free | Pro | Enforce layer |
|---|---|---|---|---|
| `ocr_scan` | quota | 3/day | unlimited | Middleware |
| `create_card` | quota | 20/day | unlimited | Middleware |
| `pronunciation` | quota | 3/day | unlimited | Middleware |
| `recall_writing` | quota | 5/day | unlimited | Middleware |
| `ai_chat` | feature | disabled | enabled | Middleware |
| `mastery_check` | feature | disabled | enabled | Middleware |
| `speed_writing` | feature | disabled | enabled (Phase 2) | Middleware |
| `weakness_report` | feature | disabled | enabled | Middleware |
| `hsk_content` | scope | max_level: 3 | max_level: 9 | Use case |
| `flashcard_type` | scope | ["text"] | ["text","image","video"] | Use case |
| `grammar` | scope | tips_only | full | Use case |

**Tổng: 4 quota + 4 feature + 3 scope = 11 entitlements.**

**Ví dụ mở rộng tương lai (không cần sửa code):**

```yaml
# Thêm plan Basic — chỉ cần thêm block YAML
basic:
  display_name: "Basic"
  entitlements:
    ocr_scan:      { type: quota, limit: 10, period: day }
    ai_chat:       { type: feature, enabled: true }     # Basic có AI Chat
    hsk_content:   { type: scope, max_level: 6 }        # Basic mở đến HSK 6

# Thêm resource mới "video_flashcard" — thêm 1 dòng per plan
free:
  video_flashcard: { type: feature, enabled: false }
pro:
  video_flashcard: { type: feature, enabled: true }
basic:
  video_flashcard: { type: quota, limit: 10, period: day }  # Basic: 10 video/ngày
```

→ Thêm plan: thêm block YAML. Thêm resource: thêm 1 dòng/plan + 1 check point trong code (1 lần). Thay quota: sửa số trong YAML.

---

## 11. Chi phí nếu không enforce entitlements

| Resource | Cost/call | Free user worst case/ngày | Cost/user/ngày | 10K Free users |
|---|---|---|---|---|
| OCR Scan | $0.0015 | 50 scans | $0.075 | $750/ngày |
| AI Chat | ~$0.01 | 100 messages | $1.00 | $10,000/ngày |
| Pronunciation | ~$0.005 | 50 checks | $0.25 | $2,500/ngày |
| **Total** | | | **$1.325** | **$13,250/ngày** |

Với entitlement enforcement:

| Resource | Free limit | Cost/user/ngày | 10K Free users |
|---|---|---|---|
| OCR Scan | 3/ngày | $0.0045 | $45/ngày |
| AI Chat | Disabled | $0 | $0 |
| Pronunciation | 3/ngày | $0.015 | $150/ngày |
| **Total** | | **$0.0195** | **$195/ngày** |

→ **Entitlement system giảm cost ~68x** ($13,250 → $195/ngày cho 10K users).

---

## References

| Source | URL | Relevance |
|---|---|---|
| Stripe Entitlements Docs | https://docs.stripe.com/billing/entitlements | Pattern: Feature → Product → Active Entitlement. Recommend persist locally + webhook sync |
| Stripe Active Entitlement API | https://docs.stripe.com/api/entitlements/active-entitlement | API contract reference |
| Stigg Domain Model | https://docs.stigg.io/docs/domain-model | Đầy đủ nhất: 4 feature types, credit system, sidecar cache, promotional overrides |
| Stigg — Engineer's Guide to Entitlements | https://www.stigg.io/blog-posts/the-engineers-guide-to-entitlements | Tại sao entitlement ≠ feature flag ≠ authorization |
| OpenMeter — Entitlements & Grants (Go, open-source) | https://github.com/openmeterio/openmeter | Closest Go reference: Postgres + ClickHouse + Kafka, grant priority, rollover, soft limits |
| OpenMeter Entitlement Docs | https://openmeter.io/docs/billing/entitlements/entitlement | 3 entitlement types: boolean, static, metered |
| Schematic — Feature Flags for Entitlements | https://schematichq.com/blog/guide-how-to-use-feature-flags-to-manage-entitlements-without-writing-code | Pattern: "Smart Flags" auto-evaluate entitlement state |
| LaunchDarkly — Entitlements with Feature Flags | https://launchdarkly.com/docs/guides/flags/entitlements | Segment-based plan gating. Risk: connection fail → single fallback |
| Entitlements Model for Customer Tiers | https://appmaster.io/blog/entitlements-model-plans-limits-flags | Canonical schema pattern (plans, features, entitlements, usage) |
| Feature Gating Approaches Comparison (Stigg) | https://dev.to/getstigg/how-to-gate-end-user-access-to-features-shortcomings-of-plan-identifiers-authorization-feature-flags-38dh | Tại sao hardcode plan identifiers là anti-pattern |
| Apache Casbin (Go) | https://github.com/apache/casbin | ACL/RBAC/ABAC policy engine — authorization layer, không phải entitlement |
| Flexprice — Open-source Billing + Entitlements | https://flexprice.io | Plan-level + user-level entitlements from billing setup |
