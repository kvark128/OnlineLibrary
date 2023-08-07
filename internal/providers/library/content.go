package library

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/dodp"
)

type ContentItem struct {
	library  *Library
	label    string
	metadata *dodp.ContentMetadata
	conf     config.Book
}

func NewContentItem(library *Library, id string) *ContentItem {
	return NewContentItemWithLabel(library, id, "")
}

func NewContentItemWithLabel(library *Library, id, label string) *ContentItem {
	return &ContentItem{
		library: library,
		label:   label,
		conf:    library.service.RecentBooks.Book(id, player.DEFAULT_SPEED),
	}
}

func (ci *ContentItem) Name() (string, error) {
	md, err := ci.ContentMetadata()
	if err != nil {
		return "", err
	}
	return md.Metadata.Title, nil
}

func (ci *ContentItem) Label() string {
	return ci.label
}

func (ci *ContentItem) ID() string {
	return ci.conf.ID
}

func (ci *ContentItem) Resources() ([]dodp.Resource, error) {
	r, err := ci.library.GetContentResources(ci.conf.ID)
	if err != nil {
		return nil, err
	}
	return r.Resources, nil
}

func (ci *ContentItem) ContentMetadata() (*dodp.ContentMetadata, error) {
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

func (ci *ContentItem) Issue() error {
	_, err := ci.library.IssueContent(ci.conf.ID)
	return err
}

func (ci *ContentItem) Return() error {
	_, err := ci.library.ReturnContent(ci.conf.ID)
	return err
}

func (ci *ContentItem) Config() *config.Book {
	return &ci.conf
}

func (ci *ContentItem) SaveConfig() {
	ci.library.service.RecentBooks.SetBook(ci.conf)
}
