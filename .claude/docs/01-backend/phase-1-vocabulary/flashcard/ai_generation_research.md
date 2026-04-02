# AI-Powered Flashcard Generation — Research

## 1. Các platform đang triển khai AI Flashcard

| Platform | AI Feature | Mô tả |
|---|---|---|
| **Quizlet** | Magic Notes | Convert handwritten/digital notes, PDFs, Google Docs → flashcards. ~85% accuracy không cần edit |
| **Knowt** | AI from notes/PDF/YouTube | ~90% accuracy trích xuất key concepts. Free, 700K+ students |
| **AnkiAIUtils** | GPT-4 + DALL-E plugin | Auto-generate mnemonics, illustrations, example sentences, cloze cards. Multi-model (OpenAI, Anthropic, local) |
| **NovaCards** | LLM card generation | Tìm relevant cards từ existing decks dùng lecture notes |
| **Revisely** | AI editing | Natural language prompts: "Keep it short", "Provide a simple example" |
| **Wisdolia** | Chrome extension | Generate flashcards từ any web content: slides, PDFs, YouTube, research papers |
| **StudyFetch** | Multi-format | Term/definition, MCQ, fill-in-the-blank, audio flashcards. AI tutor voice |
| **Memrizz** | Batch generation | 80-100 cards/lecture (~20 per 500 words). Basic, Cloze, Image Occlusion. 20+ languages |

---

## 2. Nguồn sinh Flashcard bằng AI

### 2.1 Từ documents (PDF, text, notes)
```
Document → Text extraction → Chunking (~12K tokens) → LLM summarization → Flashcard generation → Quality validation
```
- Map-Reduce pattern: Map phase trích concepts/terms/definitions; Reduce phase tổng hợp thành cloze sentences
- Ví dụ thực tế: 242 notes (534 cards) từ sách 10 chương trong ~1 giờ (dùng local LLM)

### 2.2 Từ images (OCR + AI)
- OCR trích text từ scanned documents/handwritten notes → LLM xử lý
- Image occlusion: AI nhận diện diagrams, auto-generate occlusion cards che key labels
- Quizlet Magic Notes xử lý handwritten notes trực tiếp

### 2.3 Từ audio/video
- YouTube videos transcribed → LLM xử lý (Knowt, Wisdolia)
- Memrizz nhận audio/video files trực tiếp

### 2.4 Từ vocabulary lists (phù hợp nhất cho Lu)
- AI enriches danh sách từ vựng với: example sentences, mnemonics, pronunciation, part of speech, context
- Multi-direction card generation: L2→L1, L1→L2, listening, reading→phonetic, sentence cloze

### 2.5 Từ learning history (adaptive)
- Cards tự điều chỉnh difficulty dựa trên user responses
- Track "likely difficulty" per card dựa trên word frequency và syntactic complexity

---

## 3. AI Features cho Language Learning cụ thể

### 3.1 Auto-generate example sentences
- LLM tạo câu ví dụ phù hợp ngữ cảnh, nhiều mức độ khó
- Bao gồm cultural context và natural usage patterns

### 3.2 Auto-generate mnemonics/memory hooks
- AnkiAIUtils: nhiều mnemonic options per card, vivid imagery, personal memory anchors
- Semantic similarity matching tìm existing mnemonics liên quan đến card mới
- Format: word, translation, example sentence, mnemonic association

### 3.3 Auto-generate cloze deletions
- AI phân tích text → xác định key information để blanking
- Grammar cloze: blanking particles, verb conjugations, sentence patterns
- Ví dụ: `"我{{c1::喜欢}}吃{{c2::苹果}}"` → mỗi cloze tạo 1 review card riêng

### 3.4 Difficulty estimation
- Word frequency analysis + syntactic complexity scoring
- Item Response Theory (IRT) cho proficiency-item matching
- FSRS difficulty parameter (1-10) trained từ review history

### 3.5 Image generation
- DALL-E 2/3 và Stable Diffusion cho custom mnemonic images
- Phân tích card content → xác định key visual concepts

---

## 4. Implementation Patterns

### 4.1 LLM Prompting Strategies

| Strategy | Mô tả | Hiệu quả |
|---|---|---|
| **Few-shot prompting** | Cung cấp 2-3 example flashcards trong prompt | Cải thiện đáng kể chất lượng |
| **Structured output (JSON Schema)** | OpenAI Structured Outputs / constrained decoding | 99%+ schema adherence, loại bỏ parsing failures |
| **XML streaming** | Dùng `<card></card>` tags thay JSON cho streaming | Cards available cho client ngay khi stream |
| **Temperature control** | Lower temperature (0.4) cho factual content | Giảm hallucination |
| **Chain-of-thought** | Break thành ideation → drafting → refinement | Tốt cho complex reasoning |

### 4.2 Model comparison (từ comparative study)

| Model | "Would use" rating | Ghi chú |
|---|---|---|
| GPT-4 variants | 0.64–0.68 | Tốt nhất |
| GPT-3.5 variants | 0.54–0.58 | Khá |
| Local 13B + manual curation | 0.44–0.50 | Cần human review |
| Local 7B | 0.24 | Kém đáng kể |

### 4.3 Quality Control

- **Multi-layered evaluation**: objective metrics (syntactic compliance) + subjective (LLM-as-judge) + human-in-the-loop
- **Key quality dimensions**: self-containment, atomicity (1 concept/card), truthfulness, flashcard viability
- **Adversarial defense**: jailbreak detection, prompt leaking safeguards
- **Iterative prompt tuning**: chạy batch → analyze edge cases → tweak prompt

### 4.4 Batch vs On-demand Generation

| Mode | Khi nào dùng | Chi phí |
|---|---|---|
| **Batch (pre-generation)** | Xử lý documents/PDFs, vocabulary lists. OpenAI batch API giảm 50% | Thấp |
| **On-demand** | Enrichment cá nhân (add mnemonic, example sentence). Stream XML cho immediate display | Cao |
| **Hybrid (khuyến nghị)** | Pre-generate base cards batch; enrich on-demand khi user xem card | Tối ưu |

**Background generation pattern**: Return HTTP 202 + job ID → client polls hoặc nhận SSE notification → generate 10 cards/batch, save từng card khi stream xong.

### 4.5 Cost Optimization

| Strategy | Hiệu quả | Mô tả |
|---|---|---|
| **Semantic caching** | Giảm 68.8% API calls, 96.9% latency reduction | Convert prompts → embeddings, cache khi cosine similarity > 0.90 |
| **Prompt engineering > fine-tuning** | Giảm 30% cost | RisingStack case study |
| **Request batching** | Giảm overhead | Accumulate requests 10-100ms window |
| **Model tiering** | Giảm đáng kể | Cheap model cho translations/TTS, expensive cho mnemonics/cloze |
| **Pre-computation** | Giảm runtime cost | Generate common vocabulary enrichments (HSK wordlists) offline |

---

## 5. API Design cho AI Generation

```
# Async generation từ document
POST /api/ai/flashcards/generate
  Body: { source_type: "pdf|text|url|vocabulary_list", content: "...", target_language: "zh", options: {...} }
  Response: 202 { job_id: "...", status_url: "/api/ai/jobs/{id}" }

# Poll job status
GET /api/ai/jobs/{id}
  Response: { status: "processing|completed|failed", progress: 0.6, cards: [...] }

# On-demand enrichment cho 1 card
POST /api/ai/cards/{id}/enrich
  Body: { fields: ["mnemonic", "example_sentence", "image"] }
  Response: 200 { mnemonic: "...", example_sentence: "...", image_url: "..." }

# Batch enrichment
POST /api/ai/cards/enrich
  Body: { card_ids: [...], fields: ["mnemonic"] }
  Response: 202 { job_id: "..." }
```

---

## 6. AI-Enhanced Spaced Repetition

- **FSRS parameter optimization**: 21 trainable parameters, fit via ML trên personal review history → 20-30% fewer reviews
- **Difficulty prediction**: Neural approaches (SSP-MMC, LSTM-HLR) predict individual memory states từ large-scale review data
- **Adaptive card content**: Khi user liên tục sai → generate thêm context cards, alternative mnemonics, simpler examples
- **Smart ordering**: Mix card types (recognition, production, cloze) within sessions cải thiện transfer learning

**Lưu ý quan trọng**: Nghiên cứu cho thấy AI hints giúp faster initial mastery (trong 48h) nhưng lower 90-day recall so với pure SRS → có thể tạo dependency. Nên dùng AI hỗ trợ, không thay thế hoàn toàn SRS.

---

## Sources

- [Quizlet AI Study Era](https://quizlet.com/blog/ai-study-era)
- [Knowt AI Review](https://fritz.ai/knowt-ai-review/)
- [AnkiAIUtils - GitHub](https://github.com/thiswillbeyourgithub/AnkiAIUtils)
- [NovaCards](https://novacards.ai/)
- [Revisely Flashcard Generator](https://www.revisely.com/flashcard-generator)
- [Wisdolia AI](https://aitoolsexplorer.com/ai-tools/wisdolia-an-ai-flashcard-generator-for-efficient-study/)
- [StudyFetch AI Flashcards](https://www.studyfetch.com/features/flashcards)
- [Memrizz AI](https://www.memrizz.com/)
- [RisingStack AI Multilingual Flashcards Case Study](https://blog.risingstack.com/ai-powered-multilingual-flashcards-case-study/)
- [Comparing LLMs for Flashcard Generation](https://www.alexejgossmann.com/LLMs-for-spaced-repetition/)
- [What I Learned Building an AI SRS App](https://www.seangoedecke.com/autodeck/)
- [OpenAI Structured Outputs](https://platform.openai.com/docs/guides/structured-outputs)
- [Semantic Caching - Redis](https://redis.io/blog/what-is-semantic-caching/)
- [Reduce AI API Costs](https://ofox.ai/blog/how-to-reduce-ai-api-costs-2026/)
