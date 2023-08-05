package library

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/dodp"
)

type LibraryContentItem struct {
	library  *Library
	label    string
	metadata *dodp.ContentMetadata
	conf     config.Book
}

func NewLibraryContentItem(library *Library, id string) *LibraryContentItem {
	return NewLibraryContentItemWithLabel(library, id, "")
}

func NewLibraryContentItemWithLabel(library *Library, id, label string) *LibraryContentItem {
	return &LibraryContentItem{
		library: library,
		label:   label,
		conf:    library.service.RecentBooks.Book(id, player.DEFAULT_SPEED),
	}
}

func (ci *LibraryContentItem) Name() (string, error) {
	md, err := ci.ContentMetadata()
	if err != nil {
		return "", err
	}
	return md.Metadata.Title, nil
}

func (ci *LibraryContentItem) Label() string {
	return ci.label
}

func (ci *LibraryContentItem) ID() string {
	return ci.conf.ID
}

func (ci *LibraryContentItem) Resources() ([]dodp.Resource, error) {
	r, err := ci.library.GetContentResources(ci.conf.ID)
	if err != nil {
		return nil, err
	}
	return r.Resources, nil
}

func (ci *LibraryContentItem) ContentMetadata() (*dodp.ContentMetadata, error) {
	if ci.metadata != nil {
		return ci.metadata, nil
	}
	md, err := ci.library.GetContentMetadata(ci.conf.ID)
	if err != nil {
		return nil, err
	}
	ci.metadata = md
	return ci.metadata, nil
}

func (ci *LibraryContentItem) Issue() error {
	_, err := ci.library.IssueContent(ci.conf.ID)
	return err
}

func (ci *LibraryContentItem) Return() error {
	_, err := ci.library.ReturnContent(ci.conf.ID)
	return err
}

func (ci *LibraryContentItem) Config() *config.Book {
	return &ci.conf
}

func (ci *LibraryContentItem) SaveConfig() {
	ci.library.service.RecentBooks.SetBook(ci.conf)
}
