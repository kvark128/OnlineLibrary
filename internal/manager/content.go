package manager

import (
	daisy "github.com/kvark128/daisyonline"
)

type LibraryContentList struct {
	library *Library
	books   *daisy.ContentList
}

func NewLibraryContentList(library *Library, contentID string) (*LibraryContentList, error) {
	contentList, err := library.GetContentList(contentID, 0, -1)
	if err != nil {
		return nil, err
	}

	cl := &LibraryContentList{
		library: library,
		books:   contentList,
	}

	return cl, nil
}

func (cl *LibraryContentList) Label() daisy.Label {
	return cl.books.Label
}

func (cl *LibraryContentList) Len() int {
	return len(cl.books.ContentItems)
}

func (cl *LibraryContentList) Item(index int) daisy.ContentItem {
	return cl.books.ContentItems[index]
}

func (cl *LibraryContentList) ItemResources(index int) ([]daisy.Resource, error) {
	book := cl.books.ContentItems[index]
	r, err := cl.library.GetContentResources(book.ID)
	if err != nil {
		return nil, err
	}
	return r.Resources, nil
}
