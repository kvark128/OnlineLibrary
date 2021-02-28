package manager

import (
	daisy "github.com/kvark128/daisyonline"
)

type LibraryContentItem struct {
	library *Library
	id      string
	label   daisy.Label
}

func NewLibraryContentItem(library *Library, id, name string) *LibraryContentItem {
	return &LibraryContentItem{
		library: library,
		id:      id,
		label:   daisy.Label{Text: name},
	}
}

func (ci *LibraryContentItem) ID() string {
	return ci.id
}

func (ci *LibraryContentItem) Label() daisy.Label {
	return ci.label
}

func (ci *LibraryContentItem) Resources() ([]daisy.Resource, error) {
	r, err := ci.library.GetContentResources(ci.id)
	if err != nil {
		return nil, err
	}
	return r.Resources, nil
}

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

func (cl *LibraryContentList) Item(index int) ContentItem {
	book := cl.books.ContentItems[index]
	return NewLibraryContentItem(cl.library, book.ID, book.Label.Text)
}
