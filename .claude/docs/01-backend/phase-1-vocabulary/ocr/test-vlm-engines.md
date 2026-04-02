# Test VLM OCR Engines — DeepSeek-OCR-2 & GLM-OCR

> Mục đích: đánh giá output quality của 2 VLM OCR engine trên ảnh chữ Hán (printed + handwritten) để quyết định có dùng làm post-processing verification layer ở Phase 2+ hay không.

---

## 1. DeepSeek-OCR-2

### Setup

```bash
# 1. Lấy API key (free credits khi đăng ký)
#    https://platform.deepseek.com/api_keys

# 2. Export key
export DEEPSEEK_API_KEY=sk-...
```

### Chạy test

```bash
cd scripts/ocr-service

# Test với ảnh mặc định (test_image.png)
python test_deepseek.py

# Test với ảnh cụ thể
python test_deepseek.py /path/to/image.webp
```

### Output mong đợi

Script chạy 2 mode:
- **Extract mode** — trả raw text (danh sách chữ Hán)
- **Structured mode** — trả JSON array `[{"text": "...", "type": "printed/handwritten"}]`

Quan sát:
- Accuracy: có nhận đúng tất cả chữ không?
- Hallucination: có sinh ra chữ không có trong ảnh không?
- Latency: bao nhiêu giây?
- Token usage: bao nhiêu tokens input/output?

---

## 2. GLM-OCR (0.9B)

### Option A: Ollama local (recommend, free, offline)

```bash
# 1. Cài Ollama
brew install ollama

# 2. Pull model (~600MB)
ollama pull glm-ocr

# 3. Đảm bảo Ollama đang chạy
ollama serve   # nếu chưa chạy
```

#### Chạy test

```bash
cd scripts/ocr-service

# Test với ảnh mặc định
python test_glm.py

# Test với ảnh cụ thể
python test_glm.py /path/to/image.webp
```

### Option B: Z.AI cloud API (free tier, GLM-4V-Flash)

```bash
# 1. Đăng ký lấy API key
#    International: https://z.ai
#    China:         https://bigmodel.cn

# 2. Export key
export ZAI_API_KEY=...

# 3. Chạy test với --cloud flag
cd scripts/ocr-service
python test_glm.py /path/to/image.webp --cloud
```

---

## 3. So sánh kết quả

Sau khi chạy cả 2, so sánh theo bảng:

| Tiêu chí | DeepSeek-OCR-2 | GLM-OCR (Ollama) | GLM-4V-Flash (Cloud) |
|---|---|---|---|
| Printed accuracy | | | |
| Handwritten accuracy | | | |
| Hallucination (có/không) | | | |
| Latency (extract) | | | |
| Latency (structured) | | | |
| Token usage | | | |

### Ảnh test gợi ý

| Loại | Mô tả |
|---|---|
| Printed sách giáo khoa | Chữ in rõ ràng, font chuẩn |
| Printed + pinyin | Chữ in có kèm phiên âm (thường gặp trong sách HSK) |
| Handwritten neat | Chữ viết tay ngay ngắn |
| Handwritten cursive | Chữ viết tay nhanh/thảo |
| Mixed CN + VN + EN | Ảnh có lẫn nhiều ngôn ngữ |

---

## 4. Kết luận mong đợi

Cả 2 engine đều **không có per-character confidence** và **không có bounding box**. Test này nhằm đánh giá:

1. **Accuracy raw text** — có đọc đúng chữ không?
2. **Hallucination** — có bịa chữ không có trong ảnh không? (critical cho learning app)
3. **Latency** — có chấp nhận được cho post-processing layer không?
4. **Structured output** — có follow instruction tốt không khi yêu cầu JSON?

Nếu accuracy cao + không hallucinate → có thể dùng làm **verification layer** sau PaddleOCR/Google Vision ở Phase 2+.
