# Vocabulary Module — Database Design (Multi-language)

> Thiết kế từ requirement PRD v3 + module requirement.
> Schema được thiết kế generic để hỗ trợ đa ngôn ngữ (Chinese, Japanese, Korean, Thai, Indonesian...),
> không hardcode logic riêng cho bất kỳ ngôn ngữ nào.

---

## Design Principles

1. **Language-agnostic core**: Bảng `vocabularies` chứa field generic (`headword`, `romanization`). Data đặc thù ngôn ngữ (radicals, stroke, gender, conjugation...) vào `metadata` JSONB.
2. **Pluggable proficiency systems**: HSK (CN), JLPT (JP), TOPIK (KR), CEFR (chung)... đều nằm trong `proficiency_levels` — không hardcode level nào.
3. **N target languages**: Meaning không hardcode `meaning_vi`, `meaning_en`. Dùng bảng `vocabulary_meanings` hỗ trợ N ngôn ngữ dịch.
4. **Language-scoped content**: Topics, grammar points, proficiency levels đều thuộc về 1 `language`. Mỗi ngôn ngữ có topic set riêng, grammar set riêng.
5. **Generic pronunciation scoring**: Không hardcode `initial/final/tone` (Mandarin-specific). Dùng JSONB `dimensions` để mỗi ngôn ngữ define scoring dimensions riêng.

---

## Nhóm 1: Language & Proficiency

### `languages` — Ngôn ngữ được hỗ trợ

Bảng top-level. Mọi content đều thuộc về 1 language. Khi mở rộng sang ngôn ngữ mới, chỉ cần thêm 1 row + seed content.

```sql
CREATE TABLE languages (
    id         UUID PRIMARY KEY,
    code       VARCHAR(10) NOT NULL UNIQUE,   -- 'zh', 'ja', 'ko', 'th', 'id'
    name_en    VARCHAR(100) NOT NULL,          -- 'Chinese', 'Japanese'
    name_native VARCHAR(100) NOT NULL,         -- '中文', '日本語'
    is_active  BOOLEAN DEFAULT true,
    config     JSONB DEFAULT '{}'::jsonb,
    -- Config per language:
    --   zh: { "has_tones": true, "has_stroke": true, "writing_system": "hanzi", "ocr_supported": true }
    --   ja: { "has_tones": false, "has_stroke": true, "writing_systems": ["kanji","hiragana","katakana"], "ocr_supported": true }
    --   ko: { "has_tones": false, "has_stroke": true, "writing_system": "hangul", "ocr_supported": false }
    --   th: { "has_tones": true, "has_stroke": false, "writing_system": "thai", "ocr_supported": false }
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### `proficiency_levels` — Hệ thống trình độ per language

Thay thế `hsk_level` hardcode. Mỗi ngôn ngữ có hệ thống riêng: HSK 1-9 (CN), JLPT N5-N1 (JP), TOPIK 1-6 (KR), CEFR A1-C2 (chung).

```sql
CREATE TABLE proficiency_levels (
    id            UUID PRIMARY KEY,
    language_id   UUID NOT NULL REFERENCES languages(id),
    code          VARCHAR(20) NOT NULL,        -- 'hsk-1', 'jlpt-n5', 'topik-1', 'cefr-a1'
    name          VARCHAR(100) NOT NULL,        -- 'HSK 1', 'JLPT N5'
    stage         VARCHAR(50),                  -- 'Elementary', 'Intermediate', 'Advanced'
    sort_order    INTEGER NOT NULL,             -- 1, 2, 3... dùng để sắp xếp tăng dần
    access_tier   VARCHAR(20) DEFAULT 'free',   -- 'free', 'pro', 'pro_phase2'

    -- Stats cho tracking progress (số liệu chuẩn per level)
    total_vocabulary   INTEGER,    -- HSK 1: 300, JLPT N5: 800
    total_characters   INTEGER,    -- HSK 1: 246 (recognition)
    total_syllables    INTEGER,    -- HSK 1: 269
    total_grammar      INTEGER,    -- HSK 1: ~20

    metadata      JSONB DEFAULT '{}'::jsonb,
    -- zh: { "writing_characters": 0, "recognition_characters": 246 }
    -- ja: { "kanji_count": 80, "kana_only_words": 200 }

    created_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(language_id, code)
);

CREATE INDEX idx_pl_language ON proficiency_levels(language_id, sort_order);
```

---

## Nhóm 2: Vocabulary Content

### `vocabularies` — Từ vựng (global shared, không thuộc user)

Bảng trung tâm. Field names generic: `headword` (không phải `hanzi`), `romanization` (không phải `pinyin`).
Metadata đặc thù ngôn ngữ vào JSONB `metadata`.

```sql
CREATE TABLE vocabularies (
    id               UUID PRIMARY KEY,
    language_id      UUID NOT NULL REFERENCES languages(id),
    proficiency_level_id UUID NOT NULL REFERENCES proficiency_levels(id),

    -- Generic fields (mọi ngôn ngữ đều có)
    headword         VARCHAR(255) NOT NULL,     -- '学习', '勉強', '공부', 'เรียน'
    romanization     VARCHAR(255),              -- 'xuéxí', 'benkyō', 'gongbu', 'riian'
    audio_url        VARCHAR(500),
    frequency_rank   INTEGER,
    examples         JSONB DEFAULT '[]'::jsonb,
    -- [{ "sentence": "我每天学习中文。", "translations": { "vi": "...", "en": "..." }, "audio_url": "..." }]

    -- Language-specific metadata (JSONB — schema khác nhau per language)
    metadata         JSONB DEFAULT '{}'::jsonb,
    -- Chinese:  { "radicals": ["子","冖","习"], "stroke_count": 11, "stroke_data_url": "...",
    --             "recognition_only": true, "tone_numbers": [2,2] }
    -- Japanese: { "kanji": "勉強", "hiragana": "べんきょう", "katakana": "ベンキョウ",
    --             "jlpt_kanji_level": "N4", "stroke_count": 16, "stroke_data_url": "..." }
    -- Korean:   { "hanja": "工夫", "hangul": "공부" }
    -- Thai:     { "tone_class": "mid", "vowel_length": "long" }

    created_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at       TIMESTAMPTZ DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_vocab_headword_lang ON vocabularies(language_id, headword) WHERE deleted_at IS NULL;
CREATE INDEX idx_vocab_language ON vocabularies(language_id);
CREATE INDEX idx_vocab_proficiency ON vocabularies(proficiency_level_id);
CREATE INDEX idx_vocab_deleted_at ON vocabularies(deleted_at);
CREATE INDEX idx_vocab_examples ON vocabularies USING GIN (examples);
CREATE INDEX idx_vocab_frequency ON vocabularies(frequency_rank) WHERE frequency_rank IS NOT NULL;
CREATE INDEX idx_vocab_metadata ON vocabularies USING GIN (metadata);
```

### `vocabulary_meanings` — Nghĩa của từ theo N ngôn ngữ đích

Thay thế `meaning_vi`, `meaning_en` hardcode. Hỗ trợ N target languages.
1 từ có thể có nhiều nghĩa trong cùng 1 ngôn ngữ đích (polysemy).

```sql
CREATE TABLE vocabulary_meanings (
    id              UUID PRIMARY KEY,
    vocabulary_id   UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    target_lang     VARCHAR(10) NOT NULL,    -- 'vi', 'en', 'th', 'id', 'ko'
    meaning         TEXT NOT NULL,            -- 'học tập', 'to study'
    is_primary      BOOLEAN DEFAULT false,    -- nghĩa chính
    sort_order      INTEGER DEFAULT 0,
    UNIQUE(vocabulary_id, target_lang, meaning)
);

CREATE INDEX idx_vm_vocab ON vocabulary_meanings(vocabulary_id);
CREATE INDEX idx_vm_lang ON vocabulary_meanings(vocabulary_id, target_lang);
```

---

## Nhóm 3: Classification

### `topics` — Chủ đề per language

Mỗi ngôn ngữ có topic set riêng (Chinese có 10 topics chuẩn HSK, Japanese có topic set khác).
Tên topic hỗ trợ đa ngôn ngữ UI qua JSONB `names`.

```sql
CREATE TABLE topics (
    id          UUID PRIMARY KEY,
    language_id UUID NOT NULL REFERENCES languages(id),
    slug        VARCHAR(100) NOT NULL,
    names       JSONB NOT NULL,
    -- { "en": "Daily Life", "vi": "Cuộc sống hằng ngày", "zh": "日常生活", "ja": "日常生活" }
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(language_id, slug)
);

CREATE INDEX idx_topics_language ON topics(language_id);
```

### `vocabulary_topics` — Junction: vocabulary ↔ topic (M:N)

1 từ thuộc nhiều topic (polysemy).

```sql
CREATE TABLE vocabulary_topics (
    vocabulary_id UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    topic_id      UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (vocabulary_id, topic_id)
);

CREATE INDEX idx_vt_topic ON vocabulary_topics(topic_id);
```

### `grammar_points` — Grammar patterns per language

Mỗi ngôn ngữ có grammar set riêng. Gắn vào proficiency level (không hardcode `hsk_level`).
Example và rule hỗ trợ đa ngôn ngữ qua JSONB.

```sql
CREATE TABLE grammar_points (
    id                   UUID PRIMARY KEY,
    language_id          UUID NOT NULL REFERENCES languages(id),
    proficiency_level_id UUID NOT NULL REFERENCES proficiency_levels(id),
    code                 VARCHAR(50) NOT NULL UNIQUE,
    pattern              VARCHAR(255) NOT NULL,        -- 'S + 把 + O + V + Complement'
    examples             JSONB DEFAULT '{}'::jsonb,
    -- { "source": "我把书放在桌子上。", "translations": { "vi": "Tôi để sách lên bàn.", "en": "..." } }
    rule                 JSONB DEFAULT '{}'::jsonb,
    -- { "vi": "Dùng 把 khi tác động lên đối tượng cụ thể", "en": "Use 把 when..." }
    common_mistakes      JSONB DEFAULT '{}'::jsonb,
    -- { "vi": "Không dùng 把 với 是, 有, 知道", "en": "Don't use 把 with 是, 有, 知道" }
    created_at           TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_gp_language ON grammar_points(language_id);
CREATE INDEX idx_gp_proficiency ON grammar_points(proficiency_level_id);
```

### `vocabulary_grammar_points` — Junction: vocabulary ↔ grammar point (M:N)

```sql
CREATE TABLE vocabulary_grammar_points (
    vocabulary_id    UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    grammar_point_id UUID NOT NULL REFERENCES grammar_points(id) ON DELETE CASCADE,
    PRIMARY KEY (vocabulary_id, grammar_point_id)
);

CREATE INDEX idx_vgp_gp ON vocabulary_grammar_points(grammar_point_id);
```

---

## Nhóm 4: User Organization

### `folders` — Folder user tự tạo

User-scoped. Scoped per language (1 folder chứa từ của 1 ngôn ngữ).

```sql
CREATE TABLE folders (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    language_id UUID NOT NULL REFERENCES languages(id),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_folders_user ON folders(user_id, language_id);
CREATE INDEX idx_folders_deleted ON folders(deleted_at);
```

### `folder_vocabularies` — Junction: folder ↔ vocabulary (M:N)

```sql
CREATE TABLE folder_vocabularies (
    folder_id     UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    vocabulary_id UUID NOT NULL REFERENCES vocabularies(id) ON DELETE CASCADE,
    added_at      TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (folder_id, vocabulary_id)
);
```

---

## Nhóm 5: Learning Progress

### `user_vocabulary_progress` — Memory Score + SRS per user per word

Bảng cốt lõi cho Trụ 2 + 3. Lưu toàn bộ tiến độ học của 1 user với 1 từ:

- **Memory Score** tổng hợp từ 8 mode: `Σ(Mode_Score × Weight) + Spacing_Score / Max_Points × 100`
- **6 trạng thái**: Start Learning → Still Learning → Almost Learnt → Finish Learning → Memory Mode → Mastered
- **SM-2 data** cho spaced repetition
- **Spacing streak** cho điều kiện Memory Mode (≥1) và Mastered (≥2)
- `max_points` để handle Free→Pro migration (normalize score)

```sql
CREATE TABLE user_vocabulary_progress (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users(id),
    vocabulary_id   UUID NOT NULL REFERENCES vocabularies(id),

    -- Memory Score (0-100)
    memory_score    DECIMAL(5,2) NOT NULL DEFAULT 0,
    memory_state    VARCHAR(30) NOT NULL DEFAULT 'start_learning',
    -- start_learning, still_learning, almost_learnt,
    -- finish_learning, memory_mode, mastered

    -- Per-mode scores — generic, áp dụng cho mọi ngôn ngữ
    -- Modes nào ngôn ngữ không hỗ trợ thì giữ 0 (e.g. Thai không có stroke mode)
    score_discover       DECIMAL(3,1) DEFAULT 0,  -- max 1, weight 1
    score_recall         DECIMAL(3,1) DEFAULT 0,  -- max 2, weight 2
    score_stroke_guided  DECIMAL(3,1) DEFAULT 0,  -- max 1, weight 1 (CJK only)
    score_stroke_recall  DECIMAL(3,1) DEFAULT 0,  -- max 2, weight 2 (CJK only)
    score_pinyin_drill   DECIMAL(3,1) DEFAULT 0,  -- max 1, weight 1 (pronunciation drill)
    score_ai_chat        DECIMAL(3,1) DEFAULT 0,  -- max 2, weight 2
    score_review         DECIMAL(3,1) DEFAULT 0,  -- max 2, weight 2
    score_mastery_check  DECIMAL(3,1) DEFAULT 0,  -- max 2, weight 2
    spacing_score        DECIMAL(3,1) DEFAULT 0,  -- max 2

    -- SM-2 fields
    easiness_factor  DECIMAL(4,2) NOT NULL DEFAULT 2.50,
    interval_days    INTEGER NOT NULL DEFAULT 1,
    repetitions      INTEGER NOT NULL DEFAULT 0,
    next_review_at   TIMESTAMPTZ,
    last_reviewed_at TIMESTAMPTZ,

    -- Mastered conditions
    spacing_correct_streak INTEGER DEFAULT 0,
    last_mistake_at        TIMESTAMPTZ,

    -- Tier affects max_points: Free vs Pro (varies per language config)
    max_points       INTEGER NOT NULL DEFAULT 11,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, vocabulary_id)
);

CREATE INDEX idx_uvp_user ON user_vocabulary_progress(user_id);
CREATE INDEX idx_uvp_review ON user_vocabulary_progress(user_id, next_review_at)
    WHERE next_review_at IS NOT NULL;
CREATE INDEX idx_uvp_state ON user_vocabulary_progress(user_id, memory_state);
```

### `learning_sessions` — Session tracking

```sql
CREATE TABLE learning_sessions (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL REFERENCES users(id),
    language_id    UUID NOT NULL REFERENCES languages(id),
    mode           VARCHAR(30) NOT NULL,
    folder_id      UUID REFERENCES folders(id),

    total_words    INTEGER DEFAULT 0,
    correct_words  INTEGER DEFAULT 0,
    duration_ms    INTEGER DEFAULT 0,
    xp_earned      INTEGER DEFAULT 0,

    started_at     TIMESTAMPTZ DEFAULT NOW(),
    completed_at   TIMESTAMPTZ
);

CREATE INDEX idx_ls_user ON learning_sessions(user_id, started_at DESC);
CREATE INDEX idx_ls_language ON learning_sessions(user_id, language_id);
```

### `learning_events` — Event log cho mọi learning activity

Event sourcing. Mỗi lần user làm quiz, viết chữ, đọc phát âm, chat AI... đều log 1 event.

`event_data` JSONB cho phép mỗi mode + mỗi ngôn ngữ lưu data riêng:
- recall: `{ "quiz_type": "mcq", "options": [...], "selected": "..." }`
- stroke (CJK): `{ "stroke_accuracy": 0.85, "confusable_detected": "拨" }`
- pronunciation (Chinese): `{ "initial_score": 90, "final_score": 85, "tone_score": 70 }`
- pronunciation (Japanese): `{ "mora_scores": [90, 85, 70], "pitch_accent_correct": true }`
- ai_chat: `{ "session_id": "...", "words_used_correctly": [...] }`

```sql
CREATE TABLE learning_events (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id),
    vocabulary_id UUID NOT NULL REFERENCES vocabularies(id),

    mode          VARCHAR(30) NOT NULL,
    -- discover, recall, stroke_guided, stroke_recall, stroke_speed,
    -- pronunciation_drill, ai_chat, review, mastery_check

    score         DECIMAL(5,2),
    q_score       SMALLINT,           -- SM-2 quality (0-5), review mode only
    is_correct    BOOLEAN,
    duration_ms   INTEGER,
    event_data    JSONB DEFAULT '{}'::jsonb,
    session_id    UUID REFERENCES learning_sessions(id),

    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_le_user_vocab ON learning_events(user_id, vocabulary_id);
CREATE INDEX idx_le_user_mode ON learning_events(user_id, mode, created_at DESC);
CREATE INDEX idx_le_session ON learning_events(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_le_created ON learning_events(created_at);
```

### `pronunciation_scores` — Pronunciation tracking per unit

Mỗi ngôn ngữ có đơn vị phát âm khác nhau: syllable (CN), mora (JP), syllable (KR/TH).
Scoring dimensions cũng khác: Chinese có initial/final/tone, Japanese có pitch accent, Thai có tone class...
→ Dùng JSONB `dimensions` thay vì hardcode columns.

```sql
CREATE TABLE pronunciation_scores (
    id                UUID PRIMARY KEY,
    user_id           UUID NOT NULL REFERENCES users(id),
    vocabulary_id     UUID NOT NULL REFERENCES vocabularies(id),
    learning_event_id UUID REFERENCES learning_events(id),

    unit_index        SMALLINT NOT NULL,         -- 0-based, vị trí đơn vị phát âm trong word
    unit_text         VARCHAR(50) NOT NULL,       -- 'xué', 'べん', '공'
    overall_score     SMALLINT,                   -- 0-100

    -- Language-specific scoring dimensions
    dimensions        JSONB DEFAULT '{}'::jsonb,
    -- Chinese:  { "initial": 90, "final": 85, "tone": 70 }
    -- Japanese: { "mora": 90, "pitch_accent": 80, "length": 95 }
    -- Korean:   { "onset": 90, "nucleus": 85, "coda": 80 }
    -- Thai:     { "consonant": 90, "vowel": 85, "tone": 75 }

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ps_user ON pronunciation_scores(user_id, created_at DESC);
CREATE INDEX idx_ps_weakness ON pronunciation_scores(user_id, overall_score)
    WHERE overall_score < 70;
```

---

## Nhóm 6: OCR & Rate Limiting

### `ocr_scans` — OCR scan history

```sql
CREATE TABLE ocr_scans (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id),
    language_id  UUID NOT NULL REFERENCES languages(id),
    image_url    VARCHAR(500) NOT NULL,
    engine_used  VARCHAR(30) NOT NULL,

    detected_count    INTEGER DEFAULT 0,
    confirmed_count   INTEGER DEFAULT 0,
    duplicate_count   INTEGER DEFAULT 0,
    results           JSONB DEFAULT '[]'::jsonb,
    -- [{ "headword": "学", "confidence": 0.95, "status": "confirmed|edited|deleted" }]

    folder_id    UUID REFERENCES folders(id),
    status       VARCHAR(20) DEFAULT 'pending',

    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ocr_user ON ocr_scans(user_id, created_at DESC);
```

### `user_daily_counters` — Rate limiting cho Free tier

```sql
CREATE TABLE user_daily_counters (
    user_id       UUID NOT NULL REFERENCES users(id),
    counter_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    counter_type  VARCHAR(30) NOT NULL,
    -- scan, card_create, pronunciation, recall_writing
    count         INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (user_id, counter_date, counter_type)
);
```

---

## Nhóm 7: Dashboard Cache

### `user_learning_stats` — Materialized stats per user per language

Bảng denormalized cho Dashboard. Scoped per language vì user có thể học nhiều ngôn ngữ cùng lúc.

```sql
CREATE TABLE user_learning_stats (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users(id),
    language_id      UUID NOT NULL REFERENCES languages(id),

    total_words_learned   INTEGER DEFAULT 0,
    total_xp              INTEGER DEFAULT 0,
    current_streak_days   INTEGER DEFAULT 0,
    longest_streak_days   INTEGER DEFAULT 0,
    last_active_date      DATE,

    -- Memory State breakdown
    count_start_learning   INTEGER DEFAULT 0,
    count_still_learning   INTEGER DEFAULT 0,
    count_almost_learnt    INTEGER DEFAULT 0,
    count_finish_learning  INTEGER DEFAULT 0,
    count_memory_mode      INTEGER DEFAULT 0,
    count_mastered         INTEGER DEFAULT 0,

    -- Proficiency level progress (per level tracking)
    level_progress JSONB DEFAULT '{}'::jsonb,
    -- { "hsk-1": { "vocabulary": 250, "characters": 180, "syllables": 200, "grammar": 15 },
    --   "hsk-2": { "vocabulary": 80, "characters": 60, "syllables": 70, "grammar": 5 } }
    -- Japanese: { "jlpt-n5": { "vocabulary": 500, "kanji": 60, "grammar": 30 } }
    -- Totals per level lấy từ proficiency_levels table

    words_due_today    INTEGER DEFAULT 0,

    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, language_id)
);

CREATE INDEX idx_uls_user ON user_learning_stats(user_id);
```

---

## ERD

```
languages
  |
  |--< proficiency_levels
  |       |
  |       |--< vocabularies --< vocabulary_meanings
  |       |       |
  |       |       |--< vocabulary_topics >-- topics --< languages
  |       |       |
  |       |       |--< vocabulary_grammar_points >-- grammar_points
  |       |       |                                      |
  |       |       |                                      +-- languages
  |       |       |                                      +-- proficiency_levels
  |       |       |
  |       |       +--< folder_vocabularies >-- folders
  |       |       |                               |
  |       |       |                               +-- users
  |       |       |                               +-- languages
  |       |       |
  |       |       +--< user_vocabulary_progress --< users
  |       |       |
  |       |       +--< learning_events --< learning_sessions
  |       |       |         |
  |       |       |         +--< pronunciation_scores
  |       |       |
  |       |       +--< ocr_scans
  |       |
  |       +--< user_learning_stats --< users
  |
  +-- users --< user_daily_counters
```

## Tổng hợp: Thay đổi so với bản Chinese-only

| Bảng | Thay đổi |
|---|---|
| **MỚI `languages`** | Bảng top-level, mọi content thuộc về 1 language |
| **MỚI `proficiency_levels`** | Thay `hsk_level` INT. Hỗ trợ HSK, JLPT, TOPIK, CEFR... |
| **MỚI `vocabulary_meanings`** | Thay `meaning_vi`/`meaning_en` hardcode. N target languages |
| `vocabularies` | `hanzi`→`headword`, `pinyin`→`romanization`, thêm `language_id`, `proficiency_level_id`. CJK-specific fields (radicals, stroke...) → `metadata` JSONB |
| `topics` | Thêm `language_id`. `name_cn/vi/en` → `names` JSONB |
| `grammar_points` | Thêm `language_id`, `proficiency_level_id`. `example_cn/vi` → `examples` JSONB, `rule` → JSONB, `common_mistake` → `common_mistakes` JSONB |
| `folders` | Thêm `language_id` (1 folder = 1 ngôn ngữ) |
| `learning_sessions` | Thêm `language_id` |
| `pronunciation_scores` | `syllable_*` → `unit_*`. `initial/final/tone_score` → `dimensions` JSONB |
| `ocr_scans` | Thêm `language_id` |
| `user_learning_stats` | PK thành `(user_id, language_id)`. `dimension_progress` → `level_progress` generic |
| `user_vocabulary_progress` | Không đổi structure — mode scores giữ nguyên, modes không áp dụng cho ngôn ngữ đó thì giữ 0 |

## Multi-language Expansion Checklist

Khi thêm 1 ngôn ngữ mới:

1. Insert row vào `languages` (code, name, config JSONB)
2. Insert rows vào `proficiency_levels` (JLPT N5-N1, TOPIK 1-6...)
3. Seed `topics` cho ngôn ngữ đó
4. Seed `grammar_points` cho ngôn ngữ đó
5. Import `vocabularies` + `vocabulary_meanings` cho ngôn ngữ đó
6. Config `languages.config` JSONB để enable/disable features (OCR, stroke, tones...)

Không cần migration, không cần code change cho core logic.

## Volume Estimates (50K MAU)

| Nhóm | Bảng | Rows dự kiến |
|---|---|---|
| **Config** | `languages`, `proficiency_levels` | ~5 languages, ~30 levels |
| **Content** | `vocabularies`, `vocabulary_meanings`, `topics`, `grammar_points` | ~20K vocab (multi-lang), ~40K meanings, ~50 topics, ~200 grammar |
| **Organization** | `folders`, `folder_vocabularies` | ~250K folders, ~2.5M links |
| **Learning** | `user_vocabulary_progress` | ~5M |
| **Events** | `learning_events`, `learning_sessions` | ~75M events/month |
| **Pronunciation** | `pronunciation_scores` | ~15M/month |
| **Gating** | `user_daily_counters`, `ocr_scans` | ~1.5M counters/month |
| **Dashboard** | `user_learning_stats` | ~50K × languages learned |

> `learning_events` và `pronunciation_scores` là hot tables — cân nhắc **table partitioning by month** khi scale.
