# Google Cloud Vision API — OCR Reference

> Tổng hợp documentation cho API đang dùng trong `google_vision_service.go`.

---

## 1. API đang dùng

**Feature**: `DOCUMENT_TEXT_DETECTION`
**Go SDK**: `cloud.google.com/go/vision/v2/apiv1`
**Method**: `BatchAnnotateImages` (synchronous)
**Protocol**: gRPC (SDK tự handle, có option REST client)
**Auth**: Service Account JSON (`GOOGLE_APPLICATION_CREDENTIALS`)

### TEXT_DETECTION vs DOCUMENT_TEXT_DETECTION

| | TEXT_DETECTION | DOCUMENT_TEXT_DETECTION (đang dùng) |
|---|---|---|
| **Use case** | Ảnh chung: biển báo, ảnh đường phố | Dense text: tài liệu, sách, chữ viết tay |
| **Response** | `textAnnotations[]` — flat list phrases + words | `fullTextAnnotation` — structured hierarchy |
| **Hierarchy** | Không có | pages > blocks > paragraphs > words > symbols |
| **Handwriting** | Hỗ trợ cơ bản | Tối ưu cho handwriting |
| **Block types** | Không phân biệt | TEXT, TABLE, PICTURE, RULER, BARCODE |
| **Pricing** | Giống nhau | Giống nhau |

**Kết luận**: `DOCUMENT_TEXT_DETECTION` đúng cho use case scan vở/sách giáo khoa — cần structured hierarchy + per-word confidence.

---

## 2. Response Structure

```
fullTextAnnotation
├── text          (string — toàn bộ text UTF-8)
└── pages[]
    ├── width, height
    ├── confidence        (float 0-1)
    ├── property
    │   └── detectedLanguages[]
    └── blocks[]
        ├── blockType     (TEXT | TABLE | PICTURE | RULER | BARCODE)
        ├── boundingBox
        ├── confidence    (float 0-1)
        ├── property
        │   └── detectedLanguages[]
        └── paragraphs[]
            ├── boundingBox
            ├── confidence  (float 0-1)
            ├── property
            │   └── detectedLanguages[]
            └── words[]
                ├── boundingBox
                ├── confidence  (float 0-1)
                ├── property
                │   ├── detectedLanguages[]   ← code đang dùng ở đây
                │   └── detectedBreak         (SPACE, LINE_BREAK, HYPHEN...)
                └── symbols[]
                    ├── text        (string — 1 UTF-8 character)
                    ├── boundingBox
                    ├── confidence  (float 0-1)
                    └── property
                        ├── detectedLanguages[]
                        └── detectedBreak
```

### Confidence Score

- Có ở **mọi level**: page, block, paragraph, word, symbol
- Range: `0.0` — `1.0`
- Code đang dùng: `word.GetConfidence()` (word level)
- Chinese: cũng lấy word level confidence (word = nhóm symbols gần nhau)

### detectedLanguages

- Có ở mọi level qua `property.detectedLanguages[]`
- Mỗi entry: `{ languageCode: "en", confidence: 0.95 }`
- Format: **BCP-47** language codes
- Code đang dùng: `word.GetProperty().GetDetectedLanguages()` → lấy top language
- Mapping trong code: `zh`, `zh-Hans`, `zh-Hant`, `zh-CN`, `zh-TW` → `"zh"`

### detectedBreak

- Cho biết break sau symbol: `SPACE`, `SURE_SPACE`, `EOL_SURE_SPACE`, `HYPHEN`, `LINE_BREAK`
- Code hiện tại chưa dùng — có thể dùng để reconstruct câu/dòng

---

## 3. Request Format

### Go SDK (đang dùng)

```go
resp, err := client.BatchAnnotateImages(ctx, &visionpb.BatchAnnotateImagesRequest{
    Requests: []*visionpb.AnnotateImageRequest{
        {
            Image: &visionpb.Image{
                Content: imageBytes,                    // []byte — raw image
                // hoặc: Source: &visionpb.ImageSource{ImageUri: "gs://bucket/image.jpg"}
            },
            Features: []*visionpb.Feature{
                {Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION},
            },
            // Optional:
            ImageContext: &visionpb.ImageContext{
                LanguageHints: []string{"zh-Hans"},     // BCP-47, thường để trống cho auto-detect
            },
        },
    },
})
```

### Image input options

| Source | Field | Ghi chú |
|---|---|---|
| Raw bytes | `Image.Content` | Đang dùng. Max 20MB (base64 ~10MB do JSON limit) |
| GCS URI | `Image.Source.ImageUri` | `gs://bucket/path`. Không qua network, nhanh hơn |
| Public URL | `Image.Source.ImageUri` | `https://...`. Google download về xử lý |

### Language Hints

```go
ImageContext: &visionpb.ImageContext{
    LanguageHints: []string{"en-t-i0-handwrit"},  // English handwriting
}
```

- Thường **để trống** — auto-detect cho kết quả tốt nhất
- Format: BCP-47 extended. VD: `"en-t-i0-handwrit"` = English transformed from handwriting
- Code hiện tại: không dùng language hints

---

## 4. Limits & Quotas

### Image Limits

| Limit | Value |
|---|---|
| Max image size | **20 MB** |
| Max JSON request | **10 MB** (base64 encoded image nặng hơn raw ~33%) |
| Max PDF size | 1 GB |
| Supported formats | JPEG, PNG, GIF, BMP, WebP, RAW, ICO, PDF, TIFF |

### Rate Limits (per project)

| Limit | Value |
|---|---|
| Requests per minute | **1,800** |
| Batch size (sync) | **16 images** per request |
| Batch size (async) | 2,000 images per request |
| Concurrent async images | 8,000 |
| Concurrent async pages | 10,000 |

### Lưu ý

- Base64 image trong JSON có thể vượt 10MB JSON limit → dùng GCS URI cho ảnh lớn
- 1,800 req/min = 30 req/s — dư cho MVP, cần monitor khi scale
- Batch 16 images/request — code đang gửi 1 image/request, có thể batch nếu cần

---

## 5. Pricing

| Tier | Units/tháng | Giá |
|---|---|---|
| Free | 0 — 1,000 | **$0** |
| Standard | 1,001 — 5,000,000 | **$1.50 / 1,000 units** |
| High volume | 5,000,001+ | **$0.60 / 1,000 units** |

- 1 unit = 1 image (bất kể image có bao nhiêu text)
- Multi-page PDF: mỗi page = 1 unit
- Multi-feature (VD: TEXT_DETECTION + LABEL_DETECTION cùng image): tính phí từng feature

### Cost estimate cho project

| Giai đoạn | Requests/tháng | Chi phí |
|---|---|---|
| MVP | ~3,000 | ~$3/tháng (2,000 units × $1.50/1K) |
| Growth | ~100,000 | ~$148.50/tháng |
| Scale | ~1,000,000 | ~$1,498.50/tháng |

---

## 6. Supported Languages

### Ngôn ngữ project đang dùng

| Ngôn ngữ | BCP-47 codes | Ghi chú |
|---|---|---|
| Chinese Simplified | `zh`, `zh-Hans` | Chữ giản thể (PRC) |
| Chinese Traditional | `zh-Hant` | Chữ phồn thể (Taiwan) |
| Vietnamese | `vi` | Đầy đủ dấu |
| English | `en` | |

### Một số ngôn ngữ khác hỗ trợ (mở rộng sau)

Japanese (`ja`), Korean (`ko`), Thai (`th`), Indonesian (`id`), French (`fr`), German (`de`), Spanish (`es`), Portuguese (`pt`), Russian (`ru`), Arabic (`ar`), Hindi (`hi`)

> Google Vision hỗ trợ 100+ ngôn ngữ. Full list: https://cloud.google.com/vision/docs/languages

---

## 7. Code mapping hiện tại

| Phần | Code location | Cách dùng API |
|---|---|---|
| **Client init** | `NewGoogleVisionService()` | `vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile(credFile))` |
| **Feature** | `Recognize()` | `Feature_DOCUMENT_TEXT_DETECTION` |
| **Image input** | `Recognize()` | `Image.Content = req.Image` (raw bytes) |
| **Response parse** | `Recognize()` | `GetFullTextAnnotation().GetPages()` → blocks → paragraphs |
| **Chinese extract** | `extractChinese()` | Word level → filter CJK symbols → concat → dedup |
| **Other lang extract** | `extractWordsWithLang()` | Word level → filter by `detectedLanguages` → dedup |
| **Language detect** | `detectWordLanguage()` | `word.Property.DetectedLanguages[0].LanguageCode` → map BCP-47 |
| **Confidence** | extract functions | `word.GetConfidence()` (0.0 — 1.0) |

---

## 8. Features chưa dùng nhưng có thể hữu ích

| Feature | Mô tả | Use case tiềm năng |
|---|---|---|
| **languageHints** | Gợi ý ngôn ngữ cho OCR engine | Tăng accuracy khi biết trước ngôn ngữ |
| **detectedBreak** | Break type sau mỗi symbol (SPACE, LINE_BREAK) | Reconstruct câu/dòng từ symbols |
| **boundingBox** | Tọa độ text trên ảnh | Highlight text trên ảnh preview cho mobile |
| **blockType** | TABLE, PICTURE, BARCODE... | Phân loại vùng ảnh, skip non-text blocks |
| **Async batch** | `AsyncBatchAnnotateImages` | Batch xử lý nhiều ảnh cùng lúc (up to 2,000) |
| **GCS input** | `ImageSource.ImageUri = "gs://..."` | Upload ảnh lên GCS trước → tránh 10MB JSON limit |
| **PDF/TIFF** | `AsyncBatchAnnotateFiles` | OCR scan toàn bộ PDF sách giáo khoa |
