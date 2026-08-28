# ainovel-cli

Engine viết tiểu thuyết dài bằng AI, chạy hoàn toàn tự động. Phần engine mang tính tất định chạy trọn cuốn sách, còn mô hình chỉ được gọi đúng ở những chỗ cần phán đoán: Engine định tuyến theo sự thật đã ghi xuống đĩa để điều phối ba tác nhân sáng tác tự chủ Architect / Writer / Editor, và đánh thức Arbiter khi cần một phán quyết ngữ nghĩa. Từ một câu yêu cầu tới một cuốn tiểu thuyết hoàn chỉnh, không cần can thiệp thủ công.

<p align="center">
  <img src="scripts/sample.gif" alt="ainovel-cli demo" width="800">
  <img src="scripts/novel.png" alt="ainovel-cli bg" width="800">
</p>

## Tính năng

- **Engine tất định + nhiều tác nhân phối hợp** — Engine điều phối Architect / Writer / Editor theo bảng quyết định dựa trên sự thật; vòng lặp chính không tốn một lệnh gọi LLM nào và hành vi có thể kiểm thử vét cạn
- **Phán quyết ngữ nghĩa có thể kiểm toán** — Chọn nhà quy hoạch, phân loại can thiệp của người dùng, tìm lối ra khi thất bại đều do Arbiter quyết trong một lệnh gọi duy nhất; mỗi phán quyết đều được ghi lại và phát lại được. Càng đơn giản càng ổn định, từ chối điều phối phức tạp
- **Khôi phục ở mức step** — Mỗi công cụ chạy thành công là ghi một checkpoint; sau sự cố khôi phục chính xác tới bước plan/draft/check/commit
- **Quy hoạch cuộn hai tầng tập–vòng cung** — Truyện dài không quy hoạch toàn bộ chương ngay từ đầu. Ban đầu chỉ dựng khung 2 tập đầu + chương chi tiết của vòng cung 1; các vòng cung/tập sau do Architect mở rộng khi viết tới, mỗi lần mở rộng đều tham chiếu tóm tắt và trạng thái nhân vật đã có
- **Gợi ý chương liên quan** — Mỗi chương tự động truy ngược lịch sử theo bốn chiều:mạch cài cắm, lần xuất hiện nhân vật, thay đổi trạng thái, quan hệ — bảo đảm tính liên tục cho truyện 500+ chương
- **Chiến lược ngữ cảnh thích ứng** — Tự chuyển giữa toàn phần / cửa sổ trượt / tóm tắt phân tầng theo tổng số chương
- **Thẩm định bảy chiều** — Editor chấm theo nhất quán thiết định, hành vi nhân vật, nhịp, mạch tự sự, mạch cài cắm, hook, và chất lượng văn chương; riêng chiều văn chương chia nhỏ thành chất cảm miêu tả / thủ pháp tự sự / độ phân biệt thoại / chất lượng dùng từ / sức lay động, mỗi mục **bắt buộc trích dẫn chính văn** làm bằng chứng
- **Can thiệp thời gian thực** — Gõ ý kiến sửa vào ô nhập bất cứ lúc nào (không cần tạm dừng); hệ thống tự đánh giá phạm vi ảnh hưởng và viết lại các chương bị tác động
- **Nghiệm thu từng chương (tuỳ chọn)** — Mặc định vẫn tự động hoàn toàn; cần kiểm soát chặt thì bật `/review on`, mỗi lần `/next` chỉ cho qua đúng một chương mới
- **Song lối vào TUI + Headless** — Vừa quan sát và can thiệp trực tiếp trên giao diện, vừa chạy dài hạn không giao diện trên server, NAS hay CI
- **Đa ngôn ngữ, tách bạch hai tầng** — `language` quyết định **ngôn ngữ tác phẩm** (chính văn, tiêu đề, dàn ý, tóm tắt), `ui_language` quyết định **ngôn ngữ giao diện**. Hai thứ độc lập: hoàn toàn có thể dùng giao diện tiếng Việt để viết một cuốn tiếng Trung
- **Hỗ trợ nhiều LLM** — OpenRouter / Anthropic / Gemini / OpenAI … chuyển đổi tuỳ ý

## Kiến trúc

Thiết kế cốt lõi: **tầng sự thật thì tất định, tầng ngữ nghĩa thì tự chủ**. Những chuyển trạng thái liệt kê được thì để mã tất định thực thi (Engine + Route); những phán đoán có ranh giới rõ ràng thì hỏi hàm LLM khi cần (Arbiter); còn sáng tác mở thì giao hẳn cho vòng lặp LLM tự chủ (Workers). Tóm một câu: một Engine tuần tự tất định, ba Worker tự chủ, vài hàm Arbiter gọi theo nhu cầu, và một tầng sự thật nằm trên hệ tệp.

```
┌─────────────────────────────────────────────────┐
│            Host / Engine (tất định)             │
│  đọc Store → Route → chạy Worker → lặp lại      │
│  khởi động / can thiệp / bế tắc → hỏi Arbiter   │
└────┬──────────┬──────────┬─────────────┬────────┘
     │          │          │             │
 ┌───▼─────┐ ┌──▼────┐ ┌───▼─────┐  ┌────▼────┐
 │Architect│ │Writer │ │ Editor  │  │ Arbiter │
 │(vòng LLM)│ │(vòng LLM)│ │(vòng LLM)│  │(hàm LLM)│
 └───┬─────┘ └──┬────┘ └───┬─────┘  └─────────┘
     └──────────┼──────────┘
                │ gọi công cụ (IO + checkpoint)
┌───────────────▼─────────────────────────────────┐
│                     Store                       │
│  Progress / Checkpoint / Outline / Drafts / ... │
└─────────────────────────────────────────────────┘
```

- **Engine** — Mỗi vòng đọc sự thật từ Store, phái Worker theo bảng quyết định Route; chỉ thực thi, không tham gia phán đoán văn chương. Khôi phục sau sự cố = đọc store rồi chạy tiếp, không có phiên hội thoại nào cần khôi phục
- **Arbiter** — Phán quyết ngữ nghĩa được đánh thức theo nhu cầu (chọn nhà quy hoạch, phân loại can thiệp, lối ra khi thất bại/bế tắc). Sự thật vào, quyết định có cấu trúc ra, mỗi lần đều ghi xuống đĩa để kiểm toán và phát lại
- **Workers** — Architect / Writer / Editor, mỗi tác nhân một context riêng, phối hợp qua các artifact trong Store
- **Tools** — IO nguyên tử một tệp + phát lại idempotent; commit chương dùng Saga bền vững + checkpoint, chỉ trả về JSON sự thật, không kèm chỉ thị

### Trách nhiệm từng tác nhân

| Vai | Trách nhiệm | Công cụ |
|--------|------|------|
| **Arbiter** | Phán quyết ngữ nghĩa: chọn nhà quy hoạch khi khởi động, phân loại can thiệp, lối ra khi thất bại/bế tắc | Không có (một lệnh gọi LLM, trả về quyết định có cấu trúc) |
| **Architect** | Sinh tên sách, giới thiệu, tiền đề, dàn ý, hồ sơ nhân vật, quy tắc thế giới | `novel_context` `save_book` `save_foundation` `revise_outline` `resolve_outline_feedback` `audit_foundation` |
| **Writer** | Tự chủ hoàn thành một chương:phác ý, viết, tự soát, nộp | `novel_context` `read_chapter` `plan_chapter` `draft_chapter` `edit_chapter` `check_consistency` `commit_chapter` |
| **Editor** | Đọc chính văn, soi vấn đề ở cả tầng cấu trúc lẫn tầng văn chương | `novel_context` `read_chapter` `save_review` `save_arc_summary` `save_volume_summary` |

Ngoài ra `reopen_book` phục vụ việc mở lại một cuốn đã hoàn thành đểlàm lại. Tổng cộng 16 công cụ.

### Luồng sáng tác

```
Yêu cầu → Arbiter chọn nhà quy hoạch → Architect dựng khung + vòng cung 1 → Writer viết từng chương → Editor thẩm định vòng cung
              (ghi phán quyết)                                                  ↑                        │
                                                                               ├── viết lại/trau chuốt ◄┘
                                                                               │
                                                                    Architect mở rộng vòng cung/tập kế
```

Mỗi bước "kế tiếp phái ai" đều do bảng quyết định Route của Engine suy ra từ sự thật trong Store (được khoá bằng bài kiểm thử vét cạn hàng vạn tổ hợp), không tốn một lệnh gọi LLM nào.

Writer hoàn thành mỗi chương theo thứ tự cố định (nội dung viết thì hoàn toàn tự chủ, nhưng thứ tự gọi công cụ là nghiêm ngặt):

1. `novel_context` — nạp ngữ cảnh (tóm tắt tiền sự, mạch cài cắm, trạng thái nhân vật, quy tắc văn phong, gợi ý chương liên quan)
2. `read_chapter` — đọc lại đoạn trước để lấy lại giọng và nhịp
3. `plan_chapter` — phác mục tiêu, xung đột, đường cong cảm xúc của chương
4. `draft_chapter` — viết trọn chính văn
5. `check_consistency` — đối chiếu dữ liệu trạng thái để kiểm tra nhất quán (bắt buộc sau draft)
6. `commit_chapter` — nộp bản cuối, ghi các trường sự thật (`arc_end` / `next_chapter` / hồ phản hồi dàn ý…); bước kế tiếp do Engine suy ra từ Route

### Quy tắc chuyển trạng thái

Hệ thống tách trạng thái chạy thành hai tầng:

- **Phase** — Giai đoạn lớn: đang thiết định, đang viết, hay đã hoàn thành
- **Flow** — Luồng đang hoạt động: đang viết bình thường, đang thẩm định, viết lại, trau chuốt, hay đang xử lý can thiệp

#### Phase

`Phase` theo quy tắc "chỉ tiến, không lùi":

```text
init -> premise -> outline -> writing -> complete
  \-------> outline ------^
  \--------------> writing
```

- `init` — Đã tạo tác vụ, chưa có thiết định ổn định
- `premise` — Đã lưu tiền đề câu chuyện
- `outline` — Đã lưu dàn ý, có thể vào giai đoạn viết
- `writing` — Đang trong giai đoạn sáng tác chương
- `complete` — Toàn bộ quy trình đã kết thúc

Cho phép cập nhật cùng trạng thái (`writing -> writing`) và tiến lên (`outline -> writing`); không cho lùi (`writing -> premise`, `complete -> writing`).

#### Flow

`Flow` chỉ mô tả luồng đang hoạt động bên trong giai đoạn viết:

```text
writing   -> reviewing / rewriting / polishing / steering / writing
reviewing -> writing / rewriting / polishing / steering / reviewing
rewriting -> writing / steering / rewriting
polishing -> writing / steering / polishing
steering  -> writing / reviewing / rewriting / polishing / steering
```

- `writing` — Đẩy tiếp chương kế
- `reviewing` — Editor đang thẩm định
- `rewriting` — Xử lý chương buộc phải viết lại
- `polishing` — Xử lý chương chỉ cần trau chuốt
- `steering` — Đang đánh giá và xử lý can thiệp của người dùng

Các quy tắc này được ràng buộc bằng một lớp kiểm tra nhẹ trong mã, tránh lùi trạng thái hoặc nhảy sang nhánh vô lý.

### Quy hoạch cuộn cho truyện dài

Cách làm truyền thống quy hoạch hết chương ngay từ đầu; tới 300+ chương thì dàn ý rỗng tuếch và nhịp truyện như chạy cho kịp tiến độ. Hệ thống này dùng **la bàn + tầm nhìn cuộn**, mô phỏng cách tác giả truyện mạng thực sự làm việc:

```
Quy hoạch ban đầu           Khi kết vòng cung          Khi kết tập
┌────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│ Hướng kết cục (la bàn)│   │ Editor thẩm định vòng│    │ Editor thẩm định tập │
│ Khởi 2 tập, sau tuỳ  │   │ Tóm tắt + ảnh nhân vật│   │ Tóm tắt tập          │
│ Vòng cung 1 chi tiết │ → │ Architect mở vòng kế │ → │ Architect tự tạo tập │
│ Nhân vật + thế giới  │   │ Writer viết tiếp     │    │ kế + cập nhật la bàn │
└────────────────────┘    └─────────────────────┘    └─────────────────────┘
```

- **La bàn (Compass)** — Hướng kết cục + các tuyến dài đang mở + ước lượng quy mô; được Architect cập nhật ở mỗi ranh giới tập nên hướng truyện tiến hoá theo quá trình viết
- **Sinh theo nhu cầu** — Viết xong tập hiện tại, Architect tự tạo tập kế dựa trên nội dung đã viết
- **Vòng cung khung** — Chỉ có goal + số chương ước lượng; tới nơi mới mở chi tiết
- **Tinh chỉnh dần** — Mỗi lần mở rộng đều tham chiếu tóm tắt trước đó, ảnh chụp nhân vật, quy tắc văn phong; càng về sau càng chính xác
- **Mẫu nhịp dùng chung** — Vòng cung đột phá / đối kháng / khám phá / ân oán / đời thường chuyển tiếp, mỗi loại có mật độ tham khảo và ánh xạ thể loại phù hợp

### Quản lý ngữ cảnh cho truyện dài

Truyện 500+ chương dùng tóm tắt ba tầng + đường ống nén bốn cấp + gợi ý thông minh:

```
Tập (Volume) → tóm tắt tập
└── Vòng cung (Arc) → tóm tắt vòng cung + ảnh chụp nhân vật + quy tắc văn phong
    └── Chương (Chapter) → tóm tắt chương (cửa sổ trượt 3 chương gần nhất)
```

- **Tóm tắt phân tầng** — Gần thì dùng tóm tắt chương, trung bình dùng tóm tắt vòng cung, xa dùng tóm tắt tập
- **Gợi ý chương liên quan** — Truy ngược lịch sử theo bốn chiều (mạch cài cắm, xuất hiện nhân vật, đổi trạng thái, quan hệ) để Writer đọc lại khi cần
- **Báo trước chương kế** — Nạp dàn ý chương kế, giúp Writer thiết kế hook cuối chương và nối mạch cài cắm
- **Phát hiện ranh giới vòng cung** — Tự nhận ra kết vòng cung/tập để kích hoạt thẩm định, sinh tóm tắt và mở rộng phần kế

#### Đường ống nén ngữ cảnh

Khi hội thoại vượt cửa sổ ngữ cảnh của mô hình, nén theo thứ tự chi phí từ thấp lên cao:

```
ToolResultMicrocompact → LightTrim → StoreSummaryCompact → FullSummary
   dọn kết quả công cụ cũ   cắt văn dài   nén từ store, 0 LLM   tóm tắt bằng LLM
```

- **StoreSummaryCompact** — Riêng cho Writer: dùng tóm tắt chương, ảnh chụp nhân vật, sổ cài cắm sẵn có trong store để thay thẳng các tin nhắn cũ, không tốn LLM
- **FullSummary tuỳ biến cho tiểu thuyết** — Writer dùng prompt tóm tắt hướng tới tính liên tục tự sự, yêu cầu rõ phải giữ trạng thái nhân vật, đầu mối cài cắm, hạng mục chờ sửa, và neo văn phong
- **Gói khôi phục sau nén** — Sau FullSummary tự động tiêm lại kế hoạch chương hiện tại, dàn ý và ảnh chụp nhân vật, tránh Writer "mất trí nhớ"
- **Cầu chì** — Nén thất bại liên tiếp thì tự bỏ qua kèm cảnh báo rõ ràng, theo chế độ nửa mở, vòng sau tự thử lại
- **Ước lượng token theo ngôn ngữ** — Tiếng Trung tính `số ký tự × 1.5`, không bị `bytes/4` ước thiếu khiến nén kích hoạt trễ
- **Thanh sức khoẻ trên TUI** — Mức chiếm dụng ngữ cảnh hiển thị xanh (<70%) → vàng (70-85%) → đỏ (>85%)

## Bắt đầu nhanh

```bash
# Cài một lệnh (macOS / Linux, không cần Go)
curl -fsSL https://raw.githubusercontent.com/dongphuongman/novel-cli/main/scripts/install.sh | sh

# Cài phiên bản chỉ định
curl -fsSL https://raw.githubusercontent.com/dongphuongman/novel-cli/v1.2.3/scripts/install.sh | sh -s -- v1.2.3

# Hoặc cài qua Go
go install github.com/dongphuongman/novel-cli/cmd/ainovel-cli@latest

# Xem phiên bản / cập nhật lên bản mới nhất
ainovel-cli --version
ainovel-cli update

# Chạy lần đầu, tự vào luồng hướng dẫn (chọn Provider → nhập API Key → Base URL → tên model)
ainovel-cli
```

> Windows hoặc cài thủ công: tải gói tương ứng ở [Releases](https://github.com/dongphuongman/novel-cli/releases/latest).
> Script cài sẽ tải bảng kê SHA256 từ cùng GitHub Release, xác thực xong mới giải nén và cài binary.

Build từ mã nguồn:

```bash
go build -o ./ainovel-cli ./cmd/ainovel-cli
```

### Chế độ Headless

`--headless` không cần TUI, hợp để chạy dài trên server, NAS, CI hay tác vụ nền. Chế độ này **không** có hướng dẫn cấu hình lần đầu, nên hãy chạy `ainovel-cli` một lần để cấu hình, hoặc tự tạo `~/.ainovel/config.json`.

```bash
# Khởi tạo tác vụ mới bằng một câu yêu cầu
ainovel-cli --headless --prompt "Viết một bộ huyền huyễn phương Đông, nhân vật chính khởi đầu từ thị trấn biên thuỳ"

# Đọc yêu cầu từ tệp
ainovel-cli --headless --prompt-file prompt.txt

# Khôi phục tác vụ dang dở trong cùng thư mục
ainovel-cli --headless
```

`--prompt` và `--prompt-file` chỉ dùng được ở chế độ Headless và không được chỉ định đồng thời. Đầu ra streaming của model ghi ra stdout, sự kiện chạy ghi ra stderr, log đầy đủ nằm ở `logs/headless.log` trong thư mục tác phẩm.

### Docker

```bash
mkdir -p config workspace

# TUI
docker run --rm -it \
  -v "$PWD/config:/root/.ainovel" \
  -v "$PWD/workspace:/workspace" \
  ghcr.io/dongphuongman/novel-cli:latest

# Headless
docker run --rm \
  -v "$PWD/config:/root/.ainovel" \
  -v "$PWD/workspace:/workspace" \
  ghcr.io/dongphuongman/novel-cli:latest \
  --headless --prompt "Viết một bộ huyền huyễn phương Đông"
```

Hoặc dùng Compose:

```bash
docker compose run --rm ainovel
docker compose run --rm ainovel --headless --prompt "Viết một truyện trinh thám ngắn"
```

### Ba cách bắt đầu một cuốn sách

Vào TUI, màn hình chào hỗ trợ:

- **Bắt đầu nhanh** — Một câu là vào sáng tác luôn
- **Đồng sáng tác** — Đối thoại nhiều lượt với AI để làm rõ nhu cầu; **bên phải đồng bộ theo thời gian thực bản nháp chỉ thị sáng tác**. Mỗi lượt AI chủ động đưa 1-3 gợi ý, bấm phím số để ghép vào ô nhập, sửa rồi gửi; bấm `Ctrl+S` để vào sáng tác chính thức
- **Tạo từ tệp** — Đã có sẵn thiết định thế giới hoặc dàn ý dài:

```text
/start ./outline.md
```

`/start` lấy toàn văn tệp làm yêu cầu sáng tác ban đầu, giao Architect chỉnh lý thành thiết định nội bộ và dàn ý động; **không** coi nội dung tệp là chương đã hoàn thành. Muốn nhập một cuốn đã có rồi viết tiếp thì dùng `/import`.

### Quản lý nhiều cuốn sách

Mỗi cuốn gắn với thư mục khởi động, sản phẩm nằm ở `{cwd}/output/novel/`. Đổi thư mục khởi động = đổi cuốn khác; `cd` về chỗ cũ rồi chạy = tự khôi phục từ checkpoint gần nhất. Cấu hình `~/.ainovel/config.json` dùng chung toàn cục, không cần sao chép.

### Ngôn ngữ

Hai trường độc lập trong `~/.ainovel/config.json`:

```jsonc
{
  "language": "vi",      // ngôn ngữ TÁC PHẨM: chính văn, tiêu đề chương, tên nhân vật, dàn ý, tóm tắt, ý kiến thẩm định
  "ui_language": "vi"    // ngôn ngữ GIAO DIỆN: chữ hiển thị trong terminal
}
```

- Giá trị hợp lệ: `zh` (简体中文) / `vi` (Tiếng Việt) / `en` (English). `ui_language` còn nhận `auto` để suy từ biến môi trường `LC_ALL` / `LC_MESSAGES` / `LANG`.
- Hai tầng này **không liên quan nhau**: hoàn toàn có thể để giao diện tiếng Việt mà viết một cuốn tiếng Trung, và ngược lại.
- `language` không chỉ đổi ngôn ngữ chữ. Nó là nguồn sự thật duy nhất cho mọi hành vi máy móc phụ thuộc ngôn ngữ: cách **đếm độ dài** (tiếng Trung đếm ký tự, tiếng Việt/Anh đếm từ — chênh nhau khoảng 5 lần), cách ngắt câu, dạng tiêu đề chương, và danh sách đen "câu văn AI" theo từng ngôn ngữ. Cấu hình sai thì các lớp phòng vệ máy móc **không báo lỗi, chỉ luôn trả về 0** — im lặng mất tác dụng, nguy hiểm hơn là báo lỗi thẳng.
- Prompt và tài liệu tham chiếu trong `assets/` viết bằng **tiếng Anh** — đó là ngôn ngữ làm việc của mô hình, không phải ngôn ngữ của sách. Một chỉ thị ngôn ngữ cứng được nối vào cuối prompt của mọi Worker (và cả đường nhập/mô phỏng/sửa tay) để ghìm ngôn ngữ sản phẩm.

### Tệp cấu hình

Lần chạy đầu sẽ tự hướng dẫn sinh `~/.ainovel/config.json`. Vào TUI gõ `/config` để thêm/sửa Provider, lưu nhiều model và đặt cửa sổ ngữ cảnh cho từng model; lưu xong có hiệu lực ngay. `/model` dùng để chuyển giữa các model đã lưu.

Cũng có thể tự tạo tệp, tham khảo `config.example.jsonc` ở gốc kho. Luồng hướng dẫn lần đầu cũng chép một bản sang `~/.ainovel/config.example.jsonc` để xem offline.

```jsonc
{
  "provider": "openrouter",
  "model": "google/gemini-2.5-flash",
  "reasoning_effort": "medium",
  "language": "vi",
  "ui_language": "vi",
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-xxx",
      "base_url": "https://openrouter.ai/api/v1",
      "models": [
        { "name": "google/gemini-2.5-flash", "context_window": 200000 },
        { "name": "google/gemini-2.5-pro", "context_window": 1000000 }
      ],
      "extra": {
        "user_agent": "my-client/1.0",
        "headers": { "X-Custom-Client": "my-client" }
      }
    }
  },
  "style": "default"
}
```

#### Thứ tự tìm cấu hình (sau đè lên trước)

1. `~/.ainovel/config.json` — cấu hình toàn cục
2. `./.ainovel/config.json` — đè ở mức dự án (tuỳ chọn)

> `./.ainovel/` là bản sao gương của `~/.ainovel/`: cùng cấu trúc, chỉ đổi thư mục gốc. Cấu hình đặt ở `./.ainovel/config.json`, quy tắc viết đặt ở `./.ainovel/rules/*.md`. Thư mục này chứa khoá nên đã có sẵn trong `.gitignore`.

Quy tắc đè:

- Trường vô hướng thì sau đè trước: `provider`, `model`, `reasoning_effort`, `style`, `language`, `ui_language`
- `providers` và `roles` gộp theo key, mục trùng tên thì đè theo từng trường
- Trường không điền sẽ kế thừa cấu hình tầng trên — ví dụ cấu hình dự án chỉ ghi `base_url` thì vẫn giữ `api_key` toàn cục
- Không hỗ trợ dùng chuỗi rỗng để xoá giá trị tầng trên; cần xoá thì sửa thẳng tệp có ưu tiên cao hơn

> ⚠️ Giá trị của `provider` (và `roles.*.provider`) là **tên key** trong `providers` — một con trỏ, không phải tên giao thức. Nếu cấu hình dự án trỏ `provider` sang một tài khoản không tồn tại trong `providers` toàn cục thì phải bổ sung luôn thông tin xác thực (`api_key` / `base_url`) ở mức dự án, nếu không khởi động sẽ báo "chưa cấu hình thông tin xác thực".

`providers.<name>.models` là danh sách đối tượng model tuỳ chọn: `name` là tên model gửi cho Provider, `context_window` là cửa sổ nén ngữ cảnh riêng của model, `json_schema` là ghi đè ba trạng thái cho đầu ra có cấu trúc gốc (`true` xác nhận hỗ trợ, `false` xác nhận không, bỏ trống thì theo năng lực của adapter). Dùng proxy tự dựng hoặc năng lực tuỳ model thì nên ghi rõ. Cửa sổ ngữ cảnh giải theo thứ tự "giá trị riêng của model → `context_window` ở tầng trên (bản cũ) → bảng đăng ký model → mặc định 200K". Nó chỉ ảnh hưởng thời điểm nén ngữ cảnh cục bộ, không đổi giới hạn thật của API phía xa.

`/config` chỉ để **định nghĩa Provider** (giao thức / API Key / Base URL / kho model), không quyết định "đang dùng model nào" — chuyển model và mức suy luận thì dùng `/model`. Lưu sẽ **ghi ngược vào đúng tệp cấu hình đang có hiệu lực** — có `./.ainovel/config.json` thì ghi vào đó, không thì ghi toàn cục — và áp dụng nóng ngay. Model đang được tham chiếu thì không xoá trực tiếp được, phải chuyển đi bằng `/model` trước. Ô nhập API Key luôn ẩn.

`reasoning_effort` là mức suy luận mặc định: `off` / `low` / `medium` / `high` / `xhigh` / `max`; bỏ trống nghĩa là theo mặc định của model/provider. `roles.<role>.reasoning_effort` đè theo vai. Mức suy luận có hiệu lực theo "ý định × năng lực": cấu hình lưu **ý định gốc** bạn chọn, còn lúc gửi đi thì bị kẹp theo **năng lực của model hiện tại** của vai đó — đổi sang model yếu hơn chỉ làm giá trị áp dụng bị kẹp thấp, ý định lưu vẫn nguyên, quay lại model mạnh là tự khôi phục.

`providers.<name>.api` chỉ có tác dụng với `type: "openai"`, để chọn endpoint giao thức OpenAI: `chat` (mặc định, `base_url + /chat/completions`) hoặc `responses` (`base_url + /responses`). Nếu `base_url` đã có sẵn đường dẫn (như `/api/v3` của Volcengine) thì đường dẫn đó được giữ nguyên. Proxy kiểu Codex thường cần đặt `responses`.

`providers.<name>.extra` là cấu hình mức provider, truyền xuống HTTP client, hợp để đặt `user_agent`, `headers`, `anthropic_beta`; còn `providers.<name>.extra_body` mới là tham số mở rộng của thân yêu cầu — đừng lẫn hai thứ.

## Báo cáo chẩn đoán

Gõ `/diag` trong TUI để phân tích sản phẩm trong thư mục output của cuốn hiện tại, cho ra các phát hiện và đề xuất cải thiện có thể hành động.

Bao phủ bốn chiều:

- **Quy trình** — Vòng lặp viết lại bị kẹt, chỉ thị chuyển hướng chưa được tiêu thụ, trạng thái phase/flow bất thường, nhảy số chương
- **Chất lượng** — Chiều thẩm định điểm thấp kéo dài, tỉ lệ thực hiện hợp đồng chương, tỉ lệ viết lại, độ dài chương bất thường
- **Quy hoạch** — mạch cài cắm đình trệ, la bàn lỗi thời, dàn ý cạn, thiếu tóm tắt
- **Ngữ cảnh** — Nhân vật biến mất, đứt gãy dòng thời gian, dữ liệu quan hệ đứng yên

Mỗi phát hiện gồm: mô tả vấn đề, bằng chứng dữ liệu, đề xuất cải thiện (chỉ tới prompt/flow/config cụ thể).

`/diag` đồng thời ghi ra `meta/diag-export.md` **đã khử nhạy cảm** (bỏ chính văn, chỉ giữ khung hành vi: lệnh gọi công cụ, chuỗi lỗi, số lần lặp). Gặp lỗi vòng lặp chết hoặc gián đoạn thì dán tệp này vào GitHub issue là đủ để người bảo trì định vị mà không cần dữ liệu cục bộ của bạn.

## Hồ sơ mô phỏng văn phong

Đặt các bài tham khảo vào thư mục `simulate/` trong thư mục khởi động rồi gõ `/simulate` trong TUI. Hệ thống đọc đệ quy các tệp `.txt`, `.md`, `.markdown`, dùng model của architect phân tích ngữ liệu rồi ghi vào:

```text
output/novel/meta/simulation_profile.json
```

Chạy `/simulate` lần nữa sẽ bỏ qua tệp không đổi theo `relative_path + sha256`; nếu không có gì mới thì báo "hồ sơ đã mới nhất" và **không** gọi LLM. Nếu đã có hồ sơ mà `simulate/` xuất hiện bài mới hoặc sửa đổi thì hệ thống tổng hợp tiếp trên nền hồ sơ cũ.

Cũng có thể nhập lại hồ sơ đã sinh trước đó để khỏi phân tích lại:

```text
/importsim ./profile.json
```

`/importsim` chỉ nhận JSON `simulation_profile.v1` do chính tính năng này sinh, gộp theo vân tay ngữ liệu, nguồn trùng thì bỏ qua. **Chỉ nhập hồ sơ từ nguồn tin cậy** — nội dung nhập vào sẽ trở thành ngữ cảnh tham chiếu cho các Agent về sau. Hồ sơ được tiêm dạng compact vào `novel_context`; Architect, Writer, Editor đều đọc được và chỉ học cấu trúc, nhịp, hook, cách hút người đọc — **không** sao chép câu chữ hay thiết định riêng của bản gốc.

## Tiếp nhận sửa tay

Bạn có thể sửa thẳng các chương đã hoàn thành trong `output/novel/chapters/*.md`. Hệ thống nhận biết thay đổi theo SHA-256 của bản đã tiếp nhận, không dựa vào thời gian sửa tệp:

```text
/sync --check   # chỉ liệt kê chương đã thay đổi, không gọi model
/sync           # tiếp nhận sửa đổi, dựng lại tóm tắt, dòng thời gian, mạch cài cắm, quan hệ, trạng thái và ký ức văn phong
```

Khi phát hiện sửa đổi chưa tiếp nhận, việc khôi phục sáng tác, nhập tiếp và `/next` đều sẽ yêu cầu chạy `/sync` trước, tránh để sự thật cũ tiếp tục dẫn dắt chương mới. `/sync` **không** viết lại chính văn của bạn; model chỉ trích xuất lại sự thật chương và các sở thích văn phong tái dụng được từ chính văn mới. Các bản thẩm định, tóm tắt vòng cung/tập và ảnh chụp nhân vật bị ảnh hưởng sẽ mất hiệu lực và được Editor dựng lại; thay đổi cốt truyện sẽ được giao cho Architect cập nhật kế hoạch phía sau trước khi viết tiếp.

## Nhập một cuốn đã có (`/import`)

Gõ `/import <đường dẫn>` trong TUI để **biên dịch ngữ nghĩa** một cuốn tiểu thuyết sẵn có vào dự án. Một lần khởi động gắn với một cuốn (`output/novel` trong thư mục khởi động), nên thao tác này thường bắt đầu ngay ở **màn hình chào sau khi khởi động trong thư mục mới** — nó ngang hàng với "nhập yêu cầu tạo sách mới" và "đồng sáng tác", là cách thứ ba để bắt đầu một cuốn. Engine đang chạy thì lệnh này bị từ chối.

Đường ống chạy theo giai đoạn: chụp tệp nguồn (ingest) → LLM nhận diện ranh giới chương (segment) → xác nhận kết quả tách → trích xuất sự thật từng chương (analyze) → quy nạp phân tầng ra tiền đề / nhân vật / thế giới quan / dàn ý phân tầng / la bàn (synthesize) → xuất bản Foundation chính thức và ghi từng chương xuống đĩa (publish). Ranh giới chương do model phán đoán theo ngữ nghĩa, không dựa vào quy tắc tiêu đề cứng; phía Go chỉ quản toạ độ, kiểm tra phủ, tính idempotent và thứ tự.

Quy trình điển hình chỉ ba bước — nhập, đối chiếu, chờ xong:

```text
/import ~/tieu-thuyet.txt   # ① Khởi động: bảng hiển thị tiến độ, tách xong thì dừng
                            # ② Đối chiếu danh sách tiêu đề chương: bấm y để tiếp
                            # ③ Tự chạy phân tích → tổng hợp → xuất bản, xong thì dừng ở bước nghiệm thu
```

Tách sai? Bấm Esc đóng bảng, mô tả bằng lời rồi nhận diện lại (sẽ lại dừng để đối chiếu):

```text
/import --guide=Phần "Gian tấu" cũng là chương độc lập     # văn bản hướng dẫn đặt ở cuối lệnh
```

Toàn bộ tuỳ chọn (ba cái đầu được lưu bền, khôi phục sau sự cố vẫn tuân theo):

```text
/import ~/sach.txt --yes           # Không cần trực: tự chấp nhận kết quả tách và chạy hết
/import ~/sach.txt --story=closed  # Trả lời trước "trạng thái truyện còn nghi vấn": closed / open
/import ~/sach.txt --continue      # Nhập xong viết tiếp luôn, không dừng ở nghiệm thu
/import                            # Không tham số: khôi phục lần nhập dang dở
```

Điều kiện và khôi phục:

- Chỉ nhập được vào **sách rỗng** (chưa có chương hoàn thành); không hỗ trợ gộp một cuốn khác vào tác phẩm đã có. Tệp nguồn hỗ trợ `txt`/`md`, mã hoá UTF-8 / GB18030 (tự nhận diện; không giải mã tin cậy được thì báo lỗi rõ ràng).
- Sản phẩm mỗi giai đoạn nằm ở `meta/import/` và gắn theo vân tay đầu vào: gián đoạn hay thất bại thì chạy lại `/import` chỉ làm bù phần thiếu, không gọi lại model, không phải nhớ "đã nhập tới chương nào". Khi còn lần nhập dang dở, màn hình chào sẽ chủ động báo tiến độ (ví dụ "đã phân tích 210/300 chương"); trước khi khôi phục xong thì engine bị chặn, không lấy bản dở dang làm sách hoàn chỉnh để viết tiếp. Phản hồi thô của model khi thất bại được lưu ở `meta/import/failures/`.
- Nếu tổng hợp phán đoán trạng thái truyện là `uncertain`, đường ống dừng lại; dùng `--story=open|closed` rồi chạy lại.
- Mặc định sau khi xuất bản sẽ đặt một Hold nghiệm thu, chờ bạn xác nhận mới viết tiếp; `--continue` bỏ qua Hold này.
- Ba hàm ngữ nghĩa của đường ống nhập có thể chỉ định model riêng trong `roles` (xem [Dùng model khác nhau theo vai](#dùng-model-khác-nhau-theo-vai)).

> Chính văn đượcghi xuống đĩa nguyên văn thành chương đã hoàn thành, nên `/import` hợp cho việc "viết tiếp chính cuốn đó". Nếu chỉ muốn học thiết định để sáng tác hoàn toàn mới thì hãy tạo sách mới theo cách thường và mô tả phong cách mong muốn trong yêu cầu.

## Xuất bản (`/export`)

Gõ `/export` trong TUI để gộp các chương đã hoàn thành, mặc định TXT, ghi vào `{thư mục sách}/{tên sách}.txt`. Xuất là thao tác chỉ đọc, đang viết vẫn lấy được "thành phẩm hiện tại", không ảnh hưởng engine.

Định dạng quyết định bởi **đuôi của đường dẫn đầu ra** (`.txt` / `.epub`):

```text
/export                            # Mặc định TXT
/export ~/dom-sang.txt             # đuôi .txt → TXT
/export ~/dom-sang.epub            # đuôi .epub → EPUB (Apple Books / Kindle converter đọc được)
/export from=10 to=30 --overwrite  # khoảng chương + ghi đè
```

- **TXT** — `Tên sách` → phân cách tập → chính văn chương (chế độ phân tầng tự thêm phân cách tập). Hai loại dữ liệu nội bộ **không** vào bản xuất: premise (bản thiết kế sáng tác, chứa đối tượng độc giả, vùng cấm viết… vốn viết cho tác giả và engine đọc) và phân cách vòng cung (dưới góc nhìn độc giả, vòng cung là cấu trúc nội bộ quá chi tiết). Bộ xuất tự sinh dòng "Chương N Tiêu đề"; tiêu đề trùng do writer tự thêm trong chính văn sẽ bị lột bỏ.
- **EPUB** — Container chuẩn EPUB 3, gồm tên sách, metadata giới thiệu, trang bìa, mục lục và XHTML tách theo chương; định danh dẫn xuất ổn định từ nội dung (xuất lại cùng cuốn thì trình đọc nhận là bản cập nhật). Không kèm ảnh bìa.

Chương chưa hoàn thành trong khoảng sẽ bị bỏ qua và hiển thị trong kết quả, không tính là lỗi.

### Dùng model khác nhau theo vai

Dùng trường `roles` để gán model khác nhau cho từng tác nhân; vai không cấu hình thì dùng model mặc định:

```jsonc
{
  "provider": "openrouter",
  "model": "google/gemini-2.5-flash",
  "reasoning_effort": "medium",
  "providers": {
    "openrouter": { "api_key": "sk-or-v1-xxx", "base_url": "https://openrouter.ai/api/v1" },
    "anthropic": { "api_key": "sk-ant-xxx" }
  },
  "roles": {
    "writer": { "provider": "anthropic", "model": "claude-sonnet-4", "reasoning_effort": "high" },
    "architect": { "provider": "openrouter", "model": "google/gemini-2.5-pro", "reasoning_effort": "low" }
  }
}
```

Các vai cấu hình được: `architect` / `writer` / `editor`, cùng ba bậc hàm ngữ nghĩa của đường ống nhập là `import_segment` / `import_analyze` / `import_synthesize` (không cấu hình thì rơi về architect; có thể trỏ phần tách vốn máy móc hơn sang model rẻ hơn để tiết kiệm). Arbiter dùng thống nhất model mặc định, hiện chưa mở cấu hình vai riêng.

### Proxy tự dựng

Chọn Provider bất kỳ rồi điền địa chỉ proxy, hoặc dùng Custom Proxy và chỉ định loại giao thức API. `api_key` của proxy tự dựng là tuỳ chọn; proxy không cần xác thực thì bỏ trống được.

```jsonc
{
  "provider": "my-proxy",
  "model": "gpt-4o",
  "providers": {
    "my-proxy": {
      "type": "openai",
      "base_url": "https://proxy.example.com/v1",
      "extra": { "user_agent": "my-client/1.0" }
    }
  }
}
```

Provider hỗ trợ: `openrouter` / `anthropic` / `gemini` / `openai` / `deepseek` / `qwen` / `glm` / `grok` / `ollama` / `bedrock` / `freellmapi` và mọi proxy tự dựng.

Nếu proxy dùng giao thức Anthropic và chỉ cho client Claude Code truy cập, đặt `type` là `anthropic`, để `anthropic_beta` ở tầng đầu của `extra`, còn các HTTP header kiểu Stainless đặt trong `extra.headers`:

```jsonc
{
  "providers": {
    "claude-code-proxy": {
      "type": "anthropic",
      "api_key": "sk-xxx",
      "base_url": "https://proxy.example.com",
      "extra": {
        "user_agent": "claude-code/2.1.183",
        "anthropic_beta": "claude-code-20250219",
        "headers": { "X-Stainless-Lang": "js", "X-Stainless-Runtime": "node" }
      }
    }
  }
}
```

Nếu proxy dùng giao thức OpenAI/NewAPI và chỉ cho client Codex truy cập, đặt `type` là `openai`, ghi đè `extra.user_agent` và truyền các header nhận dạng Codex trong `extra.headers`; `Session_id` và `X-Codex-Turn-Metadata` nên thay bằng giá trị ngẫu nhiên ổn định:

```jsonc
{
  "providers": {
    "codex-proxy": {
      "type": "openai",
      "api_key": "sk-xxx",
      "base_url": "https://proxy.example.com/v1",
      "api": "responses",
      "models": [{ "name": "gpt-5.4", "context_window": 400000 }],
      "extra": {
        "user_agent": "codex-tui/0.142.3 (Mac OS 26.5.1; arm64)",
        "headers": {
          "Originator": "codex-tui",
          "Session_id": "replace-with-random-session-id",
          "X-Codex-Turn-Metadata": "replace-with-random-turn-metadata"
        }
      }
    }
  }
}
```

Về `api_key`:

- Các dịch vụ lưu trữ (`openrouter` / `anthropic` / `gemini` / `openai` / `deepseek` / `qwen` / `glm` / `grok`) thường bắt buộc điền
- `ollama` và `bedrock` cho phép bỏ trống; Bedrock cần cấu hình `region`, `access_key_id`, `secret_access_key` (tuỳ chọn `session_token`) trong `extra`
- Proxy tự dựng đã chỉ định `type` thì được bỏ trống
- `freellmapi` được xếp vào nhóm proxy tự dựng nên không bắt buộc, nhưng **thực tế vẫn cần** unified key; bỏ trống thì chỉ báo 401 lúc gửi yêu cầu chứ không chặn ở khâu cấu hình

Ví dụ `ollama` chạy cục bộ:

```jsonc
{
  "provider": "ollama",
  "model": "qwen3:latest",
  "providers": { "ollama": { "base_url": "http://localhost:11434/v1" } }
}
```

#### FreeLLMAPI

[FreeLLMAPI](https://github.com/tashfeenahmed/freellmapi) gom hạn mức miễn phí của vài chục nhà cung cấp về sau một endpoint `/v1` tương thích OpenAI, chạy bằng Docker trên máy bạn. Trong luồng hướng dẫn lần đầu hoặc `/config`, chọn **FreeLLMAPI (localhost:3001)** — giao thức, địa chỉ, model đều điền sẵn, chỉ cần dán unified key.

**Cần biết trước khi dùng**, ba điểm sau lấy từ tài liệu chính thức của họ và xung đột thật với cách chạy của dự án này:

1. **Router đổi backend theo từng yêu cầu** → tiền tố yêu cầu không ổn định, [cache prompt](docs/prompt-cache-design.md) gần như mất tác dụng, bảng cache trên TUI sẽ liên tục báo "đứt mạch". Đây là đòn bẩy chi phí lớn thứ hai khi chạy dài.
2. **Tài liệu API của họ không nhắc tới việc trả `usage`** → mức dùng, chi phí và ngân sách có thể về 0 hết, TUI sẽ hiện "thượng nguồn không trả usage". Khi đó cầu chì ngân sách coi như vô hiệu.
3. **Họ tự mô tả**: không có model tiên phong, càng cuối ngày chất lượng càng giảm (reset lúc 0h UTC), chỉ dành cho thử nghiệm cá nhân. Viết liền 200~500 chương chính là kịch bản dễ đụng trần hạn mức nhất — **hợp để chạy thử, nhưng cấu hình chủ lực cho truyện dài thì phải cân nhắc**.

`json_schema` mặc định tắt chính vì điểm 1: model thực thi đổi liên tục nên không thể cam kết hỗ trợ đầu ra có cấu trúc gốc; để Arbiter đi theo hợp đồng prompt sẽ ổn định hơn.

### Phong cách viết

Chuyển bằng trường `style` trong cấu hình:

- `default` — Phong cách chung
- `suspense` — Trinh thám, hồi hộp
- `fantasy` — Kỳ ảo, tiên hiệp
- `romance` — Ngôn tình

### Khử "mùi AI" và quy tắc tự định nghĩa

Có sẵn một baseline khử mùi AI: danh sách đen máy móc (câu sáo / từ mỏi, nằm trong mã tại `rules.SystemDefaults()`, kiểm tra tất định lúc commit) + tiêu chí ngữ nghĩa `assets/references/anti-ai-tone.md` (tiêm vào writer/editor để tránh và để trích dẫn làm bằng).

Muốn thêm sở thích riêng thì **không cần sửa mã nguồn**: đặt tệp `.md` bất kỳ vào `~/.ainovel/rules/` (toàn cục) hoặc `./.ainovel/rules/` (theo sách), gộp theo thứ tự tên tệp. **Cứ viết bằng lời thường** ("nhân vật chính đừng thánh mẫu", "tả nhiều cảm giác cơ thể", "mỗi chương khoảng 2000 từ", "đừng dùng cụm ở một mức độ nào đó") — không định dạng, không YAML. Hệ thống dùng model chuẩn hoá các yêu cầu tự nhiên đó thành ảnh chụp quy tắc của sách (khoảng độ dài / từ cấm / ngưỡng từ mỏi + sở thích văn phong), khi viết tự tuân theo và khi commit tự kiểm tra máy móc.

Lưu ý về đơn vị đếm: hệ thống đếm theo `language` — tiếng Trung đếm **ký tự**, tiếng Việt và tiếng Anh đếm **từ**. Viết "mỗi chương 2000 từ" trong một cuốn tiếng Việt nghĩa là 2000 từ thật, không phải 2000 ký tự.

### Văn phong tự định nghĩa (Voice Layer)

Chuẩn viết và tiêu chí khử mùi AI cũng ghi đè được, cũng **không cần sửa mã, không cần biên dịch lại**. Hai tầng ghi đè: `<thư mục output>/style/` (theo sách, đi cùng sách) > `~/.ainovel/style/` (toàn cục):

```
style/
├── voice.md                          # Đoạn nối vào chuẩn viết (bản gốc giữ nguyên, yêu cầu của bạn nối sau và ưu tiên cao hơn)
├── anti-ai-tone.md                   # Đoạn nối vào tiêu chí khử mùi AI (như trên)
├── styles/
│   └── xianxia.md                    # Thêm phong cách mới (tên tệp là tên phong cách; trùng tên bản gốc thì thay toàn bộ)
└── genres/
    └── xianxia/
        └── style-references.md       # Tham chiếu thể loại cho phong cách đó (thay toàn tệp)
```

Ghi nhớ ngắn: **văn bản mang tính chỉ dẫn (voice / anti-ai-tone) thì NỐI THÊM, còn bộ preset (styles / genres) thì THAY TOÀN TỆP**. Ưu tiên khi nối là chỉ dẫn cho model; ràng buộc cần cưỡng chế máy móc (từ cấm, độ dài) hãy viết vào thư mục rules ở trên. Sửa xong khởi động lại là có hiệu lực. Prompt thuộc loại giao thức thực thi **không** mở cho ghi đè — bất biến trong phối hợp được lớp công cụ canh giữ, và đó chính là lý do bạn có thể yên tâm đổi văn phong mà không làm hỏng hệ thống. Chi tiết xem `docs/voice-layer.md`.

## Cấu trúc đầu ra

Toàn bộ dữ liệu sáng tác (chương, dàn ý, nhân vật, tiến độ…) lưu trong thư mục output. Chạy lại sau khi gián đoạn sẽ tự viết tiếp từ tiến độ cũ. Xoá thư mục output nghĩa là sáng tác lại từ đầu.

```
output/{novel_name}/
├── book.md              # Tên sách và giới thiệu (bản chiếu dễ đọc)
├── chapters/            # Bản cuối (Markdown)
│   ├── 01.md
│   └── ...
├── summaries/           # Tóm tắt chương (JSON)
├── drafts/              # Bản nháp chương
├── reviews/             # Báo cáo thẩm định
├── timeline.jsonl       # Sự thật dòng thời gian (log nối thêm)
├── timeline.md          # Bản chiếu dễ đọc của dòng thời gian
├── premise.md           # Tiền đề câu chuyện
├── outline.json         # Dàn ý phẳng (chỉ gồm chương đã mở rộng)
├── layered_outline.json # Dàn ý phân tầng (chế độ truyện dài)
├── characters.json      # Hồ sơ nhân vật
├── world_rules.json     # Quy tắc thế giới
├── meta/
│   ├── book.json        # Nguồn sự thật duy nhất về tác phẩm
│   ├── compass.json     # La bàn hướng kết cục (chế độ truyện dài)
│   ├── progress.json    # Trạng thái tiến độ
│   ├── foreshadow.json  # Sổ cài cắm
│   ├── state_changes.jsonl  # Log nối thêm về thay đổi trạng thái nhân vật
│   ├── style_rules.json # Quy tắc văn phong (chắt lọc ở ranh giới vòng cung)
│   ├── decisions.jsonl  # Nhật ký kiểm toán mọi phán quyết ngữ nghĩa
│   ├── user_rules.json  # Ảnh chụp quy tắc người dùng đã chuẩn hoá
│   ├── snapshots/       # Ảnh chụp trạng thái nhân vật (truyện dài)
│   └── checkpoints.jsonl # Checkpoint mức step (nối thêm sau mỗi công cụ thành công)
```

## Khôi phục sau gián đoạn

Viết một bộ truyện dài có thể mất nhiều giờ hoặc nhiều ngày; sập, mất mạng, Ctrl+C đều là chuyện thường. Hệ thống **tự khôi phục khi chạy lại trong cùng thư mục**, không cần thao tác tay.

| Thời điểm gián đoạn | Hành vi khôi phục |
|---|---|
| Giai đoạn quy hoạch (đang dựng thế giới quan/dàn ý) | Kiểm tra thiết định đã lưu, tự bổ sung phần thiếu |
| Đang viết một chương (có nháp chưa nộp) | Viết tiếp chương đó, đọc nháp sẵn có |
| Đang thẩm định | Kích hoạt lại Editor |
| Hàng đợi viết lại/trau chuốt chưa rỗng | Tiếp tục xử lý các chương đang chờ |
| Mở rộng vòng cung/tập bị đứt | Tự phát hiện vòng cung/tập khung, kích hoạt Architect mở rộng |
| Can thiệp của người dùng chưa xong | Tiêm lại chỉ thị can thiệp lần trước |
| Gián đoạn khi đang viết bình thường | Viết tiếp từ chương kế |

### Nguyên lý

Mọi sản phẩm được lưu bền trong `output/`. Mỗi công cụ chạy thành công là ghi một checkpoint (`meta/checkpoints.jsonl`). Khi khởi động lại:

1. Đọc `progress.json` + checkpoint gần nhất + các tín hiệu chờ xử lý
2. Sinh chỉ thị khôi phục chính xác tới mức step (ví dụ "chương 7 đã có draft, hãy tiếp tục check_consistency")
3. Engine tính lại tuyến từ store rồi chạy tiếp — không có phiên nào cần khôi phục, tính idempotent của checkpoint bảo đảm phái lại là an toàn

> Ghi tệp dùng thao tác nguyên tử temp + fsync + rename, mất điện giữa chừng cũng không hỏng dữ liệu cũ.

## Nghiệm thu từng chương

Mặc định hệ thống chạy chế độ `auto`, sáng tác liên tục. Khi cần đọc duyệt từng chương và tránh việc hệ thống viết chương mới trong lúc bạn đang đọc, hãy bật cổng nghiệm thu tất định:

```text
/review on   # Bật nghiệm thu từng chương; xong việc hiện tại sẽ dừng trước chương mới kế tiếp
/next        # Chỉ cho qua đúng một chương; thẩm định và bảo trì cấu trúc vòng cung/tập vẫn tự chạy
/review off  # Trở lại tự động
```

Giấy phép gắn với số chương cụ thể. Chương chỉ tiêu thụ giấy phép sau khi trạng thái nộp đã sạch và checkpoint commit đã ghi xuống đĩa, nên tiến trình sập giữa lúc nộp cũng không vô tình viết dư một chương. Viết lại, trau chuốt, thẩm định và bảo trì cấu trúc **không** tính là "chương mới" nên không bị cổng chặn.

## Can thiệp thời gian thực (Steer)

Trong lúc sáng tác, gõ ý kiến sửa vào ô nhập bất cứ lúc nào, **không cần tạm dừng hay khởi động lại**.

```
❯ Đẩy tuyến tình cảm lên chương 4, tăng cảnh đối đầu giữa nam nữ chính
```

Bấm Enter, hệ thống tự động:

1. Ghi chỉ thị can thiệp vào `run.json` (để khôi phục sau sự cố)
2. Arbiter phán quyết ngay (câu hỏi tra cứu phản hồi trong vài giây; hành động điều khiển được nộp an toàn ở ranh giới chương)
3. Thực thi theo phán quyết: sửa thiết định thì giao Architect, viết lại chương đã có thì giao Editor đưa vào hàng đợi, quy tắc viết thì ghi xuống đĩa ngay — mỗi phán quyết đều kiểm toán và phát lại được

| Chỉ thị can thiệp | Phản ứng có thể có |
|---|---|
| "Đổi nhân vật chính thành nữ" | Sửa thiết định nhân vật, đánh giá xem chương đã viết có cần viết lại |
| "Đẩy tuyến tình cảm lên chương 4" | Điều chỉnh dàn ý, có thể viết lại chương 4 và các chương sau |
| "Thêm một nhân vật phản diện" | Cập nhật hồ sơ nhân vật và quy tắc thế giới, đưa vào các chương sau |
| "Nhịp chậm quá, đẩy nhanh lên" | Điều chỉnh mật độ dàn ý các chương sau |
| "Viết tới chương 20" | Sáng tác liên tục tới khi chương 20 nộp ổn định rồi dừng |

## Triết lý thiết kế

> **Tầng sự thật thì tất định, tầng ngữ nghĩa thì tự chủ.** Mô hình được tự do ở chỗ không thể kiểm chứng (viết gì, viết thế nào), và bị ràng buộc ở chỗ kiểm chứng được (thứ tự, idempotent, giai đoạn).

### Chia ba, càng đơn giản càng ổn định

- **Chuyển trạng thái liệt kê được thì giao cho mã** — "kế tiếp phái ai" là đọc sự thật rồi tra bảng (`flow.Route` là hàm thuần, kiểm thử vét cạn hàng vạn tổ hợp): tỉ lệ sai gần 0, không tốn LLM
- **Phán đoán có ranh giới rõ thì giao Arbiter** — sự thật vào, quyết định có cấu trúc ra, có kiểm tra máy móc đỡ phía sau, mỗi lần đều ghi lại để phát lại
- **Sáng tác mở thì giao Worker** — trong phạm vi một chương, Writer hoàn toàn tự chủ; công cụ thất bại thì trả về lỗi có cấu trúc kèm gợi ý lối ra để LLM tự sửa
- **Chỉ hardcode ranh giới, không hardcode phán đoán** — mã chỉ canh những bất biến chứng minh được; những đánh đổi sáng tác không liệt kê nổi thì giao cho model, không dùng từ khoá hay ngưỡng điểm để giả vờ là hiểu
- **Công cụ chỉ trả sự thật** — IO nguyên tử một tệp + lỗi tường minh + phát lại idempotent; commit chương dùng Saga bền vững + checkpoint, giá trị trả về là các trường sự thật JSON (`verdict` / `pending_rewrites` / `arc_end` / `next_chapter`), không kèm bất kỳ chuỗi chỉ thị nào
- **Rào chắn theo sự thật, không theo hành vi** — CheckpointDeltaGuard của Worker chỉ công nhận sản phẩm đã ghi xuống đĩa: chưa nộp mà đòi kết thúc là bị chặn; khi model hành xử đúng thì rào chắn không tốn gì
- **Từ chối điều phối phức tạp** — Không task queue, không policy engine. Một vòng lặp tuần tự + một bảng quyết định + vài hàm phán quyết là toàn bộ luồng điều khiển
- **Model càng mạnh, lợi càng lớn** — Chất lượng sáng tác và phán quyết tăng tuyến tính theo model; lớp vỏ tất định không cần sửa một dòng

### Vòng khép kín tự động

```
"Viết một bộ trinh thám" → dựng thế giới quan → thiết kế nhân vật → quy hoạch dàn ý
                         → viết từng chương → thẩm định chất lượng → tự viết lại
                         → tóm tắt vòng cung → ảnh chụp nhân vật → sách hoàn chỉnh
```

### Tách rời sự thật và chỉ thị

Công cụ chỉ trả sự thật; "bước kế tiếp" do Engine tính lại từ tầng sự thật ở mỗi vòng:

- `commit_chapter` / `save_review` ghi các sự thật có cấu trúc (`verdict` là một trong `accept`/`polish`/`rewrite`, cùng `pending_rewrites` / `arc_end` / hồ phản hồi dàn ý), không kèm chuỗi `[hệ thống]` nào
- `flow.Route` đọc `Progress` + `Outline` để suy ra chỉ thị kế tiếp; mỗi lần đổi bảng quyết định đều phải sửa đặc tả vét cạn trước rồi mới sửa cài đặt
- Mọi quyết định ngữ nghĩa đều rơi vào `meta/decisions.jsonl` phục vụ kiểm toán, phát lại ngoại tuyến và hồi quy A/B

Nhờ vậy chỉ thị không bị nuốt mất trong chuỗi lệnh gọi, cũng không trôi dạt trong sản phẩm của công cụ. Sửa một bug về luồng chỉ cần đổi một nhánh + một dòng đặc tả.

## Ngôn ngữ và bản địa hoá

Dự án tách rạch ròi ba loại văn bản, mỗi loại một quy tắc:

| Loại | Ngôn ngữ | Do đâu quyết định |
|---|---|---|
| Prompt và tài liệu tham chiếu gửi cho model (`assets/`, mô tả schema công cụ, lỗi công cụ trả cho model) | Tiếng Anh | Cố định |
| Sản phẩm sáng tác (chính văn, tiêu đề, dàn ý, tóm tắt, thẩm định) | Theo `language` | Chỉ thị ngôn ngữ cứng nối vào cuối prompt của mọi Worker, kể cả đường nhập / mô phỏng / sửa tay |
| Chữ trên giao diện (TUI, tiến độ `/import`, sidebar) | Theo `ui_language` | Bảng từ vựng `internal/i18n/catalog/{zh,vi,en}.json` |
| Log `slog` | Tiếng Trung | Hướng người bảo trì, giữ một ngôn ngữ để dễ tra khi bạn dán vào issue |

Vài cổng kiểm thử canh giữ ranh giới này để nó không bị xói mòn theo thời gian: chuỗi tiếng Trung hardcode trong tầng giao diện bị chặn (bắt cả dấu câu toàn rộng như `（）、；`), mọi prompt lấy từ `bundle.Prompts` bắt buộc phải đi qua `WithOutputDirective`, và hàm dựng sidebar không được viết cứng ký tự CJK.

> Lưu ý về tương thích: tiêu đề mục trong `premise.md` là **khoá được mã phân tích**. Tên chuẩn hiện là tiếng Anh (`## Genre and Tone`, `## Central Conflict`…), nhưng các tên tiếng Trung cũ vẫn được giữ làm alias, nên sách tạo từ bản trước vẫn đọc được bình thường.

## Ngăn xếp công nghệ

- **Go 1.25** — Ngôn ngữ chính
- **[agentcore](https://github.com/voocel/agentcore)** — Nhân Agent tối giản (tool-calling + streaming)
- **[litellm](https://github.com/voocel/litellm)** — Lớp thích ứng LLM thống nhất
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — Framework TUI cho terminal

## License

MIT
