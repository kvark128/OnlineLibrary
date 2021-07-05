package config

import (
	"errors"
	"time"
)

var (
	BookNotFound      = errors.New("book not found")
	BookmarkNotFound  = errors.New("bookmark not found")
	ListeningPosition = "listening_position"
)

type Bookmark struct {
	// Name of the bookmark. Reserved for future use
	Name string `json:"name,omitempty"`
	// Fragment of a book with the bookmark
	Fragment int `json:"fragment"`
	// Offset from the beginning of the fragment
	Position time.Duration `json:"position"`
}

type Book struct {
	// Human-readable title of the book
	Name string `json:"name"`
	// Unique ID of the book
	ID string `json:"id"`
	// Set of bookmarks in the book
	Bookmarks map[string]Bookmark `json:"bookmarks,omitempty"`
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
	return Bookmark{}, BookmarkNotFound
}
