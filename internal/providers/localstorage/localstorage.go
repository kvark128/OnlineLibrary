package localstorage

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/content"
	"github.com/kvark128/dodp"
	"github.com/leonelquinteros/gotext"
)

type LocalStorage struct {
	conf *config.Config
}

func NewLocalStorage(conf *config.Config) *LocalStorage {
	return &LocalStorage{conf: conf}
}

func (s *LocalStorage) ContentList(string) (*content.List, error) {
	userData := config.UserData()
	entrys, err := os.ReadDir(userData)
	if err != nil {
		return nil, err
	}

	lst := &content.List{
		ID:   dodp.Issued,
		Name: gotext.Get("Local books"),
	}

	for _, e := range entrys {
		if e.IsDir() {
			path := filepath.Join(userData, e.Name())
			item := NewContentItem(s, path)
			lst.Items = append(lst.Items, item)
		}
	}

	return lst, nil
}

func (s *LocalStorage) LastContentListID() (string, error) {
	return dodp.Issued, nil
}

func (s *LocalStorage) ContentItem(id string) (content.Item, error) {
	lst, err := s.ContentList("")
	if err != nil {
		return nil, err
	}

	for _, item := range lst.Items {
		if item.ID() == id {
			return item, nil
		}
	}
	return nil, errors.New("content item not found")
}

func (s *LocalStorage) LastContentItemID() (string, error) {
	book, err := s.conf.LocalBooks.LastBook()
	if err != nil {
		return "", err
	}
	return book.ID, nil
}

func (s *LocalStorage) Tidy(ids []string) {
	s.conf.LocalBooks.Tidy(ids)
}
