package manager

import (
	"encoding/xml"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/player"
	daisy "github.com/kvark128/daisyonline"
)

type ContentList struct {
	Name  string
	ID    string
	Items []ContentItem
}

type LibraryContentItem struct {
	lib       *Library
	name      string
	conf      config.Book
	resources []daisy.Resource
	metadata  *daisy.ContentMetadata
}

func NewLibraryContentItem(lib *Library, id string) *LibraryContentItem {
	item := NewLibraryContentItemWithName(lib, id, id)
	if metadata, err := item.ContentMetadata(); err == nil {
		item.name = metadata.Metadata.Title
	}
	return item
}

func NewLibraryContentItemWithName(lib *Library, id, name string) *LibraryContentItem {
	item := &LibraryContentItem{
		lib:  lib,
		name: name,
		conf: config.Book{
			ID:    id,
			Speed: player.DEFAULT_SPEED,
		},
	}

	if conf, err := lib.service.RecentBooks.Book(id); err == nil {
		item.conf = conf
	}
	return item
}

func (item *LibraryContentItem) ID() string {
	return item.conf.ID
}

func (item *LibraryContentItem) Name() string {
	return item.name
}

func (item *LibraryContentItem) Resources() ([]daisy.Resource, error) {
	if item.resources != nil {
		return item.resources, nil
	}
	r, err := item.lib.GetContentResources(item.conf.ID)
	if err != nil {
		return nil, err
	}
	item.resources = r.Resources
	return item.resources, nil
}

func (item *LibraryContentItem) ContentMetadata() (*daisy.ContentMetadata, error) {
	if item.metadata != nil {
		return item.metadata, nil
	}
	metadata, err := item.lib.GetContentMetadata(item.conf.ID)
	if err != nil {
		return nil, err
	}
	item.metadata = metadata
	return item.metadata, nil
}

func (item *LibraryContentItem) Issue() error {
	_, err := item.lib.IssueContent(item.conf.ID)
	return err
}

func (item *LibraryContentItem) Return() error {
	_, err := item.lib.ReturnContent(item.conf.ID)
	return err
}

func (item *LibraryContentItem) Config() config.Book {
	return item.conf
}

func (item *LibraryContentItem) SetConfig(conf config.Book) {
	item.conf = conf
	item.lib.service.RecentBooks.SetBook(conf)
}

type LocalContentItem struct {
	storage   *LocalStorage
	name      string
	conf      config.Book
	resources []daisy.Resource
	metadata  *daisy.ContentMetadata
}

func NewLocalContentItem(storage *LocalStorage, id string) *LocalContentItem {
	item := &LocalContentItem{
		storage: storage,
		name:    id,
		conf: config.Book{
			ID:    id,
			Speed: player.DEFAULT_SPEED,
		},
	}

	if metadata, err := item.ContentMetadata(); err == nil {
		item.name = metadata.Metadata.Title
	}

	if conf, err := storage.conf.LocalBooks.Book(id); err == nil {
		item.conf = conf
	}
	return item
}

func (item *LocalContentItem) ID() string {
	return item.conf.ID
}

func (item *LocalContentItem) Name() string {
	return item.name
}

func (item *LocalContentItem) Resources() ([]daisy.Resource, error) {
	if item.resources != nil {
		return item.resources, nil
	}

	rsrc := make([]daisy.Resource, 0)
	itemPath := filepath.Join(config.UserData(), item.conf.ID)
	walker := func(targpath string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		localURI, err := filepath.Rel(itemPath, targpath)
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

	if err := filepath.Walk(itemPath, walker); err != nil {
		return nil, err
	}
	item.resources = rsrc
	return item.resources, nil
}

func (item *LocalContentItem) ContentMetadata() (*daisy.ContentMetadata, error) {
	if item.metadata != nil {
		return item.metadata, nil
	}

	path := filepath.Join(config.UserData(), item.conf.ID, MetadataFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	metadata := new(daisy.ContentMetadata)
	d := xml.NewDecoder(f)
	if err := d.Decode(metadata); err != nil {
		return nil, err
	}
	item.metadata = metadata
	return item.metadata, nil
}

func (item *LocalContentItem) Config() config.Book {
	return item.conf
}

func (item *LocalContentItem) SetConfig(conf config.Book) {
	item.conf = conf
	item.storage.conf.LocalBooks.SetBook(conf)
}
