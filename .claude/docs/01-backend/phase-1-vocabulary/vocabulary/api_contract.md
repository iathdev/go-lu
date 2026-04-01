# Vocabulary Module — API Contract (Groups 1-4)

> Derived from `database_design.md` (multi-language design).
> Covers: Language & Proficiency, Vocabulary Content, Classification, User Organization.
> All endpoints follow the unified response envelope defined in `.claude/rules/api_response.md`.

---

## Conventions

- **Base path:** `/api/v1`
- **Auth:** All endpoints require JWT (`Authorization: Bearer <token>`) unless marked `[Public]`.
- **Language detection:** `lang` query param > `X-Lang` header > `Accept-Language` header (for i18n error messages + meaning language).
- **Pagination:** Query params `page` (default 1) + `page_size` (default 10, max 100). Response includes `meta.total`, `meta.page`, `meta.page_size`, `meta.total_pages`.
- **Soft delete:** Entities with `deleted_at` are excluded from queries by default.
- **UUID v7:** All IDs are UUID v7 format.

---

## Nhóm 1: Language & Proficiency

### 1.1 List Languages

```
GET /api/v1/languages
```

**[Public]** — No auth required. Cached aggressively (content rarely changes).

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `active_only` | bool | `true` | Filter by `is_active` |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "code": "zh",
      "name_en": "Chinese",
      "name_native": "中文",
      "is_active": true,
      "config": {
        "has_tones": true,
        "has_stroke": true,
        "writing_system": "hanzi",
        "ocr_supported": true
      }
    }
  ],
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

---

### 1.2 Get Language by ID

```
GET /api/v1/languages/:id
```

**[Public]**

**Response `200`:** Single language object (same shape as list item).

**Errors:** `404 NOT_FOUND` — `language.not_found`

---

### 1.3 List Categories

```
GET /api/v1/categories
```

**[Public]**

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `language_id` | UUID | — | Filter by language. Omit = all |
| `is_public` | bool | — | Filter by visibility. Omit = all |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "language_id": "019...",
      "prep_category_id": 5,
      "code": "hsk",
      "name": "HSK 3.0",
      "is_public": true
    }
  ],
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

---

### 1.4 Get Category by ID

```
GET /api/v1/categories/:id
```

**[Public]**

**Response `200`:** Single category object.

**Errors:** `404 NOT_FOUND` — `category.not_found`

---

### 1.5 List Proficiency Levels

```
GET /api/v1/proficiency-levels
```

**[Public]** — Sorted by `offset` ascending.

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `category_id` | UUID | — | Filter by category. Omit = all |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "category_id": "019...",
      "prep_level_id": 1,
      "code": "hsk-1",
      "name": "HSK 1",
      "target": 180.00,
      "display_target": "180 điểm",
      "offset": 1
    }
  ],
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

---

### 1.6 Get Proficiency Level by ID

```
GET /api/v1/proficiency-levels/:id
```

**[Public]**

**Response `200`:** Single proficiency level object.

**Errors:** `404 NOT_FOUND` — `proficiency_level.not_found`

---

## Nhóm 2: Vocabulary Content

### 2.1 Create Vocabulary

```
POST /api/v1/vocabularies
```

**Request body:**

```json
{
  "language_id": "019...",
  "proficiency_level_id": "019...",
  "word": "学习",
  "phonetic": "xuéxí",
  "audio_url": "https://...",
  "image_url": "https://...",
  "frequency_rank": 42,
  "metadata": {
    "radicals": ["子", "冖", "习"],
    "stroke_count": 11,
    "stroke_data_url": "https://...",
    "recognition_only": true
  },
  "meanings": [
    {
      "language_id": "019...(vi)",
      "meaning": "học tập",
      "word_type": "verb",
      "is_primary": true,
      "examples": [
        {
          "sentence": "我每天学习中文。",
          "phonetic": "Wǒ měitiān xuéxí zhōngwén.",
          "translations": { "vi": "Tôi học tiếng Trung mỗi ngày.", "en": "I study Chinese every day." },
          "audio_url": "https://..."
        }
      ]
    },
    {
      "language_id": "019...(en)",
      "meaning": "to study",
      "word_type": "verb",
      "is_primary": true,
      "examples": []
    }
  ]
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `language_id` | UUID | ✅ | Must exist in `languages` |
| `proficiency_level_id` | UUID | — | Must exist in `proficiency_levels` |
| `word` | string | ✅ | max 255 |
| `phonetic` | string | — | max 255 |
| `audio_url` | string | — | max 500, valid URL |
| `image_url` | string | — | max 500, valid URL |
| `frequency_rank` | int | — | ≥ 0 |
| `metadata` | object | — | Language-specific JSONB |
| `meanings` | array | ✅ | min 1 item |
| `meanings[].language_id` | UUID | ✅ | Target language |
| `meanings[].meaning` | string | ✅ | non-empty |
| `meanings[].word_type` | string | — | enum: `noun`, `verb`, `adjective`, `adverb`, `phrase` |
| `meanings[].is_primary` | bool | — | default `false` |
| `meanings[].examples` | array | — | |
| `meanings[].examples[].sentence` | string | ✅ | non-empty |
| `meanings[].examples[].phonetic` | string | — | |
| `meanings[].examples[].translations` | object | — | `{ "lang_code": "text" }` |
| `meanings[].examples[].audio_url` | string | — | max 500 |

**Response `201`:** `VocabularyResponse` (see 2.3).

**Errors:**

| Status | Code | Key | When |
|---|---|---|---|
| 400 | `BAD_REQUEST` | `vocabulary.word_required` | `word` empty |
| 400 | `BAD_REQUEST` | `vocabulary.meaning_required` | no meanings |
| 404 | `NOT_FOUND` | `language.not_found` | invalid `language_id` |
| 404 | `NOT_FOUND` | `proficiency_level.not_found` | invalid `proficiency_level_id` |
| 409 | `CONFLICT` | `vocabulary.already_exists` | duplicate `(language_id, word)` |
| 422 | `VALIDATION_FAILED` | `common.validation_failed` | binding errors |

---

### 2.2 Update Vocabulary

```
PUT /api/v1/vocabularies/:id
```

**Request body:** Same as Create (all fields overwrite). `meanings` replaces all existing meanings + examples (full replace strategy).

**Response `200`:** `VocabularyResponse`.

**Errors:** Same as Create + `404 vocabulary.not_found`.

---

### 2.3 Get Vocabulary

```
GET /api/v1/vocabularies/:id
```

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `meaning_lang` | string | — | Filter meanings by target language code (e.g. `vi`). Omit = all meanings |

**Response `200`:**

```json
{
  "success": true,
  "data": {
    "id": "019...",
    "language_id": "019...",
    "proficiency_level_id": "019...",
    "word": "学习",
    "phonetic": "xuéxí",
    "audio_url": "https://...",
    "image_url": "https://...",
    "frequency_rank": 42,
    "metadata": { "radicals": ["子", "冖", "习"], "stroke_count": 11 },
    "meanings": [
      {
        "id": "019...",
        "language_id": "019...",
        "meaning": "học tập",
        "word_type": "verb",
        "is_primary": true,
        "offset": 0,
        "examples": [
          {
            "id": "019...",
            "sentence": "我每天学习中文。",
            "phonetic": "Wǒ měitiān xuéxí zhōngwén.",
            "translations": { "vi": "Tôi học tiếng Trung mỗi ngày." },
            "audio_url": "https://..."
          }
        ]
      }
    ],
    "created_at": "2026-03-31T10:00:00Z"
  },
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

**Errors:** `404 NOT_FOUND` — `vocabulary.not_found`

---

### 2.4 Get Vocabulary Detail (with Topics + Grammar Points)

```
GET /api/v1/vocabularies/:id/detail
```

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `meaning_lang` | string | — | Filter meanings by target language code |

**Response `200`:** Same as 2.3 with additional fields:

```json
{
  "success": true,
  "data": {
    "...vocabulary fields...",
    "topics": [
      {
        "id": "019...",
        "category_id": "019...",
        "slug": "daily-life",
        "names": { "en": "Daily Life", "vi": "Cuộc sống hằng ngày", "zh": "日常生活" },
        "offset": 1
      }
    ],
    "grammar_points": [
      {
        "id": "019...",
        "category_id": "019...",
        "proficiency_level_id": "019...",
        "code": "gp_001",
        "pattern": "S + 把 + O + V + Complement",
        "examples": { "source": "我把书放在桌子上。", "translations": { "vi": "...", "en": "..." } },
        "rule": { "vi": "Dùng 把 khi...", "en": "Use 把 when..." },
        "common_mistakes": { "vi": "Không dùng 把 với 是, 有, 知道", "en": "Don't use..." }
      }
    ]
  },
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

---

### 2.5 List Vocabularies

```
GET /api/v1/vocabularies
```

Replaces current `GET /v1/vocabularies/hsk/:level` and `GET /v1/vocabularies/topic/:slug`.

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `language_id` | UUID | — | Filter by source language |
| `proficiency_level_id` | UUID | — | Filter by proficiency level |
| `topic_id` | UUID | — | Filter by topic |
| `meaning_lang` | string | — | Filter meanings by target language code |
| `page` | int | 1 | |
| `page_size` | int | 10 | max 100 |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "word": "学习",
      "phonetic": "xuéxí",
      "meanings": [
        { "meaning": "học tập", "word_type": "verb", "is_primary": true }
      ],
      "proficiency_level_id": "019...",
      "frequency_rank": 42
    }
  ],
  "meta": {
    "request_id": "...", "timestamp": "...",
    "total": 300, "page": 1, "page_size": 10, "total_pages": 30
  }
}
```

> **Note:** List response is lightweight — meanings without examples, no metadata.

---

### 2.6 Search Vocabularies

```
GET /api/v1/vocabularies/search
```

**Query params:**

| Param | Type | Required | Description |
|---|---|---|---|
| `q` | string | ✅ | Search term (matches `word`, `phonetic`, meaning text) |
| `language_id` | UUID | — | Filter by source language |
| `meaning_lang` | string | — | Filter meanings |
| `page` | int | — | |
| `page_size` | int | — | |

**Response `200`:** Same shape as 2.5.

**Errors:** `400 BAD_REQUEST` — `common.bad_request` (empty `q`)

---

### 2.7 Delete Vocabulary (Soft Delete)

```
DELETE /api/v1/vocabularies/:id
```

**Response `204`:** No content.

**Errors:** `404 NOT_FOUND` — `vocabulary.not_found`

---

### 2.8 Bulk Import Vocabularies (Admin)

```
POST /api/v1/admin/vocabularies/import
```

**Request body:**

```json
{
  "vocabularies": [
    {
      "language_id": "019...",
      "proficiency_level_id": "019...",
      "word": "学习",
      "phonetic": "xuéxí",
      "metadata": {},
      "meanings": [
        { "language_id": "019...", "meaning": "học tập", "word_type": "verb", "is_primary": true }
      ]
    }
  ]
}
```

**Response `200`:**

```json
{
  "success": true,
  "data": {
    "imported": 45,
    "skipped": 5,
    "total": 50
  },
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

---

## Nhóm 3: Classification

### 3.1 List Topics

```
GET /api/v1/topics
```

**[Public]** — Sorted by `offset` ascending.

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `category_id` | UUID | — | Filter by category. Omit = all |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "category_id": "019...",
      "slug": "daily-life",
      "names": { "en": "Daily Life", "vi": "Cuộc sống hằng ngày", "zh": "日常生活" },
      "offset": 1
    }
  ],
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

---

### 3.2 Get Topic by ID

```
GET /api/v1/topics/:id
```

**[Public]**

**Response `200`:** Single topic object (same shape as list item).

**Errors:** `404 NOT_FOUND` — `topic.not_found`

---

### 3.3 List Grammar Points

```
GET /api/v1/grammar-points
```

**[Public]**

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `category_id` | UUID | — | Filter by category. Omit = all |
| `proficiency_level_id` | UUID | — | Filter by level |
| `page` | int | 1 | |
| `page_size` | int | 10 | max 100 |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "category_id": "019...",
      "proficiency_level_id": "019...",
      "code": "gp_001",
      "pattern": "S + 把 + O + V + Complement",
      "examples": {
        "source": "我把书放在桌子上。",
        "translations": { "vi": "Tôi để sách lên bàn.", "en": "I put the book on the table." }
      },
      "rule": { "vi": "Dùng 把 khi tác động lên đối tượng cụ thể", "en": "Use 把 when..." },
      "common_mistakes": { "vi": "Không dùng 把 với 是, 有, 知道", "en": "Don't use 把 with..." }
    }
  ],
  "meta": {
    "request_id": "...", "timestamp": "...",
    "total": 80, "page": 1, "page_size": 10, "total_pages": 8
  }
}
```

---

### 3.4 Get Grammar Point by ID

```
GET /api/v1/grammar-points/:id
```

**[Public]**

**Response `200`:** Single grammar point object (same shape as list item).

**Errors:** `404 NOT_FOUND` — `grammar_point.not_found`

---

### 3.5 Manage Vocabulary ↔ Topic Associations

```
PUT /api/v1/vocabularies/:id/topics
```

Full replace — sets the exact topic list for a vocabulary.

**Request body:**

```json
{
  "topic_ids": ["019...", "019..."]
}
```

**Response `204`:** No content.

**Errors:**

| Status | Code | Key |
|---|---|---|
| 404 | `NOT_FOUND` | `vocabulary.not_found` |
| 404 | `NOT_FOUND` | `topic.not_found` (any invalid ID) |

---

### 3.6 Manage Vocabulary ↔ Grammar Point Associations

```
PUT /api/v1/vocabularies/:id/grammar-points
```

Full replace — sets the exact grammar point list for a vocabulary.

**Request body:**

```json
{
  "grammar_point_ids": ["019...", "019..."]
}
```

**Response `204`:** No content.

**Errors:**

| Status | Code | Key |
|---|---|---|
| 404 | `NOT_FOUND` | `vocabulary.not_found` |
| 404 | `NOT_FOUND` | `grammar_point.not_found` (any invalid ID) |

---

## Nhóm 4: User Organization

### 4.1 Create Folder

```
POST /api/v1/folders
```

**Request body:**

```json
{
  "language_id": "019...",
  "name": "Bài 1 - Chào hỏi",
  "description": "Từ vựng bài 1 sách giáo khoa"
}
```

| Field | Type | Required | Validation |
|---|---|---|---|
| `language_id` | UUID | ✅ | Must exist in `languages` |
| `name` | string | ✅ | max 255 |
| `description` | string | — | |

**Response `201`:**

```json
{
  "success": true,
  "data": {
    "id": "019...",
    "user_id": "019...",
    "language_id": "019...",
    "name": "Bài 1 - Chào hỏi",
    "description": "Từ vựng bài 1 sách giáo khoa",
    "created_at": "2026-03-31T10:00:00Z"
  },
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

**Errors:**

| Status | Code | Key |
|---|---|---|
| 404 | `NOT_FOUND` | `language.not_found` |
| 422 | `VALIDATION_FAILED` | `common.validation_failed` |

---

### 4.2 List User Folders

```
GET /api/v1/folders
```

Returns folders owned by the authenticated user.

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `language_id` | UUID | — | Filter by language |

**Response `200`:**

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "user_id": "019...",
      "language_id": "019...",
      "name": "Bài 1 - Chào hỏi",
      "description": "...",
      "vocabulary_count": 25,
      "created_at": "2026-03-31T10:00:00Z"
    }
  ],
  "meta": { "request_id": "...", "timestamp": "..." }
}
```

> `vocabulary_count` is a computed field (COUNT from `folder_vocabularies`).

---

### 4.3 Update Folder

```
PUT /api/v1/folders/:id
```

**Request body:**

```json
{
  "name": "Bài 1 - Updated",
  "description": "Updated description"
}
```

> `language_id` is **immutable** after creation — not included in update.

**Response `200`:** `FolderResponse`.

**Errors:**

| Status | Code | Key |
|---|---|---|
| 403 | `FORBIDDEN` | `folder.not_owner` |
| 404 | `NOT_FOUND` | `folder.not_found` |

---

### 4.4 Delete Folder (Soft Delete)

```
DELETE /api/v1/folders/:id
```

Soft deletes the folder. Does NOT delete vocabularies (they may exist in other folders).

**Response `204`:** No content.

**Errors:**

| Status | Code | Key |
|---|---|---|
| 403 | `FORBIDDEN` | `folder.not_owner` |
| 404 | `NOT_FOUND` | `folder.not_found` |

---

### 4.5 Add Vocabulary to Folder

```
POST /api/v1/folders/:id/vocabularies
```

**Request body:**

```json
{
  "vocabulary_id": "019..."
}
```

**Response `204`:** No content.

**Errors:**

| Status | Code | Key | When |
|---|---|---|---|
| 403 | `FORBIDDEN` | `folder.not_owner` | Not folder owner |
| 404 | `NOT_FOUND` | `folder.not_found` | |
| 404 | `NOT_FOUND` | `vocabulary.not_found` | |
| 409 | `CONFLICT` | `folder.vocabulary_already_added` | Duplicate |

---

### 4.6 Remove Vocabulary from Folder

```
DELETE /api/v1/folders/:id/vocabularies/:vocabulary_id
```

**Response `204`:** No content.

**Errors:**

| Status | Code | Key |
|---|---|---|
| 403 | `FORBIDDEN` | `folder.not_owner` |
| 404 | `NOT_FOUND` | `folder.not_found` |
| 404 | `NOT_FOUND` | `folder.vocabulary_not_found` |

---

### 4.7 List Vocabularies in Folder

```
GET /api/v1/folders/:id/vocabularies
```

**Query params:**

| Param | Type | Default | Description |
|---|---|---|---|
| `meaning_lang` | string | — | Filter meanings |
| `page` | int | 1 | |
| `page_size` | int | 10 | max 100 |
| `sort` | string | `added_at_desc` | `added_at_desc`, `added_at_asc`, `word_asc` |

**Response `200`:** Same shape as vocabulary list (2.5) with additional `added_at` per item.

```json
{
  "success": true,
  "data": [
    {
      "id": "019...",
      "word": "学习",
      "phonetic": "xuéxí",
      "meanings": [
        { "meaning": "học tập", "word_type": "verb", "is_primary": true }
      ],
      "proficiency_level_id": "019...",
      "frequency_rank": 42,
      "added_at": "2026-03-31T10:00:00Z"
    }
  ],
  "meta": {
    "request_id": "...", "timestamp": "...",
    "total": 25, "page": 1, "page_size": 10, "total_pages": 3
  }
}
```

**Errors:**

| Status | Code | Key |
|---|---|---|
| 403 | `FORBIDDEN` | `folder.not_owner` |
| 404 | `NOT_FOUND` | `folder.not_found` |

---

## Route Summary

| Method | Path | Auth | Group | Description |
|---|---|---|---|---|
| GET | `/languages` | Public | 1 | List languages |
| GET | `/languages/:id` | Public | 1 | Get language |
| GET | `/categories` | Public | 1 | List categories (`?language_id=`) |
| GET | `/categories/:id` | Public | 1 | Get category |
| GET | `/proficiency-levels` | Public | 1 | List levels (`?category_id=`) |
| GET | `/proficiency-levels/:id` | Public | 1 | Get level |
| POST | `/vocabularies` | JWT | 2 | Create vocabulary |
| GET | `/vocabularies` | JWT | 2 | List vocabularies (`?proficiency_level_id=`, `?topic_id=`, `?language_id=`) |
| GET | `/vocabularies/:id` | JWT | 2 | Get vocabulary |
| GET | `/vocabularies/:id/detail` | JWT | 2 | Get vocabulary detail |
| PUT | `/vocabularies/:id` | JWT | 2 | Update vocabulary |
| DELETE | `/vocabularies/:id` | JWT | 2 | Delete vocabulary |
| GET | `/vocabularies/search` | JWT | 2 | Search vocabularies |
| POST | `/admin/vocabularies/import` | Admin | 2 | Bulk import |
| GET | `/topics` | Public | 3 | List topics (`?category_id=`) |
| GET | `/topics/:id` | Public | 3 | Get topic |
| GET | `/grammar-points` | Public | 3 | List grammar points (`?category_id=`, `?proficiency_level_id=`) |
| GET | `/grammar-points/:id` | Public | 3 | Get grammar point |
| PUT | `/vocabularies/:id/topics` | JWT | 3 | Set vocabulary topics |
| PUT | `/vocabularies/:id/grammar-points` | JWT | 3 | Set vocabulary grammar points |
| POST | `/folders` | JWT | 4 | Create folder |
| GET | `/folders` | JWT | 4 | List folders (`?language_id=`) |
| PUT | `/folders/:id` | JWT | 4 | Update folder |
| DELETE | `/folders/:id` | JWT | 4 | Delete folder |
| POST | `/folders/:id/vocabularies` | JWT | 4 | Add vocabulary to folder |
| DELETE | `/folders/:id/vocabularies/:vocabulary_id` | JWT | 4 | Remove vocabulary |
| GET | `/folders/:id/vocabularies` | JWT | 4 | List folder vocabularies |

---

## Breaking Changes vs Current API

| Current Route | New Route | Reason |
|---|---|---|
| `GET /v1/vocabularies/hsk/:level` | `GET /v1/vocabularies?proficiency_level_id=` | HSK-specific → generic proficiency, flat query param |
| `GET /v1/vocabularies/topic/:slug` | `GET /v1/vocabularies?topic_id=` | Slug → UUID, flat query param |
| `GET /v1/topics` | `GET /v1/topics?category_id=` | Topics now filterable by category |
| _(new)_ | `GET /v1/languages` | New table, new endpoint |
| _(new)_ | `GET /v1/categories?language_id=` | New table, new endpoint |
| _(new)_ | `GET /v1/proficiency-levels?category_id=` | New table, new endpoint |
| All vocabulary request/response fields | `hanzi`→`word`, `pinyin`→`phonetic`, `meaning_vi/en`→`meanings[]`, `hsk_level`→`proficiency_level_id` | Multi-language generic fields |
| Folder create/response | Added `language_id` field | Folders now language-scoped |
| Topic response | `name_cn/vi/en` → `names` JSONB, added `category_id` | Multi-language i18n |
| Grammar point response | Scalar text fields → JSONB, `hsk_level` → `proficiency_level_id` | Multi-language + generic proficiency |
