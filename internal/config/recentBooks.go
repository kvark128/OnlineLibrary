package config

import (
	"time"
)

type RecentBooks []Book

func (rb RecentBooks) SetBook(id, name string, fragment int, elapsedTime time.Duration) {
	for i := range rb {
		if rb[i].ID == id {
			rb[i].Name = name
			rb[i].Fragment = fragment
			rb[i].ElapsedTime = elapsedTime
			rb.SetCurrentBook(id)
			return
		}
	}

	book := Book{
		Name:        name,
		ID:          id,
		Fragment:    fragment,
		ElapsedTime: elapsedTime,
	}

	rb = append(rb, book)
	rb.SetCurrentBook(id)

	if len(rb) > 256 {
		rb = rb[:256]
	}

	Conf.Services[0].RecentBooks = rb
}

func (rb RecentBooks) Book(id string) Book {
	for _, b := range rb {
		if b.ID == id {
			return b
		}
	}
	return Book{}
}

func (rb RecentBooks) SetCurrentBook(id string) {
	for i := range rb {
		if rb[i].ID == id {
			rb[0], rb[i] = rb[i], rb[0]
		}
	}
}

func (rb RecentBooks) CurrentBook() Book {
	if len(rb) != 0 {
		return rb[0]
	}
	return Book{}
}
