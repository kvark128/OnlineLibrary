package config

import (
	"errors"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/util"
)

var (
	BookNotFound      = errors.New("book not found")
	ListeningPosition = "listening_position"
)

type Bookmark struct {
	// Name of the bookmark. Reserved for future use
	Name string `yaml:"name,omitempty"`
	// Fragment of a book with the bookmark
	Fragment int `yaml:"fragment"`
	// Offset from the beginning of the fragment
	Position time.Duration `yaml:"position"`
}

type Book struct {
	// Unique ID of the book
	ID string `yaml:"id"`
	// Values for speed when playing the book
	Speed float64 `yaml:"speed,omitempty"`
	// Set of bookmarks in the book
	Bookmarks map[string]Bookmark `yaml:"bookmarks,omitempty"`
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
