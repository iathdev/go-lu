# Flashcard & Spaced Repetition — Research

## 1. Tổng quan các thuật toán Spaced Repetition

### 1.1 SM-2 (SuperMemo, 1987)

Thuật toán nền tảng, được sử dụng rộng rãi nhất. Theo dõi 3 biến per-card:

- **Ease Factor (EF)**: Khởi tạo 2.5, điều khiển tốc độ tăng interval. Min 1.3.
- **Repetition Number (n)**: Số lần recall thành công liên tiếp (grade >= 3).
- **Interval (I)**: Số ngày đến lần review tiếp.

**Rating scale**: 0–5 (0 = không nhớ gì, 5 = nhớ hoàn hảo).

**Tính interval**:
- quality >= 3 (thành công): n=0 → 1 ngày; n=1 → 6 ngày; n>1 → interval trước × EF
- quality < 3 (thất bại): reset n=0, interval về 1 ngày

**Cập nhật EF**:
```
EF' = EF + (0.1 - (5 - q) × (0.08 + (5 - q) × 0.02))
```

**Ưu điểm**: Đơn giản, dễ implement, đã proven qua decades.
**Nhược điểm**: Dễ rơi vào "ease hell" (EF giảm dần không hồi phục), không personalize.

### 1.2 Anki Modified SM-2

Anki cải tiến SM-2 với:
- **4 nút thay vì 6**: Again, Hard, Good, Easy
- **Again**: Reset card, giảm ease 20%, interval × "new interval" %
- **Hard**: Giảm ease 15%, interval × 1.2
- **Good**: Interval × ease, không đổi ease
- **Easy**: Interval × ease × easy bonus, tăng ease 15%
- **Learning steps**: Configurable thay vì cố định 1d/6d
- **Late review bonus**: Card quá hạn nhưng vẫn nhớ → interval bonus
- Từ v23.10, Anki hỗ trợ FSRS như alternative scheduler

### 1.3 FSRS (Free Spaced Repetition Scheduler) — State of the Art

Thuật toán hiện đại, data-driven, dựa trên mô hình DSR của MaiMemo. **Giảm 20–30% số lần review** so với SM-2 cùng mức retention.

**3 biến core**:
- **Stability (S)**: Interval mà retrievability = 90%. Càng cao → quên càng chậm.
- **Difficulty (D)**: Range [1, 10]. Độ khó của material.
- **Retrievability (R)**: Xác suất recall tại thời điểm t.

**Forgetting curve**:
```
R(t, S) = (1 + F × t/S)^C     // F = 19/81, C = -0.5 (trainable trong FSRS-6)
```

**Tính interval**:
```
I(r, S) = (S/F) × (r^(1/C) - 1)   // Khi r = 0.9, I = S
```

**Stability sau review thành công**:
```
S' = S × (1 + e^w8 × (11-D) × S^(-w9) × (e^(w10×(1-R)) - 1) × hard_penalty × easy_bonus)
```

**Stability sau lapse**:
```
S' = w11 × D^(-w12) × ((S+1)^w13 - 1) × e^(w14×(1-R))
```

**Go library**: `github.com/open-spaced-repetition/go-fsrs` (production-ready)

```go
type Card struct {
    Due, LastReview    time.Time
    Stability          float64
    Difficulty         float64
    ElapsedDays        uint64
    ScheduledDays      uint64
    Reps, Lapses       uint64
    State              State  // New=0, Learning=1, Review=2, Relearning=3
}

type Rating int8  // Again=1, Hard=2, Good=3, Easy=4

// Trả về scheduling cho cả 4 rating cùng lúc
func (p *Parameters) Repeat(card Card, now time.Time) map[Rating]SchedulingInfo
```

**FSRS-6** (latest): 21 trainable weights, có thể optimize per-user qua ML trên review history.

**Ví dụ sequence "Good"**: 0 → 4 → 14 → 44 → 125 → 328 ngày.

### 1.4 Leitner System

Đơn giản nhất, dùng hệ thống "hộp":
- 5 hộp, interval tăng dần (1d → 2d → 4d → 8d → 16d)
- Đúng → tiến 1 hộp; Sai → về hộp 1
- Binary assessment (biết/không biết)

**Chỉ phù hợp cho MVP đơn giản**, không khuyến nghị cho production.

### 1.5 So sánh

| Tiêu chí | SM-2 | Anki SM-2 | FSRS | Leitner |
|---|---|---|---|---|
| Độ phức tạp implement | Thấp | Trung bình | Cao | Rất thấp |
| Hiệu quả review | Trung bình | Khá | Tốt nhất (-20-30%) | Kém |
| Personalization | Không | Ít | Cao (trainable) | Không |
| Go library | Tự implement | Tự implement | `go-fsrs` ✅ | Tự implement |
| Adoption | Legacy | Rất phổ biến | Đang tăng nhanh | Giáo dục cơ bản |

---

## 2. Phân tích hệ thống Flashcard hiện có

### 2.1 Anki (Open Source — Gold Standard)

**Data model**:
- **Notes**: Container nội dung, nhiều fields (word, meaning, example...). 1 note → N cards qua templates.
- **Cards**: Unit review, track state/queue/due/interval/ease/reps/lapses.
- **Review Log (revlog)**: Immutable audit trail, 1 row per review.
- **Decks**: Hierarchical (e.g., "Japanese::Vocabulary::N3").

**Card states & queues**:
```
type = 0 (new), 1 (learning), 2 (review), 3 (relearning)
queue = -3 (buried/v2), -2 (buried/v1), -1 (suspended), 0 (new), 1 (learning), 2 (review), 3 (day-learn)
```

**Key index**: `ix_cards_sched ON cards(did, queue, due)` — critical cho fetching due cards.

**AnkiConnect API**: 100+ actions qua HTTP, bao gồm deck/card CRUD, answer cards, stats, media management.

### 2.2 WaniKani (Japanese/CJK Learning — Tương đồng nhất với Lu)

**9 SRS stages trong 5 groups**:
- Apprentice 1–4: 4h, 8h, 1d, 2d
- Guru 1–2: 1 tuần, 2 tuần
- Master: 1 tháng
- Enlightened: 4 tháng
- Burned: permanently retired (tương đương Mastered trong Lu)

**API v2** (RESTful, well-documented):
- `GET /v2/subjects` — radicals, kanji, vocabulary với meanings, readings, level
- `GET /v2/assignments` — user progress per subject: `srs_stage` (0–9), timestamps
- `POST /v2/reviews` — submit review: incorrect meaning/reading counts → server tính updated SRS stage
- `GET /v2/review_statistics` — correct/incorrect counts, streaks
- `GET /v2/spaced_repetition_systems` — SRS stage definitions

**Điểm nổi bật**: Progression radical → kanji → vocabulary, tương tự Lu (radical decomposition → hanzi → vocabulary).

### 2.3 Memrise

- **Fixed SRS intervals**: 4h → 12h → 24h → 6d → 12d → 48d → 96d → 6 tháng
- Sai → reset về 4h
- Course-based, nhấn mạnh native speaker video clips
- Đơn giản nhưng không flexible

### 2.4 Brainscape

- **Confidence-Based Repetition (CBR)**: 5-level self-rating (1–5)
- Low-confidence items lặp lại nhiều hơn; high-confidence spaced xa hơn
- Không public API

### 2.5 Quizlet

- Binary "know" / "don't know"
- Multiple study modes: Learn, Test, Flashcards, Match (game)
- Focus collaborative learning, không SRS thực sự

---

## 3. Data Model chuẩn cho Flashcard System

### 3.1 Card State Machine

```
    [New] ──study──▶ [Learning] ──graduate──▶ [Review]
                          ▲                       │
                          │                    (lapse)
                          │                       ▼
                          └──────────── [Relearning]

    Any state ──▶ [Suspended]  (user action)
    Any state ──▶ [Buried]     (scheduler/user, resets daily)
```

### 3.2 Core Entities

**Card / Flashcard** (unit review):
| Field | Type | Mô tả |
|---|---|---|
| id | UUID | PK |
| deck_id | UUID | FK to deck |
| vocabulary_id | UUID | FK to vocabulary (nếu link với vocabulary module) |
| state | enum | New, Learning, Review, Relearning |
| due | timestamp | Thời điểm review tiếp |
| stability | float64 | FSRS stability |
| difficulty | float64 | FSRS difficulty [1, 10] |
| elapsed_days | uint | Ngày từ lần review trước |
| scheduled_days | uint | Interval đã schedule |
| reps | uint | Tổng số lần review |
| lapses | uint | Số lần quên (Review → Relearning) |
| last_review | timestamp | Lần review gần nhất |
| suspended | bool | User tạm ẩn card |

**Deck** (collection of cards):
| Field | Type | Mô tả |
|---|---|---|
| id | UUID | PK |
| user_id | UUID | FK to user |
| name | string | Tên deck |
| description | string | Mô tả |
| new_cards_per_day | int | Giới hạn card mới/ngày (default 20) |
| review_cards_per_day | int | Giới hạn review/ngày (default 200) |

**Review Log** (immutable audit trail):
| Field | Type | Mô tả |
|---|---|---|
| id | UUID | PK |
| card_id | UUID | FK to card |
| user_id | UUID | FK to user |
| rating | enum | Again=1, Hard=2, Good=3, Easy=4 |
| state | enum | State trước review |
| scheduled_days | uint | Interval đã schedule |
| elapsed_days | uint | Ngày thực tế từ lần review trước |
| duration_ms | int | Thời gian user xem card (ms) |
| reviewed_at | timestamp | Thời điểm review |

**Study Session** (optional aggregate):
| Field | Type | Mô tả |
|---|---|---|
| id | UUID | PK |
| user_id | UUID | FK to user |
| deck_id | UUID | FK to deck |
| started_at | timestamp | Bắt đầu session |
| ended_at | timestamp | Kết thúc session |
| cards_studied | int | Tổng cards đã review |
| new_count | int | Số card mới |
| correct_count | int | Số lần đúng |
| incorrect_count | int | Số lần sai |

**User SRS Settings** (per-user configuration):
| Field | Type | Mô tả |
|---|---|---|
| user_id | UUID | PK, FK to user |
| desired_retention | float64 | Mức retention mong muốn (default 0.9) |
| max_interval | int | Interval tối đa (default 365 ngày) |
| day_start_hour | int | Giờ bắt đầu ngày mới (default 4 = 4:00 AM) |
| timezone | string | User timezone |
| fsrs_weights | float64[] | 21 FSRS weights (nullable, dùng default nếu null) |

### 3.3 Key Index

```sql
CREATE INDEX idx_card_scheduling ON flashcards(user_id, state, due);
```

---

## 4. API Patterns

### 4.1 Typical REST Endpoints

**Deck Management**:
```
POST   /api/decks                    -- Tạo deck
GET    /api/decks                    -- List decks (with card counts, due counts)
GET    /api/decks/:id                -- Chi tiết deck + stats
PUT    /api/decks/:id                -- Update deck
DELETE /api/decks/:id                -- Xóa deck
```

**Card Management**:
```
POST   /api/decks/:deck_id/cards     -- Thêm card(s) vào deck
GET    /api/decks/:deck_id/cards     -- List cards (paginated, filter by state)
PUT    /api/cards/:id                -- Update card content
DELETE /api/cards/:id                -- Xóa card
POST   /api/cards/:id/suspend        -- Tạm ẩn card
POST   /api/cards/:id/unsuspend      -- Bỏ ẩn card
```

**Study Session Flow**:
```
GET    /api/decks/:deck_id/study     -- Bắt đầu study: trả batch due cards (new + review + relearning)
POST   /api/reviews                  -- Submit review: { card_id, rating, duration_ms }
GET    /api/decks/:deck_id/study/summary  -- Tóm tắt session
```

**Statistics**:
```
GET    /api/stats/reviews?period=30d         -- Review counts theo thời gian
GET    /api/decks/:deck_id/stats             -- Deck-level stats
GET    /api/stats/forecast                   -- Dự báo review sắp tới
GET    /api/stats/heatmap?year=2026          -- Review heatmap (like GitHub contributions)
```

### 4.2 Study Session Flow (Chi tiết)

```
1. Client: GET /api/decks/:id/study
   Server: Query due cards + new cards (respect daily limits)
   Response: { cards: [...], counts: { new: 5, learning: 3, review: 12 } }

2. Client: Hiển thị card front → User flip → User chọn rating
   Client: POST /api/reviews { card_id, rating: "good", duration_ms: 4500 }
   Server: FSRS.Next(card, now, rating) → save updated card + review log
   Response: { card: { next_due, state, interval }, remaining: { new: 4, learning: 3, review: 11 } }

3. Repeat cho đến hết batch hoặc user dừng

4. Client: GET /api/decks/:id/study/summary
   Response: { studied: 20, correct: 16, accuracy: 0.8, time_spent_ms: 180000 }
```

### 4.3 Design Decisions quan trọng

1. **Server-side scheduling**: Server tính tất cả interval và due dates. Client chỉ submit rating. Đảm bảo consistency across devices, chống cheat.
2. **Batch card fetching**: Trả 10–20 cards/batch thay vì 1 card/request → giảm round trips.
3. **Idempotent reviews**: Dùng client-generated review ID hoặc timestamp để prevent duplicate submissions khi network retry.
4. **Daily limits server-side**: Per-deck, rolling 24-hour window, configurable per user.
5. **Scheduling info in response**: Khi submit review → trả updated card (next_due, interval, state) để client hiển thị ngay không cần re-fetch.

---

## 5. Best Practices cho Language Learning Flashcards

### 5.1 Multi-Field Card Structure

Vocabulary card khác với flashcard thông thường ở chỗ cần nhiều field:

| Field | Mô tả | Ví dụ |
|---|---|---|
| word | Từ target language | 你好 |
| reading/phonetic | Phiên âm | nǐ hǎo |
| part_of_speech | Loại từ | greeting |
| meanings | Nghĩa (multi-language) | Xin chào |
| example_sentences | Câu ví dụ + dịch | 你好，我叫小明。 |
| audio_url | Audio phát âm | /audio/nihao.mp3 |
| image_url | Hình minh họa | /images/greeting.jpg |
| notes | Ghi chú cá nhân (mnemonic) | Giống "nỉ háo" |
| tags | Phân loại | HSK1, daily-life |

### 5.2 Multi-Direction Cards (Reverse Cards)

Từ 1 vocabulary entry, tạo nhiều card types:

| Direction | Front | Back | Độ khó | Khi nào dùng |
|---|---|---|---|---|
| Recognition (L2→L1) | 你好 | Xin chào | Dễ | Mặc định, luôn tạo |
| Production (L1→L2) | Xin chào | 你好 | Khó | Thêm sau khi Recognition ok |
| Listening | 🔊 Audio | 你好 + Xin chào | Trung bình | Khi có audio |
| Reading→Phonetic | 你好 | nǐ hǎo | Trung bình | CJK languages |
| Sentence Cloze | 我叫___，你好。 | 小明 | Khó | Nâng cao |

**Quan trọng**: Mỗi direction là card riêng với **scheduling độc lập** vì nhận diện từ và sản xuất từ là 2 kỹ năng khác nhau.

### 5.3 Recommendations

- **1 concept per card**: Không nhồi nhiều từ/grammar vào 1 card
- **Context over translation**: Hình ảnh + câu ví dụ hiệu quả hơn dịch word-to-word
- **Start with recognition**: Giảm cognitive load ban đầu, thêm production sau
- **Include audio**: Essential cho language learning; TTS backup nếu không có audio gốc
- **Tag by topic/level**: Cho phép study filtered (e.g., "chỉ học HSK3")
- **Radical decomposition**: 40–60% faster learning khi hiểu cấu trúc bộ thủ (đã có trong requirement Lu)

---

## 6. Timezone & Daily Limits

### 6.1 Timezone Handling

- **Lưu tất cả timestamps UTC** trong database
- **"Day start" offset**: Cho user config giờ bắt đầu ngày mới (e.g., 4:00 AM local)
- **Tính day boundary**: `day_start = today_midnight_in_user_tz + offset`, convert sang UTC cho queries
- **FSRS library dùng UTC**: Luôn pass UTC times cho `Repeat()`/`Next()`
- **Due date cho review cards**: Lưu date (không datetime) khi interval >= 1 ngày; dùng epoch seconds cho intra-day learning steps

### 6.2 Daily Limits

- Lưu `new_cards_per_day` và `review_cards_per_day` trong deck/user settings
- Đếm today's reviews từ review_log: `WHERE reviewed_at >= day_start`
- Day boundary: user-configurable offset
- Anki defaults: 20 new/day, 200 review/day

---

## 7. Hệ thống tham khảo

| Platform | Algorithm | Đặc điểm nổi bật | API |
|---|---|---|---|
| Anki | SM-2 / FSRS | Open source, gold standard, 100+ API actions | AnkiConnect |
| WaniKani | Custom 9-stage | CJK-specific, radical progression | REST v2 |
| Memrise | Fixed intervals | Video clips, course-based | Không public |
| Brainscape | CBR | Confidence-based, 5-level rating | Không public |
| Quizlet | Không SRS thật | Game modes, collaborative | Deprecated |
| Mochi Cards | FSRS | Modern API, template-based | REST |
| sr-cards-api | FSRS | Open source TypeScript reference | REST |

---

## Sources

- [SM-2 Algorithm Explained](https://tegaru.app/en/blog/sm2-algorithm-explained)
- [Anki SM-2 Algorithm](https://faqs.ankiweb.net/what-spaced-repetition-algorithm)
- [FSRS Algorithm Wiki](https://github.com/open-spaced-repetition/awesome-fsrs/wiki/The-Algorithm)
- [go-fsrs GitHub](https://github.com/open-spaced-repetition/go-fsrs)
- [Implementing FSRS in 100 Lines](https://borretti.me/article/implementing-fsrs-in-100-lines)
- [AnkiDroid Database Structure](https://github.com/ankidroid/Anki-Android/wiki/Database-Structure)
- [WaniKani API Reference](https://docs.api.wanikani.com/20170710/)
- [WaniKani SRS Stages](https://knowledge.wanikani.com/wanikani/srs-stages/)
- [Mochi Cards API](https://mochi.cards/docs/api/)
- [sr-cards-api (FSRS REST API)](https://github.com/drewsamsen/sr-cards-api)
- [ABC of FSRS](https://github.com/open-spaced-repetition/awesome-fsrs/wiki/ABC-of-FSRS)
- [Flashcard Best Practices - Migaku](https://migaku.com/blog/language-fun/flashcard-best-practices-language-learning)
