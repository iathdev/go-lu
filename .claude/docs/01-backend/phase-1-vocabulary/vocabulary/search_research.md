# Bài toán Search từ nhanh — Research

> Nghiên cứu cách tối ưu search vocabulary trong ứng dụng học ngôn ngữ, đặc biệt cho CJK (Chinese/Japanese/Korean) + đa ngôn ngữ.

---

## 1. Problem Statement

**Bài toán**: User cần search từ vựng nhanh theo nhiều cách:
- Gõ hanzi: `学` → tìm `学习`, `学校`, `学生`
- Gõ pinyin: `xue` → tìm `学习 (xuéxí)`, `学校 (xuéxiào)`
- Gõ nghĩa: `học` → tìm `学习 (học tập)`, `学校 (trường học)`
- Gõ prefix (autocomplete): `xu` → suggest `学习`, `需要`
- Fuzzy (typo tolerance): `hoc tap` → tìm `học tập`

**Yêu cầu UX**: Autocomplete cần phản hồi < 100ms (p99) để user cảm nhận "instant". Trên 200ms user nhận ra delay.

**Nếu không làm gì**: Search hiện tại dùng `ILIKE %query%` — chậm ở scale, không hỗ trợ prefix pinyin, không fuzzy, không ranking.

---

## 2. Current State

### Hiện tại đang dùng: `ILIKE` substring match

```go
// vocabulary_repository.go:131-155
like := "%" + query + "%"
dbQuery.Where(
    "word ILIKE ? OR phonetic ILIKE ? OR id IN (SELECT vocabulary_id FROM vocabulary_meanings WHERE meaning ILIKE ?)",
    like, like, like,
)
```

### Vấn đề

| Vấn đề | Chi tiết |
|---|---|
| **Không dùng index** | `ILIKE '%query%'` (leading wildcard) → **full table scan**. B-tree index không giúp được. |
| **Subquery trên meanings** | `IN (SELECT ... WHERE meaning ILIKE ?)` → thêm 1 full scan trên bảng `vocabulary_meanings` |
| **Không ranking** | `ORDER BY word ASC` — không rank theo relevance. "学习" và "学而时习之" cùng rank khi search "学" |
| **Không fuzzy** | `hoc tap` (không dấu) ≠ `học tập` → không tìm thấy |
| **Không prefix pinyin** | `xu` không match `xuéxí` vì ILIKE cần exact substring |
| **Không CJK-aware** | PostgreSQL FTS `to_tsvector` mặc định không tokenize CJK (không có word boundary) |
| **Performance** | ~20K vocab hiện tại OK, nhưng sẽ chậm rõ ở 100K+ rows |

### Database schema hiện tại

- `vocabularies`: Không có FTS index, chỉ có B-tree + GIN (metadata JSONB)
- `vocabulary_meanings`: Không có index trên `meaning` text

---

## 3. Options Considered

### Option A: PostgreSQL pg_trgm (Trigram) — Nâng cấp in-place

**What:** Dùng extension `pg_trgm` của PostgreSQL, tạo GIN trigram index. Trigram chia text thành chuỗi 3 ký tự ("学习" → {"学习", " 学", "习 "}) rồi match bằng similarity.

**How it works:**
```sql
-- Enable extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram index cho fast search
CREATE INDEX idx_vocab_word_trgm ON vocabularies USING GIN (word gin_trgm_ops);
CREATE INDEX idx_vocab_phonetic_trgm ON vocabularies USING GIN (phonetic gin_trgm_ops);
CREATE INDEX idx_vm_meaning_trgm ON vocabulary_meanings USING GIN (meaning gin_trgm_ops);

-- Search query: prefix + fuzzy
SELECT * FROM vocabularies
WHERE word % 'xue'              -- similarity match (fuzzy)
   OR word ILIKE 'xue%'         -- prefix match
   OR phonetic ILIKE 'xue%'     -- pinyin prefix
ORDER BY word <-> 'xue' ASC     -- sort by distance (relevance)
LIMIT 20;

-- Hoặc dùng similarity threshold
SET pg_trgm.similarity_threshold = 0.3;
SELECT *, similarity(word, 'xue') AS sim
FROM vocabularies
WHERE word % 'xue'
ORDER BY sim DESC;
```

**CJK handling:**
- Trigram hoạt động tốt với CJK vì chia theo character, không cần word boundary
- "学习" tạo trigrams: `"  学"`, `" 学习"`, `"学习 "` → match được `学` (prefix)
- Single character search (1-2 chars) cần fallback ILIKE vì trigram cần >= 3 chars

**Pros:**
- Zero infrastructure thêm — chỉ cần `CREATE EXTENSION pg_trgm`
- GIN index hỗ trợ `ILIKE prefix%` rất tốt (dùng index scan thay vì full scan)
- Fuzzy matching built-in (`%` operator + `similarity()`)
- Relevance ranking via `<->` distance operator
- CJK works out of the box (trigram level, không cần tokenizer)
- Đã proven ở scale 100K-1M rows
- Không cần sync data, không cần maintain thêm service

**Cons:**
- Fuzzy quality thấp hơn Meilisearch/Typesense (dựa trên trigram overlap, không phải edit distance)
- Không có built-in pinyin→hanzi mapping (cần materialized column hoặc app-level)
- Performance giảm ở > 1M rows nếu query phức tạp
- GIN index tốn RAM (roughly 2-3x size of text data)
- Không có typo tolerance kiểu "did you mean?" suggestions
- Single character CJK search cần special handling

**Best when:** < 500K vocabulary entries, team muốn giữ stack đơn giản, không cần advanced features.

---

### Option B: PostgreSQL FTS + pg_trgm combo + Materialized Search Column

**What:** Kết hợp Full-Text Search (`tsvector/tsquery`) cho word boundary search + `pg_trgm` cho prefix/fuzzy. Thêm materialized column `search_text` gom tất cả searchable fields.

**How it works:**
```sql
-- Thêm search column chứa tất cả searchable text
ALTER TABLE vocabularies ADD COLUMN search_text TEXT GENERATED ALWAYS AS (
    COALESCE(word, '') || ' ' || COALESCE(phonetic, '') || ' ' ||
    COALESCE(metadata->>'pinyin_no_tone', '')  -- pinyin không dấu: "xuexi"
) STORED;

-- GIN trigram index trên search_text
CREATE INDEX idx_vocab_search_trgm ON vocabularies USING GIN (search_text gin_trgm_ops);

-- Materialized view cho meanings search (denormalized)
CREATE MATERIALIZED VIEW vocabulary_search_mv AS
SELECT
    v.id AS vocabulary_id,
    v.word,
    v.phonetic,
    v.search_text,
    v.language_id,
    string_agg(vm.meaning, ' ') AS all_meanings
FROM vocabularies v
LEFT JOIN vocabulary_meanings vm ON vm.vocabulary_id = v.id
WHERE v.deleted_at IS NULL
GROUP BY v.id, v.word, v.phonetic, v.search_text, v.language_id;

CREATE INDEX idx_vsm_search ON vocabulary_search_mv USING GIN (
    (search_text || ' ' || COALESCE(all_meanings, '')) gin_trgm_ops
);

-- Search query
SELECT vocabulary_id, word, phonetic,
       similarity(search_text, 'xue') AS sim_word,
       similarity(all_meanings, 'xue') AS sim_meaning
FROM vocabulary_search_mv
WHERE search_text ILIKE 'xue%'
   OR all_meanings ILIKE '%xue%'
   OR search_text % 'xue'
ORDER BY GREATEST(
    similarity(search_text, 'xue'),
    similarity(all_meanings, 'xue')
) DESC
LIMIT 20;
```

**Pinyin search strategy:**
```sql
-- Lưu pinyin variants trong metadata hoặc generated column:
-- metadata: { "pinyin_no_tone": "xuexi", "pinyin_abbrev": "xx" }
--
-- User gõ "xx" → match "学习" qua pinyin_abbrev
-- User gõ "xuexi" → match qua pinyin_no_tone
-- User gõ "xuéxí" → match qua phonetic (có tone)
-- User gõ "学" → match qua word
```

**Pros:**
- Vẫn zero infrastructure thêm
- `search_text` generated column — auto update, không cần app logic
- Materialized view denormalize meanings → 1 query thay vì subquery
- Hỗ trợ pinyin no-tone search (rất quan trọng cho CJK learners)
- Hỗ trợ pinyin abbreviation search (首字母: "xx" → "学习")
- Relevance ranking tốt hơn Option A (weighted similarity)

**Cons:**
- Materialized view cần `REFRESH` khi data thay đổi (sau insert/update vocab)
- Complexity cao hơn Option A — cần maintain generated columns + materialized view
- Vẫn không có "did you mean?" suggestions
- Generated column `search_text` thêm storage
- Matview refresh có thể block reads (dùng `CONCURRENTLY` để tránh, nhưng cần unique index)

**Best when:** Cần pinyin search + relevance ranking, nhưng không muốn thêm external service. Scale < 500K.

---

### Option C: Meilisearch — Dedicated Search Engine

**What:** Deploy Meilisearch (Rust-based, lightweight) bên cạnh PostgreSQL. Sync vocabulary data → Meilisearch. Client search qua Meilisearch, CRUD vẫn qua PostgreSQL.

**How it works:**
```
PostgreSQL (source of truth) → Sync → Meilisearch (search index)

Write path: Client → API → PostgreSQL → async sync → Meilisearch
Search path: Client → API → Meilisearch → return results
```

**Sync strategy:**
```go
// Sau khi create/update vocabulary trong PostgreSQL:
// 1. Publish event (hoặc call trực tiếp)
// 2. Meilisearch index document

type SearchDocument struct {
    ID         string   `json:"id"`
    Word       string   `json:"word"`
    Phonetic   string   `json:"phonetic"`
    PinyinNoTone string `json:"pinyin_no_tone"`
    PinyinAbbrev string `json:"pinyin_abbrev"`
    Meanings   []string `json:"meanings"`    // denormalized
    LanguageID string   `json:"language_id"`
    Level      string   `json:"level"`
    Topics     []string `json:"topics"`
}

// Meilisearch settings
index.UpdateSearchableAttributes([]string{
    "word", "phonetic", "pinyin_no_tone", "pinyin_abbrev", "meanings",
})
index.UpdateFilterableAttributes([]string{"language_id", "level", "topics"})
index.UpdateSortableAttributes([]string{"frequency_rank"})
```

**CJK support:**
- Meilisearch dùng `charabia` tokenizer — hỗ trợ CJK out of the box (character-level + dictionary-based)
- Chinese: dùng `jieba` dictionary segmentation
- Japanese: dùng `lindera` (MeCab-based)
- Tự động detect ngôn ngữ per document

**Pros:**
- **Typo tolerance** built-in (edit distance based, configurable per attribute)
- **Prefix search** instant — optimized cho autocomplete
- **CJK excellent** — dedicated tokenizers cho Chinese, Japanese, Korean
- **Sub-millisecond search** (< 50ms p99 ở 1M docs, thường < 20ms)
- **Faceted search** — filter by language, level, topic đồng thời search
- **Highlighting** — highlight matched text trong results
- **Ranking customizable** — words, typo, proximity, attribute, exactness
- **Go SDK** official: `github.com/meilisearch/meilisearch-go`
- **Lightweight** — single binary, ~100MB RAM cho 100K docs
- **Easy K8s deploy** — single container, persistent volume cho data

**Cons:**
- **Thêm infrastructure**: 1 container nữa trong K8s cluster
- **Data sync**: Cần implement sync PostgreSQL → Meilisearch (eventual consistency)
- **Dual write risk**: Nếu sync fail → search data stale
- **Operational overhead**: Monitor, backup, upgrade Meilisearch
- **License**: Dual licensed (MIT cho core features, commercial cho advanced features như multi-tenancy, API keys management)
- **Not ACID**: Search results có thể stale vài trăm ms sau write
- **RAM-bound**: Index phải fit in RAM (100K docs ≈ 100-200MB, 1M docs ≈ 1-2GB)

**Best when:** Cần autocomplete chất lượng cao, typo tolerance, CJK support tốt, scale > 100K docs.

---

### Option D: Typesense — Alternative Search Engine

**What:** Tương tự Meilisearch nhưng C++ based, focus performance. Self-hosted hoặc Typesense Cloud.

**How it works:** Tương tự Option C, thay Meilisearch bằng Typesense.

**Pros:**
- **Rất nhanh** — written in C++, benchmark thường nhanh hơn Meilisearch 10-30%
- **Typo tolerance** excellent (Levenshtein distance based)
- **CJK support** — từ v0.24+, dùng ICU tokenizer
- **Geo search** built-in (không cần cho vocab, nhưng có sẵn)
- **Built-in API key management** và rate limiting
- **Go SDK** official: `github.com/typesense/typesense-go`
- **Cluster mode** — built-in HA với Raft consensus (3 nodes)
- **GPLv3** — full open source, không có commercial-only features

**Cons:**
- **CJK support kém hơn Meilisearch** — ICU tokenizer generic, không có dedicated Chinese/Japanese dictionary segmentation
- **Pinyin search** cần custom handling (không có built-in pinyin mapping)
- **Ít popular hơn** — community nhỏ hơn Meilisearch
- **Schema strict** — cần define schema trước, ít flexible hơn Meilisearch
- Operational overhead tương tự Option C

**Best when:** Cần raw performance, built-in HA clustering, nhưng CJK không phải priority #1.

---

### Option E: Do Nothing (Keep ILIKE)

**What:** Giữ nguyên `ILIKE` hiện tại. Thêm index `pg_trgm` cho `ILIKE prefix%` (leading non-wildcard).

**Pros:** Zero effort, zero risk.

**Cons:** Performance tệ ở scale, không fuzzy, không pinyin search, UX kém.

**Best when:** Vocabulary stays < 10K, search không phải core feature.

---

## 4. Comparison

| Criteria | A: pg_trgm | B: FTS + pg_trgm combo | C: Meilisearch | D: Typesense |
|---|---|---|---|---|
| **Search latency (20K docs)** | 5-20ms | 5-20ms | 1-5ms | 1-5ms |
| **Search latency (500K docs)** | 20-100ms | 20-80ms | 5-20ms | 3-15ms |
| **CJK support** | Tốt (trigram level) | Tốt (trigram level) | Excellent (jieba/lindera) | Khá (ICU generic) |
| **Pinyin search** | Cần generated column | Tốt (generated column) | Tốt (custom attribute) | Cần custom handling |
| **Pinyin abbreviation** | Cần generated column | Tốt (generated column) | Tốt (custom attribute) | Cần custom handling |
| **Fuzzy/typo tolerance** | Có (trigram similarity) | Có (trigram similarity) | Excellent (edit distance) | Excellent (Levenshtein) |
| **Prefix autocomplete** | Có (ILIKE prefix%) | Có (ILIKE prefix%) | Excellent (optimized) | Excellent (optimized) |
| **Relevance ranking** | Basic (similarity) | Better (weighted) | Excellent (customizable) | Excellent (customizable) |
| **Highlighting** | Manual | Manual | Built-in | Built-in |
| **Infrastructure** | None (PG extension) | None (PG extension) | +1 container | +1 container |
| **Data sync** | None | Matview refresh | Cần implement | Cần implement |
| **Go SDK** | GORM (existing) | GORM (existing) | Official SDK | Official SDK |
| **Ops complexity** | Rất thấp | Thấp | Trung bình | Trung bình |
| **Migration effort** | Thấp (add index) | Trung bình (add columns + matview) | Cao (new service + sync) | Cao (new service + sync) |
| **Reversibility** | Rất dễ (drop index) | Dễ (drop column + matview) | Trung bình (remove service) | Trung bình (remove service) |
| **Cost** | $0 | $0 | +RAM cho container | +RAM cho container |

---

## 5. Recommendation

### Phase 1 (Ngay bây giờ): Option B — FTS + pg_trgm combo

**Confidence: High**

Lý do:
1. **Volume hiện tại ~20K vocab** — PostgreSQL native dư sức handle
2. **Zero infrastructure thêm** — không cần thêm container, không cần data sync
3. **Pinyin search** giải quyết được qua generated column `search_text` + metadata `pinyin_no_tone`
4. **Migration effort thấp** — chỉ cần migration thêm extension + index + generated column
5. **Reversible** — nếu không đủ, dễ dàng thêm Meilisearch sau
6. **Team fit** — không cần learn thêm technology, dùng GORM như hiện tại

### Phase 2 (Khi scale > 100K hoặc cần advanced features): Upgrade lên Option C — Meilisearch

Trigger để upgrade:
- Search latency p99 > 100ms
- Cần typo tolerance chất lượng cao
- Cần faceted search (filter by level + topic + search đồng thời)
- Cần highlighting trong results
- Vocabulary > 100K entries

---

## 6. Implementation Sketch

### Phase 1: pg_trgm + Generated Column

**Step 1: Migration — Add extension + indexes**
```sql
-- Migration: 000XXX_add_search_indexes.up.sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram indexes cho fast prefix + fuzzy search
CREATE INDEX idx_vocab_word_trgm ON vocabularies USING GIN (word gin_trgm_ops);
CREATE INDEX idx_vocab_phonetic_trgm ON vocabularies USING GIN (phonetic gin_trgm_ops);
CREATE INDEX idx_vm_meaning_trgm ON vocabulary_meanings USING GIN (meaning gin_trgm_ops);
```

**Step 2: Migration — Add search helper columns**
```sql
-- Migration: 000XXX_add_search_columns.up.sql

-- Pinyin no-tone column (generated hoặc trigger-maintained)
-- PostgreSQL 12+ generated columns không hỗ trợ IMMUTABLE functions phức tạp,
-- nên dùng trigger hoặc app-level update
ALTER TABLE vocabularies ADD COLUMN pinyin_no_tone VARCHAR(255);
ALTER TABLE vocabularies ADD COLUMN pinyin_abbrev VARCHAR(50);  -- 首字母: "xx" cho "学习"

CREATE INDEX idx_vocab_pinyin_no_tone_trgm ON vocabularies USING GIN (pinyin_no_tone gin_trgm_ops);
CREATE INDEX idx_vocab_pinyin_abbrev ON vocabularies (pinyin_abbrev);

-- Index cho meanings search
CREATE INDEX idx_vm_meaning_trgm ON vocabulary_meanings USING GIN (meaning gin_trgm_ops);
```

**Step 3: Backfill pinyin columns**
```sql
-- Backfill từ phonetic field (app-level, vì cần strip tone logic)
-- Go code strip tones: "xuéxí" → "xuexi", abbreviation: "xx"
```

**Step 4: Update repository search**
```go
func (repo *VocabularyRepository) Search(ctx context.Context, query string, languageID *domain.LanguageID, offset, limit int) ([]*domain.Vocabulary, error) {
    prefix := query + "%"
    like := "%" + query + "%"

    dbQuery := repo.db.WithContext(ctx).Model(&model.VocabularyModel{}).
        Select("vocabularies.*, similarity(word, ?) AS word_sim, similarity(COALESCE(phonetic,''), ?) AS phonetic_sim", query, query)

    if languageID != nil {
        dbQuery = dbQuery.Where("language_id = ?", languageID.UUID())
    }

    // Multi-field search: word, phonetic, pinyin_no_tone, pinyin_abbrev, meanings
    dbQuery = dbQuery.Where(`
        word ILIKE ? OR word % ?
        OR phonetic ILIKE ?
        OR pinyin_no_tone ILIKE ?
        OR pinyin_abbrev = ?
        OR id IN (SELECT vocabulary_id FROM vocabulary_meanings WHERE meaning ILIKE ? OR meaning % ?)
    `, prefix, query, prefix, prefix, query, like, query)

    // Sort by relevance: exact match > prefix > fuzzy
    dbQuery = dbQuery.Order(gorm.Expr(`
        CASE
            WHEN word = ? THEN 0
            WHEN word ILIKE ? THEN 1
            WHEN phonetic ILIKE ? THEN 2
            WHEN pinyin_no_tone ILIKE ? THEN 3
            WHEN pinyin_abbrev = ? THEN 4
            ELSE 5
        END, word <-> ? ASC
    `, query, prefix, prefix, prefix, query, query))

    // ... rest of query
}
```

### Phase 2 (tương lai): Thêm Meilisearch

```
1. Deploy Meilisearch container trong K8s
2. Tạo sync service: listen vocabulary CRUD events → update Meilisearch index
3. Backfill existing vocabularies vào Meilisearch
4. Thêm search port interface, implement MeilisearchSearchRepository
5. Switch search handler sang dùng Meilisearch, keep PostgreSQL cho CRUD
6. Giữ pg_trgm indexes làm fallback khi Meilisearch down (circuit breaker)
```

---

## 7. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| pg_trgm GIN index tốn RAM | Tăng memory usage PostgreSQL | Monitor `pg_stat_user_indexes`. 20K vocab ≈ negligible (< 10MB index) |
| Pinyin strip-tone logic sai | Search không tìm được từ đúng | Unit test comprehensive cho tone stripping. Bao gồm edge cases: "lǚ", "nǚ" |
| `similarity()` function chậm ở scale | Search latency tăng | Dùng `%` operator (dùng index) thay vì `similarity()` trong WHERE clause. `similarity()` chỉ dùng cho ORDER BY |
| Single character CJK search | Trigram cần >= 3 chars | Fallback sang ILIKE cho query < 3 chars |
| Materialized view stale | Search trả kết quả cũ | Nếu dùng matview: refresh sau mỗi vocabulary CRUD. Hoặc bỏ matview, dùng subquery (đủ nhanh ở 20K) |
| Future Meilisearch migration effort | Tốn thời gian refactor | Design search port interface từ đầu → swap implementation dễ dàng |

---

## 8. Open Questions

1. **Pinyin abbreviation search**: Có cần hỗ trợ "xx" → "学习" không? (首字母 rất phổ biến trong Chinese input)
2. **Cross-language search**: User Việt gõ "hoc" (không dấu) → có cần match "học tập" không? Nếu có → cần Vietnamese no-diacritics column tương tự pinyin_no_tone
3. **Search scope**: Search chỉ trong folder (deck) của user, hay search toàn bộ vocabulary global?
4. **Minimum query length**: Cho phép search 1 character (CJK) hay require >= 2?
5. **Result format**: Autocomplete trả lightweight (id, word, phonetic, primary meaning) hay full vocabulary?

---

## 9. References

- [PostgreSQL pg_trgm Documentation](https://www.postgresql.org/docs/current/pgtrgm.html) — Official docs, operators, functions, GIN index
- [PostgreSQL Full Text Search](https://www.postgresql.org/docs/current/textsearch.html) — tsvector/tsquery, text search configurations
- [Meilisearch Documentation](https://www.meilisearch.com/docs) — Getting started, CJK support, Go SDK
- [Meilisearch Go SDK](https://github.com/meilisearch/meilisearch-go) — Official Go client
- [Typesense Documentation](https://typesense.org/docs/) — Alternative search engine
- [Typesense Go SDK](https://github.com/typesense/typesense-go) — Official Go client
- [CJK Search in Meilisearch](https://www.meilisearch.com/docs/learn/engine/language) — Charabia tokenizer, jieba, lindera
- [pg_trgm Performance at Scale](https://about.gitlab.com/blog/2016/03/18/fast-search-using-postgresql-trigram-indexes/) — GitLab's experience with trigram indexes
- [Search UX: Autocomplete Latency](https://baymard.com/blog/autocomplete-design) — UX research on search latency expectations
