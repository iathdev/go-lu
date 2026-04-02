# On-Device OCR Deployment — Research

> Khả thi chạy PaddleOCR trực tiếp trên mobile/web để giảm server cost, giảm latency, và hỗ trợ offline cho app học tiếng Trung.

---

## 1. Problem Statement

Hiện tại Lu plan dùng server-side OCR (PaddleOCR Python sidecar + Google Vision fallback). Mỗi request phải upload ảnh lên server → xử lý → trả kết quả. Vấn đề:

- **Latency**: Upload ảnh (300-600KB base64) + server processing + response = p50 ~1-2s
- **Cost**: Server compute cho OCR (CPU/GPU) scale theo số request
- **Offline**: Không dùng được khi mất mạng
- **Privacy**: Ảnh user transit qua server

On-device OCR giải quyết tất cả: zero upload, zero server cost, offline capable, ảnh không rời device.

**Nếu không làm:** App vẫn hoạt động nhưng phụ thuộc server hoàn toàn, cost tăng theo user base.

---

## 2. Current State

- Backend: Go + Python sidecar (PaddleOCR) trên K8s
- Mobile app: chưa có (planned)
- Web app: chưa có (planned)
- PaddleOCR PP-OCRv5 đã chọn làm primary engine (xem `research.md`)

---

## 3. Model Sizes

### PP-OCR Mobile Models (Paddle native format)

| Version | Detection | Recognition | Classifier | **Total** | Accuracy (rec) |
|---|---|---|---|---|---|
| **PP-OCRv5** | 4.7 MB | 16 MB | — | **~21 MB** | 81.29% |
| **PP-OCRv4** | 4.7 MB | 10.5 MB | ~2 MB | **~17 MB** | 78.74% |
| **PP-OCRv3** | 2.1 MB | 10.3 MB | 2.1 MB | **~14.5 MB** | 72.96% |
| **Ultra-lite** | — | — | — | **3.5 MB** | — |

### ONNX Models (HuggingFace — monkt/paddleocr-onnx)

| Model | ONNX Size | Ghi chú |
|---|---|---|
| PP-OCRv5 Detection | 84 MB | Server variant, chưa optimize cho mobile |
| PP-OCRv5 Chinese Rec | 81 MB | Server variant |
| PP-OCRv3 Detection | 2.3 MB | Mobile-friendly |
| PP-OCRv5 English Rec | 7.5 MB | — |

> **Quan trọng:** ONNX v5 trên HuggingFace là **server models** (~165 MB total). Để dùng trên mobile cần convert PP-OCRv5 **mobile** models (~21 MB) sang ONNX qua `paddle2onnx`. Hoặc dùng PP-OCRv3/v4 ONNX (~10-17 MB).

### PP-OCRv5 Mobile Accuracy Improvement vs v4

| Metric | v4 Mobile | v5 Mobile | Delta |
|---|---|---|---|
| Detection Hmean | 0.624 | 0.770 | **+14.6 pts** |
| Recognition avg | 0.530 | 0.802 | **+27.1 pts** |
| Printed Chinese | — | 0.861 | — |
| Handwritten Chinese | 0.298 | 0.417 | **+11.9 pts** |
| Traditional Chinese | — | 0.720 | — |
| Japanese | — | 0.758 | — |

> v5 mobile cải thiện rất lớn so với v4, đặc biệt recognition (+27 pts). Trade-off: model lớn hơn 4 MB.

Ref: [PaddleX OCR Benchmarks](https://paddlepaddle.github.io/PaddleX/3.3/en/pipeline_usage/tutorials/ocr_pipelines/OCR.html), [PP-OCRv5 Introduction](http://www.paddleocr.ai/main/en/version3.x/algorithm/PP-OCRv5/PP-OCRv5.html)

---

## 4. Deployment Options

### Option A: Google ML Kit v2 (Native SDK)

- **What:** Google's on-device text recognition SDK, hỗ trợ Chinese qua v2 API. Managed hoàn toàn.
- **How it works:** SDK bundled trong app (hoặc download qua Play Services). Gọi API → trả text + confidence + bounding box.
- **Pros:**
  - Integration cực đơn giản (1 dependency)
  - Latency tốt: **~50ms** (iPhone 12), ~140ms avg
  - 6x nhanh hơn Apple Vision
  - Free, fully offline
  - Flutter plugin chính thức (`google_mlkit_text_recognition`)
  - Memory optimized: ~11 MB runtime
  - Model size: ~8-15 MB (tự quản lý download)
- **Cons:**
  - **Closed-source** — không fine-tune, không customize
  - Chinese accuracy thấp hơn PaddleOCR (không specialized cho CJK)
  - Không hỗ trợ handwritten Chinese
  - Không có per-character confidence (chỉ per-block/per-line)
  - Phụ thuộc Google Play Services (vấn đề ở China mainland)
- **Best when:** MVP nhanh, printed Chinese chủ yếu, không cần handwriting

Ref: [Google ML Kit v2](https://developers.google.com/ml-kit/vision/text-recognition/v2), [Bitfactory OCR Comparison](https://www.bitfactory.io/de/dev-blog/comparing-on-device-ocr-frameworks-apple-vision-and-google-mlkit/)

### Option B: PaddleOCR via ONNX Runtime (Cross-platform)

- **What:** Convert PaddleOCR models sang ONNX, chạy inference bằng ONNX Runtime trên iOS/Android/Web.
- **How it works:**
  1. Convert Paddle mobile models → ONNX qua `paddle2onnx`
  2. Bundle ONNX models trong app (~17-21 MB)
  3. ONNX Runtime inference (CoreML EP trên iOS, NNAPI EP trên Android)
  4. Custom pre/post-processing (image resize, normalize, CTC decode)
- **Pros:**
  - **Best Chinese accuracy** (86% printed, 81% handwritten — PP-OCRv5)
  - Cross-platform: iOS + Android + Web (cùng model)
  - Hardware acceleration: CoreML (Apple Neural Engine), NNAPI (Qualcomm DSP)
  - Model size nhỏ (~17-21 MB PP-OCRv5 mobile)
  - Có thể fine-tune model cho domain cụ thể
  - Lấy được per-character confidence (patch CTC decode)
  - RapidOCR wrapper đã có iOS + Android implementation
- **Cons:**
  - Integration effort cao — tự implement pre/post-processing
  - Phải tự quản lý model versioning, update
  - ONNX v5 mobile chưa có pre-converted models trên HuggingFace (cần tự convert)
  - Debug khó hơn ML Kit
- **Best when:** Cần accuracy cao nhất, handwritten support, customizable

Ref: [RapidOCR GitHub](https://github.com/RapidAI/RapidOCR), [ONNX Runtime Mobile](https://onnxruntime.ai/docs/tutorials/mobile/), [monkt/paddleocr-onnx](https://huggingface.co/monkt/paddleocr-onnx)

### Option C: PaddleOCR via Paddle-Lite (Native Android)

- **What:** PaddleOCR chạy native qua Paddle-Lite inference engine trên Android.
- **How it works:** Convert Paddle models → `.nb` format → Paddle-Lite runtime trên ARM64.
- **Pros:**
  - Official Baidu deployment path
  - Well-documented cho Android (demo code sẵn)
  - Optimized cho ARM (kernel fusion, memory reuse)
- **Cons:**
  - **Android only** — iOS docs gần như không có
  - Vendor lock-in (Paddle ecosystem)
  - `.nb` format không portable
  - Community nhỏ hơn ONNX Runtime
- **Best when:** Android-only app, muốn official Baidu support

Ref: [PaddleOCR On-Device Deployment](http://www.paddleocr.ai/main/en/version3.x/deployment/on_device_deployment.html)

### Option D: Web — ONNX Runtime Web / Paddle.js

- **What:** Chạy PaddleOCR trong browser qua WebAssembly hoặc WebGL.
- **How it works:**
  - **Paddle.js**: Official Baidu, backends WebGL/WebGPU/WASM. NPM packages sẵn.
  - **paddleocr-browser**: ONNX Runtime Web + OpenCV.js. NPM package sẵn.
  - **ONNX Runtime Web**: Direct ONNX inference với WebAssembly/WebGL backend.
- **Pros:**
  - Zero install — chạy trong browser
  - Models lazy-loaded từ CDN
  - Hỗ trợ camera API (live scan)
- **Cons:**
  - Performance phụ thuộc device + browser
  - WebAssembly chậm hơn native ~2-5x
  - Model download lần đầu (cache sau)
  - Memory limited trong browser
  - Không có published latency benchmarks cho Paddle.js
- **Best when:** Web-first hoặc PWA, không muốn native app

Ref: [Paddle.js GitHub](https://github.com/PaddlePaddle/Paddle.js/), [paddleocr-browser](https://github.com/xulihang/paddleocr-browser)

### Option E: Hybrid (On-device + Server fallback)

- **What:** On-device cho printed text (fast path), server cho handwritten/low-confidence (heavy path).
- **How it works:**
  1. Ảnh → on-device OCR (ML Kit hoặc PaddleOCR ONNX)
  2. Nếu confidence cao → accept ngay (zero latency, zero cost)
  3. Nếu confidence thấp hoặc user chọn "handwritten" → gửi lên server (PaddleOCR full + context-aware post-processing)
- **Pros:**
  - **Best of both worlds**: fast cho easy cases, accurate cho hard cases
  - ~80-90% requests xử lý on-device (ước tính: phần lớn scan printed textbook)
  - Server cost giảm 80-90%
  - Offline cho printed text
  - Incremental: ship on-device trước, server fallback sẵn có
- **Cons:**
  - Phức tạp nhất — 2 pipelines, sync logic
  - Model trên device + model trên server cần coordinate
  - Edge cases: khi nào trigger fallback?
- **Best when:** Production app cần balance cost/accuracy/offline

---

## 5. Competitor Comparison

| Criteria | A: ML Kit v2 | B: PaddleOCR ONNX | C: Paddle-Lite | D: Web | E: Hybrid |
|---|---|---|---|---|---|
| **Chinese accuracy (printed)** | Tốt | **Tốt nhất (86%)** | Tốt nhất | Tốt nhất | Tốt nhất |
| **Handwritten Chinese** | Không | **Có (81%)** | Có | Có | **Có** |
| **Latency (mobile)** | **~50-140ms** | ~80-200ms | ~80ms | ~200-500ms | ~50-200ms |
| **Model size** | ~8-15 MB | ~17-21 MB | ~17-21 MB | Lazy-load | ~8-21 MB |
| **iOS support** | **Có** | **Có** (ONNX RT) | Kém | N/A | **Có** |
| **Android support** | **Có** | **Có** | **Có** | N/A | **Có** |
| **Web support** | Không | Không | Không | **Có** | Partial |
| **Offline** | **Có** | **Có** | **Có** | Có (cached) | **Có** |
| **Per-char confidence** | Không | **Có** (patch) | Có (patch) | Có (patch) | **Có** |
| **Integration effort** | **Rất thấp** | Cao | TB | TB | **Cao** |
| **Fine-tunable** | Không | **Có** | Có | Có | **Có** |
| **Cost** | Free | Free | Free | Free | Free |
| **Flutter support** | **Plugin chính thức** | Platform channel | Platform channel | N/A | Mixed |
| **React Native** | Community | @gutenye/ocr | Không | N/A | Mixed |

### So với native platform OCR

| | Google ML Kit v2 | Apple Vision | Tesseract 5 |
|---|---|---|---|
| Chinese accuracy | Tốt | TB | Kém |
| Latency | **~50ms** | ~310ms | ~220ms |
| Handwriting | Không | Không | Không |
| Traditional Chinese | Có | **Không** | Có (kém) |
| Model size | ~8-15 MB | Built-in | ~50 MB |
| Platform | iOS + Android | iOS only | Cross |

---

## 6. Recommendation

**Option E: Hybrid (ML Kit on-device + PaddleOCR server fallback)** — Confidence: **cao**

### Phased approach:

**Phase 1 (MVP):** Google ML Kit v2 on-device
- Integration 1-2 ngày
- Cover 80%+ use case (printed Chinese textbook/sách)
- Zero server cost cho OCR
- Offline capable

**Phase 2:** PaddleOCR ONNX on-device (thay thế hoặc bổ sung ML Kit)
- Khi cần handwritten Chinese, per-character confidence, fine-tuning
- PP-OCRv5 mobile models via ONNX Runtime
- ~21 MB model bundled trong app

**Phase 3:** Server fallback cho edge cases
- Handwritten khó, confidence thấp → gửi server
- Context-aware post-processing (hybrid pipeline từ `context-aware-candidates-research.md`)
- Server PaddleOCR full model + KenLM + DISC scoring

### Lý do:

1. **ML Kit trước** vì integration effort gần zero — ship nhanh, validate OCR UX
2. **PaddleOCR ONNX Phase 2** vì accuracy cao hơn + handwritten + per-char confidence — nhưng effort cao hơn
3. **Server fallback vẫn cần** cho context-aware post-processing (candidate generation, DISC scoring, vocabulary DB lookup) — chạy on-device quá phức tạp
4. App học tiếng Trung → phần lớn scan **printed textbook** → ML Kit đủ cho MVP
5. Hybrid giảm server cost ~80-90% so với server-only

### Trade-off chấp nhận:

- ML Kit accuracy thấp hơn PaddleOCR cho CJK → Phase 2 bổ sung
- ML Kit không handwritten → Phase 2 bổ sung
- Hybrid phức tạp hơn single pipeline → incremental adoption giảm risk

---

## 7. Latency Benchmarks

### Mobile ARM (real-world, từ dev.to optimization article)

| Pipeline stage | Before opt | After opt |
|---|---|---|
| Image preprocessing | 800 ms | 50 ms |
| ROI detection | — | 50 ms |
| OCR inference | 2,500 ms | 400 ms |
| **Total** | **~4,100 ms** | **~800 ms (mid-range)** |

Flagship (Snapdragon 8 Gen 3): **400-500 ms total pipeline**.

**Key optimizations:**
- Downscale ảnh xuống 1280px max (3x faster, negligible accuracy loss)
- ROI detection (chỉ process 30-40% screen → OCR 2500ms → 800ms)
- Script-specific recognizer (CJK riêng, Latin riêng)
- LRU cache cho repeated screens (~10ms hit)

### PP-OCR CPU latency (server benchmark, reference)

| Version | CPU | GPU |
|---|---|---|
| PP-OCRv5 mobile | ~79 ms | ~16 ms |
| PP-OCRv4 mobile | ~74 ms | ~15 ms |
| PP-OCRv3 mobile | ~51 ms | ~14 ms |

> Đây là server CPU (Intel Xeon), không phải mobile ARM. Mobile ARM sẽ chậm hơn ~2-4x.

Ref: [Optimizing OCR on Mobile](https://dev.to/joe_wang_6a4a3e51566e8b52/optimizing-ocr-performance-on-mobile-from-5-seconds-to-under-1-second-332m), [PaddleX Benchmarks](https://paddlepaddle.github.io/PaddleX/3.3/en/pipeline_usage/tutorials/ocr_pipelines/OCR.html)

---

## 8. Model Optimization cho Mobile

### Quantization (INT8)

- PaddleSlim QAT: FP32 → INT8, giảm ~4x model size, ~2x inference speed
- Cần retraining (online QAT recommended)
- INT8 models deploy qua Paddle-Lite `.nb` format

### PP-OCR Compression (19 strategies)

Baidu đã áp dụng 19 optimization strategies để tạo ultra-lightweight models:
- Backbone pruning
- Knowledge distillation
- Quantization-aware training
- Result: **3.5 MB total** (Chinese), **2.8 MB total** (English)

### Mobile-specific tips

- Downscale ảnh xuống 960-1280px trước inference (3x faster)
- ROI detection: crop vùng có text, không process cả ảnh
- Script detection trước: nếu biết là CJK → dùng CJK recognizer (skip Latin)
- Cache model trong memory (avoid cold start ~1-2s)

Ref: [PaddleOCR Quantization Docs](https://www.paddleocr.ai/v2.9.1/en/ppocr/model_compress/quantization.html)

---

## 9. Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| ML Kit Chinese accuracy không đủ | UX kém, user phải sửa nhiều | Monitor accuracy metrics → trigger Phase 2 (PaddleOCR ONNX) sớm |
| PP-OCRv5 ONNX mobile chưa có pre-converted | Cần tự convert, có thể gặp lỗi | Dùng PP-OCRv4 ONNX (đã có sẵn) trước, upgrade v5 sau |
| Model size ~21 MB tăng app size | User không muốn download app lớn | Lazy-load model sau install, hoặc dùng v3 (~14.5 MB) |
| ONNX Runtime cold start ~1-2s | Lần scan đầu chậm | Pre-load model khi app start, background init |
| Browser WASM chậm | Web OCR latency cao | WebGL backend cho GPU acceleration, hoặc accept server-side cho web |
| Google Play Services dependency (ML Kit) | Không chạy ở China mainland | PaddleOCR ONNX as primary cho China market |

---

## 10. Open Questions

1. **Mobile framework**: Lu dùng React Native, Flutter, hay native? Ảnh hưởng lớn đến integration path
2. **ML Kit v2 Chinese accuracy benchmark**: Cần test trực tiếp trên Chinese textbook photos để validate
3. **PP-OCRv5 mobile ONNX conversion**: Đã có ai convert thành công chưa? ONNX size sau convert = bao nhiêu?
4. **Camera real-time vs photo**: Lu cần live camera scan hay chỉ chụp ảnh → upload?
5. **China mainland users**: Có cần hỗ trợ không? ML Kit phụ thuộc Google Play Services

---

## 11. References

### PaddleOCR
- [PaddleOCR GitHub](https://github.com/PaddlePaddle/PaddleOCR)
- [PaddleOCR On-Device Deployment](http://www.paddleocr.ai/main/en/version3.x/deployment/on_device_deployment.html)
- [PP-OCRv5 Introduction](http://www.paddleocr.ai/main/en/version3.x/algorithm/PP-OCRv5/PP-OCRv5.html)
- [PaddleOCR 3.0 Technical Report](https://arxiv.org/html/2507.05595v1)
- [PaddleX OCR Pipeline](https://paddlepaddle.github.io/PaddleX/3.3/en/pipeline_usage/tutorials/ocr_pipelines/OCR.html)
- [PaddleOCR Model List](https://www.paddleocr.ai/main/en/version2.x/ppocr/model_list.html)
- [PaddleOCR Quantization](https://www.paddleocr.ai/v2.9.1/en/ppocr/model_compress/quantization.html)

### ONNX & RapidOCR
- [RapidOCR GitHub](https://github.com/RapidAI/RapidOCR) — ONNX wrapper, iOS + Android
- [monkt/paddleocr-onnx (HuggingFace)](https://huggingface.co/monkt/paddleocr-onnx) — Pre-converted ONNX models
- [Paddle2ONNX](https://paddlepaddle.github.io/PaddleOCR/main/en/version2.x/legacy/paddle2onnx.html)

### Web
- [Paddle.js GitHub](https://github.com/PaddlePaddle/Paddle.js/) — Official browser deployment
- [paddleocr-browser GitHub](https://github.com/xulihang/paddleocr-browser) — ONNX Runtime Web

### Cross-platform
- [@gutenye/ocr GitHub](https://github.com/gutenye/ocr) — React Native bridge (PP-OCRv4 + ONNX)
- [EasyOCR React Native ExecuTorch](https://swmansion.com/blog/bringing-easyocr-to-react-native-executorch-2401c09c2d0c)

### Competitors
- [Google ML Kit v2 Text Recognition](https://developers.google.com/ml-kit/vision/text-recognition/v2)
- [Apple Vision vs Google MLKit](https://www.bitfactory.io/de/dev-blog/comparing-on-device-ocr-frameworks-apple-vision-and-google-mlkit/)
- [Mobile OCR Libraries 2026](https://www.designveloper.com/blog/mobile-ocr-libraries/)

### Benchmarks & Optimization
- [Optimizing OCR on Mobile — dev.to](https://dev.to/joe_wang_6a4a3e51566e8b52/optimizing-ocr-performance-on-mobile-from-5-seconds-to-under-1-second-332m)
- [PP-OCRv5 InfoQ](https://www.infoq.com/news/2025/09/baidu-pp-ocrv5/)
- [PP-OCRv5 on HuggingFace Blog](https://huggingface.co/blog/baidu/ppocrv5)
