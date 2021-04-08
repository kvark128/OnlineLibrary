package config

import (
	"errors"
	"time"
)

type Bookmark struct {
	Fragment int           `json:"fragment"`
	Position time.Duration `json:"position"`
}

type Book struct {
	Name        string              `json:"name"`
	ID          string              `json:"id"`
	Fragment    int                 `json:"fragment"`
	ElapsedTime time.Duration       `json:"elapsed_time"`
	Bookmarks   map[string]Bookmark `json:"bookmarks"`
}

func (b *Book) SetBookmark(id string, bookmark Bookmark) {
	if b.Bookmarks == nil {
		b.Bookmarks = make(map[string]Bookmark)
	}
	b.Bookmarks[id] = bookmark
}

func (b *Book) Bookmark(id string) (Bookmark, error) {
	if bookmark, ok := b.Bookmarks[id]; ok {
		return bookmark, nil
	}
	return Bookmark{}, errors.New("no bookmark")
}
