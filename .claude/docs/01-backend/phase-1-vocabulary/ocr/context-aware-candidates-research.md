# Context-Aware Candidate Generation cho OCR Post-Processing — Research

> Nghiên cứu cách generate và rank chữ Hán tương tự (形近字) có xét ngữ cảnh khi OCR confidence thấp, cho app học tiếng Trung.

---

## 1. Problem Statement

OCR engine trả 1 kết quả per character, không trả alternatives. Khi confidence thấp, cần suggest chữ tương tự cho user chọn. Hiện tại doc chỉ có **static lookup table** (形近字 pairs) — purely visual similarity, không xét context.

**Vấn đề:**
- Cùng chữ bị nhận sai nhưng candidate đúng **phụ thuộc ngữ cảnh**. VD: OCR nhận "打雪话" → "雪" sai, nhưng candidate nào đúng? Nếu không có context → trả random 形近字. Có context → biết "打___话" → ưu tiên "电" (打电话)
- **Polysemy**: cùng chữ khác nghĩa tuỳ context — 了 trong "吃了饭" (particle) vs "了解" (understand)
- Static ranking (visual similarity → frequency → HSK) bỏ lỡ context signals

**Nếu không làm gì:** Candidate list sẽ random, user phải tự tìm đúng chữ trong top-3 — UX kém, đặc biệt cho người mới học.

---

## 2. Current State

Từ `research.md` section 5:
- Pre-built `map[rune][]SimilarChar` từ similar_chinese_characters CSV + makemeahanzi + Wiktionary
- ~5K chars × 5 candidates = ~300KB RAM, O(1) lookup
- Ranking: visual similarity → character frequency → HSK level
- "Context-based ranking (bigram) là Phase 2" — chưa có implementation

Architecture hiện tại: Go backend + Python sidecar (PaddleOCR). Candidate generation sẽ chạy trong Python sidecar hoặc Go.

---

## 3. Options Considered

### Option A: KenLM N-gram Language Model

- **What:** Statistical n-gram model (bigram/trigram) trained trên large Chinese corpus. Cho mỗi low-confidence character, generate visually similar candidates, form candidate sentences, score bằng n-gram probability.
- **How it works:**
  1. Train character-level bigram/trigram trên Chinese corpus (Wikipedia zh, CC-100)
  2. Khi OCR confidence thấp tại vị trí i, lấy candidates từ confusion set
  3. Với mỗi candidate, tính P(candidate | context) bằng n-gram LM
  4. Rank theo score, trả top-3
- **Pros:**
  - Cực nhanh (~1ms per query), không cần GPU
  - Model file nhỏ (~50-200MB cho trigram)
  - Deterministic, dễ debug
  - KenLM mature, production-ready (C++ với Python bindings)
  - pycorrector có sẵn Kenlm-CSC backend
- **Cons:**
  - Accuracy thấp nhất (F1 ~0.34 trên benchmark pycorrector)
  - Context window ngắn (2-3 chars) → miss long-range dependencies
  - Không hiểu semantics, chỉ statistical co-occurrence
- **Best when:** MVP, latency-critical, không có GPU

### Option B: MacBERT4CSC (BERT-based)

- **What:** Pre-trained MacBERT fine-tuned cho Chinese Spelling Correction. Detect + correct errors dựa trên bidirectional context.
- **How it works:**
  1. Input: OCR text (cả câu/dòng)
  2. Model predict probability distribution cho mỗi vị trí
  3. Vị trí có confidence thấp (từ OCR) → lấy top-K predictions từ model
  4. Filter predictions qua confusion set (chỉ giữ visually/phonetically similar)
  5. Rank theo model probability
- **Pros:**
  - Bidirectional context (cả câu) → hiểu semantics tốt hơn n-gram
  - Pre-trained model sẵn trên HuggingFace (`shibing624/macbert4csc-base-chinese`)
  - pycorrector hỗ trợ out-of-box
  - Latency acceptable (~50-100ms/sentence trên GPU, ~200-500ms CPU)
- **Cons:**
  - F1 ~0.40 general — **tụt mạnh trên domain-specific** (83% SIGHAN → 20% MCSC)
  - Cần fine-tune trên domain data để tốt
  - ~400MB model size
  - Có thể "overcorrect" — sửa chữ đúng thành chữ phổ biến hơn (nguy hiểm cho learning app)
- **Best when:** Cần context awareness tốt hơn n-gram, chấp nhận GPU/CPU inference cost

### Option C: LLM Fine-tuned (Qwen3-4B-CTC)

- **What:** LLM (Qwen3-4B) fine-tuned cho Chinese Text Correction qua 2-stage SFT. SOTA accuracy.
- **How it works:**
  1. Input: OCR text + prompt template (giống FeOCR: "đây là output OCR, sửa lỗi nhận dạng")
  2. LLM generate corrected text
  3. So sánh input vs output → identify changed positions → đó là candidates
  4. Caching kết quả cho recurring patterns
- **Pros:**
  - **SOTA accuracy** — F1 0.85 avg across domains (pycorrector benchmark)
  - Hiểu context sâu, xử lý polysemy tốt
  - Qwen3-4B đủ nhỏ để self-host (8GB VRAM)
  - pycorrector hỗ trợ (`Qwen3-4B-CTC`)
- **Cons:**
  - Latency cao (~200-500ms/sentence GPU, >1s CPU)
  - Cần GPU cho production (A10/T4 minimum)
  - Overcorrection risk — LLM "sửa" chữ hiếm thành chữ phổ biến (language prior mạnh hơn visual evidence)
  - Ops complexity: model serving (vLLM/TGI), GPU nodes trên K8s
  - Cost: GPU instance ~$200-500/tháng
- **Best when:** Accuracy là priority #1, có GPU infra, scale phase

### Option D: Hybrid Pipeline (Recommend)

- **What:** Kết hợp nhiều layer — fast path cho easy cases, heavy path cho hard cases.
- **How it works:**
  1. **Layer 1 — Static lookup** (hiện tại): confusion set + visual similarity. O(1), ~0ms
  2. **Layer 2 — KenLM re-rank**: Score candidates từ Layer 1 bằng character n-gram. ~1ms
  3. **Layer 3 — DISC formula** (plug-and-play): `Score = P(y|context) + 1.1 × (0.7 × PinyinSim + 0.3 × GlyphSim)`. Tích hợp vào Layer 2 scoring
  4. **Layer 4 — Domain constraint**: Filter candidates against app's vocabulary DB (HSK levels, user's learned words)
  5. **Layer 5 (optional, Phase 2+)** — MacBERT/LLM fallback: Chỉ invoke khi Layer 1-4 không confident. Cache results
- **Pros:**
  - Phần lớn requests xử lý ở Layer 1-3 (fast, no GPU)
  - Accuracy tăng dần qua từng layer
  - Domain constraint (Layer 4) là unique advantage cho learning app — chỉ suggest chữ user đang/sẽ học
  - Incremental adoption — ship Layer 1-3 trước, thêm Layer 5 sau
  - DISC formula proven (+1-4 F1 points, ACL 2025)
- **Cons:**
  - Nhiều layer = phức tạp hơn single model
  - KenLM vẫn limited context (nhưng DISC formula bù)
  - Layer 5 cần GPU nếu muốn
- **Best when:** Production system cần balance accuracy/latency/cost

---

## 4. Comparison

| Criteria | A: KenLM | B: MacBERT4CSC | C: LLM (Qwen3-4B) | D: Hybrid |
|---|---|---|---|---|
| **Accuracy (F1)** | ~0.34 | ~0.40 (general) | **~0.85** | ~0.50-0.70 (est.) |
| **Latency** | **~1ms** | ~50-500ms | ~200-1000ms | ~2-5ms (Layer 1-4) |
| **GPU required** | Không | Optional (CPU ok) | **Bắt buộc** | Không (Layer 1-4) |
| **Ops complexity** | Thấp | Trung bình | **Cao** | Thấp-TB |
| **Domain adaptability** | Thấp | Cần fine-tune | Tốt | **Tốt** (Layer 4) |
| **Overcorrection risk** | Thấp | Trung bình | **Cao** | Thấp |
| **Cost (infra/tháng)** | **~$0** | ~$50-100 | ~$200-500 | **~$0** (no GPU) |
| **Implementation effort** | 1-2 ngày | 2-3 ngày | 3-5 ngày + GPU setup | 3-5 ngày |
| **pycorrector support** | Có | Có | Có | Partial (Layer 1-3 custom) |
| **Incremental adoption** | Không | Không | Không | **Có** |

---

## 5. Recommendation

**Option D: Hybrid Pipeline** — Confidence: **cao**

Lý do:
1. **Learning app ≠ general OCR correction.** App biết vocabulary DB, HSK level, user's progress → domain constraint (Layer 4) là advantage lớn mà không model nào có
2. **Overcorrection là risk #1** cho learning app — LLM/BERT có thể "sửa" chữ hiếm (HSK 5-6) thành chữ phổ biến (HSK 1-2). Hybrid cho phép control chặt hơn
3. **MVP không cần GPU** — Layer 1-4 chạy hoàn toàn trên CPU, fit vào Python sidecar hiện tại
4. **DISC formula** (ACL 2025) proven effective, plug-and-play, không cần train
5. **Incremental** — ship basic context-awareness (KenLM + DISC) trước, thêm MacBERT/LLM sau khi có user data

**Trade-off chấp nhận:** Accuracy Layer 1-4 (~0.50-0.70) thấp hơn LLM (~0.85). Nhưng cho learning app, **suggest đúng chữ trong top-3** quan trọng hơn **auto-correct đúng ngay**. User vẫn phải confirm — chỉ cần candidate list tốt.

---

## 6. Implementation Sketch

### Phase 1 (MVP) — Layer 1-3

```
OCR result (per-char confidence)
    │
    ▼
[Confidence < threshold?] ──No──▶ Accept character
    │ Yes
    ▼
[Layer 1: Static Confusion Set]
    Generate candidates từ 形近字 + phonetic similar
    │
    ▼
[Layer 2-3: KenLM + DISC Re-rank]
    Score = P_ngram(candidate|context) + 1.1 × (0.7 × PinyinSim + 0.3 × GlyphSim)
    │
    ▼
Return top-3 candidates với scores
```

**Implementation trong Python sidecar:**

| Component | Library/Data | Size | Notes |
|---|---|---|---|
| Confusion set | SIGHAN confusion set (~19K pairs) + similar_chinese_characters + makemeahanzi | ~2MB | Build offline, load at startup |
| KenLM model | Train character trigram trên zhwiki + CC-100 zh | ~100-200MB | One-time training |
| Pinyin lookup | pypinyin library | ~5MB | Standard Chinese pinyin |
| Glyph similarity | Four-corner code + stroke sequence (from Unihan DB) | ~3MB | Pre-computed lookup table |
| DISC scoring | Custom function (10 lines) | 0 | Formula implementation |

**Estimated total: ~210MB RAM, ~2-5ms per character**

### Phase 2 — Layer 4 (Domain Constraint)

```
[Top-3 from Layer 1-3]
    │
    ▼
[Filter/boost against vocabulary DB]
    - Chữ trong user's learned list → boost score
    - Chữ trong current HSK level → boost
    - Chữ ngoài HSK 1-9 → penalize
    │
    ▼
Re-ranked top-3
```

Go backend query vocabulary DB, trả HSK level + learned status cho Python sidecar. Hoặc Python sidecar call Go API.

### Phase 3 (optional) — Layer 5 (MacBERT/LLM)

Chỉ khi:
- Layer 1-4 không confident (top-1 score < threshold)
- User feedback data cho thấy accuracy cần cải thiện
- Có GPU infra

Dùng pycorrector's MacBERT hoặc Qwen3-4B backend. Cache results.

---

## 7. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| KenLM context quá ngắn (trigram = 3 chars) | Miss long-range dependencies, ranking sai | DISC formula bù bằng phonetic + glyph similarity. Phase 2 thêm MacBERT nếu cần |
| Confusion set thiếu pairs | Candidate đúng không có trong list | Merge nhiều sources (SIGHAN + makemeahanzi + Wiktionary). Log user edits → bổ sung |
| Overcorrection (Phase 3 LLM) | Sửa chữ hiếm nhưng đúng thành chữ phổ biến | Dùng DISC's similarity constraint. Temperature 0.0. Chỉ suggest, không auto-correct |
| KenLM training data bias | Model bias toward simplified Chinese, miss traditional/rare chars | Train trên mixed corpus. Filter training data cho HSK vocabulary coverage |
| Latency budget (p50 < 1.5s total) | OCR + post-processing phải fit budget | Layer 1-4 chỉ ~2-5ms. Không ảnh hưởng total latency |

---

## 8. Open Questions

1. **KenLM corpus**: Dùng zhwiki đủ chưa hay cần thêm textbook corpus (HSK-aligned)?
2. **DISC alpha tuning**: Paper dùng alpha=1.1, cần benchmark trên OCR output thực tế của Lu
3. **User feedback loop**: Khi user sửa candidate → log lại → dùng để improve confusion set + retrain LM?
4. **Pinyin input**: Nếu user biết pinyin của chữ cần tìm → filter candidates theo pinyin match? (UX question)
5. **PaddleOCR per-line confidence**: Có cách nào extract per-character confidence từ PaddleOCR internal layers không? (giảm dependency vào cloud APIs)

---

## 9. Key Concepts

### DISC Scoring Formula (ACL 2025)

```
Score(x, i, y) = P(y | x, i) + α × Sim(xi, y)

Sim(c1, c2) = 0.7 × SimP(c1, c2) + 0.3 × SimG(c1, c2)

SimP(c1, c2) = 1 - LevenshteinDistance(pinyin(c1), pinyin(c2)) / len(pinyin(c1) + pinyin(c2))

SimG(c1, c2) = mean(
    FourCornerCodeMatch(c1, c2),
    StructureFourCornerMatch(c1, c2),
    1 - StrokeEditDistance(c1, c2) / max(len(strokes(c1)), len(strokes(c2))),
    StrokeLCS(c1, c2) / max(len(strokes(c1)), len(strokes(c2)))
)
```

- α = 1.1 (from paper, tunable)
- Phonetic weighted 0.7 vì Chinese errors thường phonetic-based
- Cho OCR (visual errors), có thể điều ngược: glyph 0.7, phonetic 0.3

### pycorrector

Library Python hỗ trợ nhiều backend:

```python
# KenLM (fast)
from pycorrector import Corrector
m = Corrector()
result = m.correct('少先队员因该为老人让坐')

# MacBERT (accurate)
from pycorrector import MacBertCorrector
m = MacBertCorrector("shibing624/macbert4csc-base-chinese")
result = m.correct('少先队员因该为老人让坐')
```

---

## 10. References

- [DISC: Decoding Intervention with Similarity of Characters (ACL 2025)](https://arxiv.org/html/2412.12863) — Plug-and-play scoring formula
- [Chinese Spelling Correction: Comprehensive Survey](https://arxiv.org/html/2502.11508v1) — 25+ models catalogued
- [Multimodal Features Impact on CSC](https://arxiv.org/html/2504.07661) — Phonetic + glyph encoding importance
- [FeOCR: Domain-Adaptive Chinese OCR](https://www.mdpi.com/2079-9292/15/6/1144) — Production LLM-based OCR correction
- [pycorrector GitHub](https://github.com/shibing624/pycorrector) — Multi-backend CSC toolkit
- [ChineseErrorCorrector3-4B (SOTA)](https://arxiv.org/html/2511.17562) — Qwen3-4B fine-tuned
- [Training-Free CSC with LLMs (EMNLP 2024)](https://arxiv.org/abs/2410.04027) — LLM as conventional LM
- [KenLM](https://github.com/kpu/kenlm) — Efficient n-gram LM toolkit
- [SpellGCN (ACL 2020)](https://aclanthology.org/2020.acl-main.81/) — GCN encoding similarity graphs
- [ReaLiSe (ACL 2021)](https://arxiv.org/abs/2105.12306) — Multimodal BERT for CSC
- [Survey of Post-OCR Processing (ACM)](https://dl.acm.org/doi/fullHtml/10.1145/3453476) — General OCR post-processing survey
- [SIGHAN Confusion Sets](https://github.com/zzboy/chinese/blob/master/%E5%BD%A2%E8%BF%91%E5%AD%97.txt) — 形近字 dataset
- [Radical Similarity for OCR Post-Correction](https://link.springer.com/chapter/10.1007/978-3-031-70533-5_10) — IDS-based similarity
