# Memory Score Research — Weighted Multi-mode Scoring + Spaced Repetition

> Research phục vụ thiết kế Memory Score system (C10 trong `technical_challenges.md`).
>
> **Context:** Mỗi từ vựng có 1 Memory Score tổng hợp từ 7 learning modes (weighted). Score quyết định trạng thái ghi nhớ (6 states) và lịch ôn tập (spaced repetition). Đây là hệ thống trung tâm — mọi mode khác (C5-C9) đều feed vào đây.

---

## 1. Yêu cầu từ PRD

### 1.1 Memory Score Formula (§5.2)

```
Memory Score (per word) =
  Σ(Mode_Score × Mode_Weight) + Spacing_Score
  ─────────────────────────────────────────────  × 100
                   Max_Points
```

### 1.2 Mode Weights

| Mode | Weight | Max Weighted | Free? |
|---|---|---|---|
| Discover | 1 | 1 | Free |
| Recall | 2 | 4 | Free |
| Stroke (Guided) | 1 | 1 | Pro |
| Stroke (Recall) | 2 | 4 | Pro |
| Pinyin Drill | 1 | 1 | Pro |
| Chat (AI) | 2 | 4 | Pro |
| Review (SM-2) | 2 | 4 | Free |
| Mastery Check | 2 | 4 | Pro |
| **Spacing Score** | — | 2 | Free |

**Max_Points:** Free = 11 (Discover 1 + Recall 4 + Review 4 + Spacing 2). Pro = 25 (all 23 + Spacing 2).

### 1.3 6 States (§5.1)

| State | Điều kiện | Review Interval |
|---|---|---|
| **Start Learning** | Score = 0 | 1 day |
| **Still Learning** | Score < 40 | 2 days |
| **Almost Learnt** | 40 ≤ Score < 60 | 3-5 days |
| **Finish Learning** | 60 ≤ Score < 80 | 7 days |
| **Memory Mode** | Score ≥ 80 + spacing đúng ≥ 1 lần | 14 days |
| **Mastered** | Score ≥ 90 + spacing đúng ≥ 2 lần + không sai 10-14 ngày | 30 days. Reset nếu không ôn 60 ngày |

### 1.4 Spacing Score

| Condition | Score |
|---|---|
| Review đúng ngày (±1 day) và q ≥ 4 | +2 |
| Review trễ ≤ 3 ngày, q ≥ 3 | +1 |
| Review sớm hơn 2 ngày | +0.5 |
| Review trễ > 3 ngày hoặc q < 3 | 0 |

---

## 2. Spaced Repetition — SM-2 vs FSRS

### 2.1 SM-2 (Classic, PRD đề cập)

Algorithm gốc từ SuperMemo (1987):

```
Interval:
  I(1) = 1 day
  I(2) = 6 days
  I(n) = I(n-1) × EF    (n > 2)

Easiness Factor update:
  EF' = EF + (0.1 - (5 - q) × (0.08 + (5 - q) × 0.02))
  EF minimum = 1.3, initial = 2.5

Reset: q < 3 → restart từ I(1), giữ EF
```

**Hạn chế:**
- Interval tăng tuyến tính (linear × EF) — không model forgetting curve thực tế
- "Ease hell": EF giảm dần khi user hay sai → interval ngắn → thấy quá nhiều → chán
- Không adapt theo user behavior — cùng EF cho mọi người
- q-score 0-5 quá granular — user khó phân biệt 3 vs 4

### 2.2 FSRS (Free Spaced Repetition Scheduler) — Modern alternative

**[open-spaced-repetition/go-fsrs](https://github.com/open-spaced-repetition/go-fsrs)** — Go library sẵn.

3 biến per card:

| Variable | Ý nghĩa | Range |
|---|---|---|
| **Difficulty (D)** | Độ khó inherent của material | 1-10 |
| **Stability (S)** | Memory storage strength. Interval mà R = 90% | Float (ngày) |
| **Retrievability (R)** | Xác suất nhớ hiện tại. Decay theo thời gian | 0-1 |

**Forgetting curve:**
```
R(t, S) = (1 + t/(9×S))^(-1)
```

**Next interval:**
```
I(r, S) = 9×S × (1/r - 1)
```
Với r = desired retention (VD: 0.9 = muốn 90% nhớ).

**4 buttons:** Again / Hard / Good / Easy (thay vì 0-5).

**Tại sao FSRS tốt hơn SM-2:**

| | SM-2 | FSRS |
|---|---|---|
| Forgetting curve | Không model | Explicitly model R(t,S) |
| Ease hell | Có — EF giảm liên tục | Không — mean reversion trên Difficulty |
| Personalization | Không — cùng formula cho mọi người | 21 params tunable từ user review history |
| Stability after lapse | Reset I(1) | Tính S' dựa trên D, S trước, R lúc fail |
| Go library | Phải tự implement | `go-fsrs/v4` sẵn |
| Anki support | Anki đang chuyển từ SM-2 sang FSRS | FSRS là default scheduler mới của Anki |

### 2.3 Duolingo — Half-Life Regression (HLR)

ML approach: `p = 2^(-delta/h)` với h = learned half-life.

**Không recommend** vì cần training data lớn + ML infrastructure. Phù hợp cho Duolingo (1B+ reviews) nhưng overkill cho MVP.

### 2.4 Recommend: FSRS

- Go library sẵn (`go-fsrs/v4`)
- Modern, đã proven ở Anki (100M+ users)
- Tránh ease hell
- 4 buttons dễ map từ mode scores
- Tunable khi có production data

---

## 3. Tích hợp FSRS với multi-mode scoring

### 3.1 Bài toán

PRD yêu cầu Memory Score = weighted sum từ 7 modes + Spacing Score. FSRS produce scheduling (khi nào review). Cần kết nối 2 hệ thống.

### 3.2 Architecture

```
Learning Event (từ bất kỳ mode nào)
  │
  ▼
[1. Update Mode Score]
  │ VD: Recall mode, user đúng 8/10 → mode_score = 0.8
  │ Weighted: 0.8 × 2 (weight) = 1.6
  │
  ▼
[2. Recalculate Memory Score]
  │ Sum tất cả mode weighted scores + spacing_score
  │ ÷ Max_Points (dựa trên plan)
  │ × 100
  │
  ▼
[3. Determine State]
  │ Score → state transition (state machine)
  │ VD: Score = 65, spacing_correct ≥ 0 → "Finish Learning"
  │
  ▼
[4. Map to FSRS Rating]
  │ Memory Score → Rating:
  │   Score ≥ 90 → Easy
  │   Score ≥ 70 → Good
  │   Score ≥ 40 → Hard
  │   Score < 40 → Again
  │
  ▼
[5. FSRS Schedule]
  │ fsrs.Next(card, now, rating) → updated card with next Due date
  │
  ▼
[6. Persist]
  │ Save: mode_scores, memory_score, state, FSRS card (stability, difficulty, due)
```

### 3.3 Ví dụ cụ thể

User Free, từ "学习":
```
Current scores:
  Discover:    1.0 × 1 = 1.0   (đã học flashcard)
  Recall:      0.6 × 2 = 1.2   (đúng 6/10 quiz)
  Review:      0.0 × 2 = 0.0   (chưa review)
  Spacing:     0

Memory Score = (1.0 + 1.2 + 0.0 + 0) / 11 × 100 = 20%
State: Still Learning (< 40)
FSRS Rating: Again (< 40) → short interval
```

User Review đúng ngày, q ≥ 4:
```
  Review:      0.8 × 2 = 1.6
  Spacing:     2.0

Memory Score = (1.0 + 1.2 + 1.6 + 2.0) / 11 × 100 = 52.7%
State: Almost Learnt (40-60)
FSRS Rating: Good (≥ 40) → medium interval
```

---

## 4. State Machine

### 4.1 States + Transitions

```
                    score < 40
Start Learning ──────────────────► Still Learning
  (score = 0)                         │
                                      │ score ≥ 40
                                      ▼
                                 Almost Learnt
                                      │
                                      │ score ≥ 60
                                      ▼
                                 Finish Learning
                                      │
                                      │ score ≥ 80 + spacing_correct ≥ 1
                                      ▼
                                 Memory Mode
                                      │
                                      │ score ≥ 90 + spacing_correct ≥ 2
                                      │ + không sai 10-14 ngày
                                      ▼
                                   Mastered
                                      │
                                      │ không ôn 60 ngày
                                      ▼
                                   (reset → Still Learning)
```

**Transition xuống (downgrade):** Có thể xảy ra khi score giảm (VD: review fail → spacing_score = 0 → score giảm → state xuống).

### 4.2 Go implementation

```go
type LearningState string

const (
    StateStartLearning LearningState = "start_learning"
    StateStillLearning LearningState = "still_learning"
    StateAlmostLearnt  LearningState = "almost_learnt"
    StateFinishLearning LearningState = "finish_learning"
    StateMemoryMode    LearningState = "memory_mode"
    StateMastered      LearningState = "mastered"
)

func DetermineState(score float64, spacingCorrect int, lastMistakeDays int, lastReviewDays int) LearningState {
    switch {
    case score == 0:
        return StateStartLearning
    case score >= 90 && spacingCorrect >= 2 && lastMistakeDays >= 10:
        if lastReviewDays > 60 {
            return StateStillLearning // reset
        }
        return StateMastered
    case score >= 80 && spacingCorrect >= 1:
        return StateMemoryMode
    case score >= 60:
        return StateFinishLearning
    case score >= 40:
        return StateAlmostLearnt
    default:
        return StateStillLearning
    }
}
```

**Tại sao không dùng FSM library (looplab/fsm)?** State transitions phụ thuộc vào **nhiều điều kiện đồng thời** (score + spacing + mistake history) — không phải simple event-driven FSM. Switch statement rõ ràng hơn, dễ test, dễ đọc.

---

## 5. Concurrency — Atomic score updates

### 5.1 Bài toán

User mở 2 mode đồng thời (VD: Recall trên tab 1, Stroke trên tab 2). Cả 2 mode update Memory Score cho cùng 1 từ → race condition.

### 5.2 Giải pháp

| Approach | Khi nào dùng | Ví dụ |
|---|---|---|
| **Atomic UPDATE expression** | Simple score increment từ 1 mode | `UPDATE SET recall_score = $1, memory_score = recompute(...) WHERE user_id = $2 AND vocab_id = $3` |
| **Optimistic locking** | Full recalculate từ tất cả mode scores | Thêm `version` column. UPDATE ... WHERE version = $old_version. Retry nếu fail |
| **SELECT FOR UPDATE** | Complex multi-step update trong 1 transaction | `BEGIN; SELECT ... FOR UPDATE; compute; UPDATE; COMMIT;` |

**Recommend: Optimistic locking** — phù hợp nhất vì Memory Score cần recalculate từ tất cả mode scores (không phải simple increment).

```go
// Domain entity
type VocabularyScore struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    VocabularyID    uuid.UUID
    DiscoverScore   float64
    RecallScore     float64
    StrokeGuided    float64
    StrokeRecall    float64
    PinyinDrill     float64
    ChatAI          float64
    ReviewSM2       float64
    MasteryCheck    float64
    SpacingScore    float64
    MemoryScore     float64        // computed
    State           LearningState
    // FSRS fields
    Stability       float64
    Difficulty      float64
    Due             time.Time
    Reps            int
    Lapses          int
    LastReviewedAt  time.Time
    // Concurrency
    Version         int            // optimistic lock
}
```

```go
// Use case: update mode score
func (uc *ScoringUseCase) UpdateModeScore(ctx context.Context, userID, vocabID uuid.UUID, mode string, score float64) error {
    // 1. Load current scores
    vs, err := uc.repo.FindByUserAndVocab(ctx, userID, vocabID)

    // 2. Update specific mode score
    vs.SetModeScore(mode, score)

    // 3. Recalculate memory score + state
    vs.Recalculate(uc.planMax) // planMax from entitlement

    // 4. FSRS schedule
    rating := vs.ToFSRSRating()
    fsrsResult := uc.fsrs.Next(vs.ToFSRSCard(), time.Now(), rating)
    vs.ApplyFSRS(fsrsResult)

    // 5. Save with optimistic lock
    err = uc.repo.UpdateWithVersion(ctx, vs) // fails if version mismatch
    if errors.Is(err, ErrOptimisticLock) {
        // retry: reload + recalculate + save
    }
}
```

### 5.3 Thực tế race condition có thường xảy ra không?

**Rất hiếm.** Mỗi user học 1 mode tại 1 thời điểm (mobile UX = 1 screen active). Race chỉ xảy ra nếu:
- User mở app trên 2 device → rất hiếm
- Background job (VD: decay check) chạy đúng lúc user đang học → handled bằng optimistic lock retry

→ Optimistic lock đủ. Không cần advisory locks hay distributed locks.

---

## 6. Plan-dependent scoring (Free vs Pro)

### 6.1 Nguyên tắc: Store raw, compute percentage at display time

```
DB lưu: raw mode scores (float64 per mode)
API trả: memory_score_percent = raw_total / plan_max × 100
```

**Không bao giờ lưu percentage** vào DB. Tại sao: plan thay đổi (upgrade/downgrade) → percentage thay đổi → nếu lưu % phải recalculate mọi row.

### 6.2 Upgrade (Free → Pro)

```
Before: raw = 8, max = 11  → display: 72.7% → Finish Learning
After:  raw = 8, max = 25  → display: 32.0% → Still Learning
```

**User thấy "giảm bậc" khi trả tiền?** Có. PRD §5.3 acknowledge: _"giảm 1 bậc state"_.

**Mitigation (từ PRD):**
- Giữ toàn bộ data cũ
- Hiển thị message: "Your progress is preserved. New modes are available to improve your score!"
- Mode mới (Stroke, Chat, Pinyin) unlock → user có thể earn thêm points → score tăng nhanh

### 6.3 Downgrade (Pro → Free)

```
Before: raw = 18, max = 25 → display: 72% → Finish Learning
After:  raw = 18, max = 11 → display: min(18, 11) / 11 × 100 = 100% → Mastered?
```

**Vấn đề:** raw > free_max → 100%+ → sai. Cần cap.

**Giải pháp:** Khi compute display score:
```go
func ComputeDisplayScore(rawTotal float64, planMax float64) float64 {
    capped := math.Min(rawTotal, planMax)
    return capped / planMax * 100
}
```

Pro mode scores (Stroke, Chat, etc.) vẫn lưu nhưng **không tính vào total** khi plan = Free. Khi upgrade lại → scores khôi phục.

```go
func ComputeRawTotal(scores *VocabularyScore, plan string) float64 {
    total := scores.DiscoverScore*1 + scores.RecallScore*2 + scores.ReviewSM2*2 + scores.SpacingScore
    if plan != "free" {
        total += scores.StrokeGuided*1 + scores.StrokeRecall*2 +
                 scores.PinyinDrill*1 + scores.ChatAI*2 + scores.MasteryCheck*2
    }
    return total
}
```

---

## 7. Decay & Reset

### 7.1 Mastered reset sau 60 ngày không ôn

**Recommend: Lazy evaluation (check on access)**

```go
func (vs *VocabularyScore) CheckDecay(now time.Time) {
    if vs.State == StateMastered {
        daysSinceReview := now.Sub(vs.LastReviewedAt).Hours() / 24
        if daysSinceReview > 60 {
            vs.State = StateStillLearning
            // Optionally reduce score based on FSRS retrievability
            vs.MemoryScore = vs.MemoryScore * vs.Retrievability(now)
        }
    }
}
```

**Tại sao lazy thay vì batch cron:**
- Zero infrastructure (không cần cron job)
- Luôn chính xác tại thời điểm đọc
- Không lãng phí compute cho inactive users (50K MAU nhưng có thể 200K registered → 150K inactive)
- Anki cũng dùng approach này: store `due` date, check `WHERE due <= today` khi query

**Optional:** Lightweight daily batch cho active users (active 90 ngày gần nhất) → trigger push notification "You have words to review!"

### 7.2 FSRS Retrievability check

FSRS đã model decay tự nhiên:

```
R(t, S) = (1 + t/(9×S))^(-1)
```

VD: Stability S = 30 ngày, sau 60 ngày không ôn:
```
R(60, 30) = (1 + 60/270)^(-1) = (1.222)^(-1) = 0.818
```

→ Retrievability = 81.8%. Dưới target 90% → cần review.

→ FSRS tự schedule review trước khi R xuống dưới target. "60 ngày reset" của PRD tương ứng với R rất thấp — FSRS sẽ flag cần review sớm hơn nhiều.

---

## 8. Score history & Analytics

### 8.1 Learning events table (time-series)

```sql
CREATE TABLE learning_events (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,
    vocab_id    UUID NOT NULL,
    mode        VARCHAR(20) NOT NULL,   -- 'discover', 'recall', 'stroke_guided', ...
    raw_score   FLOAT NOT NULL,         -- 0.0 - 1.0
    metadata    JSONB,                   -- mode-specific data: { correct: 8, total: 10, time_ms: 5000 }
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE learning_events_2026_04 PARTITION OF learning_events
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

-- Indexes
CREATE INDEX idx_learning_events_user ON learning_events (user_id, vocab_id, created_at DESC);
CREATE INDEX idx_learning_events_brin ON learning_events USING BRIN (created_at);
```

**Tại sao partition by month:**
- 50K MAU × 50 events/session × 1 session/day = **2.5M events/day** = 75M/month
- Partition pruning: query "last 7 days" chỉ scan 1 partition
- Archive: detach old partitions không ảnh hưởng active data
- VACUUM per-partition → ít lock contention

### 8.2 Materialized view cho dashboard

```sql
CREATE MATERIALIZED VIEW mv_user_progress AS
SELECT
    user_id,
    COUNT(DISTINCT vocab_id) AS total_words,
    COUNT(DISTINCT vocab_id) FILTER (WHERE state = 'mastered') AS mastered_words,
    COUNT(DISTINCT vocab_id) FILTER (WHERE state = 'memory_mode') AS memory_mode_words,
    AVG(memory_score) AS avg_memory_score
FROM vocabulary_scores
GROUP BY user_id;

-- Refresh hourly (concurrent = no downtime)
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_user_progress;
```

### 8.3 Volume estimation

| Giai đoạn | Events/ngày | Events/tháng | Storage/tháng (~200 bytes/event) |
|---|---|---|---|
| MVP (200 MAU) | 10K | 300K | ~60 MB |
| Growth (10K MAU) | 500K | 15M | ~3 GB |
| Scale (50K MAU) | 2.5M | 75M | ~15 GB |

→ MVP: không cần partition. Growth+: partition by month. Scale: xem xét move events sang ClickHouse/TimescaleDB.

---

## 9. Data model

### 9.1 vocabulary_scores table

```sql
CREATE TABLE vocabulary_scores (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    vocabulary_id   UUID NOT NULL,
    -- Per-mode scores (0.0 - 1.0, trước khi nhân weight)
    discover_score      FLOAT DEFAULT 0,
    recall_score        FLOAT DEFAULT 0,
    stroke_guided_score FLOAT DEFAULT 0,
    stroke_recall_score FLOAT DEFAULT 0,
    pinyin_drill_score  FLOAT DEFAULT 0,
    chat_ai_score       FLOAT DEFAULT 0,
    review_sm2_score    FLOAT DEFAULT 0,
    mastery_check_score FLOAT DEFAULT 0,
    spacing_score       FLOAT DEFAULT 0,
    -- Computed (tại application layer, không dùng generated column vì phụ thuộc plan)
    memory_score    FLOAT DEFAULT 0,       -- 0-100
    state           VARCHAR(20) DEFAULT 'start_learning',
    -- FSRS scheduling
    stability       FLOAT DEFAULT 0,
    difficulty      FLOAT DEFAULT 0,
    due             TIMESTAMPTZ,
    reps            INT DEFAULT 0,
    lapses          INT DEFAULT 0,
    last_reviewed_at TIMESTAMPTZ,
    -- Concurrency
    version         INT DEFAULT 1,
    -- Metadata
    spacing_correct_count INT DEFAULT 0,   -- cho state transition (Memory Mode, Mastered)
    last_mistake_at       TIMESTAMPTZ,     -- cho Mastered condition "không sai 10-14 ngày"
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, vocabulary_id)
);

CREATE INDEX idx_vocab_scores_user ON vocabulary_scores (user_id);
CREATE INDEX idx_vocab_scores_due ON vocabulary_scores (user_id, due) WHERE due IS NOT NULL;
CREATE INDEX idx_vocab_scores_state ON vocabulary_scores (user_id, state);
```

### 9.2 Tại sao 1 table thay vì nhiều?

| Alternative | Nhược điểm |
|---|---|
| Tách `mode_scores` table (1 row per mode per vocab per user) | JOIN phức tạp cho mỗi score read. 7 rows per vocab thay vì 1. Recalculate cần aggregate 7 rows |
| Tách `fsrs_cards` table | Thêm JOIN. FSRS fields luôn đọc/ghi cùng scores. Không có lý do tách |
| JSONB cho mode scores | Không query/index được per-mode. Postgres JSONB update tốn hơn column update |

→ 1 table with typed columns: đơn giản, fast read (1 row = đầy đủ info), fast update (optimistic lock trên 1 row).

---

## 10. So sánh reference implementations

| System | Algorithm | Multi-mode? | State machine? | Concurrency |
|---|---|---|---|---|
| **Anki** | SM-2 → đang migrate FSRS | Không (chỉ flashcard review) | 4 states (New, Learning, Review, Relearning) | Single-user desktop, không cần |
| **Duolingo** | Half-Life Regression (ML) | Có (lesson types: read, write, listen, speak) | Skill levels (0-5) per topic | Server-side, internal |
| **SuperMemo** | SM-18 (proprietary) | Không | Complex (15+ parameters per item) | Desktop |
| **FSRS (open-source)** | DSR model | Không (single card) | 4 states (New, Learning, Review, Relearning) | Library — caller handles |
| **Prep (planned)** | **FSRS + weighted multi-mode** | **Có (7 modes)** | **6 states (custom)** | **Optimistic locking** |

Prep system độc đáo ở chỗ kết hợp **FSRS scheduling** với **weighted multi-mode scoring** — không system nào ở trên làm cả 2. FSRS handle "khi nào review", multi-mode scoring handle "user nhớ tốt cỡ nào từ nhiều góc độ".

---

## 11. Backend scope

| Component | Effort | Chi tiết |
|---|---|---|
| **VocabularyScore domain entity** | 1 ngày | Struct + Recalculate() + DetermineState() + CheckDecay() |
| **FSRS integration** | 1 ngày | Wrap go-fsrs library. Map Memory Score → FSRS Rating |
| **Scoring use case** | 2 ngày | UpdateModeScore(), GetDueWords(), GetUserProgress(). Optimistic lock retry |
| **Repository** | 1 ngày | vocabulary_scores CRUD. FindDue(). UpdateWithVersion() |
| **Learning events logging** | 1 ngày | Async write to learning_events table |
| **Migration** | 0.5 ngày | vocabulary_scores + learning_events tables + indexes |
| **Dashboard aggregation** | 1 ngày | Materialized view + refresh logic + API endpoint |
| **Total** | **~7-8 ngày** | |

---

## References

| Source | URL | Relevance |
|---|---|---|
| SM-2 Algorithm (Original) | https://super-memory.com/english/ol/sm2.htm | Classic algorithm, PRD references |
| Anki SRS Deep Dive | https://juliensobczak.com/inspect/2022/05/30/anki-srs/ | How Anki modified SM-2 |
| Anki v3 Scheduler | https://faqs.ankiweb.net/the-2021-scheduler.html | 4-state machine, fuzz factor |
| FSRS GitHub | https://github.com/open-spaced-repetition/free-spaced-repetition-scheduler | Algorithm specification |
| FSRS Algorithm Wiki | https://github.com/open-spaced-repetition/awesome-fsrs/wiki/The-Algorithm | DSR model formulas |
| go-fsrs (Go library) | https://pkg.go.dev/github.com/open-spaced-repetition/go-fsrs/v4 | Go implementation, ready to use |
| FSRS4Anki | https://github.com/open-spaced-repetition/fsrs4anki | FSRS integrated into Anki |
| Duolingo Half-Life Regression | https://aclanthology.org/P16-1174/ | ML-based SRS alternative |
| Go State Machine Patterns | https://www.codingexplorations.com/blog/state-machine-patterns-in-go | Idiomatic Go FSM |
| looplab/fsm | https://github.com/looplab/fsm | Go FSM library (evaluated, not recommended for this case) |
| PostgreSQL Explicit Locking | https://www.postgresql.org/docs/current/explicit-locking.html | Row locks, advisory locks |
| GORM Optimistic Lock | https://github.com/go-gorm/optimisticlock | Version-based optimistic locking |
| PostgreSQL Time-Series Partitioning | https://aws.amazon.com/blogs/database/speed-up-time-series-data-ingestion-by-partitioning-tables-on-amazon-rds-for-postgresql/ | BRIN indexes, monthly partitions |
| PostgreSQL Materialized Views | https://www.epsio.io/blog/postgres-materialized-views-basics-tutorial-and-optimization-tips | Dashboard aggregation |
| Redis Atomic Scoring | https://redis.io/blog/redis-game-mechanics-scoring/ | Sorted sets, Lua scripts |
