# OCR Scan — Stateful Workflow

> Luồng hoạt động tương ứng với bảng `ocr_scans` trong `database_design.md` Nhóm 6.
> Flow hiện tại là stateless (scan → trả kết quả → xong). Document này mô tả flow stateful cần implement.

---

## Status Workflow

```mermaid
stateDiagram-v2
    [*] --> pending : User scan ảnh
    pending --> confirmed : User review & confirm
    pending --> cancelled : User huỷ
    pending --> expired : Quá 24h không confirm
    confirmed --> completed : Vocabulary đã tạo xong
    completed --> [*]
    cancelled --> [*]
    expired --> [*]
```

---

## Sequence Diagram — Full Flow

### Phase 1 — Scan & Persist

**Mục đích:** Nhận ảnh từ user, xử lý OCR, phân loại kết quả (từ mới / từ đã có / confidence thấp), lưu vào DB để user review sau.

```mermaid
sequenceDiagram
    actor User
    participant FE as Mobile App
    participant H as VocabularyHandler
    participant UC as OCRScanUseCase
    participant OCR as OCR Module
    participant VRepo as VocabularyRepository
    participant SRepo as OCRScanRepository
    participant DB as PostgreSQL

    User->>FE: Chụp ảnh / chọn gallery / paste URL
    FE->>H: POST /api/v1/vocabularies/ocr-scan<br/>(image file hoặc image_url)
    H->>H: Validate input (size, MIME type)

    H->>UC: CreateScan(ctx, input)
    UC->>OCR: ProcessScan(image bytes, type, language)
    OCR->>OCR: resolveEngine() → chọn engine
    OCR-->>UC: OCRScanOutput (items, low_confidence, metadata)

    UC->>VRepo: FindByWordList(ctx, languageID, words)
    VRepo-->>UC: existing vocabularies
    UC->>UC: Classify items → new / existing / low_confidence

    UC->>SRepo: Save(ctx, ocrScan)
    Note over SRepo, DB: status=pending, expires_at=NOW()+24h
    SRepo->>DB: INSERT ocr_scans
    DB-->>SRepo: OK

    UC-->>H: scan_id + classified results
    H-->>FE: 201 {scan_id, new_items, existing_items, low_confidence, metadata}
    FE-->>User: Preview screen
```

Phase này là bắt buộc. Không có nó thì không có gì để review.

---

### Phase 2 — Confirm & Create Vocabularies

**Mục đích:** User đã review xong (ở client), gửi danh sách items cuối cùng, server tạo vocabulary từ đó.

```mermaid
sequenceDiagram
    actor User
    participant FE as Mobile App
    participant H as VocabularyHandler
    participant UC as OCRScanUseCase
    participant VRepo as VocabularyRepository
    participant SRepo as OCRScanRepository
    participant DB as PostgreSQL

    User->>FE: Review xong, tap "Confirm" + chọn folder
    FE->>H: POST /api/v1/ocr-scans/:scan_id/confirm<br/>{folder_id?, items: [{text, pronunciation, action}]}
    H->>UC: ConfirmScan(ctx, scanID, userID, req)
    UC->>SRepo: FindByID(ctx, scanID)
    SRepo-->>UC: ocrScan (status=pending)
    UC->>UC: Validate ownership + status=pending
    UC->>UC: Apply client edits to results JSONB
    UC->>UC: Filter confirmed/edited items (skip deleted)

    loop Mỗi confirmed item
        UC->>VRepo: Create vocabulary + meanings
    end

    opt folderID provided
        UC->>VRepo: AddToFolder(folderID, vocabularyIDs)
    end

    UC->>SRepo: Update results, confirmed_count,<br/>status=completed, confirmed_at=NOW()

    UC-->>H: created vocabularies count
    H-->>FE: 200 {confirmed_count, folder_id}
    FE-->>User: "X từ đã được thêm vào folder"
```

Client gửi kèm danh sách items đã review (edited/confirmed/deleted) trong confirm request. Server apply edits + tạo vocabulary trong 1 bước.

---

## Sequence Diagram — Cancel & Expire

```mermaid
sequenceDiagram
    actor User
    participant FE as Mobile App
    participant H as VocabularyHandler
    participant UC as OCRScanUseCase
    participant SRepo as OCRScanRepository

    Note over User, SRepo: Cancel — User chủ động huỷ

    User->>FE: Tap "Huỷ" / Back
    FE->>H: DELETE /api/v1/ocr-scans/:scan_id
    H->>UC: CancelScan(ctx, scanID, userID)
    UC->>SRepo: FindByID(ctx, scanID)
    UC->>UC: Validate ownership + status=pending
    UC->>SRepo: Update status=cancelled
    UC-->>H: OK
    H-->>FE: 200

    Note over User, SRepo: Expire — Background job

    participant Job as Cleanup Job
    Job->>SRepo: FindExpired(ctx, NOW())
    Note over SRepo: WHERE status='pending' AND expires_at < NOW()
    SRepo-->>Job: expired scans
    loop Mỗi expired scan
        Job->>SRepo: Update status=expired
    end
```

---

## API Endpoints (cần implement)

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| `POST` | `/api/v1/vocabularies/ocr-scan` | Phase 1: Scan ảnh → lưu pending → trả results |
| `GET` | `/api/v1/ocr-scans/:id` | Lấy scan + results (re-open preview nếu app bị tắt) |
| `POST` | `/api/v1/ocr-scans/:id/confirm` | Phase 2: Client gửi items đã review → tạo vocabularies |
| `DELETE` | `/api/v1/ocr-scans/:id` | Cancel scan |
| `GET` | `/api/v1/ocr-scans` | List scan history (user_id, paginated) |

---

## Input Methods

```mermaid
flowchart LR
    A[User] --> B{Input method}
    B -->|Chụp ảnh / Gallery| C[File upload<br/>multipart/form-data]
    B -->|Paste link ảnh| D[Image URL<br/>application/json]
    C --> E[Handler validate<br/>size + MIME]
    D --> F[Handler fetch<br/>image from URL]
    E --> G[OCR Engine]
    F --> G
    G --> H[Save to ocr_scans<br/>status=pending]
```

| Input | `image_url` trong DB |
|-------|---------------------|
| File upload | S3 URL (nếu đã tích hợp S3) hoặc NULL |
| Image URL | URL gốc user cung cấp |
