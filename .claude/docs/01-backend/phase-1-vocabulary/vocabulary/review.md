# Vocabulary Module — Code Review

> Reviewer: Claude | Date: 2026-03-25
> Scope: DB design, code quality, Phase 1 requirement coverage, multi-language scalability

---

## 1. Phase 1 Requirement Coverage

### 1.1 Trụ 1: Nhập từ vựng thông minh

| Requirement | Status | Ghi chú |
|---|---|---|
| Vocabulary CRUD (manual input) | ✅ Done | Full Create/Read/Update/Delete |
| HSK Built-in Wordlists (HSK 1-9) | ✅ Done | `ListByHSKLevel` endpoint + pagination |
| Data Model per Word (hanzi, pinyin, meanings, examples, audio, radicals, stroke, grammar, recognition_only, frequency_rank) | ✅ Done | Đầy đủ theo PRD |
| Bulk Import | ✅ Done | `ImportVocabularies` với duplicate check by hanzi |
| OCR Scan → Auto Flashcards | ✅ Done | `ProcessOCRScan` endpoint, adapter pattern tách biệt OCR module |
| OCR duplicate detection | ✅ Done | OCR adapter trả về `ExistingItems` + `NewItems` + `LowConfidence` |
| Topic system (10 HSK topics) | ✅ Done | CRUD topics, M:N junction table |
| Grammar Context per word | ✅ Done | Grammar points M:N, code/pattern/example/rule/common_mistake |
| Character Decomposition (radicals, stroke_count, stroke_data_url) | ✅ Done | Fields exist in entity + DB |
| Folder system (user-owned decks) | ✅ Done | CRUD + Add/Remove vocabulary |
| Folder ownership verification | ✅ Done | `getOwnedFolder()` checks userID |
| Search across hanzi/pinyin/meaning | ✅ Done | LIKE search on 4 columns |
| Vocabulary detail (topics + grammar) | ✅ Done | `GetVocabularyDetail` endpoint |

### 1.2 Trụ 2 & 3: Learning Path + Memory System

| Requirement | Status | Ghi chú |
|---|---|---|
| 7 Learning Modes | ❌ Not started | Thuộc module learning riêng, chưa implement |
| Memory Score / SM-2 | ❌ Not started | DB design đã có trong `database_design.md`, chưa implement |
| Dashboard 4-dimension tracking | ❌ Not started | `user_learning_stats` designed, chưa implement |

**Kết luận:** Trụ 1 (nhập từ vựng) **hoàn thành 100%**. Trụ 2 + 3 thuộc module learning, chưa bắt đầu — đây là expected vì chúng là module riêng.

---

## 2. Database Design Review

### 2.1 Current Schema (Migrations) vs Future Schema (database_design.md)

**GAP ANALYSIS — Current DB hardcode Chinese, future cần generic:**

| Aspect | Current Migration | Future Design (database_design.md) | Gap |
|---|---|---|---|
| **Vocabulary fields** | `hanzi`, `pinyin`, `meaning_vi`, `meaning_en` | `headword`, `romanization`, `metadata` JSONB | ⚠️ Major: field names Chinese-specific |
| **Proficiency** | `hsk_level INTEGER` hardcode | `proficiency_level_id UUID` → `proficiency_levels` table | ⚠️ Major: cần migration khi mở rộng |
| **Meanings** | 2 columns: `meaning_vi`, `meaning_en` | `vocabulary_meanings` table (N languages) | ⚠️ Major: không scale cho Thai, Indonesian |
| **Language scope** | Không có `language_id` | Mọi bảng đều có `language_id` FK | ⚠️ Major: cần thêm column + FK |
| **Topics** | `name_cn`, `name_vi`, `name_en` (3 columns) | `names` JSONB (N languages) | ⚠️ Medium: cần migration |
| **Grammar points** | `example_cn`, `example_vi`, `rule`, `common_mistake` (strings) | `examples` JSONB, `rule` JSONB, `common_mistakes` JSONB | ⚠️ Medium: cần migration |
| **Folders** | Không có `language_id` | Có `language_id` (1 folder = 1 ngôn ngữ) | ⚠️ Medium: cần migration |
| **`languages` table** | Không tồn tại | Core reference table | ⚠️ Cần tạo mới |
| **`proficiency_levels` table** | Không tồn tại | Thay `hsk_level` INT | ⚠️ Cần tạo mới |
| **`vocabulary_meanings` table** | Không tồn tại | Thay meaning_vi/meaning_en | ⚠️ Cần tạo mới |

**Đánh giá:** Thiết kế future trong `database_design.md` **rất tốt** — language-agnostic, pluggable proficiency, N-language meanings. Tuy nhiên **current migration và code hoàn toàn Chinese-specific**. Migration path từ current → future sẽ là **breaking change lớn**, cần:
1. Tạo `languages`, `proficiency_levels`, `vocabulary_meanings` tables
2. Rename `hanzi` → `headword`, `pinyin` → `romanization`
3. Migrate `meaning_vi`/`meaning_en` → `vocabulary_meanings` rows
4. Migrate `hsk_level` INT → `proficiency_level_id` UUID FK
5. Move `radicals`, `stroke_count`, `stroke_data_url`, `recognition_only` → `metadata` JSONB
6. Add `language_id` to `vocabularies`, `topics`, `grammar_points`, `folders`

### 2.2 Current DB Issues

#### 2.2.1 Search Performance — LIKE với `%query%` không dùng được index

```go
// vocabulary_repository.go:104
Where("hanzi LIKE ? OR pinyin LIKE ? OR meaning_vi LIKE ? OR meaning_en LIKE ?", q, q, q, q)
```

**Vấn đề:** `%query%` (leading wildcard) = full table scan. Với 11K+ từ (HSK 1-9) thì OK, nhưng khi scale lên multi-language (50K+ vocab) sẽ chậm.

**Khuyến nghị:**
- **Short-term:** Thêm `pg_trgm` extension + GIN trigram index cho search columns. Hoặc dùng `ILIKE` thay `LIKE` cho case-insensitive (hiện tại LIKE trên PostgreSQL là case-sensitive).
- **Long-term:** Full-text search với `tsvector` hoặc Elasticsearch/Meilisearch.

#### 2.2.2 Soft Delete inconsistency

```sql
-- Migration: vocabularies có deleted_at + soft delete index
CREATE INDEX IF NOT EXISTS idx_vocabularies_deleted_at ON vocabularies(deleted_at);
```

```go
// vocabulary_repository.go:133 — Hard delete!
func (repo *VocabularyRepository) Delete(ctx context.Context, id uuid.UUID) error {
    return repo.db.WithContext(ctx).Delete(&model.VocabularyModel{}, "id = ?", id).Error
}
```

**Vấn đề:** GORM `.Delete()` trên model có `DeletedAt` field sẽ thực hiện soft delete (set `deleted_at`), **KHÔNG phải hard delete**. Behavior đúng nhưng:
- `FindByHanzi`, `FindByHanziList`, `Search` — GORM tự filter `WHERE deleted_at IS NULL` ✅
- `FindByHSKLevel`, `FindByTopicID` — GORM tự filter ✅
- **Tuy nhiên**: Unique index `idx_vocabularies_hanzi_unique` đã có `WHERE deleted_at IS NULL` — đúng pattern. Cho phép re-create hanzi đã bị soft delete.

**Kết luận:** Soft delete behavior **đúng** nhờ GORM convention. Tuy nhiên comment trong code nên clarify "soft delete" thay vì "hard delete".

#### 2.2.3 Missing indexes cho search

Hiện tại không có index cho `pinyin`, `meaning_vi`, `meaning_en` — các columns được dùng trong search. Chỉ có index cho `hsk_level`.

**Khuyến nghị:** Thêm composite trigram index:
```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_vocabularies_search ON vocabularies
    USING GIN (hanzi gin_trgm_ops, pinyin gin_trgm_ops, meaning_vi gin_trgm_ops, meaning_en gin_trgm_ops);
```

#### 2.2.4 Bulk import — không trong transaction

```go
// import_command.go:68-72
if len(newVocabs) > 0 {
    imported, err = useCase.vocabRepo.SaveBatch(ctx, newVocabs)
```

`SaveBatch` dùng `CreateInBatches(models, 100)` — nếu batch thứ 2 fail, batch thứ 1 đã committed. Partial import có thể gây inconsistency.

**Khuyến nghị:** Wrap toàn bộ import trong 1 transaction, hoặc document rõ behavior "partial import is acceptable".

---

## 3. Code Quality Review

### 3.1 Architecture — Tốt

- **Hexagonal Architecture**: Rõ ràng — domain → ports → usecases → adapters. Zero framework dependency trong domain layer.
- **CQRS**: Command/Query tách biệt cho Vocabulary và Folder.
- **Module boundary**: Vocabulary module không import auth internals. OCR integration qua adapter pattern.
- **Error flow**: Domain errors → AppError (usecase) → HTTP response (handler). Đúng pattern.
- **DI**: Manual injection trong `module.go` — clean, no framework overhead.

### 3.2 Issues

#### 3.2.1 `downloadImage` trong handler — SSRF vulnerability

```go
// handler.go:223
func downloadImage(imageURL string, maxSize int64) ([]byte, error) {
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(imageURL)
```

**Vấn đề nghiêm trọng:** User cung cấp URL → server fetch URL đó. Đây là Server-Side Request Forgery (SSRF):
- User có thể gửi `http://169.254.169.254/latest/meta-data/` (AWS metadata)
- User có thể gửi `http://localhost:5432` (internal services)
- User có thể gửi `file:///etc/passwd`

**Khuyến nghị:**
1. Validate URL scheme: chỉ cho phép `https://`
2. Resolve DNS trước, block private IP ranges (10.x, 172.16-31.x, 192.168.x, 169.254.x, 127.x)
3. Hoặc tốt hơn: client upload image trực tiếp (base64 hoặc multipart), không qua URL

#### 3.2.2 OCR endpoint là public — thiếu authentication

```go
// module.go:55
public.POST("/vocabularies/ocr-scan", middleware.TimeoutMiddleware(60*time.Second), module.handler.ProcessOCRScan)
```

**Vấn đề:** OCR scan là public endpoint (không qua JWT auth). Bất kỳ ai cũng có thể gọi OCR API → tốn tiền OCR engine (Google Vision, Baidu). Rate limiting ở public group có giúp, nhưng vẫn dễ abuse.

**Khuyến nghị:** Chuyển sang protected route. OCR cần authentication + entitlement check (Free: 3 scans/day).

#### 3.2.3 Admin import endpoint — thiếu authorization check

```go
// module.go:58
v1.POST("/admin/vocabularies/import", module.handler.ImportVocabularies)
```

**Vấn đề:** Endpoint nằm trong protected group (cần JWT), nhưng **không check role admin**. Bất kỳ authenticated user nào cũng có thể import vocabulary.

**Khuyến nghị:** Thêm admin middleware hoặc role check.

#### 3.2.4 Duplicate `normalizePagination` và `getOwnedFolder`

```go
// folder_command.go + folder_query.go — cả 2 đều define:
func getOwnedFolder(...) {...}
func normalizePagination(...) {...}
```

**Vấn đề:** 2 hàm helper bị duplicate giữa `folder_command.go` và `folder_query.go`. Cùng logic, cùng tên.

**Khuyến nghị:** Extract ra package-level shared helpers (ví dụ `internal/vocabulary/application/usecase/helpers.go`).

#### 3.2.5 `VocabularyQuery.ListByHSKLevel` — N+1 potential

```go
// vocabulary_query.go — ListByHSKLevel
// Chỉ trả VocabularyListResponse (không có topics/grammar)
```

**Đánh giá:** Hiện tại **không có N+1** vì list endpoint chỉ trả basic fields. Detail endpoint (`GetVocabularyDetail`) mới fetch topics + grammar — đúng pattern. ✅

#### 3.2.6 `SetTopics` / `SetGrammarPoints` — loop INSERT thay vì batch

```go
// vocabulary_repository.go:160-163
for _, tid := range topicIDs {
    vt := model.VocabularyTopicModel{VocabularyID: vocabID, TopicID: tid}
    if err := tx.Create(&vt).Error; err != nil {
        return err
    }
}
```

**Vấn đề:** N inserts trong loop → N round-trips tới DB. Với số lượng nhỏ (1 vocab có ~2-3 topics) thì OK, nhưng pattern không optimal.

**Khuyến nghị:** Dùng batch insert: `tx.Create(&models)` với slice.

#### 3.2.7 `NewVocabulary` — unused `topic` parameter

```go
// domain/vocabulary.go:56
func NewVocabulary(hanzi, pinyin, meaningVI, meaningEN string, hskLevel int, topic string) (*Vocabulary, error) {
```

Parameter `topic` được nhận nhưng không sử dụng (không truyền vào `VocabularyParams`). Dead code.

**Khuyến nghị:** Remove parameter `topic` hoặc update `VocabularyParams` nếu cần.

### 3.3 Minor Issues

| # | File | Issue |
|---|---|---|
| 1 | `handler.go:183` | Gin context variable named `ctx` thay vì `ginCtx` — dễ nhầm với `context.Context` |
| 2 | `dto.go:106` | `OCRScanHTTPRequest.Engine` binding `oneof=paddleocr tesseract google_vision baidu_ocr` — hardcode engine list trong DTO, nên để config |
| 3 | `import_command.go` | Không có limit cho `BulkImportRequest.Vocabularies` — user có thể gửi 100K items |
| 4 | `folder_repository.go` | `AddVocabulary` không check duplicate — nếu add cùng vocab 2 lần sẽ lỗi PK violation |
| 5 | `vocabulary_repository.go:106` | `LIKE` trên PostgreSQL là **case-sensitive**. Pinyin search "xue" không match "Xue". Nên dùng `ILIKE` |

---

## 4. Multi-language Scalability Assessment

### 4.1 Current State: Chinese-only, hardcoded

| Component | Chinese-specific? | Migration effort |
|---|---|---|
| Domain entity: `Hanzi`, `Pinyin`, `MeaningVI`, `MeaningEN` | ✅ Yes | **High** — rename fields, update all callers |
| DB columns: `hanzi`, `pinyin`, `meaning_vi`, `meaning_en`, `hsk_level` | ✅ Yes | **High** — schema migration + data migration |
| DTOs: `Hanzi`, `Pinyin`, `MeaningVI`, `MeaningEN`, `HSKLevel` | ✅ Yes | **High** — breaking API change |
| Search: hardcode 4 column names | ✅ Yes | **Medium** — update query logic |
| Topics: `NameCN`, `NameVI`, `NameEN` | ✅ Yes | **Medium** — migrate to JSONB |
| Grammar: `ExampleCN`, `ExampleVI` | ✅ Yes | **Medium** — migrate to JSONB |
| Domain validation: `HSKLevel 1-9` | ✅ Yes | **Low** — parameterize per language |
| OCR: default language "zh" | ✅ Yes | **Low** — already parameterized |

### 4.2 Migration Strategy Recommendations

**Option A: Big-bang migration (1 migration, tất cả cùng lúc)**
- Pros: Clean codebase, no legacy code
- Cons: Rủi ro cao, downtime lớn, breaking API changes

**Option B: Incremental migration (recommended)**

1. **Phase 1 (hiện tại):** Ship Chinese-only, accept technical debt. ✅ Already done.
2. **Phase 2a — DB preparation:**
   - Tạo `languages`, `proficiency_levels` tables
   - Add `language_id` columns (nullable, default = Chinese UUID)
   - Add `vocabulary_meanings` table, backfill từ `meaning_vi`/`meaning_en`
   - **Không đổi code** — old columns vẫn hoạt động
3. **Phase 2b — Code dual-read:**
   - Code đọc từ cả old columns và new tables
   - Write vào cả 2 nơi (dual-write)
   - API response giữ nguyên format
4. **Phase 2c — Cutover:**
   - Migrate all reads sang new tables
   - Drop old columns
   - Rename entity fields (`Hanzi` → `Headword`)
   - Version API (v2) với generic field names

### 4.3 Recommendation

Với mục tiêu MVP (Phase 1), **current approach là đúng**: ship fast với Chinese-specific code, optimize later. `database_design.md` đã plan rõ future schema. Migration path tuy phức tạp nhưng feasible.

**Quan trọng:** Đừng migrate DB schema trước khi có nhu cầu thực sự (YAGNI). Khi thêm ngôn ngữ thứ 2, migration effort ~2-3 sprint cho codebase này.

---

## 5. Summary

### Điểm mạnh
1. **Architecture clean**: Hexagonal + CQRS + module boundary tốt
2. **Phase 1 Trụ 1 coverage 100%**: Tất cả requirement được implement
3. **Future-proof design doc**: `database_design.md` đã plan kỹ multi-language schema
4. **Error handling consistent**: Domain → AppError → HTTP response chain
5. **i18n ready**: Error messages dùng translation keys (en + vi)
6. **Domain validation**: Constructor validation, not anemic entities
7. **Pagination normalization**: Shared, consistent across endpoints

### Cần fix (Priority order)

| Priority | Issue | Impact |
|---|---|---|
| 🔴 Critical | SSRF trong `downloadImage` — validate URL, block private IPs | Security |
| 🔴 Critical | OCR endpoint public — cần authentication | Cost / Abuse |
| 🔴 Critical | Admin import thiếu role check | Security |
| 🟡 Medium | Search dùng `LIKE` thay vì `ILIKE` — case-sensitive trên PostgreSQL | UX Bug |
| 🟡 Medium | Bulk import không trong transaction — partial import risk | Data consistency |
| 🟡 Medium | Bulk import không limit size | DoS risk |
| 🟡 Medium | `AddVocabulary` to folder không handle duplicate gracefully | UX |
| 🟢 Low | `NewVocabulary` có unused `topic` parameter | Code cleanliness |
| 🟢 Low | Duplicate helper functions (`getOwnedFolder`, `normalizePagination`) | Code DRY |
| 🟢 Low | `SetTopics`/`SetGrammarPoints` loop insert thay vì batch | Performance |

### Scalability verdict

**Current code không scale cho multi-language** — Chinese-specific field names permeate domain, DB, và API layers. Tuy nhiên `database_design.md` đã plan kỹ migration path. Với Phase 1 MVP approach, đây là acceptable trade-off. Khi cần multi-language, expect **2-3 sprint** migration effort.
