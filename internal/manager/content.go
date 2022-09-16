package manager

import (
	"encoding/xml"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/dodp"
)

type ContentList struct {
	Name  string
	ID    string
	Items []ContentItem
}

type LibraryContentItem struct {
	library *Library
	conf    config.Book
}

func NewLibraryContentItem(library *Library, id, name string) *LibraryContentItem {
	ci := &LibraryContentItem{
		library: library,
		conf: config.Book{
			Name:  name,
			ID:    id,
			Speed: player.DEFAULT_SPEED,
		},
	}

	if conf, err := ci.library.service.RecentBooks.Book(id); err == nil {
		ci.conf = conf
	}
	ci.conf.Name = name
	return ci
}

func (ci *LibraryContentItem) ID() string {
	return ci.conf.ID
}

func (ci *LibraryContentItem) Name() string {
	return ci.conf.Name
}

func (ci *LibraryContentItem) Resources() ([]dodp.Resource, error) {
	r, err := ci.library.GetContentResources(ci.conf.ID)
	if err != nil {
		return nil, err
	}
	return r.Resources, nil
}

func (ci *LibraryContentItem) ContentMetadata() (*dodp.ContentMetadata, error) {
	return ci.library.GetContentMetadata(ci.conf.ID)
}

func (ci *LibraryContentItem) Issue() error {
	_, err := ci.library.IssueContent(ci.conf.ID)
	return err
}

func (ci *LibraryContentItem) Return() error {
	_, err := ci.library.ReturnContent(ci.conf.ID)
	return err
}

func (ci *LibraryContentItem) Config() config.Book {
	return ci.conf
}

func (ci *LibraryContentItem) SetConfig(conf config.Book) {
	ci.conf = conf
	ci.library.service.RecentBooks.SetBook(conf)
}

type LocalContentItem struct {
	storage *LocalStorage
	path    string
	conf    config.Book
}

func NewLocalContentItem(storage *LocalStorage, path string) *LocalContentItem {
	ci := &LocalContentItem{
		storage: storage,
		path:    path,
		conf: config.Book{
			Name:  filepath.Base(path),
			ID:    filepath.Base(path),
			Speed: player.DEFAULT_SPEED,
		},
	}

	if conf, err := ci.storage.conf.LocalBooks.Book(ci.conf.ID); err == nil {
		ci.conf = conf
	}
	return ci
}

func (ci *LocalContentItem) Name() string {
	return ci.conf.Name
}

func (ci *LocalContentItem) ID() string {
	return ci.conf.ID
}

func (ci *LocalContentItem) Resources() ([]dodp.Resource, error) {
	rsrc := make([]dodp.Resource, 0)
	walker := func(targpath string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		localURI, err := filepath.Rel(ci.path, targpath)
		if err != nil {
			return err
		}
		r := dodp.Resource{
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

func (ci *LocalContentItem) ContentMetadata() (*dodp.ContentMetadata, error) {
	path := filepath.Join(ci.path, MetadataFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	md := new(dodp.ContentMetadata)
	d := xml.NewDecoder(f)
	if err := d.Decode(md); err != nil {
		return nil, err
	}
	return md, nil
}

func (ci *LocalContentItem) Config() config.Book {
	return ci.conf
}

func (ci *LocalContentItem) SetConfig(conf config.Book) {
	ci.conf = conf
	ci.storage.conf.LocalBooks.SetBook(conf)
}
