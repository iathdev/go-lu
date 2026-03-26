# OCR Service

Multi-engine HTTP wrapper for OCR, supporting [PaddleOCR](https://github.com/PaddlePaddle/PaddleOCR) and [Tesseract](https://github.com/tesseract-ocr/tesseract).

## Setup

```bash
cd scripts/ocr-service

# Create virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Install PaddleOCR (default engine)
pip install -r requirements.txt

# (Optional) Install Tesseract engine
# macOS:
brew install tesseract tesseract-lang
pip install pytesseract

# Ubuntu:
# sudo apt install tesseract-ocr tesseract-ocr-chi-sim
# pip install pytesseract
```

## Run

```bash
source .venv/bin/activate
uvicorn main:app --host 0.0.0.0 --port 8000

# With auto-reload (dev):
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

Swagger docs: http://localhost:8000/docs

## Test with curl

```bash
# PaddleOCR — Chinese (default)
curl -X POST http://localhost:8000/recognize \
  -H "Content-Type: application/json" \
  -d '{"image": "'$(base64 < test_image.jpg)'", "language": "zh"}'

# PaddleOCR — English
curl -X POST http://localhost:8000/recognize \
  -H "Content-Type: application/json" \
  -d '{"image": "'$(base64 < english_text.png)'", "language": "en"}'

# Tesseract engine
curl -X POST http://localhost:8000/recognize \
  -H "Content-Type: application/json" \
  -d '{"image": "'$(base64 < test_image.jpg)'", "language": "zh", "engine": "tesseract"}'

# Health check
curl http://localhost:8000/health
```

## API

### POST /recognize

JSON body:

| Field    | Type   | Default      | Description                                 |
|----------|--------|--------------|---------------------------------------------|
| image    | string | required     | Base64-encoded image data                   |
| language | string | `"zh"`       | Language: `"zh"`, `"en"`, `"vi"`            |
| engine   | string | `"paddleocr"`| Engine: `"paddleocr"`, `"tesseract"`        |

### Response

```json
{
  "characters": [
    {"text": "你好", "confidence": 0.9521, "candidates": []},
    {"text": "世界", "confidence": 0.9234, "candidates": []}
  ],
  "engine": "paddleocr"
}
```

### GET /health

```json
{"status": "ok", "engines": ["paddleocr", "tesseract"]}
```
