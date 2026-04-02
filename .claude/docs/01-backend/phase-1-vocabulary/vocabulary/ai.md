Đây là bản tài liệu **hoàn chỉnh** đã được viết lại và cập nhật theo góp ý mới của bạn: tách riêng `radicals`, thêm bảng nối `character_radicals`, và định hướng sẵn cho các bài tập kiểu phân rã/ghép bộ thủ như màn hình bạn gửi. Thiết kế này cũng giữ nguyên hướng đi offline-first với Flutter + Supabase + PowerSync và SRS bằng FSRS mà chúng ta đã chốt trước đó. [powersync](https://www.powersync.com/blog/flutter-tutorial-building-an-offline-first-chat-app-with-supabase-and-powersync)

## Tổng quan

### Mục tiêu sản phẩm
Ứng dụng là nền tảng học từ vựng tiếng Trung cho người học HSK, ưu tiên thị trường Việt Nam trước và mở rộng sang Thái Lan sau, với trọng tâm là từ vựng, cấu tạo chữ Hán, ví dụ ngữ cảnh, media flashcard, và lộ trình ôn tập theo spaced repetition. [github](https://github.com/open-spaced-repetition/dart-fsrs)
Sản phẩm cần hỗ trợ đồng thời HSK 3.0 và mapping ngược sang HSK 2.0, vì cùng một từ có thể cần xuất hiện theo nhiều khung chương trình học khác nhau. [github](https://github.com/drkameleon/complete-hsk-vocabulary)

### Định hướng kỹ thuật
Kiến trúc phù hợp nhất cho app này là offline-first: dữ liệu học và trạng thái ôn tập được đọc/ghi cục bộ trên máy để trải nghiệm mượt, còn đồng bộ nền được xử lý qua lớp sync giữa SQLite cục bộ và PostgreSQL trên cloud. [youtube](https://www.youtube.com/watch?v=hAfnkw4PEM4)
Với Flutter, tổ hợp Supabase + PowerSync là hướng triển khai hợp lý vì vừa có auth, storage, Postgres, vừa có cơ chế sync nền cho mobile. [supabase](https://supabase.com/docs/guides/getting-started/tutorials/with-flutter)

## Phạm vi dữ liệu

### Đơn vị dữ liệu cốt lõi
Database không nên chỉ coi “từ” là đơn vị duy nhất, vì tiếng Trung có nhiều lớp thông tin khác nhau: từ vựng, từng chữ Hán, bộ thủ, nghĩa, ví dụ, media, và tiến độ ôn tập của từng người dùng. [hanzistroke](https://www.hanzistroke.com/hsk)
Vì vậy mô hình đúng nên tách ít nhất các nhóm sau: `vocabulary`, `characters`, `radicals`, `definitions`, `examples`, `media`, `user_srs_reviews`, cùng các bảng nối để mô tả qua
## Tổng quan
### Mục tiêu sản phẩm
Ứng dụng là nền tảng học từ vựng tiếng Trung cho người học HSK, ưu tiên thị trường Việt Nam trước và mở rộng sang Thái Lan sau, với trọng tâm là từ vựng, cấu tạo chữ Hán, ví dụ ngữ cảnh, media flashcard, và lộ trình ôn tập theo spaced repetition. [github](https://github.com/open-spaced-repetition/dart-fsrs)
Sản phẩm cần hỗ trợ đồng thời HSK 3.0 và mapping ngược sang HSK 2.0, vì cùng một từ có thể cần xuất hiện theo nhiều khung chương trình học khác nhau. [github](https://github.com/drkameleon/complete-hsk-vocabulary)
### Định hướng kỹ thuật
Kiến trúc phù hợp nhất cho app này là offline-first: dữ liệu học và trạng thái ôn tập được đọc/ghi cục bộ trên máy để trải nghiệm mượt, còn đồng bộ nền được xử lý qua lớp sync giữa SQLite cục bộ và PostgreSQL trên cloud. [powersync](https://www.powersync.com/blog/flutter-tutorial-building-an-offline-first-chat-app-with-supabase-and-powersync)
Với Flutter, tổ hợp Supabase + PowerSync là hướng triển khai hợp lý vì vừa có auth, storage, Postgres, vừa có cơ chế sync nền cho mobile. [supabase](https://supabase.com/docs/guides/getting-started/tutorials/with-flutter)
## Phạm vi dữ liệu
### Đơn vị dữ liệu cốt lõi
Database không nên chỉ coi “từ” là đơn vị duy nhất, vì tiếng Trung có nhiều lớp thông tin khác nhau: từ vựng, từng chữ Hán, bộ thủ, nghĩa, ví dụ, media, và tiến độ ôn tập của từng người dùng. [hanzistroke](https://www.hanzistroke.com/hsk)
Vì vậy mô hình đúng nên tách ít nhất các nhóm sau: `vocabulary`, `characters`, `radicals`, `definitions`, `examples`, `media`, `user_srs_reviews`, cùng các bảng nối để mô tả quan hệ cấu thành.
### Tính chất ngôn ngữ cần phản ánh
Một mục từ tiếng Trung có thể có nhiều nghĩa, có lượng từ đi kèm, và có thể trùng mặt chữ nhưng khác âm đọc hoặc khác nghĩa, nên schema phải tránh kiểu “một chuỗi chữ = một dòng dữ liệu duy nhất”. [cc-cedict](https://cc-cedict.org/wiki/syntax)
HSK 3.0 cũng tách rõ giữa năng lực biết **từ**, biết **nhận diện chữ**, và biết **viết chữ**, nên database nên có lớp “word-level” và lớp “character-level” riêng thay vì gộp tất cả vào một bảng. [hanzistroke](https://www.hanzistroke.com/hsk)
## Thiết kế dữ liệu
### 1. Bảng `vocabulary`
Đây là bảng trung tâm ở cấp **từ vựng**, dùng để lưu các đơn vị học như `安全`, `准备`, `电脑`, là những gì người học nhìn thấy chủ yếu trong flashcard và lesson.  
Mỗi dòng nên đại diện cho một lexical entry, không chỉ là một chuỗi ký tự.

**Các cột khuyến nghị**
- `id`: khóa chính UUID.
- `simplified`: từ viết bằng giản thể.
- `pinyin`: pinyin chuẩn của entry đó.
- `part_of_speech`: từ loại, ví dụ noun, verb, adjective.
- `measure_words`: lượng từ liên quan nếu có.
- `hsk2_level`: level theo HSK 2.0, nullable.
- `hsk3_level`: band/level theo HSK 3.0, nullable.
- `created_at`, `updated_at`, `deleted_at`: phục vụ sync và soft delete. [docs.powersync](https://docs.powersync.com/integrations/supabase/guide)

**Ràng buộc quan trọng**
- Nên dùng `UNIQUE(simplified, pinyin)` thay vì chỉ unique trên `simplified`, vì một mặt chữ có thể có nhiều cách đọc hoặc nhiều entry khác nhau. [cc-cedict](https://cc-cedict.org/wiki/syntax)

**Use case**
- Khi người dùng mở trang từ “安全”, app đọc dữ liệu chính từ bảng này để hiển thị chữ, pinyin, level HSK và metadata cơ bản. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
- Khi lọc “HSK 4” hoặc “New HSK 3”, app query theo `hsk2_level` và `hsk3_level`. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
### 2. Bảng `definitions`
Bảng này tách nghĩa ra khỏi `vocabulary` để hỗ trợ đa ngôn ngữ và nhiều lớp nghĩa cho cùng một từ.  
Điều này rất quan trọng vì cùng một từ có thể cần nghĩa tiếng Việt, tiếng Anh, tiếng Thái, và có thể còn có ghi chú riêng cho người học Việt Nam.

**Các cột khuyến nghị**
- `id`
- `vocab_id`
- `lang_code`: `vi`, `en`, `th`
- `meaning`
- `sino_vietnamese`: dành riêng cho thị trường Việt Nam, nullable
- `notes`: ghi chú sư phạm, nullable
- `created_at`, `updated_at`, `deleted_at`

**Use case**
- Với người học Việt Nam, app có thể hiển thị đồng thời nghĩa tiếng Việt và âm Hán Việt để tăng tốc liên tưởng.
- Khi mở rộng sang Thái Lan, chỉ cần thêm dòng `lang_code = 'th'` cho cùng `vocab_id`, không cần nhân đôi bảng hay tạo hệ thống dữ liệu mới.
### 3. Bảng `examples`
Ví dụ câu nên nằm ở bảng riêng vì một từ có thể có nhiều câu minh họa theo nhiều mức độ khó, nhiều ngôn ngữ dịch, và nhiều mục đích học khác nhau. [github](https://github.com/clem109/hsk-vocabulary)

**Các cột khuyến nghị**
- `id`
- `vocab_id`
- `chinese_sentence`
- `pinyin`
- `translation`
- `lang_code`
- `difficulty_level`: cơ bản/nâng cao, nullable
- `source_type`: curated/ai/generated/editorial, nullable
- `created_at`, `updated_at`, `deleted_at`

**Use case**
- Ở màn hình chi tiết từ, app hiển thị nhiều câu ví dụ ngay dưới phần meaning như mẫu bạn gửi. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
- Ở chế độ luyện đọc, hệ thống có thể ưu tiên câu ngắn cho HSK thấp và câu dài hơn cho HSK cao.
### 4. Bảng `media`
Bảng này lưu toàn bộ media dùng cho flashcard hoặc lesson như image, gif, short video, audio cover image.  
Không nên nhét binary trực tiếp vào SQLite/Postgres chính, mà chỉ lưu metadata và URL/file key.

**Các cột khuyến nghị**
- `id`
- `vocab_id`
- `media_type`: `image`, `gif`, `video`, `audio_cover`
- `file_url`
- `thumbnail_url`, nullable
- `prompt_text`, nullable
- `style_tag`, nullable
- `is_primary`: boolean
- `created_at`, `updated_at`, `deleted_at`

**Use case**
- Từ “安全” có thể có một ảnh minh họa an toàn, một GIF biểu đạt cảnh báo bảo mật, và một video ngắn mô phỏng ngữ cảnh dùng từ.
- App chỉ cần đọc URL rồi cache ở client để tránh làm database phình to.
### 5. Bảng `characters`
Đây là bảng cấp **chữ Hán đơn**, dùng cho logic nhận diện, tập viết, phân tích cấu tạo, và lesson về thành phần chữ. [hanzistroke](https://www.hanzistroke.com/hsk)
Nó khác với `vocabulary`: một `vocabulary` có thể là `安全`, còn `characters` sẽ chứa riêng `安` và `全`.

**Các cột khuyến nghị**
- `id`
- `character`
- `pinyin`
- `stroke_count`
- `hsk3_recognition_level`
- `hsk3_writing_level`
- `stroke_svg_data`
- `unicode_codepoint`, nullable
- `created_at`, `updated_at`, `deleted_at`

**Use case**
- Màn hình “Comprises” của từ `安全` có thể bẻ tách ra `安` và `全`, mỗi chữ lấy dữ liệu riêng từ bảng này. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
- Tính năng tập viết dùng `stroke_svg_data` để phát từng nét.
### 6. Bảng `word_characters`
Đây là bảng nối giữa `vocabulary` và `characters`, vì một từ gồm một hay nhiều chữ và một chữ có thể xuất hiện trong rất nhiều từ khác nhau.

**Các cột khuyến nghị**
- `vocab_id`
- `character_id`
- `position_index`
- `role_note`, nullable

**Use case**
- Từ `安全` được map tới `安` ở vị trí 1 và `全` ở vị trí 2, giúp app render đúng thứ tự trong phần “Comprises”. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
### 7. Bảng `radicals`
Đây là bảng mới cần bổ sung theo góp ý của bạn, và mình đồng ý đây là hướng thiết kế đúng hơn nhiều so với việc để radical dưới dạng một field text trong `characters`.  
Tách bảng này ra giúp radical trở thành một thực thể độc lập, có thể tái sử dụng, tra cứu, hiển thị, phân nhóm, và làm bài tập.

**Các cột khuyến nghị**
- `id`
- `radical`
- `pinyin`, nullable
- `meaning`
- `sino_vietnamese`, nullable
- `stroke_count`, nullable
- `stroke_svg_data`, nullable
- `display_order`, nullable
- `created_at`, `updated_at`, `deleted_at`

**Use case**
- App có thể có trang riêng “Bộ thủ”, nơi người học xem từng bộ, ý nghĩa, số nét và các chữ liên quan.
- App có thể cho phép lọc “những chữ có liên quan đến bộ thủ nước”, “những chữ có bộ nhân đứng”, hoặc “học theo nhóm bộ thủ”.
### 8. Bảng `character_radicals`
Đây là bảng quan trọng nhất trong cập nhật mới.  
Một chữ không nhất thiết chỉ có một radical theo nghĩa khai thác học tập của app, nên việc tạo bảng nối riêng linh hoạt hơn rất nhiều.

Bạn đề xuất trước mắt chỉ cần `character_id + radical_id` là đủ, và điều đó hoàn toàn hợp lý cho phase đầu.  
Tuy nhiên nên chừa sẵn vài cột nullable để sau này không phải refactor mạnh.

**Các cột khuyến nghị tối thiểu**
- `character_id`
- `radical_id`

**Các cột mở rộng nên để nullable**
- `position`: trái/phải/trên/dưới/bao quanh
- `function_type`: `semantic`, `phonetic`, `form`, `historical`
- `reasoning`: mô tả tại sao radical này được gắn với character đó
- `is_primary`: đánh dấu radical chính nếu cần

**Use case**
- Chữ `安` có thể gắn với nhiều thành phần để phục vụ học cấu tạo, thay vì ép thành một radical duy nhất.
- Sau này bạn có thể giải thích bộ nào mang nghĩa, bộ nào gợi âm, hoặc bộ nào chỉ mang tính cấu trúc thị giác.
- Đây chính là nền tảng cho bài tập radical picker kiểu người học ghép thành phần để tạo chữ như ảnh bạn gửi. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/2dd9c52f-a1b0-44d0-a9e9-5e26d793b721/IMG_2868.jpeg)
### 9. Bảng `user_srs_reviews`
Bảng này lưu trạng thái ôn tập của từng user theo từng từ.  
Nó là dữ liệu cá nhân, không phải dữ liệu từ điển công khai.

**Các cột khuyến nghị**
- `id`
- `user_id`
- `vocab_id`
- `state`
- `due_date`
- `stability`
- `difficulty`
- `reps`
- `lapses`
- `last_review_at`
- `created_at`, `updated_at`, `deleted_at`

FSRS là thuật toán phù hợp để tính lịch ôn tập dựa trên trạng thái nhớ/quên của người học. [github](https://github.com/open-spaced-repetition/dart-fsrs)

**Use case**
- Khi người học bấm Again/Hard/Good/Easy, app cập nhật dòng tương ứng trong bảng này ngay trên local database rồi sync nền lên server. [powersync](https://www.powersync.com/blog/flutter-tutorial-building-an-offline-first-chat-app-with-supabase-and-powersync)
## Quan hệ dữ liệu
### Quan hệ chính
- `vocabulary` 1-n `definitions`
- `vocabulary` 1-n `examples`
- `vocabulary` 1-n `media`
- `vocabulary` n-n `characters` qua `word_characters`
- `characters` n-n `radicals` qua `character_radicals`
- `vocabulary` 1-n `user_srs_reviews` theo từng user
### Ý nghĩa mô hình
Mô hình này tách rõ ba lớp học khác nhau:
- lớp **word learning**: học từ và nghĩa
- lớp **character learning**: học từng chữ Hán
- lớp **radical learning**: học thành phần cấu tạo và logic ghi nhớ

Nhờ đó app không chỉ là flashcard từ vựng, mà còn có thể tiến hóa thành hệ sinh thái học chữ Hán theo chiều sâu.
## Thuật ngữ quan trọng
### Vocabulary
Đơn vị từ vựng hoàn chỉnh người học cần ghi nhớ và sử dụng trong ngữ cảnh.  
Ví dụ `安全` là một vocabulary entry.
### Character
Một chữ Hán đơn lẻ.  
Ví dụ `安` và `全` là hai character khác nhau cấu thành từ `安全`. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
### Radical
Thành phần cơ bản dùng để phân loại hoặc phân tích cấu trúc chữ Hán.  
Trong app, radical không chỉ phục vụ tra cứu mà còn phục vụ mnemonic, bài tập kéo-thả, lesson cấu tạo chữ và lọc theo nhóm.
### Character-Radical Mapping
Mối quan hệ giữa một chữ và các radical/structural components được dùng để giải thích nó.  
Đây là chỗ bạn đề xuất nên tách riêng, và đó là quyết định rất tốt vì nó giữ database đủ linh hoạt cho cả phase MVP lẫn phase nâng cao.
### Reasoning
Phần giải thích tại sao radical này liên quan đến character.  
Nó có thể để null ở giai đoạn đầu, sau đó dần dần bổ sung nội dung sư phạm.
### Function Type
Vai trò của radical trong character, ví dụ:
- semantic: thiên về nghĩa
- phonetic: thiên về âm
- form: thiên về hình
- historical: ghi chú lịch sử/từ nguyên

Phần này cũng có thể để nullable trước, rồi bổ sung sau.
## Use cases sản phẩm
### 1. Chi tiết từ
Người dùng mở từ `安全`, thấy nghĩa, pinyin, ví dụ câu, các chữ cấu thành, và từng chữ có thể bấm vào để học sâu hơn. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
### 2. Comprises
Từ `安全` được tách thành `安` và `全`; từng chữ lại nối sang radical của chính nó để giải thích cấu tạo. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/21099e81-d4f3-4e38-8a33-daa57370ca37/IMG_2869.jpeg)
### 3. Radical picker
App hiển thị vài component/radical rời, người dùng ghép để tạo thành chữ đúng, giống pattern bài tập bạn gửi. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/2dd9c52f-a1b0-44d0-a9e9-5e26d793b721/IMG_2868.jpeg)
Tính năng này phụ thuộc trực tiếp vào bảng `character_radicals`. [ppl-ai-file-upload.s3.amazonaws](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/images/42804958/2dd9c52f-a1b0-44d0-a9e9-5e26d793b721/IMG_2868.jpeg)
### 4. Học theo bộ thủ
Người dùng chọn một radical rồi học tất cả character hoặc vocabulary liên quan đến radical đó.  
Đây là một hướng gamification và pedagogy rất mạnh cho người học sơ-trung cấp.
### 5. Flashcard SRS
App đưa ra từ đến hạn ôn, hiển thị media, nghĩa, ví dụ, và cập nhật độ khó/lịch ôn sau mỗi lần trả lời. [powersync](https://www.powersync.com/blog/flutter-tutorial-building-an-offline-first-chat-app-with-supabase-and-powersync)
## Nguyên tắc triển khai
### Phase 1
Ở giai đoạn đầu, bạn có thể giữ scope gọn như sau:
- `radicals`
- `character_radicals` chỉ gồm `character_id`, `radical_id`
- chưa bắt buộc `reasoning`, `function_type`, `position`
- tập trung làm được tra cứu từ, cấu thành chữ, flashcard, ví dụ, và SRS

Đây là mức đủ để ship MVP nhanh nhưng vẫn không khóa đường phát triển sau này.
### Phase 2
Khi có thêm content và thời gian biên tập:
- thêm `reasoning`
- thêm `function_type`
- thêm lesson theo radical
- thêm bài tập build-a-character
- thêm nhóm chữ đồng bộ thủ hoặc đồng thành phần
### Phase 3
Khi app trưởng thành hơn:
- thêm etymology
- thêm variation theo font/handwriting
- thêm đồ thị quan hệ giữa radicals, characters, vocabulary
- thêm recommendation engine dựa trên lỗi người học ở cùng radical family
## Mẫu schema SQL
```sql
CREATE TABLE vocabulary (
  id UUID PRIMARY KEY,
  simplified TEXT NOT NULL,
  pinyin TEXT NOT NULL,
  part_of_speech TEXT,
  measure_words TEXT,
  hsk2_level INT,
  hsk3_level INT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ,
  UNIQUE (simplified, pinyin)
);

CREATE TABLE characters (
  id UUID PRIMARY KEY,
  character TEXT NOT NULL UNIQUE,
  pinyin TEXT,
  stroke_count INT,
  hsk3_recognition_level INT,
  hsk3_writing_level INT,
  stroke_svg_data TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ
);

CREATE TABLE radicals (
  id UUID PRIMARY KEY,
  radical TEXT NOT NULL UNIQUE,
  pinyin TEXT,
  meaning TEXT,
  sino_vietnamese TEXT,
  stroke_count INT,
  stroke_svg_data TEXT,
  display_order INT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ
);

CREATE TABLE word_characters (
  vocab_id UUID NOT NULL REFERENCES vocabulary(id),
  character_id UUID NOT NULL REFERENCES characters(id),
  position_index INT NOT NULL,
  role_note TEXT,
  PRIMARY KEY (vocab_id, character_id)
);

CREATE TABLE character_radicals (
  character_id UUID NOT NULL REFERENCES characters(id),
  radical_id UUID NOT NULL REFERENCES radicals(id),
  position TEXT,
  function_type TEXT,
  reasoning TEXT,
  is_primary BOOLEAN,
  PRIMARY KEY (character_id, radical_id)
);

CREATE TABLE definitions (
  id UUID PRIMARY KEY,
  vocab_id UUID NOT NULL REFERENCES vocabulary(id),
  lang_code TEXT NOT NULL,
  meaning TEXT NOT NULL,
  sino_vietnamese TEXT,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ
);

CREATE TABLE examples (
  id UUID PRIMARY KEY,
  vocab_id UUID NOT NULL REFERENCES vocabulary(id),
  chinese_sentence TEXT NOT NULL,
  pinyin TEXT,
  translation TEXT,
  lang_code TEXT,
  difficulty_level TEXT,
  source_type TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ
);

CREATE TABLE media (
  id UUID PRIMARY KEY,
  vocab_id UUID NOT NULL REFERENCES vocabulary(id),
  media_type TEXT NOT NULL,
  file_url TEXT NOT NULL,
  thumbnail_url TEXT,
  prompt_text TEXT,
  style_tag TEXT,
  is_primary BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ
);

CREATE TABLE user_srs_reviews (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  vocab_id UUID NOT NULL REFERENCES vocabulary(id),
  state INT NOT NULL,
  due_date TIMESTAMPTZ NOT NULL,
  stability FLOAT,
  difficulty FLOAT,
  reps INT DEFAULT 0,
  lapses INT DEFAULT 0,
  last_review_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  deleted_at TIMESTAMPTZ
);
```
## Kết luận thiết kế
Cập nhật lớn nhất và đúng nhất trong phiên bản mới là:  
`radicals` phải là một bảng riêng, và `character_radicals` phải là bảng nối độc lập.

Điểm này giúp schema phản ánh đúng hơn bản chất chữ Hán, đồng thời mở đường cho:
- tra cứu bộ thủ
- phân tích cấu tạo chữ
- bài tập ghép thành phần
- lesson theo radical
- giải thích semantic/phonetic sau này
- hệ thống ghi nhớ trực quan cho người học Việt Nam và Thái
## ERD cập nhật
ERD mới đã phản ánh đầy đủ việc tách `radicals` và thêm `character_radicals` để nối giữa `characters` và `radicals`.



Nếu bạn muốn, ở bước tiếp theo mình có thể viết tiếp ngay bản **Database Design Specification chuyên nghiệp** theo format bàn giao cho dev, gồm:
1. data dictionary chi tiết từng cột,
2. index strategy,
3. sync rules cho Supabase/PowerSync,
4. migration plan từ MVP sang phase 2.