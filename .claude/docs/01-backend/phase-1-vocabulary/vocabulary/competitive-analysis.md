# Competitive Analysis — Chinese Vocabulary Learning Apps

> Nghien cuu 2026-04-02. So sanh 7 doi thu chinh voi he thong Lu.

---

## 1. Competitor Overview

| App | USP chinh | HSK Coverage | Model |
|---|---|---|---|
| **Hanzii** | Dictionary + notebook + SRS flashcards + radical decomposition | HSK 1-9 (HSK 3.0) | Free + Premium |
| **HelloChinese** | Best structured beginner course + 1,000+ graded stories | HSK 1-2 (new standard) | Freemium |
| **Pleco** | Live camera OCR + dictionary/reader/flashcard ecosystem | N/A (reference tool) | One-time purchase |
| **Duolingo** | AI video call (Lily) + strongest gamification/habit engine | Not HSK-aligned | Freemium |
| **ChineseSkill** | Explicit pinyin rules + tone animations + stroke order from day 1 | HSK 1-4 | Freemium |
| **SuperChinese** | Scenario-based lessons + CHAO AI tutor + 170+ themes | HSK 1-5 (claim 9 levels) | Subscription (PLUS/CHAO) |
| **Skritter** | Active writing recall + stroke-level grading (3 modes) | Textbook-aligned | Subscription |
| **Anki** | FSRS algorithm (20-30% better than SM-2) + infinite customization | Community decks HSK 1-9 | Free (iOS $24.99) |
| **SuperTest** | 300K+ HSK questions + official past papers + AI mistake bank | HSK 1-9 | Ad-free, freemium |

---

## 2. Feature Comparison Matrix

| Feature | Hanzii | HelloChinese | Pleco | Duolingo | ChineseSkill | SuperChinese | Skritter | Anki | SuperTest | **Lu** |
|---|---|---|---|---|---|---|---|---|---|---|
| HSK 1-9 | Yes | No (1-2) | N/A | No | No (1-4) | No (1-5) | No | Community | Yes | **Yes** |
| Multi-language | No | No | No | Yes | No | No | JP+ZH | Any | No | **Yes** |
| OCR scan -> flashcards | No | No | Live camera | No | No | No | No | No | No | **Yes** |
| AI Chat / Tutor | No | No | No | AI video call | No | CHAO AI | No | No | No | **Yes (3 personas)** |
| Stroke writing (recall) | No | Tracing only | No | No | Basic | Basic | **3 modes** | Addon | Basic | **3 sub-modes** |
| Pronunciation scoring | No | Tone detection | No | Basic | Tone feedback | AI scoring | No | No | No | **PrepAI per-syllable** |
| SRS algorithm | Basic | Basic | SRS | Internal | Basic | Smart Review | SRS | **FSRS** | AI-adaptive | SM-2 |
| Grammar context per word | No | Inline highlights | No | No | No | In lessons | No | No | Comparison lessons | **Yes (M:N)** |
| Graded stories/reading | No | **1,000+** | Document Reader | Stories | No | Scenarios | No | No | No | No |
| Gamification (streaks/leagues) | No | Basic | No | **Best** | Game Center | Streaks+cards | No | No | Missions | Partial (XP only) |
| Mistake tracking/bank | No | No | No | No | No | No | No | No | **AI mistake bank** | No |
| Confusable word comparison | No | No | No | No | No | No | Etymology | No | **Grammar comparison** | No |
| Memory score / progress | Memory chart | Progress | No | XP/leagues | No | Level assessment | No | Retention % | Performance reports | **6-state Memory Score** |

---

## 3. Lu's Competitive Advantages

| Advantage | Detail |
|---|---|
| **HSK 1-9 full coverage** | Most competitors max out at HSK 4-5. Only SuperTest + Hanzii cover HSK 7-9 |
| **Multi-language architecture** | Generic design supports zh, ja, ko, th, id. All competitors are Chinese-only |
| **OCR scan -> auto flashcards** | No competitor combines OCR with structured vocabulary learning |
| **7 learning modes** | Most comprehensive mode set: Discover, Recall, Stroke (3 sub-modes), Pinyin Drill, AI Chat, Review, Mastery Check |
| **PrepAI pronunciation** | Per-syllable scoring (initial/final/tone) is deeper than most competitors |
| **Grammar-per-word (M:N)** | Grammar tips attached to each vocabulary word, not separate lessons |
| **Memory Score system** | 6-state progression with weighted multi-mode scoring is unique |

---

## 4. Gaps & Improvement Opportunities

### Priority 1 — Phase 1 candidates (high impact, low-medium effort)

#### 4.1 AI Mistake Bank (inspired by SuperTest)

SuperTest automatically collects every mistake, categorizes by type, and generates targeted "mistake training" sessions.

**Current gap:** `user_vocabulary_progress` tracks scores per mode but does NOT track specific error history (which word, what type of mistake, how many times).

**Proposed addition:**
```sql
CREATE TABLE user_mistakes (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    vocabulary_id   UUID NOT NULL,
    session_id      UUID,
    mode            VARCHAR(30) NOT NULL,   -- 'recall', 'stroke', 'pronunciation'
    mistake_type    VARCHAR(50),            -- 'tone_2_vs_3', 'stroke_order', 'meaning_confusion'
    context         JSONB,                  -- { "confused_with_id": "uuid", "expected": "拔", "actual": "拨" }
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

**Value:** Foundation for personalized learning — adaptive drill queues, weak area reports, targeted review sessions.

#### 4.2 Streak & Badge System (inspired by Duolingo)

Duolingo's streak/league system is the strongest habit formation engine — the primary reason users return daily.

**Current gap:** XP exists in `learning_sessions` but no streaks, badges, or leaderboard.

**Proposed addition:**
```sql
-- Track daily activity streaks
CREATE TABLE user_streaks (
    user_id          UUID PRIMARY KEY,
    current_streak   INTEGER NOT NULL DEFAULT 0,
    longest_streak   INTEGER NOT NULL DEFAULT 0,
    last_active_date DATE NOT NULL,
    shield_available BOOLEAN DEFAULT false,  -- protect streak on missed day
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

-- Achievement badges
CREATE TABLE user_badges (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    badge_code VARCHAR(50) NOT NULL,    -- 'hsk1_complete', 'streak_30', 'mastered_100'
    earned_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, badge_code)
);
```

**Value:** High retention impact with relatively simple implementation.

### Priority 2 — Phase 2 candidates

#### 4.3 FSRS Algorithm (inspired by Anki)

FSRS (Free Spaced Repetition Scheduler) replaces SM-2 with ML-based scheduling that learns per-user memory patterns. ~20-30% fewer reviews for the same retention rate.

**Current:** SM-2 is fine for MVP. Design `user_vocabulary_progress` to be swappable.

**Action:** No change now. When migrating, SM-2 fields (`easiness_factor`, `interval_days`, `repetitions`) would be replaced with FSRS parameters (`stability`, `difficulty`, `retrievability`).

#### 4.4 Confusable / Synonym Relations (inspired by SuperTest + Skritter)

SuperTest has grammar comparison lessons (side-by-side confusable words). Skritter shows character etymology/decomposition.

**Proposed addition (Phase 2):**
```sql
CREATE TABLE vocabulary_relations (
    vocabulary_id   UUID NOT NULL,
    related_id      UUID NOT NULL,
    relation_type   VARCHAR(20) NOT NULL,  -- 'synonym', 'antonym', 'confusable', 'component'
    note            JSONB,                 -- { "vi": "拔 (nho) vs 拨 (gat) - khac net cuoi" }
    PRIMARY KEY (vocabulary_id, related_id)
);
```

#### 4.5 Graded Reading / Stories (inspired by HelloChinese)

HelloChinese has 1,000+ graded stories with tap-to-lookup, audio, and comprehension quizzes. Pleco has a Document Reader.

**Current:** Only example sentences per word. No long-form reading content.

**Action:** Phase 2+. Requires new entity model (`stories`, `story_segments`). Does not affect current design.

### Priority 3 — Phase 3+

#### 4.6 Live Camera OCR (inspired by Pleco)

Real-time camera OCR with instant dictionary lookup on live video feed.

**Current:** Photo-based OCR flow (capture -> process -> preview -> confirm) is sufficient for MVP.

**Action:** Primarily a frontend + OCR engine concern. Backend design unaffected.

---

## 5. Summary

The current Lu architecture is **well-positioned** against all 7 competitors for Phase 1 MVP. The main gaps are not in the core vocabulary/learning domain model but in **engagement infrastructure** (mistake tracking, streaks/badges) and **content depth** (stories, confusable comparisons) — both addressable incrementally.
