# Vocabulary Module — Technical Challenges (Phase 1 MVP)

> Phân tích từ `internal/vocabulary/docs/requirement.md`

---

## Trụ 1: Nhập từ vựng thông minh

### C1. OCR Engine — Lựa chọn và tích hợp

**Yêu cầu:** Target ≥ 90% printed, ≥ 80% handwritten — tỷ lệ nhận diện đúng ký tự Hán tự. VD: ảnh chụp có 100 chữ Hán → printed phải nhận đúng ≥ 90 chữ, handwritten phải nhận đúng ≥ 80 chữ. Các lựa chọn engine:
- Google Cloud Vision API (primary)
- Baidu OCR API (fallback cho handwritten)
- Tesseract (open-source) fine-tuned trên dataset viết tay tiếng Trung

> Chi tiết so sánh 4 engine, single model vs dual model, PP-OCRv5 architecture: xem [`research_ocr_engine.md`](ocr/research_ocr_engine.md)

**Thách thức:**
- Latency budget 1-3s cho toàn bộ OCR processing
- Cost control: Cloud Vision tính per-request — cần monitoring để không bị bill shock khi scale
- Nếu dùng PaddleOCR: thiết kế sidecar service (Python) + gRPC/HTTP interface cho Go backend

### C2. OCR Post-processing — Confidence scoring & candidate ranking

**Yêu cầu:** Confidence < 80% → "Did you mean X?" + top-3 candidates. Confidence < 70% → show top-3 cho user chọn. Lọc chỉ lấy Hán tự từ mixed content (CN + VN + EN).

**Thách thức:**
- Mỗi OCR engine có thang confidence khác nhau (Google: 0-1 float, Baidu: 0-100 int) → chuẩn hóa thế nào cho consistent?
- Top-3 candidates: engine không phải lúc nào cũng trả candidates — cần dictionary lookup bổ sung (similar-looking characters)
- Mixed language filtering: phân biệt Hán tự vs chữ Kanji (Nhật) vs chữ Hán Nôm (Việt) — Unicode range overlap
- Auto-suggest pinyin + meaning cho detected characters: cần dictionary service real-time, latency phải thấp

### C3. Duplicate Detection — Polysemy-aware matching

**Yêu cầu:** Trùng hanzi → View / Ignore / Merge. Phân biệt "cùng chữ cùng nghĩa" vs "cùng chữ khác nghĩa" (polysemy).

**Thách thức:**
- Matching đơn thuần bằng hanzi string không đủ — VD: 打 có 20+ nghĩa tùy context
- Cần semantic matching: so sánh meaning_vi/meaning_en để phân biệt polysemy
- Merge logic phức tạp: merge vào card cũ nhưng giữ thông tin mới (examples, audio) → partial update
- Performance: khi bulk import 1,000 từ HSK, duplicate check phải efficient (batch query, not N+1)

### C4. Bulk Import — Seed 1,000 từ HSK 1-3

**Yêu cầu:** Nhập hàng loạt cho content team. Mỗi word có 12+ fields (hanzi, pinyin, meanings, examples, audio, radicals, stroke data, grammar points...).

**Thách thức:**
- Data validation phức tạp: pinyin phải đúng tone marks (xuéxí không phải xuexi), HSK level phải valid, topic phải match 10 topics chuẩn
- Relational data: mỗi word link đến grammar_points, topics, radicals — cần resolve references khi import
- Idempotency: import lại không duplicate, nhưng cũng phải update nếu data thay đổi
- Content team upload format: CSV? JSON? Excel? Cần design format dễ dùng cho non-technical users
- Transaction handling: import 1,000 words — partial failure thì rollback toàn bộ hay keep successful ones?

---

## Trụ 2: Smart Learning Path — 7 Modes

### C5. 7 Learning Modes trong 1 MVP — Surface area quá lớn

**Yêu cầu:** Discover → Recall → Stroke & Recall (3 sub-modes) → Pinyin Drill → AI Chat → Review (SM-2) → Mastery Check. Tổng cộng ~10 mode/sub-mode.

**Thách thức:**
- Mỗi mode có logic scoring riêng, UI riêng, data model riêng — backend phải expose API cho từng mode
- Mode interdependency: Memory Score tổng hợp từ tất cả modes → cần event-driven scoring system
- Session management: user có thể switch mode giữa chừng → state phải persist correctly
- Testing matrix: 7 modes × Free/Pro × edge cases = rất nhiều test scenarios cho 4-6 tuần

### C6. Stroke & Recall — Handwriting recognition + validation

**Yêu cầu:** Real-time stroke order check, AI evaluation (Perfect/Acceptable/Incorrect), confusable detection (拔 vs 拨).

**Thách thức:**
- Stroke order validation: cần parse Make Me a Hanzi SVG data → extract stroke sequence → compare user input stroke-by-stroke
- User input là touch coordinates (mobile) → cần stroke recognition algorithm: nhận diện đâu là 1 nét, hướng nét, thứ tự nét
- Confusable detection: database of similar-looking characters (拔/拨, 已/己/巳, 未/末) — không có sẵn, cần build hoặc tìm dataset
- Recall Writing sub-mode: user viết từ trí nhớ → phải recognize chữ viết tay KHÔNG CÓ reference → essentially handwriting OCR + matching
- Performance: stroke validation phải real-time (per-stroke feedback) trên mobile — latency budget gần 0

### C7. SpeechSuper API Integration — Pronunciation scoring

**Yêu cầu:** Chấm 3 chiều (Initial/Final/Tone) per syllable. Processing < 2s. Free: 3 từ/ngày, Pro: unlimited.

**Thách thức:**
- Audio recording format: WAV 16kHz — mobile phải record đúng format, handle microphone permissions, noise
- SpeechSuper API latency: < 2s target nhưng network latency + API processing có thể vượt — cần timeout + retry strategy
- Rate limiting phía SpeechSuper: 500K calls/day at scale — chưa confirm pricing, chưa có enterprise rate
- Tone visualization: cần pitch contour data từ API → render overlay diagram (user vs reference) — SpeechSuper có trả raw pitch data không?
- Weakness aggregation: accumulate scores across sessions → identify patterns (weak initials, weak tones) → cần analytics pipeline

### C8. AI Chat Mode — 3 Personas + Vocabulary tracking

**Yêu cầu:** 3 AI personas (朋友/老板/老师), detect từ đã học → highlight, grammar feedback, update Memory Score.

**Thách thức:**
- Prompt engineering cho 3 personas: mỗi persona phải consistent về ngôn ngữ, level, behavior — đồng thời phải detect vocabulary user đã học
- Vocabulary tracking trong conversation: AI response phải biết user đã học những từ nào → inject vocab list vào context → token cost tăng
- Synonym detection: user viết đúng nghĩa nhưng dùng từ khác (synonyms) → phải nhận diện để tính XP — cần NLP matching
- Grammar correction phải tự nhiên (không giống teacher mode nếu persona là 朋友)
- Cost: mỗi chat message = 1 LLM API call → với 50K MAU, cost per user phải kiểm soát
- Latency: chat phải feel real-time (< 3s response) — streaming response hay batch?

### C9. SM-2 Algorithm — Spaced Repetition implementation

**Yêu cầu:** q-score 0-5, interval calculation, tích hợp với Memory Score, Daily Review CTA.

**Thách thức:**
- SM-2 classic dùng EF (easiness factor) + interval — cần adapt cho multi-mode context (không chỉ review mà còn 6 mode khác ảnh hưởng)
- Scheduling logic: "words due today" phải tính từ tất cả words × last review date × interval → query performance khi user có 1,000+ words
- Timezone handling: "due today" phụ thuộc timezone user → server-side scheduling phải aware
- Interaction với Memory Score: SM-2 cập nhật Spacing_Score, đồng thời Memory Score cũng ảnh hưởng review interval → circular dependency?

---

## Trụ 3: Hệ thống ghi nhớ & phân loại

### C10. Memory Score — Weighted multi-mode scoring

**Yêu cầu:** Score = Σ(Mode_Score × Weight) + Spacing_Score / Max_Points × 100. Free Max = 11, Pro Max = 25. 6 trạng thái chuyển đổi.

**Thách thức:**
- Atomic scoring: mỗi learning event (quiz answer, stroke check, pronunciation score...) phải update Memory Score correctly — race condition nếu user dùng 2 mode đồng thời?
- State transitions: 6 states với điều kiện phức tạp (Score thresholds + spacing conditions + streak conditions) → state machine cần test kỹ
- Free vs Pro Max_Points khác nhau: cùng 1 user, score = 8 → Free: 8/11 = 72% (Finish Learning), Pro: 8/25 = 32% (Still Learning) — UX confusing khi upgrade
- Free → Pro migration: normalize score từ max 11 → max 25 → có thể giảm 1 bậc state → user cảm thấy bị "giáng cấp" khi trả tiền?
- Historical data: cần lưu score history per word per mode để debug + analytics — storage volume lớn

### C11. 4-Dimension Tracking — Syllables, Characters, Vocabulary, Grammar

**Yêu cầu:** Dashboard theo dõi progress theo 4 chiều HSK 3.0.

**Thách thức:**
- Data model: 4 dimensions cần tracking riêng nhưng overlap (1 vocabulary word đóng góp vào cả 4 dimensions)
- Syllables tracking: cần syllable-level data cho mỗi word (学习 = 2 syllables: xué, xí) — data model hiện tại chỉ có word-level
- Characters vs Vocabulary: 1 word có thể gồm nhiều characters (学习 = 学 + 习) — cần character-level tracking tách biệt
- Grammar tracking: 80 grammar points, mỗi point link đến nhiều words → progress = % grammar points "covered" bởi words đã learned?
- Real-time aggregation: dashboard cần aggregate data across 4 dimensions × tất cả words × tất cả modes → query phức tạp, cần caching/materialized view

---

## Cross-cutting

### C12. Content Data Pipeline — 4 external data sources

**Yêu cầu:** Unihan + CC-CEDICT + CJK Decomposition + Make Me a Hanzi → parse, map, store cho 1,000 words (MVP).

**Thách thức:**
- 4 data sources, 4 formats khác nhau, 4 update frequencies khác nhau → ETL pipeline cần build
- Data mapping: 1 hanzi → lookup radical (Unihan) + meaning (CC-CEDICT) + decomposition (CJK) + stroke SVG (Make Me a Hanzi) → join by Unicode codepoint, nhưng coverage không 100% ở mỗi source
- Memory hooks: AI-generated cho 1,000 words → batch generation + human review → workflow tool cần build
- Make Me a Hanzi license: ARPHIC License chưa confirm commercial use → nếu bị block, alternative nào?
- Data freshness: HSK 3.0 syllabus Nov 2025 — nếu có update trước July 2026 deployment, pipeline phải re-run

### C13. Free vs Pro — Feature gating ở mọi tầng

**Yêu cầu:** Rate limit (3 scan/day, 20 cards/day, 3 pronunciation/day, 5 recall writing/day) + feature lock (AI Chat, Mastery Check, full Stroke).

**Thách thức:**
- Rate limiting per-user per-feature per-day: cần counter system (Redis?) với reset logic hàng ngày
- Timezone lại ảnh hưởng: "per day" theo timezone user hay UTC?
- Feature gating phải enforce ở cả backend (API) lẫn mobile (UI) — nếu chỉ enforce ở mobile, dễ bypass
- 10+ rate limit rules × API endpoints → middleware phức tạp, dễ miss edge case
- Upgrade path: user đang ở giữa session Free → upgrade Pro → session hiện tại phải unlock ngay hay phải restart?

### C14. Learning Event Logging — Foundation cho adaptive learning

**Yêu cầu (từ main PRD mục 16.7):** Log mọi learning event: word learned, quiz result, time spent, difficulty level, pronunciation scores, stroke accuracy. Schema phải support backfill vào ATLAS/BKT sau.

**Thách thức:**
- Event volume: 50K MAU × trung bình 50 events/session × 1 session/day = 2.5M events/day
- Schema design: phải generic enough cho 7 mode types nhưng typed enough để query meaningful — EAV pattern vs typed events?
- Write performance: learning events là hot write path — nếu dùng PostgreSQL, cần partitioning hoặc separate event store
- CTO review data schema tuần đầu Sprint 1 — nếu bị reject, phải redesign ảnh hưởng toàn bộ timeline
- Backfill compatibility: schema phải compatible ATLAS/BKT mà chưa biết ATLAS/BKT schema là gì → design for flexibility

### C15. Prep User Service Integration — Shared auth, không tạo user riêng

**Yêu cầu (từ main PRD):** Dùng chung User Service của Prep platform, SSO, `learner_id` consistent.

**Thách thức:**
- Dependency vào external team (Platform team) — nếu User Service API chưa sẵn sàng hoặc thay đổi spec, bị block
- Auth flow: SSO token từ Prep platform → validate ở vocab backend → extract learner_id → mọi API call đều cần learner_id
- Local development: cần mock/stub User Service cho dev environment — stub phải đủ realistic
- Latency: mỗi request phải validate token với User Service → thêm 1 network hop → cần caching strategy cho token validation
- Data consistency: nếu user bị delete/deactivate ở Prep platform → vocab data xử lý thế nào?
