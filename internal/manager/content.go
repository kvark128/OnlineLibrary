package manager

import (
	"encoding/xml"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type ContentList struct {
	Name  string
	ID    string
	Items []ContentItem
}

type LibraryContentItem struct {
	library *Library
	book    config.Book
}

func NewLibraryContentItem(library *Library, id, name string) *LibraryContentItem {
	ci := &LibraryContentItem{
		library: library,
		book: config.Book{
			Name: name,
			ID:   id,
		},
	}

	if book, err := ci.library.service.RecentBooks.Book(id); err == nil {
		ci.book = book
	}

	ci.book.Name = name
	return ci
}

func (ci *LibraryContentItem) ID() string {
	return ci.book.ID
}

func (ci *LibraryContentItem) Name() string {
	return ci.book.Name
}

func (ci *LibraryContentItem) Resources() ([]daisy.Resource, error) {
	r, err := ci.library.GetContentResources(ci.book.ID)
	if err != nil {
		return nil, err
	}
	return r.Resources, nil
}

func (ci *LibraryContentItem) ContentMetadata() (*daisy.ContentMetadata, error) {
	return ci.library.GetContentMetadata(ci.book.ID)
}

func (ci *LibraryContentItem) Issue() error {
	_, err := ci.library.IssueContent(ci.book.ID)
	return err
}

func (ci *LibraryContentItem) Return() error {
	_, err := ci.library.ReturnContent(ci.book.ID)
	return err
}

func (ci *LibraryContentItem) Config() config.Book {
	return ci.book
}

func (ci *LibraryContentItem) SetConfig(book config.Book) {
	ci.book = book
	ci.library.service.RecentBooks.SetBook(book)
}

type LocalContentItem struct {
	path string
	book config.Book
}

func NewLocalContentItem(path string) *LocalContentItem {
	ci := &LocalContentItem{
		book: config.Book{
			Name: filepath.Base(path),
			ID:   filepath.Base(path),
		},
		path: path,
	}

	if book, err := config.Conf.LocalBooks.Book(ci.book.ID); err == nil {
		ci.book = book
	}
	return ci
}

func (ci *LocalContentItem) Name() string {
	return ci.book.Name
}

func (ci *LocalContentItem) ID() string {
	return ci.book.ID
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

func (ci *LocalContentItem) Config() config.Book {
	return ci.book
}

func (ci *LocalContentItem) SetConfig(book config.Book) {
	ci.book = book
	config.Conf.LocalBooks.SetBook(book)
}
