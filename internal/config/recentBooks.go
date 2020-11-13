package config

import (
	"time"
)

func (s *Service) SetBook(id, name string, fragment int, elapsedTime time.Duration) {
	for i := range s.RecentBooks {
		if s.RecentBooks[i].ID == id {
			s.RecentBooks[i].Name = name
			s.RecentBooks[i].Fragment = fragment
			s.RecentBooks[i].ElapsedTime = elapsedTime
			s.SetCurrentBook(id)
			return
		}
	}

	book := Book{
		Name:        name,
		ID:          id,
		Fragment:    fragment,
		ElapsedTime: elapsedTime,
	}

	s.RecentBooks = append(s.RecentBooks, book)
	s.SetCurrentBook(id)

	if len(s.RecentBooks) > 256 {
		s.RecentBooks = s.RecentBooks[:256]
	}
}

func (s *Service) Book(id string) Book {
	for _, b := range s.RecentBooks {
		if b.ID == id {
			return b
		}
	}
	return Book{}
}

func (s *Service) SetCurrentBook(id string) {
	for i, b := range s.RecentBooks {
		if b.ID == id {
			copy(s.RecentBooks[1:i+1], s.RecentBooks[0:i])
			s.RecentBooks[0] = b
			break
		}
	}
}

func (s *Service) CurrentBook() Book {
	if len(s.RecentBooks) != 0 {
		return s.RecentBooks[0]
	}
	return Book{}
}
