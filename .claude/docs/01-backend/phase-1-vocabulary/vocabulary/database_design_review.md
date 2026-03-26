# Database Design Review & Improvement Proposals

> Reviewer: Claude | Date: 2026-03-25
> Scope: Phân tích `database_design.md` — schema quality, scalability, JSONB strategy, indexing, hot table performance

---

## Tổng quan đánh giá

Design hiện tại **tốt ở mức conceptual** — 5 principles đúng hướng, table grouping rõ ràng, separation giữa content/learning/organization/gating hợp lý. Tuy nhiên có **7 vấn đề cần cải thiện** trước khi implement, chia thành 3 nhóm: Structural Issues, Performance Issues, Missing Pieces.

---

## A. Structural Issues

### A1. JSONB overuse — mất type safety và query performance

**Hiện tại:** 10 JSONB columns across 7 tables:

| Table | Column | Mục đích |
|---|---|---|
| `languages` | `config` | Feature flags per language |
| `proficiency_levels` | `metadata` | Language-specific stats |
| `vocabularies` | `metadata` | Radicals, stroke, tones... |
| `vocabularies` | `examples` | Example sentences |
| `topics` | `names` | Multi-lang display names |
| `grammar_points` | `examples` | Example sentences |
| `grammar_points` | `rule` | Multi-lang rules |
| `grammar_points` | `common_mistakes` | Multi-lang mistakes |
| `pronunciation_scores` | `dimensions` | Scoring dimensions |
| `learning_events` | `event_data` | Mode-specific event payload |

**Vấn đề:**

1. **Không có schema validation ở DB level.** Ai cũng có thể insert `{"radicals": 123}` thay vì `{"radicals": ["子"]}`. Bugs phát hiện ở runtime, không phải migration time.

2. **Query performance trên JSONB kém hơn columns.** Ví dụ: "Lấy tất cả vocabulary có `stroke_count > 10`" → phải dùng `metadata->>'stroke_count'` + cast, không dùng được B-tree index thông thường. GIN index trên `metadata` hỗ trợ containment (`@>`) nhưng **không hỗ trợ range queries** (`>`, `<`, `BETWEEN`).

3. **ORM mapping phức tạp.** GORM cần custom scanner/valuer cho mỗi JSONB field. Code phình ra với type assertions và nil checks.

**Phân loại JSONB — khi nào hợp lý, khi nào không:**

| JSONB Column | Verdict | Lý do |
|---|---|---|
| `languages.config` | ✅ Keep | Low-cardinality reference table (~5 rows). Đọc 1 lần, cache. Schema khác nhau per language → JSONB hợp lý. |
| `proficiency_levels.metadata` | ✅ Keep | Tương tự — reference data, hiếm query. |
| `vocabularies.metadata` | ⚠️ Cân nhắc | **Hot path** — vocabulary detail page, search results, learning modes đều cần access. `stroke_count`, `recognition_only` được filter/sort thường xuyên. |
| `vocabularies.examples` | ✅ Keep | Array of objects, variable length, hiếm khi filter/sort by example content. |
| `topics.names` | ✅ Keep | Display-only, không query. |
| `grammar_points.examples/rule/common_mistakes` | ✅ Keep | Display-only, multi-lang content. |
| `pronunciation_scores.dimensions` | ✅ Keep | Schema thay đổi per language, write-heavy → JSONB tránh ALTER TABLE. |
| `learning_events.event_data` | ✅ Keep | Event sourcing payload, schema varies by mode. Classic JSONB use case. |

**Đề xuất cho `vocabularies.metadata`:**

```sql
-- Thay vì nhồi tất cả vào metadata JSONB, tách ra:

-- Option A: Hybrid — extract high-query fields thành columns
ALTER TABLE vocabularies ADD COLUMN stroke_count      SMALLINT;
ALTER TABLE vocabularies ADD COLUMN recognition_only  BOOLEAN DEFAULT false;
ALTER TABLE vocabularies ADD COLUMN stroke_data_url   VARCHAR(500);
-- Giữ metadata JSONB cho phần còn lại (radicals, tone_numbers, kanji/hiragana...)

-- Option B: Giữ nguyên JSONB nhưng thêm generated columns (Postgres 12+)
ALTER TABLE vocabularies
    ADD COLUMN stroke_count SMALLINT GENERATED ALWAYS AS ((metadata->>'stroke_count')::SMALLINT) STORED;
-- Cho phép index trên generated column
CREATE INDEX idx_vocab_stroke ON vocabularies(stroke_count) WHERE stroke_count IS NOT NULL;
```

**Recommendation: Option A (Hybrid).** Lý do:
- `stroke_count` và `recognition_only` là fields mà learning modes cần filter (e.g. "chỉ lấy từ có stroke data để practice Stroke mode")
- Generated columns thêm complexity cho ORM mà không cần thiết
- Phần metadata còn lại (`radicals`, `tone_numbers`, `kanji/hiragana`) thực sự language-specific → giữ JSONB

---

### A2. `vocabulary_meanings` — unique constraint quá chặt

```sql
UNIQUE(vocabulary_id, target_lang, meaning)
```

**Vấn đề:** Unique trên `meaning` TEXT column:
1. **Meaning có thể rất dài** — "to study; to learn (especially through practice)" → unique index on TEXT rất tốn storage
2. **Không cần thiết về mặt business** — nếu admin import 2 lần cùng meaning, DB reject. Nhưng import flow đã check duplicate ở application level (by hanzi). Case 2 meanings giống hệt nhau cho cùng vocab + cùng lang thì cực hiếm.
3. **Collation-sensitive** — "to study" vs "To study" vs "to study " (trailing space) là 3 values khác nhau theo unique constraint.

**Đề xuất:**

```sql
-- Bỏ unique trên meaning, dùng composite unique trên (vocabulary_id, target_lang, sort_order)
-- Đảm bảo mỗi vocab + lang có thứ tự nghĩa duy nhất
CREATE TABLE vocabulary_meanings (
    id              UUID PRIMARY KEY,
    vocabulary_id   UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    target_lang     VARCHAR(10) NOT NULL,
    meaning         TEXT NOT NULL,
    is_primary      BOOLEAN DEFAULT false,
    sort_order      INTEGER DEFAULT 0,
    UNIQUE(vocabulary_id, target_lang, sort_order)  -- thay vì unique trên meaning
);
```

**Bổ sung:** Thêm CHECK constraint cho `is_primary` — mỗi (vocabulary_id, target_lang) chỉ có tối đa 1 `is_primary = true`:

```sql
-- Partial unique index thay vì CHECK constraint (dễ hơn, performant hơn)
CREATE UNIQUE INDEX idx_vm_primary
    ON vocabulary_meanings(vocabulary_id, target_lang)
    WHERE is_primary = true;
```

---

### A3. `user_vocabulary_progress` — 9 score columns = hardcode modes

```sql
score_discover       DECIMAL(3,1) DEFAULT 0,
score_recall         DECIMAL(3,1) DEFAULT 0,
score_stroke_guided  DECIMAL(3,1) DEFAULT 0,
score_stroke_recall  DECIMAL(3,1) DEFAULT 0,
score_pinyin_drill   DECIMAL(3,1) DEFAULT 0,
score_ai_chat        DECIMAL(3,1) DEFAULT 0,
score_review         DECIMAL(3,1) DEFAULT 0,
score_mastery_check  DECIMAL(3,1) DEFAULT 0,
spacing_score        DECIMAL(3,1) DEFAULT 0,
```

**Vấn đề:**
1. **Thêm learning mode mới = ALTER TABLE** trên bảng 5M rows. Ví dụ Phase 2 thêm "Speed Writing" mode → cần add `score_speed_writing` column.
2. **9 columns "waste" cho ngôn ngữ ít modes.** Thai không có stroke → 2 columns luôn = 0. Nhưng vẫn phải store + read.
3. **Weights hardcode ở application.** Nếu muốn A/B test weight khác nhau (e.g. Recall weight 3 thay vì 2) → phải deploy code, không thể config.

**Tuy nhiên, giữ columns có lợi thế:**
- **UPDATE atomic:** `UPDATE SET score_recall = 1.5` — 1 statement, no race condition
- **Query đơn giản:** `SELECT * FROM uvp WHERE user_id = ?` — tất cả scores trong 1 row
- **Không JOIN:** Không cần bảng phụ

**Đề xuất: Giữ nguyên columns nhưng bổ sung `version` cho optimistic locking:**

```sql
ALTER TABLE user_vocabulary_progress ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
-- Code: UPDATE ... SET score_recall = ?, version = version + 1 WHERE id = ? AND version = ?
```

Lý do giữ columns:
- 5M rows × 9 DECIMAL columns = ~180MB thêm — acceptable
- ALTER TABLE thêm 1 column trên 5M rows chỉ mất vài giây (Postgres 11+ `ADD COLUMN DEFAULT` là metadata-only)
- Hot path (update score sau mỗi learning event) cần tối thiểu hóa queries

**Nếu muốn flexible hơn (Phase 3+):** Migrate sang JSONB `mode_scores`:
```sql
mode_scores JSONB DEFAULT '{}'::jsonb
-- { "discover": 1.0, "recall": 1.5, "review": 2.0 }
-- Modes không dùng → không có key, không waste space
```
Nhưng đây là premature — đợi khi thực sự thêm mode mới thì mới cần.

---

### A4. `proficiency_levels` — stats columns nên tách ra

```sql
total_vocabulary   INTEGER,
total_characters   INTEGER,
total_syllables    INTEGER,
total_grammar      INTEGER,
```

**Vấn đề:** 4 stat columns này:
1. **Derivable** — có thể COUNT từ `vocabularies`, `grammar_points` tables
2. **Stale risk** — nếu import thêm vocab mà quên update stats → data inconsistent
3. **Language-specific** — `total_syllables` chỉ relevant cho Chinese, `total_characters` nghĩa khác cho Japanese (kanji count)

**Đề xuất:** Bỏ 4 columns này, thay bằng computed values hoặc di vào `metadata`:

```sql
CREATE TABLE proficiency_levels (
    id            UUID PRIMARY KEY,
    language_id   UUID NOT NULL REFERENCES languages(id),
    code          VARCHAR(20) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    stage         VARCHAR(50),
    sort_order    INTEGER NOT NULL,
    access_tier   VARCHAR(20) DEFAULT 'free',
    -- Bỏ total_vocabulary, total_characters, total_syllables, total_grammar
    -- Chuyển vào metadata nếu cần hardcode stats chuẩn:
    metadata      JSONB DEFAULT '{}'::jsonb,
    -- { "total_vocabulary": 300, "total_characters": 246, "total_syllables": 269, "total_grammar": 20,
    --   "writing_characters": 0, "recognition_characters": 246 }
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(language_id, code)
);
```

Lý do: Đây là **reference data** (seed 1 lần, hiếm thay đổi). Gộp vào `metadata` JSONB hợp lý hơn tách 4 columns mà mỗi language dùng khác nhau.

---

## B. Performance Issues

### B1. `learning_events` — 75M rows/month, thiếu partitioning strategy

**Volume:** 50K MAU × 50 events/day × 30 days = **75M rows/month = 900M rows/year**

**Current design:**

```sql
CREATE INDEX idx_le_user_vocab ON learning_events(user_id, vocabulary_id);
CREATE INDEX idx_le_user_mode ON learning_events(user_id, mode, created_at DESC);
CREATE INDEX idx_le_session ON learning_events(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_le_created ON learning_events(created_at);
```

**Vấn đề:**
1. **Không partition** — 900M rows trong 1 table. Index bloat, vacuum chậm, query degradation.
2. **UUID PK** — random UUID = random I/O on insert. Với 75M inserts/month, B-tree page splits liên tục.
3. **FK to `vocabularies`** — mỗi INSERT phải check FK existence. Với write-heavy table, FK overhead đáng kể.
4. **4 indexes** trên write-heavy table — mỗi INSERT cập nhật 4 indexes.

**Đề xuất:**

```sql
-- 1. Partition by month (RANGE on created_at)
CREATE TABLE learning_events (
    id            UUID NOT NULL,  -- bỏ PK constraint (partition key phải nằm trong PK)
    user_id       UUID NOT NULL,
    vocabulary_id UUID NOT NULL,
    mode          VARCHAR(30) NOT NULL,
    score         DECIMAL(5,2),
    q_score       SMALLINT,
    is_correct    BOOLEAN,
    duration_ms   INTEGER,
    event_data    JSONB DEFAULT '{}'::jsonb,
    session_id    UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)  -- created_at phải nằm trong PK cho partition
) PARTITION BY RANGE (created_at);

-- Tạo partition per month (automate qua pg_partman hoặc cron)
CREATE TABLE learning_events_2026_04 PARTITION OF learning_events
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- 2. Dùng UUIDv7 thay vì UUIDv4 — monotonic, sequential insert friendly
-- (Go: uuid.NewV7() — đã dùng sẵn trong codebase)

-- 3. Bỏ FK references — verify ở application layer
-- learning_events không cần FK to users, vocabularies, learning_sessions
-- Lý do: hot write path, FK check trên 900M-row table là bottleneck
-- Application đã validate user_id (từ JWT) và vocabulary_id (từ prior query)

-- 4. Giảm indexes — chỉ giữ essential:
CREATE INDEX idx_le_user_mode ON learning_events(user_id, mode, created_at DESC);
CREATE INDEX idx_le_session ON learning_events(session_id) WHERE session_id IS NOT NULL;
-- Bỏ idx_le_user_vocab (covered by session-based queries)
-- Bỏ idx_le_created (partition pruning thay thế)
```

**Retention policy:** Events > 12 months → archive sang cold storage hoặc DROP partition:
```sql
-- Hàng tháng:
DROP TABLE IF EXISTS learning_events_2025_04;  -- drop partition cũ 12 tháng
```

---

### B2. `pronunciation_scores` — 15M rows/month, tương tự learning_events

**Đề xuất:** Partition by month, bỏ FK, tương tự B1.

```sql
CREATE TABLE pronunciation_scores (
    id                UUID NOT NULL,
    user_id           UUID NOT NULL,
    vocabulary_id     UUID NOT NULL,
    learning_event_id UUID,  -- bỏ FK, app-level reference
    unit_index        SMALLINT NOT NULL,
    unit_text         VARCHAR(50) NOT NULL,
    overall_score     SMALLINT,
    dimensions        JSONB DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

---

### B3. `user_daily_counters` — composite PK nhưng thiếu cleanup

```sql
PRIMARY KEY (user_id, counter_date, counter_type)
```

**Vấn đề:** Bảng này grow vô hạn. 50K users × 4 counter types × 365 days = **73M rows/year**. Rows cũ hơn 1 ngày không bao giờ cần lại (rate limit chỉ check hôm nay).

**Đề xuất:**

```sql
-- Option A: TTL cleanup (cron job hàng ngày)
DELETE FROM user_daily_counters WHERE counter_date < CURRENT_DATE - INTERVAL '7 days';

-- Option B (recommended): Dùng Redis thay vì Postgres
-- Key: quota:{user_id}:{counter_type}:{date}
-- INCR + EXPIRE 86400s
-- Không cần table này trong Postgres nếu đã có Redis
```

Doc `plan_a_config_entitlement.md` đã đề xuất Redis cho rate limiting. Nếu implement theo Plan A → **bỏ bảng `user_daily_counters`** khỏi Postgres schema, dùng Redis.

---

### B4. `vocabularies` search — thiếu full-text search strategy

**Current:** Chỉ có GIN indexes trên `examples` và `metadata` JSONB. Không có index cho text search trên `headword`, `romanization`.

**Query pattern phổ biến nhất:** Search vocabulary by headword/romanization/meaning — hiện tại dùng `LIKE '%query%'`.

**Đề xuất:**

```sql
-- 1. Trigram index cho fuzzy search (pg_trgm)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_vocab_headword_trgm ON vocabularies USING GIN (headword gin_trgm_ops);
CREATE INDEX idx_vocab_romanization_trgm ON vocabularies USING GIN (romanization gin_trgm_ops);

-- 2. Full-text search trên meanings (vocabulary_meanings table)
CREATE INDEX idx_vm_meaning_trgm ON vocabulary_meanings USING GIN (meaning gin_trgm_ops);

-- 3. Search query cải thiện:
-- Thay: WHERE headword LIKE '%xue%'
-- Bằng: WHERE headword % 'xue' OR headword ILIKE '%xue%'
-- pg_trgm hỗ trợ % operator (similarity) + ILIKE với GIN index
```

---

## C. Missing Pieces

### C1. Thiếu `updated_at` trigger — stale timestamps

Nhiều tables có `updated_at` column nhưng không có trigger tự động update:

- `vocabularies.updated_at`
- `folders.updated_at`
- `user_vocabulary_progress.updated_at`
- `user_learning_stats.updated_at`

GORM auto-updates `updated_at` ở application level, nhưng nếu có direct SQL updates (admin scripts, data fixes) thì `updated_at` stale.

**Đề xuất:**

```sql
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Áp dụng cho mỗi table có updated_at:
CREATE TRIGGER set_updated_at BEFORE UPDATE ON vocabularies
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at BEFORE UPDATE ON folders
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at BEFORE UPDATE ON user_vocabulary_progress
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at BEFORE UPDATE ON user_learning_stats
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
```

---

### C2. Thiếu `version` column cho optimistic locking trên `user_vocabulary_progress`

Doc `research_memory_score.md` đã đề cập "optimistic locking for concurrent score updates" nhưng schema thiếu `version` field.

**Scenario:** User hoàn thành Recall mode và Review mode gần như đồng thời (2 tabs, hoặc mobile background sync) → 2 concurrent UPDATEs trên cùng row → last-write-wins → mất 1 mode score.

**Đề xuất:**

```sql
ALTER TABLE user_vocabulary_progress ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
```

Application code:
```go
// UPDATE user_vocabulary_progress
// SET score_recall = ?, memory_score = ?, version = version + 1
// WHERE id = ? AND version = ?
// If rows_affected = 0 → retry (re-read + re-calculate + re-update)
```

---

### C3. Thiếu index trên `folder_vocabularies`

```sql
CREATE TABLE folder_vocabularies (
    folder_id     UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    vocabulary_id UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    added_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (folder_id, vocabulary_id)
);
```

PK `(folder_id, vocabulary_id)` → index on `(folder_id, vocabulary_id)`. Query "list vocabularies in folder" sử dụng `folder_id` → OK.

**Nhưng thiếu reverse index:** Query "folder nào chứa vocabulary X?" (cần cho delete vocabulary, hiển thị "this word appears in N folders"):

```sql
CREATE INDEX idx_fv_vocabulary ON folder_vocabularies(vocabulary_id);
```

---

### C4. `grammar_points.code` — UNIQUE nhưng thiếu language scope

```sql
code VARCHAR(50) NOT NULL UNIQUE,
```

**Vấn đề:** `code` unique globally, nhưng Chinese grammar code "ba-structure" có thể trùng concept name với Korean grammar. Khi thêm ngôn ngữ mới, phải nghĩ tên code không trùng → friction.

**Đề xuất:**

```sql
-- Thay UNIQUE(code) toàn cục bằng UNIQUE per language:
UNIQUE(language_id, code)
-- Cho phép: zh:ba-structure, ko:ba-structure (nếu có)
```

---

### C5. `ocr_scans` — thiếu constraints và cleanup strategy

```sql
status VARCHAR(20) DEFAULT 'pending',
```

**Vấn đề:**
1. Không có CHECK constraint cho `status` → có thể insert giá trị bất kỳ
2. Không có cleanup — OCR scan history grow vô hạn, nhưng user hiếm khi review scan cũ

**Đề xuất:**

```sql
-- 1. CHECK constraint
status VARCHAR(20) DEFAULT 'pending'
    CHECK (status IN ('pending', 'completed', 'failed', 'expired')),

-- 2. Retention: archive scans > 90 days
-- Cron: DELETE FROM ocr_scans WHERE created_at < NOW() - INTERVAL '90 days' AND status != 'pending';
```

---

### C6. Thiếu `learning_sessions.proficiency_level_id`

```sql
CREATE TABLE learning_sessions (
    ...
    language_id UUID NOT NULL REFERENCES languages(id),
    mode        VARCHAR(30) NOT NULL,
    folder_id   UUID REFERENCES folders(id),
    ...
);
```

**Vấn đề:** Session biết `language_id` nhưng không biết `proficiency_level_id`. Dashboard cần aggregate "user đã học bao nhiêu từ HSK 3?" → phải JOIN qua `learning_events → vocabularies → proficiency_level_id`. Denormalize vào session sẽ tránh JOIN này.

**Đề xuất:**

```sql
ALTER TABLE learning_sessions
    ADD COLUMN proficiency_level_id UUID REFERENCES proficiency_levels(id);
-- Nullable — session có thể cover nhiều levels (e.g. review mixed HSK 1-3)
```

---

## D. Summary — Action Items theo Priority

### Must-have trước khi implement

| # | Issue | Section | Effort |
|---|---|---|---|
| 1 | Partition `learning_events` by month + bỏ FK | B1 | Medium |
| 2 | Partition `pronunciation_scores` by month + bỏ FK | B2 | Medium |
| 3 | Thêm `version` cho optimistic locking trên `user_vocabulary_progress` | C2 | Low |
| 4 | Thêm reverse index `folder_vocabularies(vocabulary_id)` | C3 | Low |
| 5 | `grammar_points.code` → `UNIQUE(language_id, code)` thay vì global unique | C4 | Low |

### Should-have

| # | Issue | Section | Effort |
|---|---|---|---|
| 6 | `vocabulary_meanings` — đổi unique constraint sang `(vocab_id, target_lang, sort_order)` + partial unique index cho `is_primary` | A2 | Low |
| 7 | `vocabularies.metadata` — extract `stroke_count`, `recognition_only`, `stroke_data_url` thành columns | A1 | Medium |
| 8 | Trigram indexes cho search (`headword`, `romanization`, `meaning`) | B4 | Low |
| 9 | `updated_at` triggers | C1 | Low |
| 10 | `proficiency_levels` — gộp 4 stat columns vào `metadata` | A4 | Low |

### Nice-to-have

| # | Issue | Section | Effort |
|---|---|---|---|
| 11 | Bỏ `user_daily_counters` → dùng Redis (nếu implement Plan A entitlement) | B3 | Medium |
| 12 | `ocr_scans` — CHECK constraint + retention policy | C5 | Low |
| 13 | `learning_sessions.proficiency_level_id` denormalization | C6 | Low |
| 14 | `user_vocabulary_progress` mode scores → JSONB (Phase 3+, khi thêm modes mới) | A3 | High |

---

## E. Improved Schema — Key Tables

Dưới đây là schema cải thiện cho **3 tables quan trọng nhất** (thay đổi so với bản gốc được đánh dấu `-- [CHANGED]` hoặc `-- [NEW]`):

### `vocabularies` (improved)

```sql
CREATE TABLE vocabularies (
    id                   UUID PRIMARY KEY,
    language_id          UUID NOT NULL REFERENCES languages(id),
    proficiency_level_id UUID NOT NULL REFERENCES proficiency_levels(id),

    headword         VARCHAR(255) NOT NULL,
    romanization     VARCHAR(255),
    audio_url        VARCHAR(500),
    frequency_rank   INTEGER,
    examples         JSONB DEFAULT '[]'::jsonb,

    -- [CHANGED] Extract high-query fields from metadata → columns
    stroke_count     SMALLINT,
    stroke_data_url  VARCHAR(500),
    recognition_only BOOLEAN DEFAULT false,

    -- Language-specific metadata (phần còn lại: radicals, tone_numbers, kanji, hiragana...)
    metadata         JSONB DEFAULT '{}'::jsonb,

    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_vocab_headword_lang ON vocabularies(language_id, headword) WHERE deleted_at IS NULL;
CREATE INDEX idx_vocab_language ON vocabularies(language_id);
CREATE INDEX idx_vocab_proficiency ON vocabularies(proficiency_level_id);
CREATE INDEX idx_vocab_deleted_at ON vocabularies(deleted_at);
CREATE INDEX idx_vocab_frequency ON vocabularies(frequency_rank) WHERE frequency_rank IS NOT NULL;
CREATE INDEX idx_vocab_metadata ON vocabularies USING GIN (metadata);
-- [NEW] Trigram indexes for search
CREATE INDEX idx_vocab_headword_trgm ON vocabularies USING GIN (headword gin_trgm_ops);
CREATE INDEX idx_vocab_romanization_trgm ON vocabularies USING GIN (romanization gin_trgm_ops);
-- [NEW] Stroke filter for learning modes
CREATE INDEX idx_vocab_stroke ON vocabularies(stroke_count) WHERE stroke_count IS NOT NULL;
```

### `learning_events` (improved — partitioned)

```sql
CREATE TABLE learning_events (
    id            UUID NOT NULL,
    user_id       UUID NOT NULL,    -- [CHANGED] bỏ FK
    vocabulary_id UUID NOT NULL,    -- [CHANGED] bỏ FK
    mode          VARCHAR(30) NOT NULL,
    score         DECIMAL(5,2),
    q_score       SMALLINT,
    is_correct    BOOLEAN,
    duration_ms   INTEGER,
    event_data    JSONB DEFAULT '{}'::jsonb,
    session_id    UUID,             -- [CHANGED] bỏ FK
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)    -- [CHANGED] composite PK for partition
) PARTITION BY RANGE (created_at); -- [NEW]

-- [CHANGED] Giảm từ 4 → 2 indexes
CREATE INDEX idx_le_user_mode ON learning_events(user_id, mode, created_at DESC);
CREATE INDEX idx_le_session ON learning_events(session_id) WHERE session_id IS NOT NULL;
```

### `user_vocabulary_progress` (improved)

```sql
CREATE TABLE user_vocabulary_progress (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    vocabulary_id   UUID NOT NULL REFERENCES vocabularies(id),

    memory_score    DECIMAL(5,2) NOT NULL DEFAULT 0,
    memory_state    VARCHAR(30) NOT NULL DEFAULT 'start_learning',

    score_discover       DECIMAL(3,1) DEFAULT 0,
    score_recall         DECIMAL(3,1) DEFAULT 0,
    score_stroke_guided  DECIMAL(3,1) DEFAULT 0,
    score_stroke_recall  DECIMAL(3,1) DEFAULT 0,
    score_pinyin_drill   DECIMAL(3,1) DEFAULT 0,
    score_ai_chat        DECIMAL(3,1) DEFAULT 0,
    score_review         DECIMAL(3,1) DEFAULT 0,
    score_mastery_check  DECIMAL(3,1) DEFAULT 0,
    spacing_score        DECIMAL(3,1) DEFAULT 0,

    easiness_factor  DECIMAL(4,2) NOT NULL DEFAULT 2.50,
    interval_days    INTEGER NOT NULL DEFAULT 1,
    repetitions      INTEGER NOT NULL DEFAULT 0,
    next_review_at   TIMESTAMPTZ,
    last_reviewed_at TIMESTAMPTZ,

    spacing_correct_streak INTEGER DEFAULT 0,
    last_mistake_at        TIMESTAMPTZ,
    max_points       INTEGER NOT NULL DEFAULT 11,

    version          INTEGER NOT NULL DEFAULT 0,  -- [NEW] optimistic locking

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, vocabulary_id)
);

CREATE INDEX idx_uvp_user ON user_vocabulary_progress(user_id);
CREATE INDEX idx_uvp_review ON user_vocabulary_progress(user_id, next_review_at)
    WHERE next_review_at IS NOT NULL;
CREATE INDEX idx_uvp_state ON user_vocabulary_progress(user_id, memory_state);
```
