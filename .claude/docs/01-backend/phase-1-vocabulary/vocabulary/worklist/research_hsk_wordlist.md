# HSK 3.0 Wordlist (11,000 từ) — Research

> Research cách đưa 11,000 từ vựng HSK 3.0 (9 levels) vào hệ thống. So sánh crawl vs download dataset vs API.

---

## 1. Problem Statement

Hệ thống cần seed 11,000 từ vựng HSK 3.0 (9 levels) với đầy đủ: hanzi, pinyin, meaning (vi/en), word type, HSK level, radicals, stroke count, frequency rank, examples. Hiện tại bulk import endpoint đã sẵn sàng (`POST /api/v1/vocabularies/import`) nhưng **chưa có data source**.

Nếu không làm: app không có content để học → không launch được.

---

## 2. Current State

### Đã có trong hệ thống

| Component | Status | Location |
|---|---|---|
| Bulk import endpoint | Done | `POST /api/v1/vocabularies/import` |
| Import use case | Done | `internal/vocabulary/application/usecase/import_command.go` |
| SaveBatch (batch 100) | Done | `internal/vocabulary/adapter/repository/vocabulary_repository.go` |
| Dedup by (language, word) | Done | `FindByWordList` check trước khi insert |
| Schema vocabularies | Done | migrations 000005-000013 |
| HSK categories + levels | Done | migrations 000003-000004 |
| Seed data files | **Missing** | Chưa có CSV/JSON nào |

### Fields cần populate per vocabulary

```
word, phonetic, language_id, proficiency_level_id, frequency_rank,
audio_url, image_url, metadata (radicals, stroke_count, shared_root),
meanings[] (language_id, meaning, word_type, is_primary),
  examples[] (sentence, phonetic, translations, audio_url),
topic_ids[], grammar_point_ids[]
```

---

## 3. Options Considered

### Option A: Download Open Dataset (CSV/JSON từ GitHub)

- **What:** Tải trực tiếp từ GitHub repos đã có sẵn HSK 3.0 data, parse → transform → call bulk import endpoint
- **How it works:**
  1. Tải `ivankra/hsk30` CSV (11,092 terms, pinyin, POS, HSK level)
  2. Enrich với CC-CEDICT (124K entries, English definitions)
  3. Enrich với `drkameleon/complete-hsk-vocabulary` JSON (radicals, frequency, multiple transcriptions)
  4. Enrich với Unihan Database (stroke count, radical decomposition)
  5. Viết Go CLI tool parse + transform → JSON → call `/api/v1/vocabularies/import`
- **Data sources:**

| Source | Entries | Fields | Format | License |
|---|---|---|---|---|
| [ivankra/hsk30](https://github.com/ivankra/hsk30) | 11,092 | hanzi, traditional, pinyin, POS, HSK level | CSV | MIT (code), data from PRC gov |
| [drkameleon/complete-hsk-vocabulary](https://github.com/drkameleon/complete-hsk-vocabulary) | HSK 2.0 + 3.0 | radical, frequency, 5 transcription systems, definitions, classifiers | JSON | MIT |
| [CC-CEDICT](https://www.mdbg.net/chinese/dictionary?page=cedict) | 124,839 | traditional, simplified, pinyin, English definitions | Text | CC BY-SA 4.0 |
| [Unihan Database](https://unicode.org/reports/tr38/) | All CJK | radical, stroke count, readings, definitions | TSV | Unicode ToU (free) |

- **Pros:**
  - Không cần crawl, không risk bị block
  - Data đã được clean và validate bởi community
  - Reproducible — cùng input → cùng output
  - License rõ ràng (MIT, CC BY-SA 4.0)
  - Fastest to implement (1-2 ngày cho CLI tool)
- **Cons:**
  - Thiếu Vietnamese meanings → cần translate riêng
  - Thiếu audio URLs → cần generate/source riêng
  - Thiếu example sentences tiếng Việt
- **Best when:** Cần seed data nhanh, đã có import infrastructure sẵn

### Option B: Crawl từ website HSK

- **What:** Scrape từ các website có HSK wordlist (mandarinbean.com, hsklord.com, hsk.academy, etc.)
- **How it works:**
  1. Dùng Colly (Go scraping framework) hoặc Rod (headless Chrome) crawl từng page
  2. Parse HTML → extract hanzi, pinyin, meaning, level
  3. Transform → call bulk import endpoint
- **Go libraries:**

| Library | Type | Use case |
|---|---|---|
| [Colly](https://github.com/gocolly/colly) | Framework | Static HTML, rate limiting built-in |
| [Goquery](https://github.com/PuerkitoBio/goquery) | Parser | DOM extraction, jQuery-like |
| [Rod](https://github.com/go-rod/rod) | Headless browser | JS-rendered pages, stealth mode |
| [Chromedp](https://github.com/chromedp/chromedp) | Headless browser | Chrome DevTools Protocol |

- **Pros:**
  - Có thể lấy data mới nhất từ source gốc
  - Một số site có Vietnamese meanings sẵn
- **Cons:**
  - **Fragile** — HTML structure thay đổi → crawler break
  - **Legal risk** — ToS hầu hết sites cấm scraping
  - **Rate limiting / blocking** — IP bị ban nếu crawl nhanh
  - **Incomplete data** — không site nào có đủ 11,000 từ với tất cả fields cần thiết
  - **Maintenance cost** — phải fix khi site thay đổi
  - **Slower** — phải handle pagination, retry, dedup
  - **Unnecessary** — data đã có sẵn dạng CSV/JSON
- **Best when:** Cần data từ nguồn không có download option (không áp dụng ở đây)

### Option C: API Services

- **What:** Gọi API có sẵn để lấy vocabulary data
- **APIs available:**

| API | Coverage | Rate | Cost |
|---|---|---|---|
| [pepebecker/pinyin-rest](https://github.com/pepebecker/pinyin-rest) | Per-word lookup (pinyin, definition) | Self-hosted | Free (MIT) |
| [pinyin-word-api](https://pinyin-word-api.vercel.app/) | Segmentation + examples | Unknown | Free |
| Google Translate API | Translation vi/en/th/id | 500K chars/month free | $20/1M chars |
| Azure Translator | Translation | 2M chars/month free | $10/1M chars |

- **Pros:**
  - Structured response, no parsing needed
  - Translation APIs solve Vietnamese meaning gap
- **Cons:**
  - **Không có bulk HSK API** — không API nào trả 11K từ một lần
  - Per-word lookup = 11K API calls = chậm + có thể bị rate limit
  - Translation API accuracy cho vocabulary context không cao
  - Cost nếu dùng premium translation (Google/Azure)
- **Best when:** Enrichment (thêm translation) sau khi đã có base data

---

## 4. Comparison

| Criteria | A: Download Dataset | B: Crawl Website | C: API Services |
|---|---|---|---|
| **Functional fit** | 90% — thiếu VI meanings, audio | 60% — fragile, incomplete | 40% — no bulk HSK API |
| **Data quality** | High — community validated | Medium — depends on source | Medium — translation accuracy |
| **Implementation effort** | 1-2 ngày (CLI parse + transform) | 3-5 ngày (crawler + error handling) | 2-3 ngày (API client + rate limiting) |
| **Legal risk** | Low — MIT, CC BY-SA 4.0 | **High** — ToS violations | Low — official APIs |
| **Reliability** | High — static files, reproducible | **Low** — sites change, IP blocking | Medium — API uptime |
| **Maintenance** | Near zero | **High** — fix khi HTML thay đổi | Low |
| **Coverage (fields)** | hanzi, pinyin, POS, level, radical, frequency, EN meaning | Varies by site | Per-word only |
| **Missing data** | VI/TH/ID meanings, audio, examples | Same + more gaps | Same |

---

## 5. Recommendation

**Option A: Download Dataset** — confidence: **high**

Lý do:
1. Data HSK 3.0 đã có sẵn, clean, 11,092 terms — **crawl là unnecessary**
2. Combine `ivankra/hsk30` (base) + `drkameleon/complete-hsk-vocabulary` (enrichment) + CC-CEDICT (definitions) + Unihan (stroke) → cover 90% fields
3. Gaps (VI meanings, audio, examples) phải giải quyết riêng bất kể option nào
4. Fastest, most reliable, legally safest

**Cho gaps còn lại:**
- Vietnamese meanings → Option C (Google/Azure Translate API) hoặc manual content team
- Audio → TTS API (Google Cloud TTS, Azure TTS) hoặc forvo.com data
- Example sentences → CC-CEDICT có basic examples, hoặc LLM-generate

---

## 6. Implementation Sketch

### Phase 1: Base Import (1-2 ngày)

```
cmd/seed/main.go  ← CLI tool, không phải HTTP handler
├── Download ivankra/hsk30 CSV
├── Parse CSV → []RawVocabulary
├── Enrich với drkameleon JSON (radical, frequency)
├── Enrich với CC-CEDICT (English definitions)
├── Enrich với Unihan (stroke_count)
├── Map to BulkImportRequest DTOs
└── Call import use case directly (bypass HTTP)
```

**Hoặc** viết standalone Go script:

```go
// cmd/seed/main.go
func main() {
    // 1. Parse hsk30.csv
    records := parseHSK30CSV("data/hsk30.csv")          // 11,092 records

    // 2. Load enrichment data
    cedict := parseCCEDICT("data/cedict_ts.u8")          // 124K entries
    complete := parseCompleteHSK("data/complete.json")    // radicals, frequency
    unihan := parseUnihan("data/Unihan_RadicalStrokeCounts.txt")

    // 3. Merge & transform
    vocabs := mergeData(records, cedict, complete, unihan)

    // 4. Batch import (reuse existing SaveBatch)
    repo := repository.NewVocabularyRepository(db)
    for batch := range chunk(vocabs, 100) {
        repo.SaveBatch(ctx, batch)
    }
}
```

### Phase 2: Vietnamese Meanings (1-2 ngày)

| Approach | Accuracy | Cost | Speed |
|---|---|---|---|
| Google Translate API | ~80% | ~$20 cho 11K words | Fast (batch) |
| Azure Translator | ~80% | Free tier (2M chars) | Fast (batch) |
| Content team review | ~98% | Human effort | Slow |
| **Hybrid: API translate → human review** | **~95%** | API + partial human | **Balanced** |

### Phase 3: Audio & Examples (optional, later)

- Audio: Google Cloud TTS hoặc Azure TTS — generate per word, upload to S3/GCS
- Examples: CC-CEDICT basic examples hoặc LLM-generate with human review

### Data Flow

```
GitHub CSV/JSON ──→ Go CLI (cmd/seed/) ──→ Transform ──→ SaveBatch ──→ PostgreSQL
       ↓                                        ↑
CC-CEDICT + Unihan ─────── Enrich ──────────────┘
       ↓
Google Translate API ──── Vietnamese meanings ───┘
```

---

## 7. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| HSK 3.0 word list không chính xác | Import sai level/word | Cross-validate giữa 2+ sources (ivankra + krmanik + drkameleon) |
| CC-CEDICT thiếu entry cho một số HSK từ | Missing English meaning | Fallback to drkameleon JSON (có definitions) |
| Google Translate không chính xác cho vocab context | Sai nghĩa tiếng Việt | Hybrid: API → human review. Flag low-confidence translations |
| Duplicate handling khi re-run seed | Double entries | Đã có dedup by (language_id, word) trong import use case |
| License issue CC-CEDICT (CC BY-SA 4.0) | Legal | Thêm attribution trong app credits. SA clause: nếu distribute modified data phải giữ SA |
| Data source repos bị xoá | Mất source | Download và commit vào `data/seed/` hoặc private storage |

---

## 8. Open Questions

| Question | How to resolve |
|---|---|
| Vietnamese meanings: API translate hay content team? | Estimate cost + timeline cả 2 → decide |
| Audio: TTS hay human recording? | TTS cho MVP, human recording cho HSK 1-3 (most used) |
| Seed data commit vào repo hay external storage? | < 10MB → repo (`data/seed/`). > 10MB → S3/GCS |
| CLI tool hay admin API endpoint cho seeding? | CLI (`cmd/seed/`) — one-time operation, không cần HTTP |
| Cần seed topics + grammar_points trước hay cùng lúc? | Trước — vocabularies reference proficiency_level_id và topic_ids |

---

## 9. References

### Primary Data Sources
- [ivankra/hsk30](https://github.com/ivankra/hsk30) — 11,092 HSK 3.0 terms, CSV, MIT
- [drkameleon/complete-hsk-vocabulary](https://github.com/drkameleon/complete-hsk-vocabulary) — HSK 2.0+3.0, JSON with radicals/frequency, MIT
- [krmanik/HSK-3.0](https://github.com/krmanik/HSK-3.0) — All 9 levels, TXT format
- [tonghuikang/HSK-3.0-words-list](https://github.com/tonghuikang/HSK-3.0-words-list) — CSV/JSON with Wiktionary meanings

### Enrichment Sources
- [CC-CEDICT Download](https://www.mdbg.net/chinese/dictionary?page=cedict) — 124K Chinese-English entries, CC BY-SA 4.0
- [Unicode Unihan Database](https://unicode.org/reports/tr38/) — Radical, stroke count, readings
- [willfliaw/hsk-dataset (HuggingFace)](https://huggingface.co/datasets/willfliaw/hsk-dataset) — 5K words with audio URLs, CC-BY-4.0

### Go Libraries (nếu cần crawl)
- [Colly](https://github.com/gocolly/colly) — Go web scraping framework, rate limiting, concurrency
- [Goquery](https://github.com/PuerkitoBio/goquery) — jQuery-like HTML parser
- [Rod](https://github.com/go-rod/rod) — Headless Chrome, stealth mode

### Translation APIs
- [Google Cloud Translation](https://cloud.google.com/translate) — 500K chars/month free
- [Azure Translator](https://azure.microsoft.com/en-us/products/ai-services/ai-translator) — 2M chars/month free
