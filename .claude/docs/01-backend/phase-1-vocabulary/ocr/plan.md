# OCR Plan — System Design & Integration

> Gộp từ: `plan_ocr_engine.md`, `plan_google_vision_integration.md`, `plan_baidu_ocr_integration.md`, `plan_paddleocr_integration.md`, `plan_tesseract_integration.md`, `plan_ocr_module_separation.md`

---

## 1. Tổng quan

User chụp ảnh (vở ghi chép, sách giáo khoa) → OCR nhận diện Hán tự → phân loại từ mới/từ đã có → user confirm → tạo flashcards.

**Input/Output:**
```
Input:  { image: multipart, type: "printed"|"handwritten"|"auto", language: "zh"|"vi"|"en" }
Output: { new_items[], existing_items[], low_confidence_items[], metadata }
```

---

## 2. Ước lượng quy mô

| Giai đoạn | MAU | Req/ngày | QPS peak (3x) |
|---|---|---|---|
| MVP (tháng 1-3) | 50-200 | 100-600 | ~0.03 |
| Growth (tháng 4-12) | 1K-10K | 3K-50K | ~2-10 |
| Scale (năm 2) | 10K-50K | 50K-500K | ~10-100 |

- ~30% requests là handwritten Chinese → Baidu
- OCR pipeline gần như không tạo storage (ảnh không lưu, flashcards dùng vocabulary table)
- Spike: 18h-22h homework time 3-5x, trước kỳ thi 10x

---

## 3. API Contract

```
POST /api/vocabularies/ocr-scan
Content-Type: multipart/form-data
Headers: X-Idempotency-Key: {uuid}

Fields:
  image: file (JPEG/PNG, max 5MB)
  type: "printed" | "handwritten" | "auto"
  language: "zh" | "vi" | "en"
```

Response `200`:
```json
{
  "data": {
    "new_items": [{ "hanzi": "新词", "confidence": 0.92, "candidates": [] }],
    "existing_items": [{ "id": "uuid", "hanzi": "你好", "pinyin": "nǐ hǎo", "confidence": 0.98 }],
    "low_confidence_items": [{ "hanzi": "鑫", "confidence": 0.65, "candidates": ["鑫", "森", "淼"] }],
    "metadata": { "engine_used": "google_vision", "total_detected": 15, "processing_time_ms": 1234 }
  }
}
```

---

## 4. Engine routing

| Request | Engine chain |
|---|---|
| `printed` (any lang) | Google Vision → PaddleOCR → Tesseract |
| `handwritten + zh` | Baidu → PaddleOCR → Google Vision |
| `handwritten + other` | Google Vision |
| `auto + zh` | Google Vision → cascade Baidu (nếu avg conf < 75%) |
| `auto + other` | Google Vision → PaddleOCR → Tesseract |

Tesseract **không bao giờ** dùng cho handwritten (accuracy gần 0).

---

## 5. Architecture

### Data flow

```
Mobile → [Middleware: Auth + RateLimit] → OCR Handler → OCRCommand (Use Case)
                                                              │
                                          ┌───────────────────┼──────────────┐
                                          ▼                   ▼              ▼
                                   OCRServicePort    VocabularyRepoPort   Redis
                                    (Output Port)                      (idempotency,
                                     │       │                          Baidu token)
                              ┌──────┘       └──────┐
                              ▼                     ▼
                    GoogleVisionAdapter      BaiduOCRAdapter
                    (Go SDK, gRPC)          (REST API, HTTP)
                              │                     │
                              ▼                     ▼
                    Google Cloud Vision      Baidu OCR API
```

PaddleOCR + Tesseract qua Python sidecar (`scripts/ocr-service/`), cùng implement `OCRServicePort`.

### Port interface

```go
type OCRServicePort interface {
    Recognize(ctx context.Context, req OCRRequest) (*OCRResult, error)
}

type OCREngineRegistry map[OCREngineKey]OCRServicePort
```

Mỗi engine là 1 adapter implement cùng interface. DI container đăng ký vào registry. Use case `resolveEngine()` chọn theo routing rules.

---

## 6. Engine integration decisions

### Google Cloud Vision

| Quyết định | Chọn | Lý do |
|---|---|---|
| SDK vs REST | Official Go SDK (`cloud.google.com/go/vision/v2`) | Type-safe, auto-retry, gRPC, connection pooling |
| Auth | Service Account JSON (mọi môi trường) | Đồng nhất dev/staging/prod |
| Feature | `DOCUMENT_TEXT_DETECTION` | Structured hierarchy, per-symbol confidence |
| Extract level | Symbol level (Chinese), Word level (other) | Mỗi symbol = 1 Hán tự |

### Baidu OCR

| Quyết định | Chọn | Lý do |
|---|---|---|
| API approach | REST API (HTTP POST) | Không có official Go SDK |
| Auth | OAuth2 (API Key + Secret → Access Token) | Cách duy nhất Baidu hỗ trợ |
| Token caching | Redis, TTL 29 ngày | Token valid 30 ngày, tránh round trip ~200-500ms |
| Per-character | `recognize_granularity=small` | Mặc định per-line, param này bật per-character |
| Confidence | ÷ 100 (0-100 → 0.0-1.0) | Đồng nhất với Google (0.0-1.0) |
| Endpoints | Handwriting (zh only) + General (printed) | Route theo type |

### PaddleOCR + Tesseract

Chạy qua Python sidecar (`scripts/ocr-service/`), 1 adapter `OCRService` dùng chung với field `engineName` khác nhau. Giao tiếp HTTP, cùng request/response format.

Tesseract cần `TESSERACT_ENABLED=true` và cài system deps (`tesseract`, `pytesseract`).

---

## 7. Cache & Storage

| Cache | Nơi | TTL | Lý do |
|---|---|---|---|
| Idempotency | Redis (`idem:{key}`) | 5 phút | Double-tap → tránh duplicate API calls |
| Baidu access token | Redis (`baidu_ocr:access_token`) | 29 ngày | Tránh round trip token mỗi request |
| OCR results | **Không cache** | — | Privacy (fingerprinting), hit rate < 1%, OCR non-deterministic |

Không cần schema DB mới — OCR stateless, flashcards dùng vocabulary table. Cần verify index `idx_vocabularies_hanzi` cho `FindByHanziList`.

---

## 8. Fault tolerance

### Circuit breaker (gobreaker v2, per-engine)

| Config | Value |
|---|---|
| MaxRequests (half-open) | 3 |
| Interval (counter reset) | 60s |
| Timeout (open → half-open) | 30s |
| ReadyToTrip | 5 consecutive failures HOẶC > 50% failure trong 10 req |

### Failure scenarios

| Scenario | Xử lý |
|---|---|
| Google down + printed | CB open → 503 |
| Google down + auto + zh | CB open → route Baidu |
| Baidu down + handwritten zh | CB open → fallback Google Vision (degraded, `engine_degraded: true`) |
| Cả 2 down | 503 "OCR service unavailable" — user import thủ công |

### Retry: 1 lần, backoff 500ms, timeout 3s per attempt. Chỉ 1 lần vì latency budget 1-3s.

---

## 9. Rate limiting & spike

| Layer | Config |
|---|---|
| Per-user per-day | Redis counter `ocr:{user_id}:{date}`. Free: 3, Pro: 50 |
| Global concurrent | Semaphore `OCR_MAX_CONCURRENT=50`. Vượt → 429 |
| Google Vision quota | 1,800 req/phút. Monitor → request increase khi đạt 70% |
| Baidu QPS | Default 10 (MVP đủ). Scale → mua package |

---

## 10. Observability

### Metrics

| Metric | Alert khi |
|---|---|
| `ocr.requests.total` (by engine, status) | Error rate > 5% / 5 phút |
| `ocr.latency.p50` / `p99` | p50 > 2s hoặc p99 > 5s |
| `ocr.circuit_breaker.state` | State = open > 5 phút → P1 |
| `ocr.cascading.fallback_rate` | > 30% → P2 |
| `ocr.cost.google_estimated` | > 80% monthly budget → P3 |

### Accuracy tracking

Implicit feedback: user edit ở preview screen → `actual_accuracy = 1 - (edits / total_detected)`. Weekly report by engine + confidence bucket.

---

## 11. OCR module separation

OCR tách ra `internal/ocr/` — chỉ lo recognize text từ ảnh, không biết vocabulary. Vocabulary module orchestrate: gọi OCR → classify new/existing.

**Multilang response:** `hanzi`/`pinyin` → `text`/`pronunciation` — FE render giống nhau cho mọi ngôn ngữ.

```
internal/ocr/
├── application/
│   ├── port/          ← OCRCommandPort, OCRServicePort, OCREngineRegistry
│   ├── dto/           ← Multilang DTOs (text, pronunciation, confidence)
│   ├── usecase/       ← recognize + enrich pronunciation + classify confidence
│   └── mapper/        ← ConvertToPinyin
├── adapter/
│   ├── handler/       ← ProcessOCRScan + downloadImage
│   └── service/       ← google_vision, baidu_ocr, ocr_service (paddle/tesseract), retry
└── module.go
```

Vocabulary module bỏ toàn bộ OCR code. Nếu mobile cần 1 endpoint trả new/existing → vocabulary import OCR exported port hoặc FE gọi 2 endpoint riêng.

---

## 12. Implementation status

| Feature | Status | Ghi chú |
|---|---|---|
| **Phase 1: PaddleOCR** | | |
| Python sidecar (`scripts/ocr-service/`) | Done | FastAPI, PaddleOCR + Tesseract, jieba segmentation |
| `OCRServicePort` interface | Done | `Recognize()` trong `ocr/application/port/outbound.go` |
| `OCREngineRegistry` + routing logic | Done | `resolveEngine()` route theo type + language |
| **Phase 2: Google Vision** | | |
| `GoogleVisionAdapter` | Done | Go SDK, `DOCUMENT_TEXT_DETECTION`, CJK extraction |
| Circuit breaker + retry decorator | Done | gobreaker per-engine, exponential backoff (3 retries, 200ms base) |
| Pinyin enrichment (use case layer) | Done | `mozillazg/go-pinyin`, chung cho mọi engine |
| **Phase 3: Baidu OCR** | | |
| `BaiduOCRAdapter` | Done | REST API, OAuth2 token caching Redis |
| Handwriting + General endpoints | Done | Route theo type |
| **Phase 4: Tesseract** | | |
| `TesseractAdapter` | Done | Qua Python sidecar, cùng interface |
| **Phase 5: Module separation** | | |
| Tách `internal/ocr/` module riêng | Done | Ports, usecase, adapters, mapper |
| Multilang response (`text`/`pronunciation`) | Done | OCR DTO + pronunciation mapper |
| OCR handler trong vocabulary module | Done | Vocabulary handler gọi OCR module qua `OCRScannerPort` bridge |
| **Phase 6: Vocabulary classify** | | |
| OCR result → `FindByHanziList()` → split `new_items` / `existing_items` | Pending | DTO đã có (`OCRScanCharacterItem`, `OCRScanExistingItem`), chưa gọi DB classify |
| Response trả `new_items` / `existing_items` / `low_confidence_items` | Pending | Hiện trả flat `items` + `low_confidence` từ OCR module, chưa check DB |
| **Phase 7: Observability & hardening** | | |
| OpenTelemetry spans cho OCR pipeline | Pending | Infra OTEL có sẵn, chưa tạo spans trong OCR code |
| Accuracy tracking (user edits → implicit accuracy) | Pending | Chưa có mechanism track user corrections |
| Custom OCR metrics (latency, fallback rate, cost) | Pending | |
| **Phase 8: Rate limiting & dedup** | | |
| Per-user per-day OCR limit (Free: 3, Pro: 50) | Pending | Chỉ có global rate limit, chưa có OCR-specific |
| Idempotency cache (client UUID) | Pending | Chưa implement `X-Idempotency-Key` + Redis cache |
| Image hash dedup (`user_id:sha256`) | Pending | Layer 2 chống duplicate ảnh |
