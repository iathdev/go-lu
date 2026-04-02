# OCR Research — Engine Selection & Post-processing

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
| Latency | p50 < 1.5s, p99 < 3s | Cascading p99 có thể lên 5-6s |
| Scalability | MVP 1K → target 500K req/ngày | Cloud APIs managed → scale theo demand |
| Durability | Ảnh gốc KHÔNG lưu server | Chỉ flashcards persist |

---

## 2. So sánh engine

### 2.1 Accuracy

| Factor | Google Cloud Vision | Baidu OCR API | PaddleOCR (self-hosted) | Tesseract | DeepSeek-OCR-2 | GLM-OCR (0.9B) | Gemini Flash Lite |
|---|---|---|---|---|---|---|---|
| Printed Chinese | Cao (~98%) | Rất cao | Rất cao (PP-OCRv5 avg 0.84) | Trung bình | Rất cao (~97%) | Rất cao (OmniDocBench 94.62, #1) | Không rõ (không có benchmark CN riêng, user report kém hơn Flash) |
| Handwritten Chinese | Trung bình | **Tốt nhất** | **Tốt nhất** (~0.81 1-EditDist, #1 vượt GPT-4o/72B VLMs) | Gần 0 | Tốt (~90% neat, kém cursive) | Tốt (Handwritten-KIE 86.1) | Không rõ (không benchmark, hallucination risk) |

### 2.2 Performance & Limits

| Factor | Google Cloud Vision | Baidu OCR API | PaddleOCR (self-hosted) | Tesseract | DeepSeek-OCR-2 | GLM-OCR (0.9B) | Gemini Flash Lite |
|---|---|---|---|---|---|---|---|
| Latency | **~1-2s** (base64), ~0.5-1s (GCS URL) | ~1-3s (từ VN qua TQ) | ~0.6s GPU / ~1.75s CPU | ~0.2s clean text (CPU) | ~5s/page (self-hosted), <1s API | ~0.54s/page (self-hosted) | ~1-5s (LLM inference) |
| Rate limit | **1,800 req/min** | Free: **2 QPS**; Paid: 10-50 QPS | Tuỳ infra | Tuỳ infra | Không công khai, dynamic | Không công khai | Free: 15 RPM; Paid: 150-1,500 RPM |
| Free tier | 1K req/tháng | 1K req/tháng | Free (ops cost) | Free (ops cost) | Free (self-host) | Free (self-host, MIT license) | 1K req/ngày (nhưng rất hạn chế) |
| Batch support | **16 images/request** | Không | Không | Không | Không | Không | Không |
| Confidence granularity | **Per-symbol** (native) | **Per-character** (cần param) | **Per-char** (cần patch, xem 7.5) | Per-word | **Không** (generative model) | **Không** (generative model) | **Không** (generative model) |

### 2.3 Operations & Compliance

> **Giải thích tiêu chí:**
> - **Data residency**: Ảnh gửi đi được xử lý ở server nào? Ảnh hưởng latency và tuân thủ luật dữ liệu (PDPA VN/TH). Lu scan ảnh sách/chữ Hán — không phải dữ liệu nhạy cảm nên risk thấp.
> - **Privacy concern**: Mức độ rủi ro khi ảnh user đi qua server bên thứ ba. Liên quan data residency.
> - **SLA**: Cam kết uptime của provider. VD: 99.5% = tối đa ~3.6h downtime/tháng. Nếu vi phạm, provider bồi thường credit. Self-hosted thì tự chịu trách nhiệm uptime.
> - **Layout/bounding box**: Toạ độ vùng text trên ảnh (x, y, w, h). Cần khi muốn highlight/crop vùng chữ. Lu hiện chỉ cần text + confidence, chưa cần vị trí.

| Factor | Google Cloud Vision | Baidu OCR API | PaddleOCR (self-hosted) | Tesseract | DeepSeek-OCR-2 | GLM-OCR (0.9B) | Gemini Flash Lite |
|---|---|---|---|---|---|---|---|
| Data residency | **Global** (chọn region Singapore) | **China only** | **Self-hosted** | **Self-hosted** | **China** (API) / Self-hosted | **China** (API) / Self-hosted (MIT) | **Global** (Vertex AI chọn region Singapore) |
| Privacy concern | Thấp | **Cao** | Không có | Không có | **Cao** (API) / Không (self-host) | **Cao** (API, bị phạt thu thập data) / Không (self-host) | Thấp (Vertex AI) |
| SLA | **99.5%** (financial credit) | Không có | Self-managed | Self-managed | Không có | Không có | Không có |
| Go SDK | **Official** | REST only | Python sidecar | cgo wrapper | Community (OpenAI-compatible) | REST only, không Go SDK | **Official** (`google.golang.org/genai`) |
| Ops overhead | Zero | Zero | Cao (Python sidecar) | Trung bình | Cao (GPU A100 recommended) | TB (0.9B, chạy được CPU/MPS) | Zero |
| Bounding box | **Block→Para→Word→Symbol** | Line + word | Line + box | Line + word + box | **Không** | **Không** | **Không** |
| Pricing | $1.50/1K | ~5K free/tháng (CNY) | Free (ops cost) | Free (ops cost) | ~$0.0003/image (API) / Free (self-host) | Free (self-host) / ~$0.002/page (API) | ~$0.15-0.27/1K images |

### 2.4 Tóm tắt điểm mạnh/yếu

| Engine | Điểm mạnh | Điểm yếu |
|---|---|---|
| **Google Vision** | Multi-language tốt nhất, per-symbol confidence, batch 16 img, SLA 99.5%, chọn region Singapore | Đắt nhất ở scale lớn |
| **Baidu OCR** | Handwritten Chinese tốt nhất | Data qua TQ (privacy risk), QPS thấp (free 2), không SLA, không batch, latency cao từ VN |
| **PaddleOCR** | Free, accuracy cao (CN/JP), model nhỏ, tự kiểm soát data, per-char confidence (patch) | Cần maintain infra, Python sidecar |
| **Tesseract** | Free, nhanh nhất trên clean text, 100+ ngôn ngữ | Accuracy thấp nhất, yếu Vietnamese diacritics, không handwriting |
| **DeepSeek-OCR-2** | SOTA document OCR (91.09 OmniDocBench), rẻ, open-source, 25M Chinese training pages | **Không có per-char confidence**, không bounding box, hallucination risk, data qua TQ (API) |
| **GLM-OCR (0.9B)** | OmniDocBench #1 (94.62), siêu nhẹ 0.9B, MIT license, self-host dễ | **Không có per-char confidence**, không bounding box, không Go SDK, Zhipu bị phạt data collection |
| **Gemini Flash Lite** | Cực rẻ (~10x rẻ hơn Vision), Go SDK chính thức, free tier khá | **Không có per-char confidence**, hallucination nghiêm trọng, OCR regression 2.5 vs 2.0, model deprecated nhanh, không SLA |

> **Per-character confidence là yếu tố quyết định.** 3 engine mới (DeepSeek, GLM, Gemini Flash Lite) đều là VLM generative → **không có per-char confidence native**. Chỉ Google Vision, Baidu OCR, và PaddleOCR (patch) có. Đây là blocker cho UX "Did you mean X?".
>
> **Hallucination là risk mới.** VLM-based OCR có thể "sửa" chữ sai thay vì report chính xác — ngược với yêu cầu learning app cần biết user viết gì.
>
> **Data residency:** DeepSeek và GLM (cloud API) đều gửi data qua TQ. Self-host giải quyết được nhưng tăng ops overhead. Google Vision và Gemini (Vertex AI) cho chọn Singapore region.

### Recommend

| Loại | Engine | Lý do |
|---|---|---|
| Printed (mọi ngôn ngữ) | **Google Cloud Vision** | Official Go SDK, zero ops, accuracy cao, per-symbol confidence |
| Handwritten Chinese | **Baidu OCR API** | Accuracy cao nhất + per-character confidence + zero ops |
| Handwritten khác | **Google Cloud Vision** | Baidu tối ưu cho Chinese, accuracy ngôn ngữ khác không đảm bảo |
| Classification | **User-specified + Cascading** | Tiết kiệm cost (~1.1-1.2x), user biết content |
| Dev/fallback | **PaddleOCR** (Python sidecar) | Free, per-char confidence (patch), evaluate trước khi commit cloud |
| Last fallback (printed) | **Tesseract** | Qua Python sidecar, chỉ printed |

> **Tại sao không chọn DeepSeek-OCR-2, GLM-OCR, Gemini Flash Lite?**
>
> Cả 3 đều là **VLM generative** — output text tokens, không có per-character confidence hay bounding box. Đây là blocker cho core UX:
> - Confidence < 80% → "Did you mean X?" + top-3 candidates
> - Confidence < 70% → show top-3, user phải chọn
>
> Ngoài ra, VLM có **hallucination risk** — model có thể "sửa" chữ sai thay vì report chính xác. Ngược hoàn toàn với yêu cầu learning app.
>
> **Trường hợp có thể dùng (Phase 2+):** Post-processing verification — sau khi OCR truyền thống extract characters với confidence, gửi ảnh + OCR result cho VLM để context-aware correction cho low-confidence characters. Tận dụng language understanding của VLM mà không phụ thuộc vào nó cho raw extraction.

### Chi phí crossover

| Quy mô | Google Vision | Baidu | PaddleOCR | DeepSeek-OCR-2 (API) | GLM-OCR (self-host) | Gemini Flash Lite |
|---|---|---|---|---|---|---|
| 1K req/ngày | **$43** | ~$39 | ~$124 | ~$0.3 | ~$124 (GPU) | ~$5 |
| 10K req/ngày | $449 | **~$253** | ~$384 | ~$3 | ~$384 (GPU) | ~$50 |
| 100K req/ngày | $4,499 | ~$1,826 | **~$768-$1,152** | ~$30 | ~$768 (GPU) | ~$500 |
| 500K req/ngày | $13,499 | ~$7,917 | **~$3,072-$3,840** | ~$150 | ~$3,072 (GPU) | ~$2,500 |

- < 10K/ngày: cloud APIs rẻ hơn self-hosted (trừ DeepSeek API cực rẻ nhưng không có per-char confidence)
- \> 50K/ngày: PaddleOCR/GLM-OCR self-hosted rẻ nhất trong nhóm có per-char confidence
- VLM engines (DeepSeek, GLM, Gemini) rẻ hơn nhưng **thiếu per-char confidence** → không thay thế được cho core OCR flow
- Tesseract đắt hơn PaddleOCR ở mọi quy mô + accuracy kém → không chọn

### Giới hạn kích thước ảnh (URL vs Base64)

| Constraint | Google Cloud Vision | Baidu OCR API | PaddleOCR (self-hosted) |
|---|---|---|---|
| **Max file (URL)** | **20 MB** | **~10 MB** | Không giới hạn (tuỳ server config) |
| **Max file (Base64)** | **~7.3 MB raw** (10 MB JSON body limit) | **~3 MB raw** (4 MB sau encode) | Không giới hạn (tuỳ server config) |
| **Max dimensions** | 75 MP (auto-resize) | 4,096 px cạnh dài (recommend ≤ 1,024 px) | Default 960 px (configurable `limit_side_len`) |
| **Min dimensions** | 640×480 (recommend 1024×768 cho OCR) | 15 px cạnh ngắn | Không có |
| **Auto-compress** | Không | Có — ảnh > 1 MB hoặc > 1,024 px bị compress server-side | Có — resize theo `limit_side_len` |

> **Lưu ý:** Base64 encode inflate ~37% so với raw bytes. Ảnh 5 MB raw → ~6.85 MB base64. Google cho phép 10 MB JSON body → raw tối đa ~7.3 MB. Baidu giới hạn 4 MB base64 → raw tối đa ~3 MB.

### Ảnh lớn hơn có tốn thêm chi phí không?

| Engine | Tính phí theo | Ảnh lớn = đắt hơn? |
|---|---|---|
| **Google Cloud Vision** | Per-request ($1.50/1K units) | **Không.** 1 MB hay 20 MB cùng giá. Không charge per-byte |
| **Baidu OCR** | Per-call (free tier + package) | **Không.** Chỉ tính call thành công. Ảnh lớn/nhỏ cùng giá |
| **PaddleOCR** | Infrastructure cost | **Gián tiếp có.** Ảnh lớn → nhiều RAM hơn, GPU time dài hơn, latency cao hơn |

→ Cloud APIs: kích thước ảnh **không ảnh hưởng giá**, nhưng ảnh lớn tăng latency do upload/transfer.
→ Self-hosted: ảnh lớn tăng compute cost (RAM, CPU/GPU time).

### Trade-off: Gửi URL vs Base64

| Factor | URL | Base64 |
|---|---|---|
| **File size limit** | Cao hơn (20 MB Google, ~10 MB Baidu) | Thấp hơn (~7.3 MB Google, ~3 MB Baidu) |
| **Latency** | GCS URL nhanh hơn ~50% (theo Google benchmark). URL ngoài thêm 1 round-trip fetch | Payload lớn hơn 37% → upload chậm hơn |
| **Bandwidth client→server** | Thấp — chỉ gửi string URL | Cao — toàn bộ ảnh encode trong JSON body |
| **Bandwidth server→OCR** | OCR provider tự fetch ảnh từ URL | Server gửi full payload đến OCR provider |
| **Reliability** | Phụ thuộc URL accessibility. URL expire hoặc hosting down → fail | Self-contained. Không phụ thuộc bên ngoài |
| **Security** | Ảnh phải public hoặc dùng signed URL. URL có thể leak qua logs | Ảnh nằm trong HTTPS POST body — an toàn hơn cho sensitive content |
| **Infra cần thêm** | Cần image hosting (S3/GCS/CDN) + signed URL | Không cần — encode rồi gửi |
| **Phù hợp khi** | Ảnh > 4 MB, ảnh đã có trên cloud, batch processing | Ảnh < 4 MB, sensitive content, mobile upload, không có cloud storage |

**Recommendation cho project này:**
- **MVP:** Dùng **base64**. Ảnh sẽ resize server-side xuống ~300-600 KB (target 2048 px, JPEG 85%). Ở size này base64 chỉ thêm ~200 KB overhead — nằm trong limit của mọi API. Không cần setup cloud storage.
- **Scale phase:** Chuyển sang **URL mode** với signed GCS/S3 URLs nếu ảnh đã lưu trên cloud hoặc cần batch processing.

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

**Pre-built lookup table (MVP)**

Offline build `map[rune][]SimilarChar` từ:
- `similar_chinese_characters` CSV (形近字 pairs)
- `makemeahanzi` NDJSON (chars sharing ≥ 2 components)
- Wiktionary confusables

~5K chars × 5 candidates = ~300KB RAM. O(1) lookup. Zero latency.

Ranking: visual similarity → character frequency → HSK level. Context-based ranking (bigram) là Phase 2.

---

### OCR error types

| Type | Tần suất | Xử lý |
|---|---|---|
| Substitution (wrong char) | ~70% | Confusable candidates + confidence threshold |
| Insertion (extra char) | ~20% | Low confidence + không form word → flag noise |
| Deletion (missed char) | ~10% | Khó detect server-side → user thêm thủ công |


## 7. PaddleOCR — Tăng accuracy & customization

> PP-OCRv5 đã rất mạnh cho cả printed lẫn handwritten Chinese (#1 across all models, vượt GPT-4o và 72B VLMs). Per-character confidence giải quyết được qua patch postprocessor hoặc ONNX (section 7.5). Handwritten ~0.81 (1-EditDist) có thể cải thiện thêm qua preprocessing + context-aware post-processing.

### 7.1 Config tuning (zero effort)

Thay đổi config inference, không cần train lại model.

| Parameter | Mặc định (v5) | Recommend | Effect |
|---|---|---|---|
| `text_det_limit_side_len` | 64 | 960–1280 | Nhận diện chữ nhỏ tốt hơn, latency tăng |
| `use_doc_orientation_classify` | off | on | Xoay ảnh đúng hướng (99% accuracy) |
| `use_doc_unwarping` | off | on | Flatten ảnh cong/bị warp |
| `use_textline_orientation` | off | on | Xử lý text dọc/xoay |

Bật hết 3 module phụ: latency tăng từ ~0.6s → ~1.1s/ảnh (GPU).

**Ref:**
- [PP-OCRv5 Official Docs — Configuration](https://www.paddleocr.ai/main/en/version3.x/algorithm/PP-OCRv5/PP-OCRv5.html)
- [PaddleX Document Preprocessing Pipeline](https://paddlepaddle.github.io/PaddleX/3.3/en/pipeline_usage/tutorials/ocr_pipelines/doc_preprocessor.html)

### 7.2 Image preprocessing (ROI cao nhất)

Preprocessing trước khi đưa vào PaddleOCR. Có thể cải thiện 5-15% accuracy cho ảnh chụp thực tế. Implement trong Python sidecar hoặc Go (imaging library).

| Bước | Kỹ thuật | Tác dụng |
|---|---|---|
| 1 | Grayscale conversion | Loại bỏ noise màu |
| 2 | Bilateral filter | Giữ edge, giảm noise |
| 3 | CLAHE (Contrast Limited Adaptive Histogram Equalization) | Cân bằng contrast cho ảnh chụp điều kiện sáng kém |
| 4 | Adaptive thresholding | Binarization, đặc biệt tốt cho handwriting |
| 5 | Deskew correction | Sửa ảnh nghiêng |

**Ref:**
- [Image Preprocessing Impact on PaddleOCR — ScienceDirect](https://www.sciencedirect.com/science/article/pii/S1877050925027383)

### 7.3 Fine-tune model (medium effort)

Khi nào cần: font đặc biệt, handwriting style cụ thể, domain text (y khoa, pháp lý).

| Yếu tố | Chi tiết |
|---|---|
| Data tối thiểu | Detection: ~500 ảnh, Recognition: ~50 ảnh |
| Tỉ lệ data | 5:1 → 10:1 (original : custom) để không quên knowledge cũ |
| Learning rate | 0.0001 (1/10 so với train from scratch) |
| Epochs | 5–10 |
| Compute | GPU recommended, ~vài giờ cho small dataset |
| Tool annotation | PPOCRLabel, 8-point bounding boxes |

**Pitfalls:**
- **Character dictionary mismatch** giữa training và inference → #1 cause accuracy tụt sau export
- **`max_text_length` sai** → accuracy plateau ở 50%
- Cold start overhead ~4.2s

**Ref:**
- [OCR Fine-Tuning: From Raw Data to Custom Model — HackerNoon](https://hackernoon.com/ocr-fine-tuning-from-raw-data-to-custom-paddle-ocr-model)
- [Fine-tuning PaddleOCR Text Recognition — tim's blog](https://timc.me/blog/finetune-paddleocr-text-recognition.html)
- [Train Your Own OCR Model — DataGet](https://dataget.ai/blogs/train-your-own-ocr-model-paddleocr/)

### 7.4 Train from scratch (high effort)

Chỉ nên khi domain rất khác (Giáp Cốt Văn, thư pháp cổ), cần model cực nhỏ cho mobile/edge, hoặc có hàng chục ngàn ảnh labeled. **Không recommend cho Lu** — PP-OCRv5 pretrained đã đủ mạnh cho CJK printed.

### 7.5 Per-character confidence từ PaddleOCR

PaddleOCR mặc định chỉ trả per-line confidence (`np.mean(conf_list)`). Có 5 cách lấy per-character:

| # | Cách | Effort | Chất lượng | Per-char? |
|---|------|--------|------------|-----------|
| 1 | Patch `rec_postprocess.py` — return `conf_list` thay vì `np.mean()` | **Easy** | Tốt (CTC greedy) | Có |
| 2 | ONNX raw logits + tự viết CTC postprocessor | Medium | Tốt (+ full distribution, top-K candidates) | Có |
| 3 | `return_word_box=True` | Easy | Không cải thiện | Chỉ bounding box, confidence vẫn averaged |
| 4 | Attention-based decoder (SAR, NRTR) thay CTC | Hard | Tốt hơn (context-aware) | Có (cần patch) |
| 5 | CTC beam search + Chinese Language Model | Hard | Tốt nhất (LM-calibrated) | Có |

**Cách 1 — Patch postprocessor (recommend cho dev/fallback):**

Trong `ppocr/postprocess/rec_postprocess.py`, method `decode()` đã tính per-char probability nhưng average lại:

```python
# Dòng gốc
result_list.append((text, np.mean(conf_list)))
# Sửa thành
result_list.append((text, conf_list))
```

Cần patch thêm downstream code (expect `float` → giờ là `list`). Maintainer LDOUBLEV confirm cách này.

**Cách 2 — ONNX raw logits (recommend nếu dùng PaddleOCR làm primary):**

Export model sang ONNX → chạy inference → lấy raw tensor `(1, time_steps, num_classes)` → tự softmax + CTC decode. Lợi thế: lấy **full probability distribution** → top-K candidates per position (phục vụ feature "Did you mean X?").

**Cách 3 — `return_word_box=True`:** Với CJK, mỗi character = 1 word → trả per-char bounding box. Nhưng confidence **vẫn averaged**. Chỉ hữu ích khi kết hợp cách 1.

**Cách 4 — Attention decoder:** Predict tuần tự từng char → confidence per-char tự nhiên, context-aware. Nhưng phải đổi model (không dùng PP-OCRv5 pretrained), chậm hơn.

**Cách 5 — Beam search + LM:** Thay greedy decode bằng beam search + Chinese n-gram LM. Confidence chính xác nhất + top-K alternatives. Chậm nhất, cần integrate `pyctcdecode`.

> **Cho Lu:** Dev/fallback → **Cách 1** (patch 1 dòng). Scale >50K req/ngày muốn primary → **Cách 2** (ONNX + top-K candidates). Production hiện tại dùng Google Vision/Baidu đã có sẵn per-character.

**Ref:**
- [GitHub Issue #5932 — Per-char confidence (maintainer confirm)](https://github.com/PaddlePaddle/PaddleOCR/issues/5932)
- [PaddleOCR ONNX export docs](https://github.com/PaddlePaddle/PaddleOCR/blob/main/deploy/paddle2onnx/readme.md)
- [RapidAI/PaddleOCRModelConvert](https://github.com/RapidAI/PaddleOCRModelConvert)
- [Discussion #15011 — return_word_box](https://github.com/PaddlePaddle/PaddleOCR/discussions/15011)
- [Discussion #11352 — Recognition confidence & attention decoder](https://github.com/PaddlePaddle/PaddleOCR/discussions/11352)
- [Paddle Issue #2230 — CTC beam search decoder](https://github.com/PaddlePaddle/Paddle/issues/2230)

### 7.6 Kết luận cho Lu

| Cách | Effort | Impact | Áp dụng khi |
|---|---|---|---|
| Config tuning | Zero | Thấp–TB | Ngay từ đầu |
| Preprocessing | Thấp | **Cao (5-15%)** | MVP — implement trong sidecar |
| Fine-tune | TB–Cao | Tuỳ data | Phase 2+ khi có labeled data từ user |
| Train scratch | Rất cao | Tuỳ domain | Không áp dụng |

> **Per-character confidence đã giải quyết được** qua patch postprocessor (Cách 1) hoặc ONNX raw logits (Cách 2) — xem section 7.5. Handwritten Chinese ~0.81 (1-EditDist) đã là #1 across all models — cải thiện thêm qua preprocessing (section 7.2) + context-aware post-processing (xem `context-aware-candidates-research.md`). PaddleOCR có thể dùng làm **primary engine**, không chỉ dev/fallback.

**Ref tổng hợp:**
- [PP-OCRv5 Technical Report — arXiv](https://arxiv.org/html/2507.05595v1)
- [PP-OCRv5 on Hugging Face](https://huggingface.co/blog/baidu/ppocrv5)
- [PaddleOCR in Production — Medium](https://medium.com/@ankitladva11/what-it-really-takes-to-use-paddleocr-in-production-systems-d63e38ded55e)
- [OCR API Benchmarks — Nanonets](https://nanonets.com/blog/identifying-the-best-ocr-api/)
- [PaddleOCR GitHub](https://github.com/PaddlePaddle/PaddleOCR)

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
| [DeepSeek-OCR-2 on HuggingFace](https://huggingface.co/deepseek-ai/DeepSeek-OCR-2) | Model card, benchmarks |
| [DeepSeek-OCR Paper — arXiv](https://arxiv.org/abs/2510.18234) | Architecture, training data 25M Chinese pages |
| [DeepSeek API Pricing](https://api-docs.deepseek.com/quick_start/pricing) | $0.30/M input tokens |
| [DeepSeek OCR Handwriting Accuracy — Skywork](https://skywork.ai/blog/llm/deepseek-ocr-for-handwriting-recognition-accuracy-test-and-tips/) | ~90% neat, kém cursive |
| [The Truth About DeepSeek OCR — Medium](https://medium.com/intelligent-document-insights/the-truth-about-deepseek-ocr-and-what-to-use-instead-b4167ddcf21d) | Benchmark-to-production gap 15-25% |
| [GLM-OCR GitHub](https://github.com/zai-org/GLM-OCR) | 0.9B model, MIT license |
| [GLM-OCR on HuggingFace](https://huggingface.co/zai-org/GLM-OCR) | Model card, OmniDocBench #1 (94.62) |
| [DeepSeek-OCR-2 vs GLM-OCR vs PaddleOCR Benchmark — Regolo](https://regolo.ai/deepseek-ocr-vs-glm-ocr-vs-paddleocr-benchmark-2026/) | Head-to-head comparison |
| [8 Top Open-Source OCR Models Compared — Modal](https://modal.com/blog/8-top-open-source-ocr-models-compared) | GLM-OCR, PaddleOCR-VL so sánh |
| [Zhipu Data Collection Violations — SCMP](https://www.scmp.com/tech/tech-trends/article/3311175/chinas-ai-tigers-zhipu-moonshot-accused-collecting-excessive-data-chatbot-apps) | Privacy concern |
| [Gemini Flash Lite OCR Forum — Google AI](https://discuss.ai.google.dev/t/ocr-gemini-2-0-flash-lite-vs-2-5-flash-lite/106599) | User reports: Lite kém hơn Flash cho OCR |
| [Gemini Developer API Pricing](https://ai.google.dev/gemini-api/docs/pricing) | $0.10/M input tokens (2.5 Flash Lite) |
| [Why OCR Quality Worse in Gemini 2.5 — Blog](https://genuineartificialintelligence.com/2025/05/30/why-is-ocr-quality-so-much-worse-in-the-2-5-thinking-gemini-models-vs-2-0-flash/) | OCR regression 2.5 vs 2.0 |
| [Gemini 2.0 Flash Lite Deprecation — Google Forum](https://discuss.ai.google.dev/t/extend-eol-for-gemini-flash-cost-effective-models/121751) | EOL June 2026 |
| [Japanese Handwriting OCR 19-Model Comparison](https://nyosegawa.github.io/posts/japanese-handwriting-ocr-comparison/) | Gemini 3.1 Flash Lite NLS 0.899 vs Pro 0.924 |
| [Best OCR Models 2026 Benchmarks — CodeSOTA](https://www.codesota.com/ocr) | Comprehensive benchmark ranking |
