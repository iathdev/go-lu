# Phân tích ERD — PDF + AI.md vs Lu Database Design

> So sánh thiết kế database từ 2 nguồn AI-generated (`erd.pdf` và `ai.md`) với `database_design.md` hiện tại của Lu. Rút ra điểm hay, thống nhất đề xuất.
>
> **Nguồn:**
> - `erd.pdf` — PRD do Perplexity generate, 6 bảng, Chinese-only
> - `ai.md` — Bản mở rộng từ PDF, thêm `radicals` + `character_radicals`, Chinese-only
> - `database_design.md` — Thiết kế hiện tại của Lu, multi-language, no FK constraints
> - `flashcard/database_design.md` — Thiết kế flashcard module của Lu (FSRS)

---

## 1. PDF nói gì?

File `erd.pdf` là 1 PRD (Product Requirements Document) cho app học tiếng Trung, gồm 3 phần chính:

### 1.1 Product Overview

- **Tên app:** HSK Mastery App (chưa đặt tên chính thức)
- **Đối tượng:** Người học tiếng Trung ở Việt Nam (Phase 1) và Thái Lan (Phase 2)
- **Mục tiêu:** Ôn thi HSK 2.0 và HSK 3.0

### 1.2 Core Features

**a) Spaced Repetition (SRS) — Hệ thống ôn tập lặp lại có giãn cách**

- Dùng thuật toán **FSRS** (Free Spaced Repetition Scheduler) — thuật toán tính toán ngày tối ưu để ôn lại 1 từ dựa trên mức độ nhớ của user
- User review flashcard → chấm điểm (Again / Hard / Good / Easy) → app tính ngày review tiếp theo
- Chạy **trên device** (không cần server) → load flashcard cực nhanh, không phụ thuộc mạng

**b) Rich Media Flashcards — Flashcard có hình ảnh, âm thanh**

- Data từ CC-CEDICT (từ điển Trung-Anh mã nguồn mở, ~120K entries)
- Flashcard hiển thị: ảnh AI-generated, pinyin (phiên âm), Âm Hán Việt, câu ví dụ
- Ảnh host trên CDN, cache trên điện thoại → tiết kiệm data mobile

**c) Offline-First — Ưu tiên chạy offline**

- App đọc/ghi vào **SQLite** (database local trên điện thoại), không chờ server
- Use case: User học trên xe buýt, mất sóng → vẫn học bình thường
- Khi có mạng trở lại → **PowerSync** (engine đồng bộ) tự push tiến độ lên cloud (Supabase PostgreSQL)

### 1.3 Tech Stack

| Component | Công nghệ |
|---|---|
| Frontend | Flutter (Dart) |
| DB local | SQLite (qua PowerSync SDK) |
| DB cloud | PostgreSQL (Supabase) |
| Sync | PowerSync (đồng bộ 2 chiều SQLite ↔ PostgreSQL) |
| Media | Supabase Storage (CDN cho ảnh/GIF) |

### 1.4 Database Schema — 6 bảng

PDF thiết kế 6 bảng, mỗi bảng có `id` (UUID), `created_at`, `updated_at`, `deleted_at` (soft delete — đánh dấu xoá thay vì xoá thật, để sync engine biết ẩn record trên device).

**Bảng 1: `vocabulary` — Cấp độ từ (词)**

Lưu **từ/cụm từ** hoàn chỉnh — đơn vị mà HSK kiểm tra.

| Column | Giải thích |
|---|---|
| `simplified` | Chữ giản thể, ví dụ: "电脑" (máy tính) |
| `pinyin` | Phiên âm Latin, ví dụ: "diànnǎo" |
| `measure_words` | **Lượng từ** — trong tiếng Trung, danh từ phải dùng kèm từ đếm riêng. VD: "一**条**狗" (một con chó) — "条" là lượng từ. App trích từ CC-CEDICT để hiển thị riêng trên flashcard |
| `hsk2_level` | Cấp độ HSK phiên bản cũ (2.0) |
| `hsk3_level` | Cấp độ HSK phiên bản mới (3.0) |

**Đặc biệt:** `UNIQUE(simplified, pinyin)` — 2 cột tạo unique key kết hợp. Lý do: xử lý **đa âm** (多音字) — cùng 1 chữ nhưng đọc khác nhau tuỳ nghĩa. VD: 行 đọc "xíng" (đi) hoặc "háng" (hàng). Nếu chỉ unique trên `simplified` thì không lưu được 2 nghĩa khác nhau.

**Bảng 2: `characters` — Cấp độ chữ đơn (字)**

Lưu **từng chữ Hán riêng lẻ** — vì HSK 3.0 kiểm tra cả khả năng đọc/viết từng chữ, không chỉ từ.

| Column | Giải thích |
|---|---|
| `character` | 1 chữ Hán, ví dụ: "脑" |
| `radical` | **Bộ thủ** (部首) — thành phần gốc cấu tạo nên chữ Hán. VD: "氵" (bộ thuỷ/nước). Mỗi chữ Hán đều có 1 bộ thủ chính. Biết bộ thủ giúp đoán nghĩa: thấy "氵" → liên quan đến nước (河 sông, 海 biển, 湖 hồ). App hiển thị bộ thủ để tạo "móc nhớ" cho người học |
| `hsk3_recognition_level` | Cấp HSK 3.0 yêu cầu **nhận biết** (đọc được chữ này) |
| `hsk3_writing_level` | Cấp HSK 3.0 yêu cầu **viết được** chữ này. Thường cao hơn recognition vì viết khó hơn đọc |
| `stroke_svg_data` | Dữ liệu vector (SVG) mô tả **thứ tự nét viết** (stroke order) của chữ. App dùng data này để animate từng nét một, giúp user học viết đúng thứ tự. VD: chữ "大" viết 3 nét: ngang → phẩy trái → mác phải |

**Bảng 3: `word_characters` — Bảng cầu nối từ ↔ chữ**

Bảng trung gian nối **từ** (vocabulary) với **từng chữ đơn** (characters). Quan hệ nhiều-nhiều (1 từ chứa nhiều chữ, 1 chữ xuất hiện trong nhiều từ).

| Column | Giải thích |
|---|---|
| `vocab_id` | FK đến bảng vocabulary |
| `char_id` | FK đến bảng characters |
| `position` | Vị trí chữ trong từ (0, 1, 2...) |

VD: Từ "电脑" → 2 rows: (电, position 0) + (脑, position 1).

**Use case:** User đang xem flashcard "电脑" → tap vào chữ "脑" → app dùng bridge table này tìm ra character "脑" → hiển thị bộ thủ (月 = bộ nhục/thịt) + animation nét viết.

**Bảng 4: `definitions` — Nghĩa / bản dịch đa ngôn ngữ**

| Column | Giải thích |
|---|---|
| `vocab_id` | FK đến vocabulary |
| `lang_code` | Mã ngôn ngữ ISO: 'vi' (Việt), 'th' (Thái) |
| `meaning` | Nghĩa dịch: "máy tính", "computer" |
| `sino_vietnamese` | **Âm Hán Việt** — cách đọc chữ Hán theo hệ thống âm Việt. VD: 准备 → "Chuẩn bị". Nhiều từ tiếng Việt gốc Hán phát âm gần giống → giúp người Việt nhớ cực nhanh. Feature đặc thù cho thị trường VN |

**Bảng 5: `user_srs_reviews` — Dữ liệu ôn tập của user**

Mỗi user có 1 row per flashcard, chứa trạng thái học tập. Bảo mật bằng PowerSync rules — user A không thấy data của user B.

| Column | Giải thích |
|---|---|
| `state` | Trạng thái: New (chưa học), Learning (đang học), Review (ôn tập), Relearning (học lại) |
| `stability` | "Độ bền trí nhớ" — FSRS tính ra, càng cao = nhớ càng lâu |
| `difficulty` | Độ khó nội tại của từ — từ khó thì ôn thường xuyên hơn |
| `due_date` | Ngày app hiện lại flashcard này |

**Bảng 6: `media` — Ảnh/GIF AI-generated**

| Column | Giải thích |
|---|---|
| `vocab_id` | FK đến vocabulary |
| `file_url` | Link CDN đến ảnh/GIF. SQLite chỉ lưu URL (text nhẹ), app Flutter download + cache ảnh thật trên device → tránh database phình to |

### 1.5 ERD Diagram

PDF có sơ đồ quan hệ giữa 6 bảng, tóm tắt:

```
vocabulary (từ)
    ├── 1:N → definitions (nghĩa đa ngôn ngữ)
    ├── 1:N → examples (câu ví dụ)
    ├── 1:N → media (ảnh/GIF)
    ├── M:N → characters (chữ đơn) qua word_characters
    └── 1:N → user_srs_reviews (tiến độ học per user)

characters (chữ đơn)
    └── M:N → vocabulary qua word_characters
```

---

## 2. AI.md bổ sung gì so với PDF?

File `ai.md` (cũng do AI generate) mở rộng PDF thêm 2 bảng quan trọng:

### 2.1 Bảng `radicals` — Bộ thủ / thành phần cấu tạo chữ

PDF ban đầu lưu `radical` là 1 column TEXT trên `characters`. `ai.md` tách thành **bảng riêng** — đúng hơn vì radical trở thành entity độc lập, có thể tra cứu, hiển thị, phân nhóm.

| Column | Giải thích |
|---|---|
| `radical` | Ký hiệu: "氵", "女", "讠" |
| `pinyin` | Cách đọc bộ thủ |
| `meaning` | Nghĩa: "nước", "phụ nữ", "lời nói" |
| `sino_vietnamese` | Âm Hán Việt của bộ: "thuỷ", "nữ", "ngôn" |
| `stroke_count` | Số nét viết của bộ thủ |
| `stroke_svg_data` | Dữ liệu animation nét viết |

### 2.2 Bảng `character_radicals` — Nối chữ ↔ bộ thủ/thành phần

1 chữ có thể gắn với **nhiều** thành phần (không chỉ 1 bộ thủ chính). Bridge table với các cột:

| Column | Phase | Giải thích |
|---|---|---|
| `character_id` | MVP | Chữ nào |
| `radical_id` | MVP | Thành phần nào |
| `position` | Phase 2 | Vị trí: trái/phải/trên/dưới/bao quanh |
| `function_type` | Phase 2 | Vai trò: `semantic` (biểu ý), `phonetic` (biểu âm), `form` (hình dạng), `historical` (lịch sử) |
| `reasoning` | Phase 2 | Giải thích tại sao thành phần này liên quan đến chữ |
| `is_primary` | Phase 2 | Đánh dấu bộ thủ chính |

### 2.3 Use cases từ ai.md

| Feature | Mô tả |
|---|---|
| **Tra cứu bộ thủ** | App có trang riêng liệt kê 214 bộ thủ, tap vào → xem tất cả chữ liên quan |
| **Comprises** | Xem từ "安全" → tách thành "安" + "全" → mỗi chữ hiện bộ thủ + giải thích cấu tạo |
| **Radical picker** | Bài tập kéo-thả: app hiện vài thành phần rời (宀 + 女) → user ghép thành chữ đúng (安) |
| **Học theo nhóm bộ thủ** | Chọn bộ 氵 → hiện tất cả chữ liên quan nước: 河, 海, 湖, 清... |
| **Ghi nhớ trực quan** | Hiển thị "bộ 氵 = nước → 清 = trong/sạch (nước trong)" → mnemonic cho người học |

### 2.4 Phân phase (ai.md)

| Phase | Scope |
|---|---|
| **Phase 1** | `radicals` + `character_radicals` chỉ gồm `character_id`, `radical_id`. Chưa cần `reasoning`, `function_type`, `position` |
| **Phase 2** | Thêm `reasoning`, `function_type`, `position`. Thêm lesson theo radical, bài tập ghép chữ |
| **Phase 3** | Etymology (nguồn gốc chữ), variation theo font/handwriting, đồ thị quan hệ radicals ↔ characters |

---

## 3. So sánh tổng hợp: PDF + AI.md vs Lu

### 3.1 Lu đã làm tốt hơn

| Aspect | PDF/AI.md | Lu |
|---|---|---|
| **Đa ngôn ngữ** | Chinese-only, hardcode HSK | Generic: `languages` + `categories` + `proficiency_levels`. Thêm ngôn ngữ mới = thêm data, không sửa code |
| **Nghĩa từ** | `definitions(lang_code, meaning)` đơn giản | `vocabulary_meanings` + `vocabulary_examples` phong phú: phân biệt word_type, is_primary, nhiều câu ví dụ per nghĩa |
| **SRS/Flashcard** | 1 bảng `user_srs_reviews` lẫn lộn mọi thứ | Tách 3 bảng: `flashcard_scheduling` (FSRS state) + `review_logs` (lịch sử ôn tập) + `user_srs_settings` (cài đặt per user) |
| **Media** | Bảng `media` riêng (thêm join query) | `image_url`/`audio_url` trực tiếp trên `vocabularies` — đơn giản, ít join |
| **Grammar** | Không có | `grammar_points` + junction table liên kết từ vựng ↔ ngữ pháp |
| **Chủ đề** | Không có | `topics` + junction table |
| **Tổ chức** | Không có folder/deck | `folders` + `folder_vocabularies` — user tự tạo bộ từ |
| **Proficiency** | Hardcode `hsk2_level`, `hsk3_level` | `proficiency_levels` pluggable: HSK, JLPT, TOPIK, CEFR... |
| **FK constraints** | Dùng `REFERENCES` / FK | Không dùng FK ở DB level — referential integrity ở application layer (phù hợp horizontal scaling) |

### 3.2 PDF/AI.md có mà Lu chưa có

| Điểm | Nguồn | Đánh giá |
|---|---|---|
| Bảng `characters` (chữ đơn) | PDF + AI.md | **Rất đáng xem xét** — cần cho OCR, stroke animation, HSK 3.0 |
| Bảng `radicals` (bộ thủ/thành phần) | AI.md | **Đáng xem xét** — entity độc lập, tra cứu, bài tập ghép chữ |
| Bridge `word_characters` (từ ↔ chữ) | PDF + AI.md | **Cần thiết** nếu có bảng characters |
| Bridge `character_radicals` (chữ ↔ thành phần) | AI.md | **Cần thiết** nếu có bảng radicals |
| `sino_vietnamese` (Âm Hán Việt) | PDF + AI.md | **Quan trọng cho VN market** — killer feature |
| `measure_words` (Lượng từ) | PDF | Giữ trong JSONB metadata được |

---

## 4. Đề xuất thống nhất: Characters + Radicals Schema

### 4.1 Bộ thủ (Radical) vs Bộ kiện (Component) — Cần phân biệt

| | Bộ thủ (Radical / 部首) | Bộ kiện (Component / 偏旁) |
|---|---|---|
| **Số lượng per chữ** | Đúng **1** (dùng tra từ điển) | **Nhiều** (cấu thành chữ) |
| **Ví dụ: 清** | 氵 (bộ thuỷ) — chỉ 1 | 氵 (nghĩa: nước) + 青 (âm: qīng) — 2 thành phần |
| **Ví dụ: 想** | 心 (bộ tâm) — chỉ 1 | 木 + 目 + 心 — 3 thành phần |
| **Tổng** | 214 bộ Khang Hy (cố định) | ~500+ thành phần |

Mỗi thành phần (component) đóng 1 vai trò trong cấu tạo chữ:

- **Biểu ý (semantic):** Gợi ý nghĩa. VD: 氵 trong "清" → liên quan nước (清 = trong/sạch)
- **Biểu âm (phonetic):** Gợi cách đọc. VD: 青 (qīng) trong "清" (qīng)
- **Cấu trúc (structural/form):** Không mang nghĩa/âm rõ ràng, chỉ là thành phần hình dạng

Bộ thủ (radical) chỉ là 1 trong các component — component được chọn làm "đại diện" để tra từ điển.

**Vấn đề terminology:** `ai.md` gọi tất cả là "radicals" và dùng `is_primary` để đánh dấu bộ thủ chính. Cách này hoạt động nhưng về mặt khái niệm không chính xác — "radical" chỉ 1 per character, "component" mới là nhiều. Có 2 hướng đặt tên:

| Hướng | Bảng chứa thành phần | Bridge | Bộ thủ chính |
|---|---|---|---|
| **A: Dùng "radicals"** (như ai.md) | `radicals` | `character_radicals` | `is_primary` trên bridge |
| **B: Dùng "components"** | `components` | `character_components` | `radical_id` trên `characters` |

Hướng A đơn giản hơn (1 bảng chứa cả radical + component, dùng `is_primary` phân biệt). Hướng B chính xác hơn về khái niệm. **Cần quyết định trước khi implement.**

### 4.2 Schema đề xuất (Hướng A — đơn giản, aligned với ai.md)

```
radicals (bảng mới — lưu tất cả bộ thủ + thành phần cấu tạo)
    id, symbol, name, stroke_count, stroke_svg_data, ...
    VD: { symbol: "氵", name: "bộ thuỷ", stroke_count: 3 }
    VD: { symbol: "青", name: "thanh", stroke_count: 8 }

characters (bảng mới — mỗi chữ Hán 1 row)
    id, character, stroke_count, stroke_svg_data, ...
    VD: { character: "清", stroke_count: 11 }

character_radicals (bridge 1:N — chữ tách thành nhiều thành phần)
    character_id, radical_id
    + Phase 2: is_primary, position, function_type, reasoning
    VD: ("清", "氵")
    VD: ("清", "青")

word_characters (bridge M:N — từ ↔ chữ đơn)
    vocabulary_id, character_id, position
    VD: ("电脑", "电", 0)
    VD: ("电脑", "脑", 1)
```

### 4.3 Schema đề xuất (Hướng B — chính xác hơn)

```
components (bảng mới — lưu tất cả thành phần cấu tạo chữ Hán)
    id, symbol, name, stroke_count, stroke_svg_data, ...

characters (bảng mới — mỗi chữ Hán 1 row)
    id, character, radical_id → components, stroke_count, stroke_svg_data, ...
    radical_id = bộ thủ chính (1:1, tra từ điển)

character_components (bridge 1:N — chữ tách thành nhiều thành phần)
    character_id, component_id
    + Phase 2: position, role, reasoning

word_characters (bridge M:N — từ ↔ chữ đơn)
    vocabulary_id, character_id, position
```

### 4.4 Phân phase (thống nhất từ ai.md + đề xuất trước)

| Phase | Làm gì | Ghi chú |
|---|---|---|
| **MVP** | `radicals/components` + `characters` + bridge `character_radicals/character_components(character_id, radical_id/component_id)` + `word_characters(vocabulary_id, character_id, position)` | Chỉ cần 2 FK columns trên bridge, chưa cần `role`, `position`, `reasoning` |
| **Phase 2** | Thêm `is_primary`/`function_type` (semantic/phonetic/structural/historical) + `position` (left/right/top/bottom) + `reasoning` trên bridge. Thêm bài tập radical picker, lesson theo bộ thủ | Cần khi muốn hiển thị "thành phần này đóng vai trò gì" |
| **Phase 3** | Recursive decomposition (component tách tiếp: 青 = 主 + 月). Etymology. Đồ thị quan hệ | Chỉ cần khi muốn decomposition sâu |

### 4.5 Data source

**makemeahanzi** (GitHub, open-source) có sẵn:
- Decomposition per character (IDS format — chuỗi mô tả cấu trúc chữ)
- Radical per character (214 bộ Khang Hy)
- Stroke SVG data (dữ liệu animation nét viết)
- ~9K characters — đủ cover toàn bộ HSK 1-9

### 4.6 Use cases

| Use case | Bảng dùng |
|---|---|
| User tap chữ "脑" trong flashcard "电脑" → xem bộ thủ + animation nét viết | `word_characters` → `characters` (stroke_svg_data) → `character_radicals` → `radicals` |
| OCR confidence thấp → tìm chữ giống "脑" dựa trên thành phần chung | `character_radicals` → tìm chữ cùng thành phần → candidate list |
| DISC scoring: tính độ giống hình dạng giữa 2 chữ cho OCR post-processing | `characters` (stroke data) + `character_radicals` (shared components) |
| Tìm tất cả từ chứa chữ "电" | `word_characters WHERE character_id = "电"` → list vocabularies |
| Hiển thị "bộ 月 trong 脑 biểu ý thịt/cơ thể" (Phase 2) | `character_radicals` (function_type = semantic, reasoning) + `radicals` (name) |
| HSK 3.0: chữ nào cần viết ở level 3? | `characters WHERE hsk3_writing_level = 3` |
| Bài tập radical picker: ghép 宀 + 女 → 安 (Phase 2) | `character_radicals` → lấy thành phần của chữ → hiện cho user ghép |
| Học theo nhóm bộ thủ: "tất cả chữ liên quan nước" | `character_radicals WHERE radical_id = "氵"` → list characters |

### 4.7 Lưu ý khi áp dụng vào Lu

| Khác biệt | AI.md/PDF | Lu | Cần điều chỉnh |
|---|---|---|---|
| **Multi-language** | Chinese-only (`hsk2_level`, `hsk3_level` hardcode) | Multi-language | `characters` nên có `language_id`. HSK levels nên dùng `proficiency_levels` thay vì hardcode. Tuy nhiên characters + radicals chủ yếu là CJK-specific → cần cân nhắc |
| **FK constraints** | Dùng `REFERENCES` | Không dùng FK ở DB level | Bỏ FK, referential integrity ở application layer |
| **Naming** | `simplified`, `pinyin` (Chinese-specific) | `word`, `phonetic` (generic) | Giữ naming generic của Lu |
| **Existing tables** | Không có `languages`, `categories`, `proficiency_levels` | Đã có | Tận dụng, không tạo lại |

---

## 5. Tóm tắt

| Điểm | Đánh giá | Hành động |
|---|---|---|
| Bảng `characters` + bridge `word_characters` | **Rất đáng xem xét** — cần cho OCR (DISC scoring, chữ giống nhau), stroke animation, HSK 3.0 read/write levels | Cân nhắc thêm vào design |
| Bảng `radicals` + bridge `character_radicals` | **Đáng xem xét** — tra cứu bộ thủ, bài tập ghép chữ, học theo nhóm thành phần. Cả PDF, ai.md, và thảo luận đều đồng ý tách riêng | Cân nhắc thêm vào design |
| Naming: "radicals" vs "components" | **Cần quyết định** — Hướng A (radicals + is_primary) đơn giản hơn, Hướng B (components + radical_id) chính xác hơn | Quyết định trước khi implement |
| `sino_vietnamese` (Âm Hán Việt) | **Quan trọng cho VN market** — killer feature cho Phase 1 | Cần thiết kế field/storage |
| `measure_words` (Lượng từ) | Có thể giữ trong JSONB metadata | Không cần thay đổi |
| FSRS / SRS design | Lu đã tốt hơn nhiều (tách 3 bảng, FSRS mapping rõ ràng) | Giữ nguyên |
| Multi-language | Lu đã tốt hơn nhiều (generic, pluggable) | Giữ nguyên |
