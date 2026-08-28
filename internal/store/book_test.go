package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/lang"
)

func TestBookStorePersistsCanonicalDataAndReadableProjection(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Book.Save(domain.BookMetadata{Title: " 长夜将明 ", Synopsis: " 少年守住最后一盏灯。 "}); err != nil {
		t.Fatal(err)
	}
	book, err := s.Book.Load()
	if err != nil {
		t.Fatal(err)
	}
	if book.Title != "长夜将明" || book.Synopsis != "少年守住最后一盏灯。" {
		t.Fatalf("unexpected book metadata: %+v", book)
	}
	projection, err := os.ReadFile(filepath.Join(dir, "book.md"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(projection); !strings.Contains(text, "《长夜将明》") || !strings.Contains(text, "少年守住最后一盏灯。") {
		t.Fatalf("unexpected book projection: %s", text)
	}
	if err := s.Book.Save(domain.BookMetadata{Title: "空简介"}); err == nil {
		t.Fatal("empty synopsis must be rejected")
	}
}

// book.md 是读者会看到的产物，排版必须跟创作语言走。
//
// 这条来自实机：跑越南语小说时 book.md 的小标题是"## 简介"，
// 书名还套着中文书名号《》——正文是越南语，包装是中文。
func TestBookProjectionFollowsLanguage(t *testing.T) {
	cases := []struct {
		lang        string
		wantTitle   string
		wantSection string
	}{
		{"zh", "# 《光斑》", "## 简介"},
		{"vi", "# 光斑", "## Giới thiệu"},
		{"en", "# 光斑", "## Synopsis"},
	}
	for _, c := range cases {
		t.Run(c.lang, func(t *testing.T) {
			got := renderBook(domain.BookMetadata{Title: "光斑", Synopsis: "S"}, lang.Resolve(c.lang))
			if !strings.HasPrefix(got, c.wantTitle+"\n") {
				t.Errorf("标题行 = %q，期望以 %q 开头", strings.SplitN(got, "\n", 2)[0], c.wantTitle)
			}
			if !strings.Contains(got, c.wantSection) {
				t.Errorf("缺少小节标题 %q，实际：\n%s", c.wantSection, got)
			}
		})
	}
}

// 零值 Profile 必须等价中文——NewStore 的只读调用点就是这么用的，
// 一旦退化成空标题，诊断导出会把 book.md 写坏。
func TestBookProjectionZeroValueIsChinese(t *testing.T) {
	var zero lang.Profile
	got := renderBook(domain.BookMetadata{Title: "光斑", Synopsis: "S"}, zero)
	want := renderBook(domain.BookMetadata{Title: "光斑", Synopsis: "S"}, lang.Resolve("zh"))
	if got != want {
		t.Errorf("零值 Profile 的投影与中文不一致：\n%q\nvs\n%q", got, want)
	}
}
