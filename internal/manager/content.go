package manager

import (
	"encoding/xml"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type LibraryContentItem struct {
	library *Library
	book    config.Book
	id      string
	label   daisy.Label
}

func NewLibraryContentItem(library *Library, id, name string) *LibraryContentItem {
	ci := &LibraryContentItem{
		library: library,
		book: config.Book{
			Name: name,
			ID:   id,
		},
		id:    id,
		label: daisy.Label{Text: name},
	}
	if book, err := ci.library.service.RecentBooks.Book(ci.id); err == nil {
		ci.book = book
	}
	return ci
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

func (ci *LibraryContentItem) Bookmark(bookmarkID string) (config.Bookmark, error) {
	return ci.book.Bookmark(bookmarkID)
}

func (ci *LibraryContentItem) SetBookmark(bookmarkID string, bookmark config.Bookmark) {
	ci.book.SetBookmark(bookmarkID, bookmark)
	ci.library.service.RecentBooks.SetBook(ci.book)
}

func (ci *LibraryContentItem) ContentMetadata() (*daisy.ContentMetadata, error) {
	return ci.library.GetContentMetadata(ci.id)
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

func (cl *LibraryContentList) ID() string {
	return cl.books.ID
}

func (cl *LibraryContentList) Len() int {
	return len(cl.books.ContentItems)
}

func (cl *LibraryContentList) Item(index int) ContentItem {
	book := cl.books.ContentItems[index]
	return NewLibraryContentItem(cl.library, book.ID, book.Label.Text)
}

type LocalContentItem struct {
	book  config.Book
	id    string
	label daisy.Label
	path  string
}

func NewLocalContentItem(path string) *LocalContentItem {
	ci := &LocalContentItem{
		book: config.Book{
			Name: filepath.Base(path),
			ID:   filepath.Base(path),
		},
	}
	ci.path = path
	ci.label.Text = filepath.Base(path)
	if book, err := config.Conf.LocalBooks.Book(ci.book.ID); err == nil {
		ci.book = book
	}
	return ci
}

func (ci *LocalContentItem) Label() daisy.Label {
	return ci.label
}

func (ci *LocalContentItem) ID() string {
	return ci.label.Text
}

func (ci *LocalContentItem) Resources() ([]daisy.Resource, error) {
	rsrc := make([]daisy.Resource, 0)

	walker := func(targpath string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		localURI, err := filepath.Rel(ci.path, targpath)
		if err != nil {
			return err
		}
		r := daisy.Resource{
			LocalURI: localURI,
			Size:     info.Size(),
		}
		rsrc = append(rsrc, r)
		return nil
	}

	if err := filepath.Walk(ci.path, walker); err != nil {
		return nil, err
	}
	return rsrc, nil
}

func (ci *LocalContentItem) Bookmark(bookmarkID string) (config.Bookmark, error) {
	return ci.book.Bookmark(bookmarkID)
}

func (ci *LocalContentItem) SetBookmark(bookmarkID string, bookmark config.Bookmark) {
	ci.book.SetBookmark(bookmarkID, bookmark)
	config.Conf.LocalBooks.SetBook(ci.book)
}

func (ci *LocalContentItem) ContentMetadata() (*daisy.ContentMetadata, error) {
	path := filepath.Join(ci.path, MetadataFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	md := new(daisy.ContentMetadata)
	d := xml.NewDecoder(f)
	if err := d.Decode(md); err != nil {
		return nil, err
	}
	return md, nil
}

type LocalContentList struct {
	books []string
	id    string
	label daisy.Label
}

func NewLocalContentList() (*LocalContentList, error) {
	cl := &LocalContentList{
		label: daisy.Label{Text: "Локальные книги"},
	}
	userData := config.UserData()
	entrys, err := os.ReadDir(userData)
	if err != nil {
		return nil, err
	}
	for _, e := range entrys {
		if e.IsDir() {
			path := filepath.Join(userData, e.Name())
			cl.books = append(cl.books, path)
		}
	}
	return cl, nil
}

func (cl *LocalContentList) Label() daisy.Label {
	return cl.label
}

func (cl *LocalContentList) ID() string {
	return cl.id
}

func (cl *LocalContentList) Len() int {
	return len(cl.books)
}

func (cl *LocalContentList) Item(index int) ContentItem {
	path := cl.books[index]
	return NewLocalContentItem(path)
}
