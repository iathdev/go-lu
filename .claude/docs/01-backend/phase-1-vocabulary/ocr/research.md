# OCR Research — Engine Selection & Post-processing

> Gộp từ: `research_ocr_engine.md`, `research_ocr_postprocessing.md`

---

## 1. Yêu cầu

- Printed ≥ 90%, handwritten Chinese ≥ 80% accuracy
- Latency: p50 < 1.5s, p99 < 3s
- Confidence < 80% → "Did you mean X?" + top-3 candidates
- Confidence < 70% → show top-3, user phải chọn
- Lọc chỉ lấy Hán tự từ mixed content (CN + VN + EN)

**NFR:**

| Tiêu chí | Target | Ghi chú |
|---|---|---|
| Availability | 99.5% | Dual-engine tăng availability |
| Latency | p50 < 1.5s, p99 < 3s | Cascading p99 có thể lên 5-6s — chấp nhận |
| Scalability | MVP 1K → target 500K req/ngày | Cloud APIs managed → scale theo demand |
| Durability | Ảnh gốc KHÔNG lưu server | Chỉ flashcards persist |

---

## 2. So sánh engine

| Factor | Google Cloud Vision | Baidu OCR API | PaddleOCR (self-hosted) | Tesseract |
|---|---|---|---|---|
| Printed Chinese | Cao | Rất cao | Rất cao (cùng PP-OCRv5) | Trung bình |
| Handwritten Chinese | Trung bình | **Tốt nhất** | **Tốt nhất** (cùng model) | Gần 0 |
| Confidence granularity | **Per-symbol** (native) | **Per-character** (cần param) | **Per-line only** | Per-word |
| Pricing | $1.50/1K (free 1K/tháng) | ~5K free/tháng (CNY) | Free (ops cost) | Free (ops cost) |
| Go SDK | Official | Không — REST only | Không — Python sidecar | cgo wrapper |
| Ops overhead | Zero | Zero | Cao | Trung bình |

> **Per-character confidence là yếu tố quyết định.** PaddleOCR accuracy ngang Baidu nhưng chỉ trả per-line confidence → classify confirmed/low_confidence sai. Google Vision/Baidu trả per-character → classify chính xác. Production phải dùng cloud APIs.

### Quyết định đã chốt

| Loại | Engine | Lý do |
|---|---|---|
| Printed (mọi ngôn ngữ) | **Google Cloud Vision** | Official Go SDK, zero ops, accuracy cao |
| Handwritten Chinese | **Baidu OCR API** | PP-OCRv5 vượt GPT-4o cho handwritten zh |
| Handwritten khác | **Google Cloud Vision** | Baidu tối ưu cho Chinese, accuracy ngôn ngữ khác không đảm bảo |
| Classification | **User-specified + Cascading** | Tiết kiệm cost (~1.1-1.2x), user biết content |
| Dev/fallback | **PaddleOCR** (Python sidecar) | Free, cùng model, evaluate trước khi commit cloud |
| Last fallback (printed) | **Tesseract** | Qua Python sidecar, chỉ printed |

### Chi phí crossover

| Quy mô | Google Vision | Baidu | PaddleOCR |
|---|---|---|---|
| 1K req/ngày | **$43** | ~$39 | ~$124 |
| 10K req/ngày | $449 | **~$253** | ~$384 |
| 100K req/ngày | $4,499 | ~$1,826 | **~$768-$1,152** |
| 500K req/ngày | $13,499 | ~$7,917 | **~$3,072-$3,840** |

- < 10K/ngày: cloud APIs rẻ hơn self-hosted
- \> 50K/ngày: PaddleOCR self-hosted rẻ nhất
- Tesseract đắt hơn PaddleOCR ở mọi quy mô + accuracy kém → không chọn

### Apps tương tự

- Không app Chinese learning nào dùng Google Cloud Vision — đa số on-device hoặc self-developed
- Công ty TQ lớn đều tự build (Baidu → PaddleOCR open-source, Tencent, NetEase, iFlytek)
- **Youdao** (NetEase) là reference tốt nhất: self-developed, offline + cloud, 97% printed
- **PaddleOCR** mạnh nhất open-source: 3.5MB mobile, offline, 100+ ngôn ngữ

---

## 3. Confidence scoring

### Engine trả gì

| | Google Cloud Vision | Baidu OCR | PaddleOCR |
|---|---|---|---|
| Range | 0.0 - 1.0 | 0.0 - 1.0 | 0.0 - 1.0 |
| Per-character | Có (Symbol level) | Có (`recognize_granularity=small`) | **Không — per-line only** |
| Comparable cross-engine? | **Không.** Model khác, calibration khác. Google 0.85 ≠ Baidu 0.85 |

### Engine-specific thresholds (MVP)

| Engine | High (confirmed) | Medium (suggest) | Low (manual pick) |
|---|---|---|---|
| Google Vision | ≥ 0.90 | 0.75 – 0.90 | < 0.75 |
| Baidu OCR | ≥ 0.85 | 0.70 – 0.85 | < 0.70 |

Baidu threshold thấp hơn vì handwriting inherently có confidence thấp hơn.

**Tuning strategy:** MVP hardcode → log character + confidence + user edit → sau 2 tuần phân tích false positive/negative → adjust.

---

## 4. Mixed language filtering

CJK characters filter bằng `unicode.Is(unicode.Han, r)` trong Go — cover tất cả CJK blocks. MVP chỉ cần core block U+4E00–U+9FFF (đủ cho toàn bộ HSK 1-9).

Han Unification: CN/JP/KR/VN share codepoints → không phân biệt được, nhưng không cần — tất cả CJK đều hợp lệ cho flashcard.

---

## 5. Candidate generation (chữ giống nhau)

OCR engine chỉ trả 1 kết quả per character, không trả alternatives → cần server-side candidate generation.

**Chọn: Pre-built lookup table (MVP)**

Offline build `map[rune][]SimilarChar` từ:
- `similar_chinese_characters` CSV (形近字 pairs)
- `makemeahanzi` NDJSON (chars sharing ≥ 2 components)
- Wiktionary confusables

~5K chars × 5 candidates = ~300KB RAM. O(1) lookup. Zero latency.

Ranking: visual similarity → character frequency → HSK level. Context-based ranking (bigram) là Phase 2.

---

## 6. Auto-suggest pinyin + meaning

**CC-CEDICT dictionary**: ~120K entries, CC BY-SA 4.0. Build hashmap at startup (~20-30MB RAM, ~50ms load).

**Polysemy** (chữ có nhiều nghĩa): Longest-match word segmentation + HSK priority.
VD: "打电话" match → meaning = "to make a phone call" (unambiguous), không show 20+ meanings của "打".

---

## 7. Post-processing pipeline

```
OCR Raw Output
  → [1] Normalize (struct format + Unicode NFC)
  → [2] Filter CJK (unicode.Han)
  → [3] Deduplicate (giữ confidence cao nhất)
  → [4] Word segmentation + Enrich (CC-CEDICT longest-match → pinyin, meaning, HSK)
  → [5] Classify by confidence (confirmed / suggest / low_confidence)
  → [6] Generate candidates (confusable_map cho medium + low)
  → [7] Match with DB (FindByHanziList → new vs existing)
```

**Latency budget:** Tổng post-processing < 25ms (trong budget 300ms). Bottleneck: DB query ~10ms.

### OCR error types

| Type | Tần suất | Xử lý |
|---|---|---|
| Substitution (wrong char) | ~70% | Confusable candidates + confidence threshold |
| Insertion (extra char) | ~20% | Low confidence + không form word → flag noise |
| Deletion (missed char) | ~10% | Khó detect server-side → user thêm thủ công |

---

## 8. Memory footprint

| Component | RAM |
|---|---|
| Confusable map | ~300KB |
| CC-CEDICT dictionary | ~20-30MB |
| **Total** | **~30MB** (load 1 lần at startup) |

---

## References

| Source | Relevance |
|---|---|
| [Google Cloud Vision — Text detection](https://docs.cloud.google.com/vision/docs/fulltext-annotations) | Confidence per Symbol |
| [Baidu OCR API](https://intl.cloud.baidu.com/en/doc/BOS/s/akce62nbw-intl-en) | Response format, probability |
| [Make Me a Hanzi](https://github.com/skishore/makemeahanzi) | Component similarity |
| [similar_chinese_characters](https://github.com/kris2808/similar_chinese_characters) | Pre-built 形近字 dataset |
| [CC-CEDICT](https://cc-cedict.org/wiki/) | Dictionary, CC BY-SA 4.0 |
| [CJK Unified Ideographs](https://en.wikipedia.org/wiki/CJK_Unified_Ideographs) | Unicode block ranges |
