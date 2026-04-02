# OCR Service — Testing Guide

So sanh tat ca OCR engines. Test qua endpoint `POST /test` (upload file truc tiep).

## Setup

```bash
cd lu-backend/scripts/ocr-service
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# Chay server
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

Swagger UI: http://localhost:8000/docs

---

## 1. PaddleOCR (default)

**Khong can setup them gi. Chay ngay.**

```bash
curl -F image=@test_image.png http://localhost:8000/test
```

- Type: OCR
- Output: characters + confidence (per-char) + bounding box
- Model: PP-OCRv5

---

## 2. Tesseract

**Can cai them tesseract.**

```bash
# Setup (1 lan)
brew install tesseract tesseract-lang

# Test
curl -F image=@test_image.png -F engine=tesseract http://localhost:8000/test
```

- Type: OCR
- Output: characters + confidence + bounding box

---

## 3. Google Cloud Vision

**Khong can set env var. Dung service account `.gcp/ocr.json` (da co san).**

```bash
# Setup (1 lan, trong venv)
pip install google-auth google-cloud-vision

# Test
curl -F image=@test_image.png -F engine=google http://localhost:8000/test
```

- Type: OCR
- Output: characters + confidence + bounding box
- Auth: service account tu `.gcp/ocr.json` (auto detect)

---

## 4. GLM — Local via Ollama

**Free. Khong can key.**

```bash
# Setup (1 lan)
brew install ollama
ollama serve          # de terminal nay mo
ollama pull glm-ocr   # ~600MB, chi can 1 lan

# Test
curl -F image=@test_image.png -F engine=glm http://localhost:8000/test
```

- Type: VLM
- Output: raw text (khong co confidence, khong co bounding box)
- Model: `glm-ocr` (0.9B) — nhe, chay duoc tren Mac

---

## 5. GLM — Cloud via Zhipu AI

**Can key. Free tier.**

```bash
# 1. Dang ky tai https://open.bigmodel.cn (can email)
# 2. Tao API key

# 3. Khoi dong server voi key
export ZAI_API_KEY=your-key-here
uvicorn main:app --host 0.0.0.0 --port 8000 --reload

# 4. Test
curl -F image=@test_image.png -F engine=glm http://localhost:8000/test
```

- Type: VLM
- Output: raw text
- Model: `glm-4v-flash` (mien phi)
- Luu y: neu co `ZAI_API_KEY` -> tu dong dung cloud, khong thi fallback ve Ollama local

---

## 6. DeepSeek

**Can key. ~$2 free credit.**

```bash
# 1. Dang ky tai https://platform.deepseek.com
# 2. Vao API Keys -> tao key

# 3. Khoi dong server voi key
export DEEPSEEK_API_KEY=sk-...
uvicorn main:app --host 0.0.0.0 --port 8000 --reload

# 4. Test
curl -F image=@test_image.png -F engine=deepseek http://localhost:8000/test
```

- Type: VLM
- Output: raw text
- Model: `deepseek-chat` (DeepSeek-V3)
- **Luu y:** DeepSeek chua public vision API -> co the fail khi gui image

---

## Test tat ca engines cung luc

```bash
# Chay lan luot, so sanh ket qua
for engine in paddleocr tesseract google glm deepseek; do
  echo "=== $engine ==="
  curl -s -F image=@test_image.png -F engine=$engine http://localhost:8000/test | python3 -m json.tool
  echo ""
done
```

---

## So sanh

| Engine      | Type | Free | Can key          | Confidence | Bounding box | Toc do  |
|-------------|------|------|------------------|------------|--------------|---------|
| `paddleocr` | OCR  | Yes  | Khong            | Per-char   | Co           | Nhanh   |
| `tesseract` | OCR  | Yes  | Khong            | Per-word   | Co           | Nhanh   |
| `google`    | OCR  | Free tier | Khong (SA auto) | Co      | Co           | Nhanh   |
| `glm`       | VLM  | Yes  | Khong (Ollama)   | Khong      | Khong        | Tuy may |
| `deepseek`  | VLM  | ~$2  | Co               | Khong      | Khong        | Nhanh   |

## Thu tu test khuyen nghi

1. **PaddleOCR** — da co san, chay ngay
2. **Google Cloud Vision** — SA da co, chi can pip install
3. **GLM Ollama** — free, khong can key, can cai Ollama
4. **Tesseract** — can brew install
5. **GLM Cloud / DeepSeek** — can dang ky lay key
