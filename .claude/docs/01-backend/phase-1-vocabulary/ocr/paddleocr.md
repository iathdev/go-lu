# PaddleOCR — Thách thức & Định hướng

## Vai trò hiện tại

Dev/fallback engine. Production dùng Google Vision (printed) + Baidu (handwritten zh). PaddleOCR chỉ hợp lý làm primary khi > 50K req/ngày (crossover point chi phí).

---

## Thách thức triển khai

### Ops overhead

| Thách thức | Chi tiết |
|-----------|---------|
| Python sidecar | PaddleOCR là Python library → Go backend không gọi trực tiếp. Phải chạy 1 Python service riêng (FastAPI) bên cạnh Go server |
| Deployment phức tạp | K8s cần thêm 1 container/pod cho Python sidecar. Quản lý 2 runtime (Go + Python) |
| Model size | PaddleOCR model ~150-300MB. Container image lớn, cold start chậm |
| Resource | CPU inference chậm (~2-5s/ảnh). GPU inference nhanh (~200ms) nhưng GPU trên K8s đắt |
| Scaling độc lập | Go server scale khác Python sidecar. Sidecar là bottleneck → cần autoscale riêng |
| Health check | Phải monitor 2 service. Sidecar crash → Go server vẫn healthy nhưng OCR chết |

### Per-line confidence only

Đây là hạn chế kỹ thuật lớn nhất:

```
Google Vision:  "学习好" → 学(0.95) 习(0.92) 好(0.88)  ← per-character
PaddleOCR:      "学习好" → confidence: 0.91              ← per-line only
```

- Không biết chữ nào OCR sai → không classify `low_confidence` chính xác
- User thấy "confidence 91%" nhưng 1 trong 3 chữ có thể sai hoàn toàn
- Không đủ granularity cho UX requirement: highlight từng chữ xanh/đỏ

---

## So sánh cơ chế PaddleOCR vs Google Vision

| | Google Cloud Vision | PaddleOCR |
|---|---|---|
| Runtime | Cloud API, managed | Self-hosted Python |
| Giao tiếp | Go SDK → gRPC đến Google server | HTTP POST đến Python sidecar |
| Model | Proprietary, trained trên billions of images | Open-source PP-OCRv5 |
| Detection | DOCUMENT_TEXT_DETECTION → Page → Block → Paragraph → Word → Symbol hierarchy | DB text detection → CRNN recognition → per-line |
| Output granularity | Symbol level (mỗi chữ Hán = 1 symbol + confidence + bounding box) | Line level (1 dòng text + 1 confidence) |
| Language detection | Tự detect mixed content (CN+EN+VN) | Cần chỉ định language, mixed kém hơn |
| Handwritten | Trung bình | Rất tốt (PP-OCRv5 vượt GPT-4o cho handwritten zh) |
| Latency | ~500ms-1.5s (network đến Google) | ~200ms GPU / ~2-5s CPU (local) |
| Cost | $1.50/1K requests | Infra cost (server + GPU) |
| Availability | 99.9% SLA | Tự quản lý uptime |

---

## Sequence Diagram

```mermaid
sequenceDiagram
    participant H as OCR Handler
    participant UC as OCRCommandUseCase
    participant CB as Circuit Breaker
    participant PS as Python Sidecar (FastAPI)
    participant PP as PaddleOCR Model

    H->>UC: ProcessScan(image, lang=zh)
    UC->>CB: paddleocr breaker check
    CB->>PS: POST /recognize<br/>{image: base64, lang: "zh"}

    PS->>PS: Decode base64 → resize (limit_side_len)
    PS->>PP: PaddleOCR.ocr(image)

    Note over PP: 1. DB text detection (locate text regions)<br/>2. CRNN recognition (read text)<br/>3. Return per-line results

    PP-->>PS: [<br/>  {text: "我每天学习中文", conf: 0.93},<br/>  {text: "你好世界", conf: 0.87}<br/>]

    PS->>PS: Split lines → individual characters<br/>Assign line confidence to each char
    PS-->>CB: [{text:"我",conf:0.93}, {text:"每",conf:0.93}, ...]
    CB-->>UC: OCR result

    UC->>UC: enrichPronunciation(zh) → go-pinyin
    UC->>UC: classifyByConfidence(0.70)
    UC->>UC: generateCandidates (confusable map)
    UC-->>H: OCRScanOutput

    Note over H: Per-line confidence only<br/>→ tất cả chữ cùng dòng có cùng score<br/>→ không phân biệt chữ nào sai
```

### Deployment topology

```mermaid
graph TB
    subgraph k8s["K8s Cluster"]
        subgraph go_pods["Go API Pods (autoscale)"]
            G1[Go Pod 1]
            G2[Go Pod 2]
            G3[Go Pod N]
        end

        subgraph py_pods["Python Sidecar Pods (autoscale riêng)"]
            P1[Sidecar 1<br/>PaddleOCR + Tesseract]
            P2[Sidecar 2<br/>PaddleOCR + Tesseract]
        end

        G1 -->|HTTP POST /recognize| P1
        G2 -->|HTTP POST /recognize| P1
        G3 -->|HTTP POST /recognize| P2

        SVC[K8s Service] --> P1
        SVC --> P2
    end

    G1 --> SVC
    G2 --> SVC
    G3 --> SVC

    style py_pods fill:#fff3e0,stroke:#e65100
    style go_pods fill:#e3f2fd,stroke:#1565c0
```

---

## Giao tiếp Go ↔ Python sidecar

### Hiện tại: HTTP REST

```
Go server → POST /recognize (JSON, base64 image) → Python FastAPI → PaddleOCR → response
```

Đủ dùng cho dev/fallback. Bottleneck là OCR inference (2-5s CPU), không phải serialization.

### Khi nào chuyển sang gRPC

Khi PaddleOCR thành primary engine (> 50K req/ngày):

| Yếu tố | Tại sao gRPC lúc đó hợp lý |
|---------|---------------------------|
| Bandwidth | 37% saving trên mỗi request. 50K req × 500KB = tiết kiệm ~9GB/ngày transfer |
| Multiplexing | HTTP/2 — nhiều request trên 1 connection, giảm connection overhead giữa Go pods ↔ Python pods |
| Contract | Proto file là single source of truth — Go client + Python server auto-gen, không lệch DTO |
| Health check | gRPC health check protocol chuẩn — K8s liveness/readiness probe native support |
| Deadline propagation | gRPC deadline tự truyền từ Go → Python — timeout handling nhất quán, không cần tự implement |

---

## Chi phí crossover

| Quy mô | Google Vision | PaddleOCR |
|---|---|---|
| 1K req/ngày | **$43** | ~$124 |
| 10K req/ngày | **$449** | ~$384 |
| 50K req/ngày | $2,249 | **~$768** |
| 100K req/ngày | $4,499 | **~$1,152** |
| 500K req/ngày | $13,499 | **~$3,840** |

< 50K/ngày → cloud APIs rẻ hơn. > 50K/ngày → PaddleOCR self-hosted rẻ nhất.
