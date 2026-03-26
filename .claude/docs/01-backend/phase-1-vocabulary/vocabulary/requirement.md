# Vocabulary Module — Requirement (trích từ PRD v3)

> Nguồn gốc: `docs/requirement.md` — Prep Chinese Vocab PRD v3

## Phase 1 — MVP

**Timeline:** Sprint 0 (Mar 15-31) → Sprint 1 (Apr 1-14) → Sprint 2 (Apr 15-28) → Sprint 3 stretch ship (Apr 29) hoặc buffer + polish (May 13).

Tập trung vào **3 trụ cột cốt lõi:**

**Trụ 1: Nhập từ vựng thông minh** *(→ mục 1, 2)*
- OCR Scan Hán tự → auto flashcards (printed ≥ 90%, handwritten ≥ 80% accuracy)
- Import thủ công + HSK Built-in Wordlists (HSK 1-9, theo chuẩn HSK 3.0 Nov 2025 syllabus)

**Trụ 2: Chu trình học đa dạng (Smart Learning Path) — 7 Modes** *(→ mục 4)*
- Discover (flashcards) → Recall → Stroke & Recall → Pinyin Drill → AI Chat → Review (SM-2) → Mastery Check
- Kết hợp SRS (SM-2) + Memory Score tracking
- Grammar gắn context: mỗi từ vựng kèm cấu trúc ngữ pháp liên quan

**Trụ 3: Hệ thống ghi nhớ & phân loại (Vocabulary Retention Logic)** *(→ mục 5)*
- Memory Score per word, 6 trạng thái: Start Learning → Still Learning → Almost Learnt → Finish Learning → Memory Mode → Mastered
- Dashboard theo dõi tiến độ 4-dimension (Syllables, Characters, Vocabulary, Grammar)

---

## 1. Vocabulary Data & Content

### 1.1 HSK Built-in Wordlists — Cấu trúc HSK 3.0 (syllabus Nov 2025)

| HSK Level | Stage | Từ vựng (tích lũy) | Hán tự nhận diện | Hán tự viết | Syllables | Access |
|---|---|---|---|---|---|---|
| HSK 1 | Elementary (A1) | 300 | 246 | — | 269 | Free |
| HSK 2 | Elementary (A1) | 500 | 424 | — | 468 | Free |
| HSK 3 | Elementary (A2) | 1,000 | 636 | — | 608 | Free |
| HSK 4 | Intermediate (B1) | 2,000 | 1,096 | — | 724 | Pro |
| HSK 5 | Intermediate (B1) | 3,600 | 1,527 | 150 | 822 | Pro |
| HSK 6 | Intermediate (B2) | 5,400 | 1,940 | 300 | 908 | Pro |
| HSK 7 | Advanced (C1) | 7,000 | 2,421 | 400 | 1,020 | Pro (Phase 2) |
| HSK 8 | Advanced (C1) | 9,000 | 2,753 | 450 | 1,070 | Pro (Phase 2) |
| HSK 9 | Advanced (C2) | 11,000 | 3,088 | 500 | 1,110 | Pro (Phase 2) |

### 1.2 Data Model per Word

```json
{
  "hanzi": "学习",
  "pinyin": "xuéxí",
  "meaning_vi": "học tập",
  "meaning_en": "to study",
  "examples": [
    { "cn": "我每天学习中文。", "vi": "Tôi mỗi ngày học tiếng Trung.", "audio_url": "..." }
  ],
  "audio_url": "...",
  "hsk_level": 1,
  "topic": "学习教育",
  "radicals": ["子", "冖", "习"],
  "stroke_count": 11,
  "stroke_data_url": "...",
  "grammar_points": ["gp_001"],
  "recognition_only": true,
  "frequency_rank": 42
}
```

### 1.3 Topic & Category System

10 topic chuẩn HSK:

| # | Tiếng Trung | Slug | Nội dung |
|---|---|---|---|
| 1 | 日常生活 | daily-life | Chào hỏi, gia đình, thời gian, số đếm |
| 2 | 饮食 | food-drink | Món ăn, gọi món, nấu ăn |
| 3 | 交通旅行 | transportation | Phương tiện, hỏi đường |
| 4 | 学习教育 | education | Trường học, thi cử |
| 5 | 工作商务 | work-career | Công việc, thương mại |
| 6 | 健康医疗 | health | Bệnh viện, triệu chứng |
| 7 | 科技 | technology | Internet, thiết bị |
| 8 | 自然环境 | nature | Thời tiết, động vật |
| 9 | 文化娱乐 | culture | Phim, nhạc, lễ hội |
| 10 | 社会 | society | Luật pháp, kinh tế |

**Quy tắc:**
- Tách khái niệm **Topic** (system-defined) và **Folder/Deck** (user-created)
- User tự tạo folder khi scan/import, có thể chọn topic tag (optional)
- 1 từ có thể thuộc nhiều topic nếu polysemy
- Không giới hạn số từ per learning card

### 1.4 Character Decomposition System

Giúp học viên hiểu cấu trúc Hán tự thay vì ghi nhớ thuần hình ảnh. Nghiên cứu cho thấy hiểu radical tăng tốc ghi nhớ 40-60%.

Mỗi flashcard hiển thị:

1. **Radical (bộ thủ):** Thành phần chính → nhóm nghĩa. VD: 语 → bộ 讠(ngôn ngữ) + 五 + 口
2. **Breakdown animation:** Tách chữ thành từng thành phần
3. **Memory hook:** Câu chuyện ghi nhớ. VD: 休 = 人 + 木 → "Người dựa vào cây = nghỉ ngơi"
4. **Related characters:** Các chữ cùng bộ thủ. VD: 讠→ 说, 话, 语, 读, 认

**Data sources:**
- Unihan database (Unicode Consortium) → radical data
- CC-CEDICT → nghĩa + pinyin
- CJK Decomposition Data Project → thành phần cấu tạo
- Memory hooks: AI-generated, Phase 2 thêm human review cho top 500 từ

### 1.5 Grammar Context System (MVP)

Không xây module riêng. Grammar gắn vào từng từ vựng dưới dạng Grammar Tips.

Mỗi learning card có section "Grammar":

1. **Pattern:** VD: 把 → `S + 把 + O + V + Complement`
2. **Example:** 1-2 câu highlight pattern. VD: 我**把**书**放在**桌子上。
3. **Rule ngắn:** 1-2 câu giải thích.
4. **Common mistake:** Lỗi người Việt hay mắc. VD: "Không dùng 把 với 是, 有, 知道."

**Data:** Chinese Grammar Wiki (AllSet Learning, CC) + HSK Standard Course index. MVP cover **80 grammar points** cho HSK 1-3.

### 1.6 Stroke Order Data

- **Make Me a Hanzi** (open-source, ARPHIC License, 9,000+ characters, SVG paths)
- Stroke validation: compare user input vs. reference (order + direction)
- Standard: GB (Trung Quốc đại lục)
- ⚠️ Cần confirm commercial use license với legal team

---

## 2. Vocabulary Input

**Trụ 1: Nhập từ vựng thông minh** — 2 cách nhập từ vựng vào hệ thống.

### 2.1 Import thủ công

- Tạo từ vựng đơn lẻ qua form nhập: hanzi, pinyin, meaning_vi, meaning_en, examples, hsk_level, topic
- Nhập hàng loạt (bulk import) qua admin endpoint cho việc seed HSK wordlists + content team upload
- Assign to folder/topic khi tạo (optional)
- Duplicate check: trùng hanzi → thông báo → View / Ignore / Merge

### 2.2 OCR Scan Hán tự → Auto Flashcards

Chụp ảnh vở ghi chép / sách giáo khoa → OCR nhận diện Hán tự → tự động tạo flashcard gồm: Hán tự, pinyin, nghĩa tiếng Việt, ví dụ câu, hình ảnh minh họa, audio phát âm.

**Flow:**

```
User tap "Scan" → Camera opens → Chụp ảnh / import từ gallery
    → OCR processing (1-3s)
    → Preview screen: hiển thị detected characters
        → Mỗi character có: confidence indicator, suggested pinyin, suggested meaning
        → User có thể: Edit / Delete / Add missing / Confirm all
    → Duplicate check: so sánh với existing collection
        → Trùng: hiện "X từ đã có" + option View / Ignore / Merge
    → Assign to folder/topic (optional, có thể chọn "Unsorted")
    → Generate flashcards → Done
```

**Edge Cases:**

| Case | Solution |
|---|---|
| OCR nhận sai chữ Hán (viết tay, font lạ) | Preview + "Confirm or Edit" từng mục. Cảnh báo từ sai chính tả → "Did you mean X?" khi confidence < 80%. Hiển thị top-3 candidates. |
| Mix tiếng Trung + Việt + Anh | Lọc chỉ lấy Hán tự. Hiển thị "Detected X Chinese characters" trước khi tạo cards. |
| Chữ quá xấu / ký tự đặc biệt | UI review với thông tin tối thiểu: word, pinyin (auto-suggest), definition, IPA, example (opt). Hệ thống so sánh từ điển → đưa IPA + audio + ảnh nếu có. |
| Scan trùng từ đã có | Check duplicate → thông báo → View / Ignore / Merge. Merge luôn vào card cũ, hiển thị số vocab trùng + CTA review. |
| Note hs ghi sai | Tôn trọng nội dung user viết. Chỉ gợi ý nếu phát hiện sai sót (ưu tiên thấp hơn so với user input). |

**Technical:**
- OCR Engine: Google Cloud Vision API (primary) hoặc Baidu OCR API (fallback cho handwritten). Tesseract (open-source) fine-tuned trên dataset viết tay tiếng Trung
- Target accuracy: ≥ 90% printed, ≥ 80% handwritten
- Fallback: confidence < 70% → show top-3 candidates

---

## 3. Vocabulary Organization

### 3.1 Folder System

- User tự tạo **Folder** (deck) để tổ chức từ vựng
- Folder thuộc sở hữu user (chỉ owner mới CRUD được)
- 1 từ có thể nằm trong nhiều folder
- Sắp xếp theo ngày thêm mới nhất trong folder
- Phase 2: Folder shareable → link mở app. Xem list từ nhưng phải login mới học được. CTA: "Save to track your own score."

### 3.2 Polysemy & Duplicate Handling

| Case | Solution |
|---|---|
| 1 Hán tự nhiều nghĩa → topic nào? | Check duplicate word ID. Cùng nghĩa → 1 bản, "also appears in X topic." Khác nghĩa → flashcard riêng, user chọn topic. |
| Scan trùng từ ở nhiều folder | Same word ≠ meaning → keep both. Same word = meaning → confirm override? |
| Cùng pinyin khác Hán tự (homophone) | Recall quiz pinyin → hiển thị tất cả Hán tự cùng pinyin để phân biệt |
| Traditional vs Simplified | MVP: Simplified only. Settings toggle để hiển thị Traditional as reference (Phase 2). |

---

## 4. Smart Learning Path — 7 Modes

Chu trình học đa dạng kết hợp SRS (SM-2) + Memory Score tracking. Grammar gắn context: mỗi từ vựng kèm cấu trúc ngữ pháp liên quan.

### 4.1 Discover Mode (Free & Pro)

Flashcard dọc full-screen. Mặt trước: Hán tự (lớn) + pinyin + tone marks + hình ảnh minh họa. Tap flip → mặt sau: nghĩa Việt, ví dụ câu, grammar tip.

| Element | Spec |
|---|---|
| Flashcard Layout | Full-screen dọc, swipe up = next card |
| Audio Button | Góc trên phải — phát âm chuẩn (male/female voice toggle) |
| Radical Badge | Badge nhỏ hiển thị bộ thủ + nghĩa. Tap → character decomposition popup |
| Progress | "Card 2 of 20" |
| Phase 2 Upgrade | **Video Flashcard (Seedance 2.0)** — thay hình tĩnh bằng video 3-5s AI-generated |

### 4.2 Recall Mode (Free & Pro)

| Element | Spec |
|---|---|
| Quiz Types | (1) Nghe audio → chọn Hán tự, (2) Xem Hán tự → chọn nghĩa (MCQ), (3) Xem pinyin → viết Hán tự, (4) Matching: nối Hán tự-nghĩa, (5) Fill pinyin tones, (6) Odd-one-out, (7) Group words by category |
| Feedback | Chấm tức thì + mini-tip giải thích (VD: "休 = 人 + 木 → nghỉ dưới gốc cây") |
| Accuracy Tracker | Tính % chính xác → cập nhật Memory Score |
| Polysemy handling | Đáp án đúng = nghĩa user đã save. Nếu từ trùng ở nhiều folder → dạng "pick from list", chọn đủ các nghĩa đã học mới tính pass |

### 4.3 Stroke & Recall Mode

Mode kết hợp 3 sub-modes trong flow liền mạch, giải quyết pain point: "học nhưng không viết → quên mặt chữ."

**Sub-mode A: Guided Writing (mặc định cho từ mới, Memory Score 0-39)**

| Element | Spec |
|---|---|
| Stroke Animation | Animation viết từng nét theo đúng thứ tự. User xem rồi viết theo. |
| Writing Canvas | Vùng viết tay + grid ô vuông (田字格) hướng dẫn tỷ lệ. |
| Stroke Order Check | Đúng = xanh, sai = đỏ + gợi ý nét tiếp theo. |
| Stroke Count | "Nét 3/8" |
| Radical Highlight | Sau viết xong → highlight bộ thủ + giải thích |
| Retry Logic | Sai > 2 nét → "Viết lại". Đạt → "Next Character" |

**Sub-mode B: Recall Writing (auto-unlock khi Memory Score ≥ 40)**

| Element | Spec |
|---|---|
| Purpose | Viết Hán tự từ trí nhớ — KHÔNG gợi ý stroke, KHÔNG animation. Chuyển từ "nhận diện" sang "tái tạo." |
| Prompt | Hiển thị: pinyin + nghĩa Việt + audio. KHÔNG hiển thị Hán tự. |
| Canvas | Grid ô vuông, không hướng dẫn stroke order. |
| AI Evaluation | So sánh output vs. chuẩn: (1) Đúng chữ? (2) Đúng stroke order? (3) Tỷ lệ nét chính xác. 3 mức: Perfect / Acceptable (đúng chữ, sai 1-2 stroke) / Incorrect. |
| Hint System | Stuck >10s → hint L1: hiện radical. +10s → hint L2: hiện số nét. "Show answer" → hiện Hán tự + animation (trừ điểm recall). |
| Interleaving | Mỗi từ viết đúng 3 lần xen kẽ = Pass. |
| Confusable Detection | Viết sai ra chữ Hán hợp lệ khác → so sánh 2 chữ cạnh nhau, highlight nét khác biệt (VD: 拔 vs 拨). |

**Sub-mode C: Speed Writing (Phase 2, auto-unlock khi Memory Score ≥ 70)**

| Element | Spec |
|---|---|
| Purpose | Viết nhanh liên tục 10 chữ trong time pressure. Train automaticity — viết không cần nghĩ. |
| Flow | Hiện nghĩa/pinyin → user viết → 5s timeout → next. Chấm binary: pass/fail. |
| Gamification | Timer bar + streak counter. Personal best tracking. |

**Flow chuyển đổi giữa sub-modes:**

| Điều kiện | Sub-mode |
|---|---|
| Từ mới (Memory Score = 0) | Guided Writing |
| Memory Score < 40 (Still Learning) | Guided Writing (option tự chuyển Recall) |
| Memory Score ≥ 40 (Almost Learnt+) | Auto-switch → Recall Writing |
| Memory Score ≥ 70 (Finish Learning+) | Auto-switch → Speed Writing (Phase 2) |
| Recall fail 2 lần liên tiếp | Quay Guided, 1 session sau thử lại Recall |

**Session Summary:** Số từ viết đúng/tổng, Guided vs Recall ratio, danh sách từ cần ôn, confusable pairs.

**Access:**

| Free | Pro |
|---|---|
| Guided Writing: xem animation only | Guided Writing: viết tay + AI check |
| Recall Writing: 5 từ/ngày | Recall Writing: unlimited |
| Speed Writing: N/A | Speed Writing: unlimited (Phase 2) |

### 4.4 Pinyin & Tone Drill Mode — PrepAI Pronunciation

Sử dụng PrepAI scoring engine để đánh giá phát âm chi tiết đến từng âm/thanh.

| Element | Spec |
|---|---|
| Prompt | Hiển thị từ vựng 1-by-1, phân tách các âm đọc kèm IPA + trọng âm. Speaker button nghe mẫu. |
| Recording | User tap mic icon → ghi âm → AI processing (< 2s) |
| **PrepAI Scoring** | Chấm 3 chiều riêng biệt: **Initial** (âm đầu: zh/z, sh/s, ch/c...), **Final** (vần), **Tone** (thanh điệu 1-4 + thanh nhẹ). Mỗi chiều cho điểm 0-100. |
| Tone Visualization | Contour diagram 4 thanh (+ thanh nhẹ). Overlay: thanh user đọc (đỏ) vs. thanh đúng (xanh). Real-time pitch contour nếu possible. |
| Result Display | Xanh (đúng) / Đỏ (sai) trên từng âm phân tách. VD: "**x**uéx**í**" → x=đúng, ué=sai tone 2, xí=đúng |
| Common Mistakes | Gợi ý lỗi phổ biến người Việt: nhầm zh/z, sh/s, thanh 2 vs 3 |
| **Personalized Weakness Report** | Sau mỗi session → tổng hợp: "Bạn hay sai thanh 2 (rising tone) và âm đầu zh. Gợi ý drill thêm 5 từ có zh + thanh 2." |
| **Learning Plan** | Tích lũy data pronunciation → tạo personalized improvement plan: focus vào âm/thanh yếu nhất. Hiển thị progress over time. |
| CTA | "Retry" nếu sai hoàn toàn. "Next Word." "Skip Mode" → chuyển thẳng AI Chat. |
| Access | Free: trial 3 từ/ngày. Pro: unlimited. |

**Technical — PrepAI Integration:**

| Component | MVP | Phase 2 |
|---|---|---|
| Scoring Engine | SpeechSuper API (SDK có sẵn cho Mandarin tone scoring) | Custom PrepAI model fine-tuned cho Mandarin tones (tái sử dụng architecture DeBERTa/PANN từ IELTS Speaking) |
| Granularity | Initial + Final + Tone scoring per syllable | + Fluency + Rhythm + Sentence-level prosody |
| Data Collection | Ghi âm user pronunciation (opt-in consent) → training data cho custom model | 10,000+ audio samples × 4 tones × common syllables |
| Weakness Detection | Per-session summary | Cross-session learning plan + adaptive drill |

**Integration flow:**
```
User tap mic → Record audio (WAV, 16kHz)
    → Send to SpeechSuper API
    → Receive: per-syllable scores (initial, final, tone)
    → Display results on UI
    → Update Memory Score (Pinyin Drill weight = 1)
```

**Pronunciation Weakness Detection & Improvement Plan:**
```
After N sessions (N ≥ 5):
    → Aggregate pronunciation scores across all words
    → Identify: Top 3 weakest initials, Top 2 weakest tones
    → Generate: "Your Pronunciation Focus"
        - "Thanh 2 (rising): 62% accuracy → Drill 10 words with tone 2"
        - "Âm đầu zh: 55% accuracy → Practice: 中, 知, 张, 住, 找"
    → Track improvement over time (weekly chart)
    → Adjust drill queue to prioritize weak areas
```

### 4.5 AI Chat Mode (Pro only)

| Element | Spec |
|---|---|
| Interface | Messenger-style. AI bên trái, user bên phải. |
| Topic Context | Gắn chủ đề learning card (VD: 点餐 gọi món, 旅行 du lịch) |
| AI Personas | 3 persona: (1) **朋友 Bạn bè** — tiếng Trung đơn giản, gần gũi, không quan trọng ngữ pháp; (2) **老板 Sếp** — từ vựng công việc, yêu cầu trang trọng; (3) **老师 Gia sư** — sửa lỗi ngữ pháp, yêu cầu dùng synonyms cao cấp hơn |
| Interaction | Text input (ưu tiên) + voice input. AI phát hiện đúng từ đã học → highlight xanh, thả tim + cộng Recall. |
| Grammar Feedback | Phản hồi tự nhiên, gợi ý sửa lỗi / từ đồng nghĩa. VD: "把 dùng khi tác động lên đối tượng: 把书放在桌子上" |
| Session Summary | Words used correctly, New words introduced, XP gained |
| Scoring Link | Update Memory Score (mode weight 2) |

### 4.6 Review Mode (Free & Pro)

| Element | Spec |
|---|---|
| Purpose | Spaced Repetition theo SM-2. |
| Daily Review CTA | Tự động hiển thị khi spacing due. Nút CTA review bên ngoài learning path. |
| Mini Games | Word scramble, True/False, Crosswords, Matching word-picture, Odd-one-out |
| SM-2 Integration | Chấm q-score 0-5 → cập nhật Spacing_Score (+0–2) |
| Feedback | Kết thúc: "Next Review In: X days" |
| Access | Free: chỉ review Still Learning & Almost Learnt. Pro: tất cả trạng thái. |

### 4.7 Mastery Check (Pro only)

| Element | Spec |
|---|---|
| Goal | Kiểm tra toàn bộ từ trong learning card, xác nhận "Mastered." |
| Structure | (1) Fill-in-the-blank, (2) 5 short-sentence chunking — tập phát âm câu ví dụ, (3) Viết pinyin đúng tone, (4) Sắp xếp từ thành câu đúng ngữ pháp |
| Result | "Mastery Achieved: 87%" + spacing tiếp theo (30 days) |
| Gamification | Double XP + badge "Mastered Island" theo HSK level |
| Trùng lặp Recall | Max 15% câu hỏi trùng dạng bài với Recall mode |

---

## 5. Vocabulary Classification & Dashboard

### 5.1 Trạng thái ghi nhớ (Memory Score)

| Trạng thái | Điều kiện | Review Interval |
|---|---|---|
| **Start Learning** | Memory Score = 0. Chưa hoàn thành mode nào. | I(1) = 1 day |
| **Still Learning** | Memory Score < 40. Đã học nhưng sai nhiều. | I(2) = 2 days |
| **Almost Learnt** | 40 ≤ Memory Score < 60. Nhớ cơ bản, chưa ổn định. | I(3) = 3-5 days |
| **Finish Learning** | 60 ≤ Memory Score < 80. Nhớ trong đa phần context. | I(4) = 7 days |
| **Memory Mode** | Memory Score ≥ 80 + spacing đúng ≥ 1 lần. Củng cố dài hạn. | I(5) = 14 days |
| **Mastered** | Memory Score ≥ 90 + spacing đúng ≥ 2 lần + không sai 10-14 ngày. | I(6) = 30 days. Reset nếu không ôn 60 ngày. |

### 5.2 Memory Score Formula

```
Memory Score (per word) =
  Σ(Mode_Score × Mode_Weight) + Spacing_Score
  ─────────────────────────────────────────────  × 100
                   Max_Points
```

**Mode Weights:**

| Mode | Weight | Max Weighted | Ghi chú |
|---|---|---|---|
| Discover | 1 | 1 | Recognition |
| Recall | 2 | 4 | Active recall — mode chính |
| Stroke (Guided) | 1 | 1 | Muscle memory |
| Stroke (Recall) | 2 | 4 | Deep character retention |
| Pinyin Drill | 1 | 1 | Secondary reinforcement |
| Chat (AI) | 2 | 4 | Deep retention, productive |
| Review (SM-2) | 2 | 4 | Long-term retention |
| Mastery Check | 2 | 4 | Validation |

**Max_Points:** Free = 11 (Discover 1 + Recall 4 + Review 4 + Spacing 2). Pro = 25 (all modes 23 + Spacing 2).

**Spacing_Score:**

| Condition | Score |
|---|---|
| Review đúng ngày (±1 day) và q ≥ 4 | +2 |
| Review trễ ≤ 3 ngày, q ≥ 3 | +1 |
| Review sớm hơn 2 ngày | +0.5 |
| Review trễ > 3 ngày hoặc q < 3 | 0 |

### 5.3 Upgrade Free → Pro Migration

Khi user upgrade: giữ toàn bộ data cũ → chuẩn hóa điểm từ thang max 11 sang 25 → nếu score sau chuẩn hóa < 40 → giảm 1 bậc state → giữ spacing history → mode mới sẽ update score dần.

### 5.4 Learning Stats Dashboard

| Component | Spec |
|---|---|
| Header Stats | Words to Review Today + "Review Now" button. Conversations practiced count. |
| Memory State Breakdown | Tổng từ đã học + breakdown 6 trạng thái (bar chart hoặc donut) |
| HSK Level Progress | Progress bar per level (VD: HSK 1: 245/300) |
| **4-Dimension Tracking** | Progress theo 4 chiều HSK 3.0: Syllables, Characters, Vocabulary, Grammar |
| XP Tracker | Tổng XP tích lũy |
| Review Progress | Topics đã/đang/chưa review + "Finish Review Now" CTA |
| Pronunciation Insight | (Pro) Weak phonemes/tones summary + improvement trend |

---

## 6. Free vs Pro & Monetization

### 6.1 Giới hạn theo tier

| Feature | Free | Pro |
|---|---|---|
| HSK Wordlists | HSK 1-3 | HSK 1-9 |
| Scan/day | Max 3 ảnh | Unlimited |
| Cards/day | Max 20 | Unlimited |
| Flashcard type | Text only | Text + Images (Phase 2: + Video) |
| Stroke & Recall | Guided xem only + Recall 5 từ/ngày | Full Guided + unlimited Recall + Speed Writing (Phase 2) |
| Pronunciation | Trial 3 từ/ngày | Unlimited + Weakness Report |
| Learning Modes | Discover + Recall + Review | All 7 modes (incl. Chat, Mastery) |
| Grammar | Tips giới hạn | Full context + Phase 2 module |
| AI Chat | Không | Unlimited |
| Ads | Non-intrusive | Ad-free |

### 6.2 Conversion Triggers

1. **Soft paywall tại Stroke Practice:** Free xem animation nhưng không viết → CTA "Unlock Stroke Practice"
2. **HSK Level Gate:** Hoàn thành HSK 3 → "Ready for HSK 4? Upgrade."
3. **AI Chat preview:** Free 1 session/week → "Want more? Go Pro."
4. **Memory Score ceiling:** Free pathway rất chậm → "Reach Mastered faster with Pro."
5. **Weekly progress email:** "You learned X words. Unlock AI Chat to learn 2x faster." (data-driven nudge)

---

## 7. HSK 3.0 Alignment & Phase 2

### 7.1 Key Changes Affecting Vocab (HSK 3.0)

| Change | Impact | Action |
|---|---|---|
| 9 levels (thay vì 6) | Progression system, UI, wordlists, badges | P0: Implement 9-level system |
| Vocabulary +120% (11,000 từ) | Database, content scope | P0: Dùng syllabus Nov 2025 |
| 4-dimension assessment | Dashboard, progress tracking | P0: Track Syllables + Characters + Vocab + Grammar |
| Recognition vs. Writing | Stroke Practice mode | P1: "Recognition Drill" (L1-4), "Writing Drill" (L5+) |
| Speaking mandatory (L3+) | Pinyin Drill + AI Chat | P1: Stronger speaking component từ HSK 3 |
| Translation (L4+, mới) | Phase 2 opportunity | P0 Phase 2: CN↔VN → CN↔TH, CN↔ID |

### 7.2 Competitive Gap

Từ phân tích sơ bộ 7 đối thủ (SuperTest, HelloChinese, Duolingo, ChineseSkill, SuperChinese, HSK Lord, Pleco):

1. Không ai fully cover HSK 3.0 với 9 levels
2. Không có AI Writing + Speaking scoring engine
3. Zero translation practice (kỹ năng mới HSK 3.0)
4. Weak SEA localization
5. Không có transition support HSK 2.0 → 3.0

> ⚠️ Cần nghiên cứu thêm: Nhi (Commercial Lead) sẽ thực hiện competitive analysis chuyên sâu trong Sprint 0 Market Brief.

### 7.3 Video Flashcard — Seedance 2.0 (Phase 2)

Thay hình ảnh tĩnh trong Discover mode bằng short video 3-5 giây AI-generated, thể hiện nghĩa từ trong context thực tế.

VD: 跑步 (chạy bộ) → video người chạy trong công viên. 吃饭 (ăn cơm) → video người ngồi ăn.

**USP: "Flashcard sống"** — không app nào trên thị trường có. Viral potential cao khi user share video flashcard lên MXH.

**Technical Evaluation (cần trước Phase 2):**

| Factor | Question | Target |
|---|---|---|
| Cost per video | Seedance 2.0 pricing per generation? | < $0.05/video để viable ở scale |
| Latency | Time to generate 1 video? | < 10s (acceptable), < 5s (ideal) |
| Quality consistency | Video output stable chất lượng? | ≥ 85% usable without manual review |
| Caching strategy | Pre-generate cho HSK wordlists vs. on-demand? | Pre-generate HSK 1-6 (~5,400 words), on-demand cho user-created |
| Storage & CDN | Video storage cost + delivery? | Estimate cho 5,400 × 3-5s videos |
| Fallback | Nếu video generation fail? | Fallback to static image |

**Viral Potential:**
- User share video flashcard lên TikTok/Douyin/Instagram Reels → organic reach
- "Xem video flashcard của tôi" → CTA download app
- GV share bộ video flashcard cho cả lớp

---

## 8. Key Flows & Edge Cases

### 8.1 Key Flows

**Flow 1: First-time User**

```
Download → Onboarding (chọn HSK level hoặc "I don't know")
    → Nếu "I don't know" → Quick placement test (20 words, 2 min)
    → Chọn level → Load wordlist → Discover mode (first 10 words)
    → Complete Discover → Prompt: "Ready to test yourself?" → Recall mode
    → Complete Recall → Dashboard showing progress
    → Push notification next day: "5 words to review!"
```

**Flow 2: Scan Notes**

```
Home → Tap "Scan" → Camera / Gallery
    → OCR processing → Preview detected characters
    → Edit/Confirm → Duplicate check → Assign folder
    → Generate flashcards → Start learning
```

**Flow 3: Daily Learning Session**

```
Open app → Dashboard shows: "12 words to review" + "Continue HSK 2"
    → Option A: "Review Now" → Review mode (SM-2 scheduled words)
    → Option B: "Continue" → Next mode in Smart Learning Path
    → Session complete → Summary: words learned, XP, streak
```

**Flow 4: Pronunciation Improvement**

```
Pinyin Drill → Record pronunciation → PrepAI scores per syllable
    → Red/Green overlay on each phoneme
    → Session end → Weakness summary
    → After 5+ sessions → "Your Pronunciation Focus" generated
    → Adaptive drill queue prioritizes weak areas
```

### 8.2 Edge Cases

| Case | Solution |
|---|---|
| Discover: nghĩa câu ví dụ không khớp user input | Ưu tiên nghĩa user đã save. Câu ví dụ phải match meaning context. |
| AI Chat: phản hồi lạc topic | Kỹ thuật check chặt topic. Rate limiting + context buffer cho spam. |
| AI Chat: user dùng synonym không tính XP | NLP matching linh hoạt, nhận diện synonyms |
| Mastery trùng Recall | Max 15% câu trùng dạng bài |
| Stroke order nhiều chuẩn | Mặc định GB (Trung Quốc đại lục). Ghi chú nếu khác biệt phổ biến. |
| Pronunciation AI nghe sai (noise/accent) | Retry 2 lần. Vẫn fail → "Mark as correct" (không tính điểm) + suggest headphones. |
