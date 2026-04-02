# OCR Service — API Contract

Base URL: `http://localhost:8000`

---

## POST /test

Upload image truc tiep, ho tro tat ca engines.

### Parameters (multipart form)

| Field    | Type   | Default      | Description                              |
|----------|--------|--------------|------------------------------------------|
| image    | file   | required     | File anh (png, jpg, webp)                |
| engine   | string | `paddleocr`  | Engine OCR (xem bang ben duoi)            |
| language | string | `zh`         | Ngon ngu: `zh`, `en`, `vi`               |

### Engines

| Engine      | Type | Auth                          | Output                        |
|-------------|------|-------------------------------|-------------------------------|
| `paddleocr` | OCR  | Khong can                     | characters + confidence + bbox |
| `tesseract` | OCR  | Khong can                     | characters + confidence + bbox |
| `google`    | OCR  | Service account `.gcp/ocr.json` (auto) | characters + confidence + bbox |
| `glm`       | VLM  | Ollama local (free) hoac `ZAI_API_KEY` | raw text                  |
| `deepseek`  | VLM  | `DEEPSEEK_API_KEY`            | raw text                      |

---

### Curl — OCR engines

```bash
# PaddleOCR (default)
curl -F image=@test_image.png \
     http://localhost:8000/test

# PaddleOCR — English
curl -F image=@test_image.png \
     -F engine=paddleocr \
     -F language=en \
     http://localhost:8000/test

# Tesseract
curl -F image=@test_image.png \
     -F engine=tesseract \
     http://localhost:8000/test

# Google Cloud Vision
curl -F image=@test_image.png \
     -F engine=google \
     http://localhost:8000/test
```

### Curl — VLM engines

```bash
# GLM local (Ollama — khong can key)
# Setup: ollama serve && ollama pull glm-ocr
curl -F image=@test_image.png \
     -F engine=glm \
     http://localhost:8000/test

# GLM cloud (Zhipu AI — can ZAI_API_KEY)
# Server phai chay voi: export ZAI_API_KEY=...
curl -F image=@test_image.png \
     -F engine=glm \
     http://localhost:8000/test

# DeepSeek (can DEEPSEEK_API_KEY)
# Server phai chay voi: export DEEPSEEK_API_KEY=sk-...
curl -F image=@test_image.png \
     -F engine=deepseek \
     http://localhost:8000/test
```

---

### Response — OCR engines (paddleocr, tesseract, google)

```json
{
  "engine": "paddleocr",
  "type": "ocr",
  "characters": [
    {
      "text": "你好",
      "pinyin": "nǐ hǎo",
      "confidence": 0.9521,
      "char_confidences": [
        {"char": "你", "confidence": 0.9612},
        {"char": "好", "confidence": 0.9430}
      ],
      "candidates": [],
      "location": [[10, 20], [100, 20], [100, 50], [10, 50]]
    }
  ],
  "elapsed_seconds": 1.23
}
```

### Response — VLM engines (glm, deepseek)

```json
{
  "engine": "glm",
  "type": "vlm",
  "text": "你好\n世界",
  "mode": "local (ollama glm-ocr)",
  "tokens": {
    "input": 256,
    "output": 12
  },
  "elapsed_seconds": 2.45
}
```

---

## POST /recognize

OCR voi base64 image. Chi ho tro paddleocr, tesseract, google.

```bash
curl -X POST http://localhost:8000/recognize \
  -H "Content-Type: application/json" \
  -d '{"image": "'$(base64 < test_image.png)'", "language": "zh", "engine": "paddleocr"}'
```

### Response

```json
{
  "characters": [
    {
      "text": "你好",
      "pinyin": "nǐ hǎo",
      "confidence": 0.9521,
      "char_confidences": [
        {"char": "你", "confidence": 0.9612},
        {"char": "好", "confidence": 0.9430}
      ],
      "candidates": [],
      "location": [[10, 20], [100, 20], [100, 50], [10, 50]]
    }
  ],
  "engine": "paddleocr"
}
```

---

## POST /extract-text

Tra ve raw text lines (khong segment words). Chi ho tro paddleocr, tesseract.

```bash
curl -X POST http://localhost:8000/extract-text \
  -H "Content-Type: application/json" \
  -d '{"image": "'$(base64 < test_image.png)'", "language": "zh"}'
```

### Response

```json
{
  "blocks": [
    {
      "text": "你好世界",
      "location": [[10, 20], [200, 20], [200, 50], [10, 50]],
      "confidence": 0.9412
    }
  ],
  "engine": "paddleocr"
}
```

---

## GET /health

```bash
curl http://localhost:8000/health
```

### Response

```json
{
  "status": "ok",
  "engines": ["paddleocr", "tesseract", "google"]
}
```

---

## Setup nhanh

```bash
cd scripts/ocr-service
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# (Optional) Tesseract
brew install tesseract tesseract-lang

# (Optional) GLM local
brew install ollama && ollama serve && ollama pull glm-ocr

# Chay server
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

Swagger UI: http://localhost:8000/docs
