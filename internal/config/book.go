package config

import (
	"errors"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/util"
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
	// Values for speed when playing the book
	Speed int `json:"speed,omitempty"`
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

type BookSet []Book

func (setP *BookSet) Book(id string) (Book, error) {
	for _, b := range *setP {
		if b.ID == id {
			return b, nil
		}
	}
	return Book{}, BookNotFound
}

func (setP *BookSet) SetBook(book Book) {
	set := *setP
	defer func() { *setP = set }()

	for i, b := range set {
		if b.ID == book.ID {
			set[i] = book
			set[0], set[i] = set[i], set[0]
			return
		}
	}
	set = append(set, book)
	i := len(set) - 1
	set[0], set[i] = set[i], set[0]
}

func (setP *BookSet) LastBook() (Book, error) {
	if len(*setP) == 0 {
		return Book{}, BookNotFound
	}
	return (*setP)[0], nil
}

func (setP *BookSet) Tidy(ids []string) {
	books := make([]Book, 0, len(ids))
	for _, b := range *setP {
		if util.StringInSlice(b.ID, ids) {
			books = append(books, b)
		}
	}
	*setP = books
}
