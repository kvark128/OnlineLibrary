package config

import (
	"time"
)

type RecentBooks []Book

func (rb RecentBooks) Update(id string, fragment int, elapsedTime time.Duration) {
	for i, b := range rb {
		if b.ID == id {
			rb[i].Fragment = fragment
			rb[i].ElapsedTime = elapsedTime
			return
		}
	}

	book := Book{
		ID:          id,
		Fragment:    fragment,
		ElapsedTime: elapsedTime,
	}
	rb = append(rb, book)

	if len(rb) > 8 {
		rb = rb[len(rb)-8:]
	}

	Conf.Services[len(Conf.Services)-1].RecentBooks = rb
}

func (rb RecentBooks) GetPosition(id string) (int, time.Duration) {
	for i, b := range rb {
		if b.ID == id {
			rb[i], rb[len(rb)-1] = rb[len(rb)-1], rb[i]
			return b.Fragment, b.ElapsedTime
		}
	}
	return 0, 0
}
