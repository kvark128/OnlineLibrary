package config

type RecentBooks []Book

func (rb RecentBooks) Update(id string, fragment, elapsedTime int) {
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
