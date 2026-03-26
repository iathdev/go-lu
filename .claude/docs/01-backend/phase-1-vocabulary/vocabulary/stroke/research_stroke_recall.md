# Stroke & Recall Research — Handwriting Recognition + Validation

> Research phục vụ thiết kế Stroke & Recall mode (C6 trong `technical_challenges.md`).
>
> **Context:** Mode có 2 sub-modes trong MVP:
> - **Guided Writing** — user xem animation rồi viết theo, hệ thống check real-time per-stroke
> - **Recall Writing** — user viết từ trí nhớ (không có mẫu), hệ thống recognize chữ rồi chấm
>
> Hai sub-mode cần **công nghệ khác nhau hoàn toàn**: Guided = template matching (biết target), Recall = handwriting recognition (không biết target).

---

## 1. Yêu cầu từ PRD

### Guided Writing (Sub-mode A, Memory Score 0-39)

| Element | Spec |
|---|---|
| Animation | Viết từng nét theo đúng thứ tự. User xem rồi viết theo |
| Canvas | Vùng viết tay + grid ô vuông (田字格) |
| Per-stroke check | Đúng = xanh, sai = đỏ + gợi ý nét tiếp theo |
| Retry | Sai > 2 nét → "Viết lại". Đạt → "Next Character" |
| Radical highlight | Sau viết xong → highlight bộ thủ + giải thích |

### Recall Writing (Sub-mode B, Memory Score ≥ 40)

| Element | Spec |
|---|---|
| Prompt | Hiển thị pinyin + nghĩa + audio. KHÔNG hiển thị Hán tự |
| Canvas | Grid ô vuông, không gợi ý |
| AI Evaluation | (1) Đúng chữ? (2) Đúng stroke order? (3) Tỷ lệ nét chính xác. 3 mức: Perfect / Acceptable / Incorrect |
| Hint system | Stuck >10s → hiện radical. +10s → hiện số nét. "Show answer" → hiện Hán tự + animation (trừ điểm) |
| Confusable detection | Viết sai ra chữ hợp lệ khác → so sánh 2 chữ cạnh nhau, highlight nét khác biệt |
| Interleaving | Mỗi từ viết đúng 3 lần xen kẽ = Pass |

### Phân chia backend vs mobile

| Việc | Ai làm | Tại sao |
|---|---|---|
| Stroke animation rendering | Mobile | Cần render SVG trên canvas, touch interaction |
| Touch input → stroke capture | Mobile | Touch events là native API |
| Per-stroke matching (Guided) | Mobile | Real-time feedback, latency ~0 — không thể gửi server |
| Handwriting recognition (Recall) | Mobile (on-device) | Latency ~100ms cần real-time. Google ML Kit chạy offline |
| Confusable detection | Backend hoặc mobile | Lookup table, nhỏ, có thể embed trong app |
| Scoring + Memory Score update | Backend | Cần persist, consistent, anti-cheat |
| Stroke data serving | Backend | Serve Make Me a Hanzi data per character |

→ **Backend scope cho C6 hẹp hơn các challenge khác:** chủ yếu serve data + scoring API. Logic nặng nằm ở mobile.

---

## 2. Make Me a Hanzi — Stroke data source

### 2.1 Dataset

Repository: [skishore/makemeahanzi](https://github.com/skishore/makemeahanzi)

Gồm 2 file NDJSON, mỗi dòng = 1 character:

**`graphics.txt`** — dữ liệu vẽ:

| Field | Mô tả | Ví dụ |
|---|---|---|
| `character` | Unicode character | `"学"` |
| `strokes` | Array SVG path strings, **ordered by stroke order** | `["M 200 500 Q 300 400 ...", ...]` |
| `medians` | Array stroke medians (centerline), mỗi median = array `[x,y]` points | `[[[200,500],[300,400]], ...]` |

- Coordinate system: 1024×1024. Upper-left = (0, 900), lower-right = (1024, -124)
- **Stroke order = array order**: `strokes[0]` = nét thứ 1, `strokes[1]` = nét thứ 2, ...
- **Medians là key cho matching**: HanziWriter dùng medians để animate và compare user input

**`dictionary.txt`** — metadata:

| Field | Mô tả |
|---|---|
| `decomposition` | IDS (Ideograph Description Sequence): `"⿰亻本"` cho 体 |
| `radical` | Bộ thủ chính |
| `matches` | Map stroke index → component trong decomposition tree |
| `pinyin`, `definition` | Phát âm + nghĩa |

### 2.2 Coverage

- 9,000+ simplified + traditional characters
- License: ARPHIC Public License (⚠️ cần confirm commercial use — PRD §1.6)
- HSK 1-9 toàn bộ 11,000 từ → đa số characters đều có trong dataset

---

## 3. Guided Writing — Công nghệ

### 3.1 HanziWriter — Giải pháp chính

| | Chi tiết |
|---|---|
| **Library** | [chanind/hanzi-writer](https://github.com/chanind/hanzi-writer) |
| **Size** | 10KB gzipped (library). Data: ~3KB per character |
| **Chức năng** | Stroke animation + Quiz mode (per-stroke feedback) |
| **React Native** | [`@jamsch/react-native-hanzi-writer`](https://github.com/jamsch/react-native-hanzi-writer) — wrapper với gesture handler + reanimated + SVG |
| **Data source** | [hanzi-writer-data](https://github.com/chanind/hanzi-writer-data) — transformed từ Make Me a Hanzi |
| **Coverage** | 9,000+ characters |

### 3.2 Stroke matching algorithm (từ HanziWriter source code)

User vẽ 1 nét → HanziWriter evaluate bằng **5 criteria, TẤT CẢ phải pass**:

| # | Criterion | Đo gì | Threshold |
|---|---|---|---|
| 1 | **Average Distance** | Khoảng cách trung bình giữa user points và reference stroke | `350 × distMod × leniency` px (trong 1024 space) |
| 2 | **Start/End Point** | Cả 2 endpoints phải gần reference | `250 × leniency` px |
| 3 | **Direction** (Cosine Similarity) | Hướng vẽ đúng không? Trích direction vectors → cosine similarity | Average similarity > 0 |
| 4 | **Shape** (Frechet Distance) | Hình dạng nét giống không? Normalize curves (Procrustes), test 5 rotation angles | Frechet distance ≤ `0.4 × leniency` |
| 5 | **Length** | Nét đủ dài không? | `leniency × (userLen + 25) / (refLen + 25) ≥ 0.35` |

**`leniency` parameter**: float, default 1.0. Tăng = dễ hơn (tolerant hơn), giảm = khó hơn. Có thể config per user level.

**Stroke order intelligence**: Nếu user vẽ nét 3 trước nét 2, HanziWriter check xem nét đó match nét nào trong tương lai. Nếu match nét 3 → tính leniency adjustment để tránh accept nhầm out-of-order.

**Backward stroke detection**: Option `acceptBackwardsStrokes: true` → reverse user points rồi test lại (hữu ích cho nét 撇/捺 user hay vẽ ngược).

### 3.3 Quiz mode API

```js
writer.quiz({
  leniency: 1.0,           // 0.5 = strict, 1.5 = lenient
  showHintAfterMisses: 3,  // show hint sau 3 lần sai
  onCorrectStroke: (data) => {
    // data: { totalMistakes, strokeNum, mistakesOnStroke, strokesRemaining, drawnPath }
  },
  onMistake: (data) => { ... },
  onComplete: (data) => {
    // data: { totalMistakes, character }
    // → Gửi về backend: character, totalMistakes → tính score
  }
})
```

### 3.4 Tại sao dùng HanziWriter thay vì tự build?

| Tự build | HanziWriter |
|---|---|
| Parse SVG paths + medians | Đã làm sẵn |
| Implement 5-criterion matching | Đã implement + tuned |
| Handle edge cases (backward strokes, out-of-order) | Đã handle |
| React Native integration | Có wrapper sẵn |
| Maintain + fix bugs | Community maintained, 3.6K stars |

→ **Không cần tự build.** HanziWriter đã giải quyết toàn bộ Guided Writing.

---

## 4. Recall Writing — Công nghệ

### 4.1 Vấn đề

Recall Writing **không biết trước** user định viết chữ gì (user viết từ trí nhớ). Cần:
1. Recognize chữ viết tay → trả character
2. Compare recognized character vs expected answer
3. Nếu sai → check confusable

Đây là **handwriting recognition** — khác hẳn Guided Writing (template matching).

### 4.2 Các giải pháp

| Solution | On-device | Accuracy | Maintained | Size | Platform |
|---|---|---|---|---|---|
| **Google ML Kit Digital Ink Recognition** | Có (offline) | Tốt nhất | Active (Google) | ~20MB model | Android + iOS |
| Zinnia | Có | Tốt | Dead (2010) | 3-25MB | C/C++, có iOS port |
| Tegaki | Có | Tốt | Dead (2010) | Tương tự | Desktop |
| Server-side OCR (Google Vision/Baidu) | Không | Tốt | Active | N/A | Any |

### 4.3 Google ML Kit Digital Ink Recognition (Recommend)

| | Chi tiết |
|---|---|
| **Là gì** | Cùng technology với Gboard handwriting input và Google Translate |
| **Cách hoạt động** | Input: touch points `(x, y, timestamp)` per stroke → Output: ranked list candidate characters + confidence |
| **Offline** | Hoàn toàn. Model download 1 lần ~20MB, chạy on-device |
| **Latency** | ~100ms recognition |
| **Chinese support** | Simplified, Traditional, HK variant, TW variant |
| **300+ ngôn ngữ** | Có thể mở rộng cho multi-language sau này |

**Input format:**

```
Touch ACTION_DOWN → new Stroke, add Point(x, y, timestamp)
Touch ACTION_MOVE → add Points to current Stroke
Touch ACTION_UP   → finalize Stroke, add to Ink
...repeat per stroke...
Recognition request: Ink (array of Strokes) → candidates[]
```

**Enhancement**: `RecognitionContext` với writing area dimensions + pre-context (tối đa 20 chars trước) → cải thiện disambiguation.

**Output**: Array of candidates sorted by confidence. VD: user viết → candidates = [{text: "学", score: 0.95}, {text: "字", score: 0.12}, ...]

### 4.4 Tại sao không dùng server-side OCR cho Recall?

| Server-side (Google Vision/Baidu) | On-device (ML Kit) |
|---|---|
| Cần network call per stroke/character → 200-500ms latency | ~100ms, offline |
| Cost per call ($0.0015) × N characters/session → expensive | Free (on-device) |
| Ảnh gốc phải render → capture → upload | Touch points trực tiếp → recognize |
| Overkill — full page OCR khi chỉ cần recognize 1 character | Designed cho single character input |

### 4.5 Flow: Recall Writing

```
1. Mobile hiển thị: pinyin + nghĩa + audio (KHÔNG hiển thị Hán tự)
2. User viết trên canvas
3. Mỗi stroke: capture touch points → add to Ink
4. User indicate "done" (hoặc auto-detect pause >2s)
5. ML Kit recognize(Ink) → candidates[]
6. Compare candidates[0] vs expected answer:
   a. Match → check stroke order (xem section 5)
   b. No match → check confusable (xem section 6)
7. Gửi result về backend: { character, expected, result: perfect/acceptable/incorrect, strokes: [...] }
8. Backend update Memory Score
```

---

## 5. Stroke order validation (cho cả 2 sub-modes)

### 5.1 Guided Writing

HanziWriter **đã xử lý** — mỗi nét match theo thứ tự, sai thứ tự → reject.

### 5.2 Recall Writing — Validate stroke order sau khi recognize

Sau khi ML Kit recognize đúng character, cần validate stroke order:

```
1. User strokes: [(stroke1 points), (stroke2 points), ...]  (từ Ink object)
2. Reference strokes: Make Me a Hanzi medians cho character đó
3. Match user stroke[i] → reference stroke[j] (nearest match by Frechet distance)
4. Nếu mapping i→j tăng dần (1→1, 2→2, 3→3) → correct order
5. Nếu không (1→2, 2→1, 3→3) → wrong order nhưng correct character
```

### 5.3 Scoring

| Grade | Điều kiện |
|---|---|
| **Perfect** | Đúng chữ + đúng stroke order + tỷ lệ nét chính xác > 80% |
| **Acceptable** | Đúng chữ + sai 1-2 stroke order HOẶC tỷ lệ nét 60-80% |
| **Incorrect** | Sai chữ HOẶC tỷ lệ nét < 60% |

### 5.4 Alternative stroke orders

Một số characters có nhiều stroke order hợp lệ (khác nhau giữa Mainland China, Taiwan, Hong Kong, Japan):
- 必 (bi), 出 (chu), 万, 方, 戈, 升: có variant orders
- Mặc định dùng GB standard (Mainland China) — theo PRD §1.6

**Xử lý:** Make Me a Hanzi dùng GB standard. Nếu cần support variant → Phase 2 thêm alternative orderings per character.

---

## 6. Confusable detection — Viết nhầm chữ giống

### 6.1 Bài toán

User được yêu cầu viết 拔 (bá - nhổ) nhưng viết thành 拨 (bō - quay số). Cả 2 đều hợp lệ, chia sẻ bộ 扌, chỉ khác 1 component (犮 vs 发).

Hệ thống cần:
1. Phát hiện user viết chữ khác hợp lệ (không phải viết bậy)
2. Hiển thị 2 chữ cạnh nhau, highlight nét khác biệt

### 6.2 Datasets

| Dataset | Nội dung | Dùng cho |
|---|---|---|
| **[similar_chinese_characters](https://github.com/kris2808/similar_chinese_characters)** | 形近字 (visually similar) + 同音字 + 近音字. CSV | Pre-built lookup table |
| **Make Me a Hanzi `matches` field** | Map stroke → component. So sánh 2 chars → tìm component khác nhau | Highlight nét khác biệt |
| **[Wiktionary confusables](https://en.wiktionary.org/wiki/Appendix:Easily_confused_Chinese_characters)** | Curated pairs: 未/末, 人/入, 已/己/巳 | Bổ sung |
| **IDS decomposition** (Make Me a Hanzi `decomposition` field) | Compare cấu trúc: chars với IDS edit distance ≤ 3 = morphologically similar | Tự động phát hiện confusables |

### 6.3 Flow confusable detection

```
ML Kit recognize → candidates = [拨(0.85), 拔(0.72), 找(0.15)]
Expected answer = 拔

1. candidates[0] = 拨 ≠ 拔 (expected)
2. Check: 拔 có trong candidates? → Có (candidates[1], score 0.72)
3. Check: 拨 và 拔 có phải confusable pair? → Lookup dataset → Có (形近字)
4. → Return: "Bạn viết 拨 (bō - quay số), nhưng câu hỏi yêu cầu 拔 (bá - nhổ)"
5. Highlight nét khác:
   - Load Make Me a Hanzi strokes cho cả 拨 và 拔
   - Compare component by component (matches field)
   - Highlight component khác nhau (发 vs 犮) bằng màu đỏ
```

### 6.4 Nếu user viết chữ không liên quan?

```
ML Kit recognize → candidates = [大(0.90), 太(0.45), ...]
Expected answer = 拔

1. candidates[0] = 大 ≠ 拔
2. Check: 拔 có trong candidates? → Không
3. Check: 大 và 拔 confusable? → Không (hoàn toàn khác)
4. → Return: Incorrect. "Bạn viết 大. Đáp án đúng là 拔."
   → Show animation viết 拔
```

---

## 7. Backend API cho Stroke & Recall

### 7.1 Endpoints cần build

```
# Serve stroke data (cho mobile render animation + validation)
GET /api/characters/:hanzi/strokes
→ { strokes: [SVG paths], medians: [[[x,y]]], radical, decomposition, stroke_count }

# Submit writing result (mobile gửi sau khi evaluate)
POST /api/learning/stroke-result
→ { character, expected, result: "perfect"|"acceptable"|"incorrect",
    stroke_count_user, stroke_order_correct, mistakes, mode: "guided"|"recall" }
→ Backend update Memory Score + Learning Event Log

# Get confusable pairs (cho mobile lookup)
GET /api/characters/:hanzi/confusables
→ { confusables: [{ char: "拨", similarity: 0.92, diff_components: ["发/犮"] }] }
```

### 7.2 Backend scope — Gì KHÔNG làm ở backend

| Việc | Tại sao không ở backend |
|---|---|
| Stroke animation rendering | Mobile canvas + SVG |
| Touch input → stroke recognition | Mobile native gesture handler |
| Per-stroke matching (Guided) | HanziWriter đã xử lý ở mobile, latency ~0 |
| Handwriting recognition (Recall) | Google ML Kit on-device, latency ~100ms, offline |
| Stroke order comparison | Mobile có cả user strokes + reference strokes, compare locally |

→ **Backend cho C6 chủ yếu là data serving + scoring persistence.** Logic nặng ở mobile.

---

## 8. So sánh các app tương tự

| App | Guided Writing | Recall Writing | Stroke matching | Technology |
|---|---|---|---|---|
| **Skritter** | Có — per-stroke feedback, replace user stroke bằng idealized version | Có — viết từ trí nhớ | Proprietary algorithm. Configurable strictness | Closed-source |
| **Pleco** | Không | Có — handwriting input cho dictionary lookup | Template matching, 10K+ chars. Tolerant với stroke order | Closed-source |
| **HanziWriter demos** | Có — quiz mode | Không (chỉ template matching) | 5-criterion algorithm (distance, endpoints, direction, shape, length) | Open-source |
| **写汉字 apps** | Có — animation + guided | Một số có recall | Thường dùng HanziWriter hoặc tương tự | Varies |

### Nhận xét

- **Skritter** là reference tốt nhất cho cả 2 modes nhưng closed-source
- **HanziWriter** giải quyết hoàn toàn Guided Writing (open-source, React Native ready)
- **Recall Writing** cần thêm handwriting recognition — không app open-source nào combine HanziWriter + ML Kit sẵn, nhưng cả 2 đều mature và có thể integrate

---

## 9. Tóm tắt quyết định

| Sub-mode | Công nghệ | Ở đâu | Effort |
|---|---|---|---|
| **Guided Writing** | HanziWriter (`@jamsch/react-native-hanzi-writer`) | Mobile | Thấp — library sẵn, cần integrate |
| **Recall Writing — Recognition** | Google ML Kit Digital Ink Recognition | Mobile (on-device) | Trung bình — cần implement ink capture + recognition flow |
| **Recall Writing — Stroke order validation** | Compare user strokes vs Make Me a Hanzi medians (Frechet distance) | Mobile | Trung bình — cần implement comparison |
| **Confusable detection** | Pre-built lookup table từ similar_chinese_characters + Make Me a Hanzi decomposition | Mobile hoặc Backend | Thấp — dataset sẵn, cần parse + load |
| **Scoring API** | Backend endpoint nhận result → update Memory Score | Backend | Thấp — nhận result + persist |
| **Stroke data API** | Backend serve Make Me a Hanzi data per character | Backend | Thấp — serve static JSON |

**Backend effort cho C6: ~1-2 ngày** (data serving + scoring API). Logic nặng ở mobile side.

---

## References

| Source | URL | Relevance |
|---|---|---|
| Make Me a Hanzi | https://github.com/skishore/makemeahanzi | Stroke data: SVG paths, medians, decomposition. 9K+ chars |
| HanziWriter | https://hanziwriter.org | Animation + quiz mode. 5-criterion stroke matching |
| HanziWriter GitHub | https://github.com/chanind/hanzi-writer | Source code: stroke matching algorithm |
| HanziWriter Data | https://github.com/chanind/hanzi-writer-data | Transformed Make Me a Hanzi data |
| React Native HanziWriter | https://github.com/jamsch/react-native-hanzi-writer | React Native wrapper |
| Google ML Kit Digital Ink | https://developers.google.com/ml-kit/vision/digital-ink-recognition | Handwriting recognition, offline, 300+ languages |
| ML Kit Android guide | https://developers.google.com/ml-kit/vision/digital-ink-recognition/android | Implementation guide |
| similar_chinese_characters | https://github.com/kris2808/similar_chinese_characters | 形近字/同音字/近音字 dataset |
| ChineseCharacterSimilarity | https://github.com/liu-hanwen/ChineseCharacterSimilarity | Multi-metric similarity scoring |
| Wiktionary confusables | https://en.wiktionary.org/wiki/Appendix:Easily_confused_Chinese_characters | Curated confusable pairs |
| Zinnia | https://taku910.github.io/zinnia/ | Open-source HWR engine (legacy) |
| Chinese character strokes — Wikipedia | https://en.wikipedia.org/wiki/Chinese_character_strokes | 6 basic + 25 compound stroke types |
| Skritter stroke order guide | https://blog.skritter.com/2021/03/the-ultimate-guide-to-chinese-character-stroke-order/ | How Skritter approaches stroke validation |
| animCJK | https://github.com/parsimonhi/animCJK | Alternative stroke animation library |
| Taiwan MOE stroke order FAQ | https://stroke-order.learningweb.moe.edu.tw/page.jsp?ID=47&la=1 | Alternative stroke orders per region |
