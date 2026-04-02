# Vocabulary Relations — Research & Implementation Guide

> Research synonym/antonym/related word relationships cho vocabulary module.
> Áp dụng vào AI Chat (synonym detection), Learning Mode (synonym drill), và Discovery (suggest related words).

---

## 1. Tổng quan

| Aspect | Decision | Lý do |
|---|---|---|
| **DB Modeling** | Junction table `vocabulary_relationships` | Fit PostgreSQL + GORM, không thêm infra, đủ cho ~20K vocab |
| **Relation Types** | `synonym`, `antonym`, `related`, `hypernym`, `hyponym` | Cover use case học từ vựng |
| **Weight** | `DECIMAL(4,3)` 0.0–1.0 | Phân biệt exact (1.0) / near (0.7) / loose (0.3) |
| **Data Source** | ConceptNet + CiLin + Manual | Multilingual 36+ lang, CiLin chuyên Chinese |
| **Synonym Level** | Word-level (explicit) + Meaning-level (implicit query) | Meaning-level free từ `vocabulary_meanings` có sẵn |
| **Phase** | 3 phases: Manual → Auto-enrich → Embedding discovery | Tăng dần complexity |

---

## 2. Database Schema

### 2.1 Junction Table — `vocabulary_relationships`

```sql
CREATE TABLE vocabulary_relationships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vocabulary_id_a UUID NOT NULL,           -- always < vocabulary_id_b for symmetric relations
    vocabulary_id_b UUID NOT NULL,
    relation_type   VARCHAR(30) NOT NULL,    -- 'synonym', 'antonym', 'related', 'hypernym', 'hyponym'
    weight          DECIMAL(4,3) NOT NULL DEFAULT 1.000,
    source          VARCHAR(30) NOT NULL DEFAULT 'manual',  -- 'manual', 'conceptnet', 'cilin', 'cedict', 'embedding'
    metadata        JSONB DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT chk_ordered CHECK (vocabulary_id_a < vocabulary_id_b),
    CONSTRAINT chk_weight CHECK (weight > 0 AND weight <= 1),
    CONSTRAINT uq_vocab_relation UNIQUE(vocabulary_id_a, vocabulary_id_b, relation_type)
);

CREATE INDEX idx_vr_vocab_a ON vocabulary_relationships(vocabulary_id_a);
CREATE INDEX idx_vr_vocab_b ON vocabulary_relationships(vocabulary_id_b);
CREATE INDEX idx_vr_type ON vocabulary_relationships(relation_type);
CREATE INDEX idx_vr_source ON vocabulary_relationships(source);
```

### 2.2 Design Notes

| Rule | Chi tiết |
|---|---|
| **Canonical ordering** | `CHECK (vocabulary_id_a < vocabulary_id_b)` — tránh duplicate (A,B) và (B,A) cho symmetric relations (synonym, antonym) |
| **No FK constraints** | Theo design principle hiện tại — referential integrity ở application layer |
| **Asymmetric relations** | `hypernym`/`hyponym`: A = source, B = target. VD: A="动物" hypernym of B="狗". Khi query cần xét direction |
| **Weight ranges** | `1.0` = exact synonym (高兴↔开心), `0.7` = near-synonym (高兴↔愉快), `0.3` = loosely related |
| **Source priority** | `manual` > `cilin` > `conceptnet` > `cedict` > `embedding`. Khi conflict, ưu tiên source cao hơn |

### 2.3 Meaning-Level Synonyms (Implicit — No Table Needed)

Hai vocabulary share cùng meaning text trong `vocabulary_meanings` → implicitly synonymous:

```sql
-- Tìm synonyms của vocab X qua shared meanings
SELECT DISTINCT vm2.vocabulary_id
FROM vocabulary_meanings vm1
JOIN vocabulary_meanings vm2
  ON vm1.meaning = vm2.meaning
  AND vm1.language_id = vm2.language_id
  AND vm1.vocabulary_id != vm2.vocabulary_id
WHERE vm1.vocabulary_id = $1;
```

Dùng bổ trợ junction table, không thay thế.

---

## 3. Domain Model

### 3.1 Entity

```go
// domain/vocabulary_relationship.go

type RelationType string

const (
    RelationSynonym  RelationType = "synonym"
    RelationAntonym  RelationType = "antonym"
    RelationRelated  RelationType = "related"
    RelationHypernym RelationType = "hypernym"  // A is broader than B (动物 → 狗)
    RelationHyponym  RelationType = "hyponym"   // A is narrower than B (狗 → 动物)
)

type RelationSource string

const (
    SourceManual     RelationSource = "manual"
    SourceCiLin      RelationSource = "cilin"
    SourceConceptNet RelationSource = "conceptnet"
    SourceCEDICT     RelationSource = "cedict"
    SourceEmbedding  RelationSource = "embedding"
)

type VocabularyRelationship struct {
    ID            VocabularyRelationshipID
    VocabularyIDA VocabularyID
    VocabularyIDB VocabularyID
    RelationType  RelationType
    Weight        float64         // 0.0 < weight <= 1.0
    Source        RelationSource
    Metadata      map[string]any
}
```

### 3.2 Port

```go
// application/port/outbound.go — thêm vào existing file

type VocabularyRelationshipRepositoryPort interface {
    Save(ctx context.Context, rel *domain.VocabularyRelationship) error
    SaveBatch(ctx context.Context, rels []*domain.VocabularyRelationship) (int, error)
    FindByVocabularyID(ctx context.Context, vocabID domain.VocabularyID, relationType *domain.RelationType) ([]*domain.VocabularyRelationship, error)
    FindSynonyms(ctx context.Context, vocabID domain.VocabularyID) ([]*domain.VocabularyRelationship, error)
    Delete(ctx context.Context, id domain.VocabularyRelationshipID) error
    DeleteByPair(ctx context.Context, vocabIDA, vocabIDB domain.VocabularyID, relationType domain.RelationType) error
}
```

### 3.3 Query Pattern

```go
// Tìm tất cả synonyms của 1 vocabulary (cả 2 chiều vì symmetric)
func (repo *VocabularyRelationshipRepository) FindSynonyms(ctx context.Context, vocabID domain.VocabularyID) ([]*domain.VocabularyRelationship, error) {
    var models []VocabularyRelationshipModel
    err := repo.db.WithContext(ctx).
        Where("(vocabulary_id_a = ? OR vocabulary_id_b = ?) AND relation_type = ?",
            vocabID, vocabID, domain.RelationSynonym).
        Order("weight DESC").
        Find(&models).Error
    // ...
}
```

---

## 4. Data Sources

### 4.1 ConceptNet (Primary — Multilingual)

| Field | Value |
|---|---|
| URL | `https://api.conceptnet.io` |
| Languages | 36+ (zh, ja, vi, th, id, ko, en) |
| Relations | `Synonym`, `Antonym`, `RelatedTo`, `SimilarTo`, `IsA` |
| Rate limit | ~3600 req/hour |
| Cost | Free |

```
GET https://api.conceptnet.io/query?node=/c/zh/学习&rel=/r/Synonym&limit=20

Response:
{
  "edges": [
    {
      "start": { "label": "学习", "language": "zh" },
      "end":   { "label": "学", "language": "zh" },
      "rel":   { "label": "Synonym" },
      "weight": 2.0
    }
  ]
}
```

**Mapping ConceptNet → vocabulary_relationships:**
- ConceptNet `weight` (0-∞, thường 1-10) → normalize thành 0-1: `min(weight / 10, 1.0)`
- `Synonym` → `synonym`, `Antonym` → `antonym`, `RelatedTo`/`SimilarTo` → `related`, `IsA` → `hypernym`
- Lookup vocab by `word` + `language_id` → get `vocabulary_id`

### 4.2 CiLin — 同义词词林 (Chinese Only)

| Field | Value |
|---|---|
| Coverage | ~70K Chinese words |
| Structure | Hierarchical code: `Aa01A01=` — words sharing same fine-grained code = synonyms |
| Format | Text file, one line per group |
| Cost | Free (academic) |

**Code structure:** `[大类][中类][小类][词群][原子词群][=|#|@]`
- `=` → exact synonyms (same meaning, interchangeable)
- `#` → near-synonyms (similar but nuanced difference)
- `@` → related words (same semantic field)

**Mapping:**
- `=` groups → `synonym`, weight `1.0`
- `#` groups → `synonym`, weight `0.7`
- `@` groups → `related`, weight `0.3`

### 4.3 CC-CEDICT (Chinese-English)

Extract synonym groups từ shared English definitions. VD: 高兴 and 开心 both have definition "happy" → synonym pair.

### 4.4 Datamuse (English Only)

```
GET https://api.datamuse.com/words?rel_syn=happy
→ [{"word":"glad","score":100},{"word":"pleased","score":95},...]
```

Free, 100K req/day, no auth.

---

## 5. Multilingual — Sino-Vietnamese/Japanese Shared Roots

### 5.1 Cross-Language Synonym via Etymology

Chinese, Vietnamese (Hán-Việt), Japanese (on'yomi) share character roots:

| Chinese | Vietnamese (Hán-Việt) | Japanese | Shared Root |
|---|---|---|---|
| 学习 xuéxí | học tập | 学習 gakushū | 學習 |
| 经济 jīngjì | kinh tế | 経済 keizai | 經濟 |
| 社会 shèhuì | xã hội | 社会 shakai | 社會 |

### 5.2 Lưu trữ trong metadata JSONB

Vocabulary `metadata` field đã có sẵn. Thêm shared root info:

```json
{
  "radicals": ["子", "冖", "习"],
  "stroke_count": 8,
  "shared_root": "學習",
  "han_viet": "học tập"
}
```

### 5.3 Cross-Language Synonym Query

```sql
-- Tìm vocabularies cùng shared_root across languages
SELECT v.id, v.word, l.code AS language
FROM vocabularies v
JOIN languages l ON l.id = v.language_id
WHERE v.metadata->>'shared_root' = '學習';
```

---

## 6. Use Cases trong hệ thống

### 6.1 AI Chat — Synonym Detection (từ research_ai_chat.md)

```
User message → Word Segmentation (GSE)
  → Exact Match (against user's learned vocab)
  → Synonym Lookup (vocabulary_relationships WHERE relation_type = 'synonym')
  → Output: exact_match | synonym_match | new_word
```

| Match Type | Credit | Example |
|---|---|---|
| Exact match | 100% Memory Score | User wrote "高兴", đã học "高兴" |
| Synonym match | 50% credit | User wrote "开心", đã học "高兴" |
| New word | 0% (logged) | User wrote "愉悦", chưa học |

### 6.2 Vocabulary Detail — Related Words Section

Khi GET vocabulary detail, include related words:

```json
{
  "word": "高兴",
  "relations": {
    "synonyms": [
      { "word": "开心", "weight": 1.0 },
      { "word": "愉快", "weight": 0.7 }
    ],
    "antonyms": [
      { "word": "难过", "weight": 1.0 }
    ]
  }
}
```

### 6.3 Learning Mode — Synonym Drill

Quiz format: "Chọn từ đồng nghĩa với 高兴" → options from synonym relationships.

---

## 7. Implementation Phases

### Phase 1: Schema + Manual Seed (MVP)

| Task | Chi tiết |
|---|---|
| Migration | Tạo `vocabulary_relationships` table |
| Domain | Entity `VocabularyRelationship` + typed ID |
| Port | `VocabularyRelationshipRepositoryPort` |
| Repository | GORM implementation với canonical ordering logic |
| Use case | `RelationshipCommand` (Create, Delete) + `RelationshipQuery` (FindSynonyms, FindByVocab) |
| Handler | Admin CRUD endpoints |
| Seed | Bulk import từ CiLin (~5K-10K synonym groups cho HSK vocabulary) |
| Integration | AI Chat synonym lookup sử dụng `FindSynonyms` |

### Phase 2: Automated Enrichment

| Task | Chi tiết |
|---|---|
| ConceptNet worker | Background job query ConceptNet API per vocab entry |
| Source tracking | `source = 'conceptnet'`, weight normalized |
| Admin review | Endpoint to approve/reject auto-discovered relations |
| Dedup | Skip pairs đã có `source = 'manual'` (higher priority) |

### Phase 3: Embedding Discovery (Optional)

| Task | Chi tiết |
|---|---|
| pgvector | `CREATE EXTENSION vector` + add `embedding vector(300)` column |
| FastText | Load pre-trained vectors cho zh/ja/vi (300d) |
| Suggest endpoint | `GET /api/vocabularies/:id/suggestions` — cosine similarity top-10 |
| Promote flow | Curator review suggestions → promote to explicit `vocabulary_relationships` |

---

## 8. Alternatives Considered

| Approach | Pros | Cons | Verdict |
|---|---|---|---|
| **Junction table (chosen)** | Fits stack, explicit, weighted, auditable | Manual maintenance, limited to known pairs | MVP + long-term |
| **pgvector embeddings** | Auto-discovery, no maintenance | Catches antonyms too, needs vector loading infra | Phase 3 supplement |
| **Graph DB (Neo4j)** | Multi-hop traversal, native graph queries | New infra, breaks PG-only arch, K8s complexity | Rejected |
| **LLM-based detection** | Highest accuracy, context-aware | Expensive per-query, latency, non-deterministic | Not for lookup, only for AI Chat prompt |
| **Static synonym file (JSON)** | Simplest, no DB | No weight, no provenance, hard to update | Too limited |
