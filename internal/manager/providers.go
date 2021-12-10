package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/log"
	daisy "github.com/kvark128/daisyonline"
)

type Library struct {
	*daisy.Client
	service           *config.Service
	serviceAttributes *daisy.ServiceAttributes
}

func NewLibrary(service *config.Service) (*Library, error) {
	client := daisy.NewClient(service.URL, config.HTTPTimeout)
	success, err := client.LogOn(service.Credentials.Username, service.Credentials.Password)
	if err != nil {
		return nil, fmt.Errorf("logOn operation: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("logOn operation returned false")
	}

	serviceAttributes, err := client.GetServiceAttributes()
	if err != nil {
		return nil, fmt.Errorf("getServiceAttributes operation: %w", err)
	}

	success, err = client.SetReadingSystemAttributes(&config.ReadingSystemAttributes)
	if err != nil {
		return nil, fmt.Errorf("setReadingSystemAttributes operation: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("setReadingSystemAttributes operation returned false")
	}

	library := &Library{
		Client:            client,
		service:           service,
		serviceAttributes: serviceAttributes,
	}

	return library, nil
}

func (l *Library) ContentList(id string) (*ContentList, error) {
	contentList, err := l.GetContentList(id, 0, -1)
	if err != nil {
		return nil, err
	}

	lst := &ContentList{
		Name: contentList.Label.Text,
		ID:   contentList.ID,
	}

	for _, contentItem := range contentList.ContentItems {
		item := NewLibraryContentItem(l, contentItem.ID, contentItem.Label.Text)
		lst.Items = append(lst.Items, item)
	}

	return lst, nil
}

func (l *Library) ContentItem(id string) (ContentItem, error) {
	book, err := l.service.RecentBooks.Book(id)
	if err != nil {
		log.Warning("%v", err)
	}

	item := NewLibraryContentItem(l, id, book.Name)
	return item, nil
}

func (l *Library) LastItem() (ContentItem, error) {
	book, err := l.service.RecentBooks.LastBook()
	if err != nil {
		return nil, err
	}
	return l.ContentItem(book.ID)
}

func (l *Library) SetLastItem(contentItem ContentItem) {
	l.service.RecentBooks.SetBook(contentItem.Config())
}

type LocalStorage struct{}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

func (s *LocalStorage) ContentList(id string) (*ContentList, error) {
	userData := config.UserData()
	entrys, err := os.ReadDir(userData)
	if err != nil {
		return nil, err
	}

	lst := &ContentList{
		Name: "Локальные книги",
		ID:   id,
	}

	for _, e := range entrys {
		if e.IsDir() {
			path := filepath.Join(userData, e.Name())
			item := NewLocalContentItem(path)
			lst.Items = append(lst.Items, item)
		}
	}

	return lst, nil
}

func (s *LocalStorage) ContentItem(id string) (ContentItem, error) {
	lst, err := s.ContentList("")
	if err != nil {
		return nil, err
	}

	for _, item := range lst.Items {
		if item.ID() == id {
			return item, nil
		}
	}
	return nil, errors.New("content item not found")
}

func (s *LocalStorage) LastItem() (ContentItem, error) {
	book, err := config.Conf.LocalBooks.LastBook()
	if err != nil {
		return nil, err
	}
	return s.ContentItem(book.ID)
}

func (s *LocalStorage) SetLastItem(contentItem ContentItem) {
	config.Conf.LocalBooks.SetBook(contentItem.Config())
}
