package store

import (
	"fmt"
	"os"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/lang"
)

// BookStore 管理作品对外信息，meta/book.json 是唯一事实源，book.md 是可读投影。
// lp 决定 book.md 这份可读投影用什么语言排版（标题号、"简介"小标题）。
// 零值安全：等价于中文，存量书目逐字节不变。
type BookStore struct {
	io *IO
	lp lang.Profile
}

func NewBookStore(io *IO, lp lang.Profile) *BookStore { return &BookStore{io: io, lp: lp} }

// Load 读取作品信息；尚未生成时返回 nil。
func (s *BookStore) Load() (*domain.BookMetadata, error) {
	var book domain.BookMetadata
	if err := s.io.ReadJSON("meta/book.json", &book); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	book = book.Normalized()
	if err := book.Validate(); err != nil {
		return nil, err
	}
	return &book, nil
}

// Save 保存规范化的作品信息及其可读投影。
func (s *BookStore) Save(book domain.BookMetadata) error {
	book = book.Normalized()
	if err := book.Validate(); err != nil {
		return err
	}
	return s.io.WithWriteLock(func() error {
		if err := s.io.WriteJSONUnlocked("meta/book.json", book); err != nil {
			return err
		}
		return s.io.WriteMarkdownUnlocked("book.md", renderBook(book, s.lp))
	})
}

func renderBook(book domain.BookMetadata, lp lang.Profile) string {
	return fmt.Sprintf("%s\n\n## %s\n\n%s\n", lp.BookTitleLine(book.Title), lp.SynopsisLabel(), book.Synopsis)
}
