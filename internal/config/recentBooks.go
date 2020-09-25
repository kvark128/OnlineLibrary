package config

import (
	"time"
)

type RecentBooks []Book

func (rb RecentBooks) Update(id string, fragment int, elapsedTime time.Duration) {
	for i, b := range rb {
		if b.ID == id {
			b.Fragment = fragment
			b.ElapsedTime = elapsedTime
			if i != 0 {
				copy(rb[1:1+i], rb[:i])
			}
			rb[0] = b
			return
		}
	}

	book := Book{
		ID:          id,
		Fragment:    fragment,
		ElapsedTime: elapsedTime,
	}
	rb = append(rb, book)
	copy(rb[1:len(rb)], rb[:len(rb)-1])
	rb[0] = book

	if len(rb) > 256 {
		rb = rb[:256]
	}

	Conf.Services[0].RecentBooks = rb
}

func (rb RecentBooks) GetPosition(id string) (int, time.Duration) {
	for _, b := range rb {
		if b.ID == id {
			return b.Fragment, b.ElapsedTime
		}
	}
	return 0, 0
}
