# AI Chat Mode Research — 3 Personas + Vocabulary Tracking

> Research phục vụ thiết kế AI Chat mode (C8 trong `technical_challenges.md`).
>
> **Context:** Pro-only feature. User chat với AI bằng tiếng Trung, AI phát hiện từ đã học → highlight, sửa lỗi ngữ pháp, update Memory Score (weight 2).

---

## 1. Yêu cầu từ PRD (§4.5)

| Element | Spec |
|---|---|
| Interface | Messenger-style. AI bên trái, user bên phải |
| Topic Context | Gắn chủ đề learning card (VD: 点餐 gọi món, 旅行 du lịch) |
| 3 Personas | (1) **朋友** — đơn giản, gần gũi, không quan trọng ngữ pháp. (2) **老板** — từ vựng công việc, trang trọng. (3) **老师** — sửa lỗi ngữ pháp, yêu cầu synonyms cao cấp |
| Vocab tracking | Phát hiện từ đã học → highlight xanh, thả tim + cộng Recall |
| Grammar feedback | Phản hồi tự nhiên, gợi ý sửa lỗi / từ đồng nghĩa |
| Session summary | Words used correctly, New words introduced, XP gained |
| Scoring | Update Memory Score (mode weight 2) |
| Access | Pro only. Free: 1 session/tuần preview |

---

## 2. Architecture

### 2.1 Flow end-to-end

```
Mobile App                    Go Backend                         External
──────────                    ──────────                         ────────

1. User chọn persona + topic
   ↓
2. POST /api/chat/sessions
   { persona: "friend", topic: "food" }
   → Backend tạo session, load user vocab list
   → Return session_id
   ↓
3. POST /api/chat/messages
   { session_id, text: "我想吃中国菜" }
   ↓
                              4. Prompt Assembly:
                                 - System prompt (persona + rules)
                                 - User vocab list (filtered by topic)
                                 - Conversation history (summarized nếu dài)
                                 - User message
                              ↓
                              5. Model Router:
                                 - Simple turn → Haiku/cheap model
                                 - Grammar correction needed → Sonnet
                                 - Complex explanation → Opus
                              ↓
                              6. LLM API call (streaming)     → OpenAI / Anthropic
                              ↓
                              7. Stream response về mobile via SSE
                              ↓
                              8. Post-processing (sau khi stream xong):
                                 - Word segmentation (GSE)
                                 - Vocab detection (match user's word list)
                                 - Synonym check
                                 - Parse corrections từ LLM response
                                 - Save message + metadata to DB
                                 - Update Memory Score
                              ↓
9. Mobile render:
   - Chat bubbles (streaming)
   - Highlight từ đã học (xanh)
   - Show corrections (nếu có)
   ↓
10. Session end → GET /api/chat/sessions/:id/summary
    → Words used, XP gained, corrections made
```

### 2.2 Streaming — SSE (Server-Sent Events)

**Tại sao SSE thay vì WebSocket:**

| | SSE | WebSocket |
|---|---|---|
| Phù hợp | Server → Client streaming (đúng pattern LLM) | Bidirectional (overkill cho chat) |
| Complexity | Đơn giản, HTTP-based | Phức tạp hơn, cần upgrade protocol |
| LLM API alignment | OpenAI/Anthropic natively stream SSE | Cần translate protocol |
| Reconnection | Tự động (built into spec) | Phải tự implement |
| Load balancer | Standard HTTP, không cần config đặc biệt | Cần sticky sessions |
| Mobile support | iOS URLSession + Android okhttp-eventsource | Cả 2 support |

**Go implementation pattern:**

```go
// Handler: POST /api/chat/messages → SSE stream
func (h *ChatHandler) SendMessage(c *gin.Context) {
    // 1. Parse request
    // 2. Assemble prompt
    // 3. Call LLM with streaming

    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    // 4. Stream chunks
    c.Stream(func(w io.Writer) bool {
        chunk, ok := <-llmStream
        if !ok {
            c.SSEvent("done", finalMetadata)
            return false  // end stream
        }
        c.SSEvent("message", chunk)
        return true  // continue
    })
}
```

**Lưu ý quan trọng:**
- Disable Gzip trên SSE routes (Gzip buffer defeats streaming)
- Timeout: LLM generation có thể mất 10-30s → set idle timeout cao
- Heartbeat: gửi `event: ping` mỗi 15s tránh proxy/LB timeout
- Client disconnect: check `c.Request.Context().Done()`

---

## 3. Prompt Engineering — 3 Personas

### 3.1 Multi-layer prompt structure

```
Layer 1 — Core Identity:
  "You are [PersonaName], a [relationship] who speaks Mandarin Chinese."

Layer 2 — Behavioral Rules:
  "Respond in Chinese (Simplified). Include pinyin in parentheses for words
   above user's HSK level. Keep responses under 3 sentences."

Layer 3 — Difficulty Control:
  "User's level: HSK [N]. Use vocabulary from HSK 1-[N].
   When introducing a new word, format: 新词(xīn cí, new word)."

Layer 4 — Correction Style (per persona):
  [Khác nhau — xem 3.2]

Layer 5 — Vocab Context:
  "User has learned these words: [compressed vocab list]
   Prefer using these words. When user uses a learned word correctly,
   acknowledge it naturally."

Layer 6 — Topic:
  "Current conversation topic: [topic]. Keep conversation within this context."
```

### 3.2 Correction style per persona

| Persona | Style | Ví dụ user viết sai "我昨天去学校了学习" |
|---|---|---|
| **朋友 (Friend)** | **Recast** — nhắc lại đúng trong câu trả lời, không chỉ ra lỗi | "哦你昨天去学校**学习了**啊！学了什么？" (nhắc lại đúng, tự nhiên) |
| **老板 (Boss)** | **Clarification** — giả vờ không hiểu nếu sai quá, bắt user tự sửa | "你是说你昨天去学校学习了吗？请说清楚一点。" |
| **老师 (Teacher)** | **Explicit** — chỉ ra lỗi + giải thích rule + ví dụ đúng | "注意：'了' should come after the verb, not after the location. ✅ 我昨天去学校学习**了**。Rule: S + 去 + Place + V + 了" |

### 3.3 Ví dụ system prompt (Friend persona)

```
You are 小明 (Xiǎo Míng), a friendly Chinese college student chatting with a
Vietnamese friend who is learning Chinese. You speak casual Mandarin.

RULES:
- Respond in Simplified Chinese, 1-3 sentences per turn
- User's level: HSK 2. Use HSK 1-2 vocabulary primarily
- For words above HSK 2, add pinyin + Vietnamese meaning: 比如(bǐ rú, ví dụ)
- Keep tone casual and encouraging. Use emoji occasionally 😊
- If user makes a grammar mistake, naturally rephrase it correctly in your
  response WITHOUT explicitly pointing it out
- If user seems stuck, offer 2-3 simple response options

VOCABULARY CONTEXT:
User has learned: 你好,谢谢,高兴,学习,老师,吃饭,中国,今天,明天,很,
不,也,都,在,有,...[topic-filtered list]

When user correctly uses a learned word, respond naturally (no special marking —
the app handles highlighting).

TOPIC: 饮食 (Food & Drink) — Talk about food preferences, ordering, cooking.

STRUCTURED OUTPUT:
After your conversational response, output a JSON block:
{"corrections": [...], "vocab_used": [...], "new_words": [...]}
```

### 3.4 Caveats

- **LLM không 100% tuân theo HSK level.** Vocabulary generated thường khó hơn specified level. → Server-side validation: segment response → check words against HSK dictionary → flag words above user level
- **Persona consistency drift.** Qua nhiều turns, LLM có thể "quên" persona. → Include persona reminder trong mỗi message (không chỉ system prompt)
- **Duolingo approach:** Dùng **nhiều prompt nhỏ specialized** thay vì 1 prompt lớn. VD: 1 prompt cho tạo câu hỏi, 1 prompt cho trả lời, 1 prompt cho feedback. Mỗi prompt hyper-focused → ít hallucination

---

## 4. Vocabulary tracking trong conversation

### 4.1 Bài toán

User đã học 300 từ. AI cần:
- (a) Ưu tiên dùng từ user đã học trong response
- (b) Phát hiện khi user dùng từ đã học → highlight + cộng điểm
- (c) Giới thiệu từ mới dần dần

### 4.2 Token-efficient vocab injection

300 Chinese words → bao nhiêu tokens?

| Format | Tokens ước tính | Ví dụ |
|---|---|---|
| **Dense** (word/translation) | ~600-900 tokens | `你好/hello,谢谢/thanks,高兴/happy,...` |
| **Full** (word, pinyin, translation, example) | ~3,000-4,500 tokens | Quá lớn |
| **Category-filtered** (chỉ từ liên quan topic) | ~100-200 tokens | 30-50 từ về food & drink |

**Recommend: Category-filtered + top frequent**

```
Always in context (~150 tokens):
  - User's HSK level
  - Top 50 most recently learned words (high frequency)

Topic-filtered (~100 tokens):
  - 30-50 words related to current conversation topic

Never in prompt:
  - Full 300-word list → query server-side cho detection
```

Tổng vocab context: ~250 tokens. Với 128K context window → < 0.2% capacity.

### 4.3 Prompt caching — Giảm cost 90%

System prompt (persona + rules + vocab list) **ít thay đổi giữa các turns**. Đặt static content ở đầu prompt → maximize cache hit.

| Provider | Caching | Tiết kiệm | Điều kiện |
|---|---|---|---|
| **Anthropic** | Explicit cache breakpoints (max 4) | 90% input tokens | Cache TTL 5 phút, extend on hit |
| **OpenAI** | Tự động cho prompts ≥ 1,024 tokens | 50% input tokens | Không cần config |

**Prompt structure tối ưu cho caching:**

```
[CACHED — ít thay đổi]
  System prompt (persona definition)
  Behavioral rules
  Vocab list
  Topic context

[DYNAMIC — thay đổi mỗi turn]
  Conversation history (last N turns)
  User's current message
```

### 4.4 Conversation history management

| Strategy | Khi nào | Tiết kiệm |
|---|---|---|
| **Full history** | < 10 turns | Giữ nguyên tất cả messages |
| **Summarize** | ≥ 10 turns | Summarize turns 1-7, giữ nguyên 3 turns gần nhất. Giảm ~60-80% tokens |
| **Sliding window** | ≥ 20 turns | Chỉ giữ 5 turns gần nhất + summary of earlier. Giảm ~90% |

Summarization: dùng cheap model (Haiku) tạo 2-3 câu tóm tắt conversation so far. Include trong system prompt thay vì full history.

---

## 5. Vocabulary detection + Synonym matching

### 5.1 Chinese word segmentation trong Go

**[GSE (go-ego/gse)](https://github.com/go-ego/gse):**

| | Chi tiết |
|---|---|
| Algorithm | Double-Array Trie + shortest path (word frequency + DAG) + HMM |
| Performance | 9.2 MB/s single-threaded, 26.8 MB/s concurrent |
| Custom dictionary | Có — load HSK word list làm custom dict |
| License | Apache 2.0 |

```go
import "github.com/go-ego/gse"

seg := gse.New()
seg.LoadDict()  // default dict
// Load custom HSK dict: seg.LoadDict("hsk_words.txt")

result := seg.Cut("我很高兴认识你", true)
// → ["我", "很", "高兴", "认识", "你"]
```

Alternative: **gojieba** (Go binding for jieba).

### 5.2 Detection pipeline

```
User message: "我很开心"
  │
  ▼
[Word Segmentation — GSE]
  → ["我", "很", "开心"]
  │
  ▼
[Exact Match — against user's learned vocab set]
  → "我" ✅ matched (HSK1)
  → "很" ✅ matched (HSK1)
  → "开心" ❌ not in learned list
  │
  ▼
[Synonym Lookup — static synonym table]
  → "开心" synonyms: ["高兴", "快乐", "愉快"]
  → "高兴" IS in user's learned list
  → Result: synonym match
  │
  ▼
[Output]
  {
    "words_used": [
      { "word": "我", "status": "exact_match" },
      { "word": "很", "status": "exact_match" },
      { "word": "开心", "status": "synonym_match", "learned_word": "高兴" }
    ]
  }
```

### 5.3 Synonym sources

| Source | Mô tả | Phù hợp |
|---|---|---|
| **CiLin (同义词词林)** | Chinese synonym dictionary, hierarchical. Words sharing fine-grained code = synonyms | Tốt nhất cho Chinese synonyms. ~70K words |
| **CC-CEDICT** | Extract synonym groups từ shared English definitions | Đơn giản, đã có data. Accuracy thấp hơn CiLin |
| **Chinese Word Vectors** | Pre-trained embeddings → cosine similarity > 0.7 = synonym | Cần load large model. Fallback cho unknown words |

**Recommend MVP: Static synonym table** từ CiLin hoặc CC-CEDICT. Store trong Postgres. 5K-10K synonym groups = simple DB lookup, không cần ML infrastructure.

### 5.4 Scoring cho vocab usage

| Match type | Credit | Feedback cho user |
|---|---|---|
| **Exact match** | Full (cộng Memory Score) | Highlight xanh + tim |
| **Synonym match** | Partial (50% credit) | "Nice! 开心 means the same as 高兴 which you learned. Both mean happy!" |
| **New word from AI** | 0 (không tính, nhưng log) | Mobile highlight vàng "New word" |

---

## 6. Cost control

### 6.1 Model routing

Không phải mọi message cần model đắt. Classify message complexity → route:

| Loại message | % traffic | Model | Cost/msg |
|---|---|---|---|
| **Simple** — greetings, short replies, basic questions | ~70% | Haiku 4.5 ($1/$5 per 1M tokens) | ~$0.0008 |
| **Medium** — grammar correction needed, longer response | ~20% | Sonnet 4.5 ($3/$15 per 1M tokens) | ~$0.0023 |
| **Complex** — detailed grammar explanation, cultural context | ~10% | Opus 4.6 ($5/$25 per 1M tokens) | ~$0.0039 |

**Classification heuristic (server-side, trước khi gọi LLM):**
- User message < 10 chars + no grammar error detected → Simple
- User message có grammar pattern phức tạp (把/被/了/过) → Medium
- User explicitly hỏi "tại sao?" hoặc request explanation → Complex

### 6.2 Budget estimation — 50K MAU

```
Assumptions:
  - 30% daily active rate = 15K DAU
  - 5 messages/user/day average
  - Prompt caching: 90% hit rate (Anthropic) → effective input cost = 10% list price

Daily:
  - 15K × 5 = 75K messages/ngày
  - 70% Haiku: 52.5K × $0.0008  = $42
  - 20% Sonnet: 15K × $0.0023   = $35
  - 10% Opus:   7.5K × $0.0039  = $29

Daily total:  ~$106
Monthly total: ~$3,200
```

**Không optimize:** mọi message dùng Sonnet → 75K × $0.0023 = $172/ngày = **$5,200/tháng**. Model routing tiết kiệm ~40%.

### 6.3 Các kỹ thuật tiết kiệm khác

| Kỹ thuật | Tiết kiệm | Chi tiết |
|---|---|---|
| **Prompt caching** | ~90% input tokens | Static system prompt + vocab ở đầu. Dynamic history ở cuối |
| **Conversation summarization** | ~60-80% history tokens | Sau 10 turns: summarize earlier turns bằng Haiku |
| **Output token limit** | ~30-50% output tokens | `max_tokens: 200-300`. Chinese rất dense — 100 ký tự = nhiều thông tin |
| **Model routing** | ~40% so với single model | Classify complexity → route cheap/expensive |
| **Batch feedback** | ~50% cho non-realtime work | Post-conversation analysis (session summary, detailed scoring) dùng batch API |

---

## 7. Grammar feedback

### 7.1 LLM performance trên Chinese grammar

Từ nghiên cứu PLOS ONE (evaluating LLMs on Chinese GEC):
- ERNIE-4.0: F0.5 = 55.10 (tốt nhất)
- GPT-4: F0.5 = 43.23
- LLMs lag behind specialized GEC models
- **Vấn đề phổ biến:** LLMs tend to **overcorrect** — thêm từ không cần thiết, formal hóa quá mức

### 7.2 Hybrid approach (Recommend)

Không rely 100% vào LLM cho grammar detection:

```
Layer 1: LLM generates response + inline corrections (80% cases)
Layer 2: Rule-based checks server-side cho high-confidence rules (了/过 placement, 把 structure)
Layer 3: Log corrections → human review → build training data over time
```

### 7.3 Structured output từ LLM

Mỗi LLM response trả cả conversational text và structured metadata:

```json
{
  "response": "哦你想吃中国菜！你最喜欢什么菜？",
  "corrections": [
    {
      "original": "我想吃中国的菜",
      "corrected": "我想吃中国菜",
      "rule": "中国菜 is a compound noun, no 的 needed",
      "severity": "minor"
    }
  ],
  "vocab_used": ["想", "吃", "中国"],
  "new_words_introduced": [
    { "word": "最", "pinyin": "zuì", "meaning": "most" }
  ]
}
```

Mobile parse JSON → render:
- `response` → chat bubble (streamed)
- `corrections` → inline highlight hoặc feedback block (tùy persona)
- `vocab_used` → highlight xanh
- `new_words_introduced` → highlight vàng "New"

---

## 8. So sánh các app tương tự

| App | AI Chat feature | Personas | Vocab tracking | Grammar feedback | Model |
|---|---|---|---|---|---|
| **Duolingo Max** | Roleplay — open-ended conversation practice | Scenario-based (không persona cố định) | Tích hợp với CEFR level | Post-conversation feedback | GPT-4 |
| **Speak** | AI tutor — voice + text | Adaptive (1 tutor, adjust style) | Real-time curriculum adaptation | Inline + detailed explanation | OpenAI realtime API |
| **HelloChinese** | AI conversation practice (VIP) | Không rõ | HSK level-based | Basic correction | Không công bố |
| **SuperChinese** | AI tutor chat | 1 tutor persona | Level-based vocab | Grammar tips | Không công bố |
| **ChatGPT/Claude trực tiếp** | Không — user tự prompt | Không | Không (user phải tự inject) | Tốt nhưng không consistent | GPT-4/Claude |

**Nhận xét:**
- **Duolingo** là reference tốt nhất: multi-prompt per turn, post-conversation feedback, CEFR-aligned
- **Speak** là reference cho voice + real-time: dùng OpenAI realtime API
- Không app nào có **3 personas + vocab highlight + synonym detection** đồng thời → đây là USP

---

## 9. Backend scope — Gì cần build

| Component | Effort | Chi tiết |
|---|---|---|
| **Chat session management** | 1 ngày | Create session, store history, session summary |
| **Prompt assembler** | 2 ngày | Multi-layer prompt, persona templates, vocab injection, topic filtering, history summarization |
| **LLM API client + streaming** | 2 ngày | SSE proxy, stream chunks, handle timeouts, heartbeat |
| **Model router** | 1 ngày | Classify message complexity → pick model |
| **Post-processor** | 2 ngày | Word segmentation (GSE), vocab detection, synonym match, parse LLM structured output |
| **Scoring API** | 0.5 ngày | Update Memory Score (weight 2) per vocab word used |
| **Session summary** | 0.5 ngày | Aggregate: words used, XP gained, corrections |
| **Synonym table** | 1 ngày | Parse CiLin/CC-CEDICT → Postgres table. Endpoint cho lookup |
| **Total** | **~10 ngày** | |

### DB tables cần thêm

```sql
-- Chat sessions
CREATE TABLE chat_sessions (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,
    persona     VARCHAR(20) NOT NULL,   -- 'friend', 'boss', 'teacher'
    topic       VARCHAR(100),
    status      VARCHAR(20) NOT NULL,   -- 'active', 'completed'
    started_at  TIMESTAMPTZ DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    summary     JSONB                    -- post-session: { words_used, xp_gained, corrections }
);

-- Chat messages
CREATE TABLE chat_messages (
    id              UUID PRIMARY KEY,
    session_id      UUID NOT NULL,
    role            VARCHAR(10) NOT NULL,   -- 'user', 'assistant'
    content         TEXT NOT NULL,
    metadata        JSONB,                   -- { corrections, vocab_used, new_words, model_used, tokens }
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id, created_at);

-- Synonym table
CREATE TABLE synonyms (
    id          UUID PRIMARY KEY,
    word        VARCHAR(20) NOT NULL,
    synonym     VARCHAR(20) NOT NULL,
    source      VARCHAR(20),            -- 'cilin', 'cedict', 'manual'
    UNIQUE(word, synonym)
);
CREATE INDEX idx_synonyms_word ON synonyms(word);
```

---

## References

| Source | URL | Relevance |
|---|---|---|
| Duolingo Max — GPT-4 Integration | https://blog.duolingo.com/duolingo-max/ | Multi-prompt per turn, CEFR alignment |
| Duolingo Case Study (OpenAI) | https://openai.com/index/duolingo/ | Architecture + Birdbrain LLM |
| Speak — AI Language Tutor | https://openai.com/index/speak-connor-zwick/ | Voice + realtime API |
| GSE — Go Segmentation Engine | https://github.com/go-ego/gse | Chinese word segmentation in Go |
| Chinese Word Vectors | https://github.com/Embedding/Chinese-Word-Vectors | Pre-trained embeddings cho synonym |
| Evaluating LLMs on Chinese GEC | https://pmc.ncbi.nlm.nih.gov/articles/PMC11524451/ | LLM grammar correction performance |
| GrammarGPT | https://arxiv.org/abs/2307.13923 | Chinese GEC with LLMs |
| SSE vs WebSocket for LLM | https://compute.hivenet.com/post/llm-streaming-sse-websockets | Tại sao SSE cho streaming |
| Go SSE with Gin | https://gist.github.com/SubCoder1/3a700149b2e7bb179a9123c6283030ff | Implementation pattern |
| Prompt Caching | https://www.prompthub.us/blog/prompt-caching-with-openai-anthropic-and-google-models | 90% cost reduction |
| LLM Token Optimization | https://redis.io/blog/llm-token-optimization-speed-up-apps/ | Context management strategies |
| Role Prompting Guide | https://learnprompting.org/docs/advanced/zero_shot/role_prompting | Persona prompt design |
| Build AI Language Learning App | https://sapient.pro/blog/how-to-build-ai-based-language-learning-app | Architecture overview |
