package localstorage

import (
	"encoding/xml"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/dodp"
)

type ContentItem struct {
	storage   *LocalStorage
	resources []dodp.Resource
	metadata  *dodp.ContentMetadata
	conf      config.Book
}

func NewContentItem(storage *LocalStorage, id string) *ContentItem {
	return &ContentItem{
		storage: storage,
		conf:    storage.conf.LocalBooks.Book(id, player.DEFAULT_SPEED),
	}
}

func (ci *ContentItem) path() string {
	return filepath.Join(ci.storage.path, ci.conf.ID)
}

func (ci *ContentItem) Name() (string, error) {
	label := ci.Label()
	if dir, err := config.BookDir(label); err == nil && dir == ci.path() {
		return label, nil
	}
	return ci.conf.ID, nil
}

func (ci *ContentItem) Label() string {
	md, err := ci.ContentMetadata()
	if err != nil {
		return ci.conf.ID
	}
	return md.Metadata.Title
}

func (ci *ContentItem) ID() string {
	return ci.conf.ID
}

func (ci *ContentItem) Resources() ([]dodp.Resource, error) {
	if ci.resources != nil {
		return ci.resources, nil
	}
	path := ci.path()
	rsrc := make([]dodp.Resource, 0)

	walker := func(targpath string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		localURI, err := filepath.Rel(path, targpath)
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

	if err := filepath.Walk(path, walker); err != nil {
		return nil, err
	}
	ci.resources = rsrc
	return ci.resources, nil
}

func (ci *ContentItem) ContentMetadata() (*dodp.ContentMetadata, error) {
	if ci.metadata != nil {
		return ci.metadata, nil
	}
	path := filepath.Join(ci.path(), config.MetadataFileName)
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
	ci.metadata = md
	return ci.metadata, nil
}

func (ci *ContentItem) Config() *config.Book {
	return &ci.conf
}

func (ci *ContentItem) SaveConfig() {
	ci.storage.conf.LocalBooks.SetBook(ci.conf)
}
