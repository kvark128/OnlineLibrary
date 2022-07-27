package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
	"github.com/leonelquinteros/gotext"
)

type Library struct {
	*daisy.Client
	service           *config.Service
	serviceAttributes *daisy.ServiceAttributes
	conf              *config.Config
}

func NewLibrary(conf *config.Config, service *config.Service) (*Library, error) {
	client := daisy.NewClient(service.URL, config.HTTPTimeout)
	success, err := client.LogOn(service.Username, service.Password)
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
		conf:              conf,
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

	l.service.OpenBookshelfOnLogin = id == daisy.Issued
	return lst, nil
}

func (l *Library) LastContentListID() (string, error) {
	if !l.service.OpenBookshelfOnLogin {
		return "", errors.New("last content list not available")
	}
	return daisy.Issued, nil
}

func (l *Library) ContentItem(id string) (ContentItem, error) {
	book, _ := l.service.RecentBooks.Book(id)
	item := NewLibraryContentItem(l, id, book.Name)
	l.service.RecentBooks.SetBook(item.Config())
	return item, nil
}

func (l *Library) LastContentItemID() (string, error) {
	book, err := l.service.RecentBooks.LastBook()
	if err != nil {
		return "", err
	}
	return book.ID, nil
}

func (l *Library) GetQuestions(ur *daisy.UserResponses) (*daisy.Questions, error) {
	questions, err := l.Client.GetQuestions(ur)
	if err == nil {
		l.service.OpenBookshelfOnLogin = false
	}
	return questions, err
}

type LocalStorage struct {
	conf *config.Config
}

func NewLocalStorage(conf *config.Config) *LocalStorage {
	return &LocalStorage{conf: conf}
}

func (s *LocalStorage) ContentList(string) (*ContentList, error) {
	userData := config.UserData()
	entrys, err := os.ReadDir(userData)
	if err != nil {
		return nil, err
	}

	lst := &ContentList{
		Name: gotext.Get("Local books"),
	}

	for _, e := range entrys {
		if e.IsDir() {
			path := filepath.Join(userData, e.Name())
			item := NewLocalContentItem(s, path)
			lst.Items = append(lst.Items, item)
		}
	}

	return lst, nil
}

func (s *LocalStorage) LastContentListID() (string, error) {
	return "", nil
}

func (s *LocalStorage) ContentItem(id string) (ContentItem, error) {
	lst, err := s.ContentList("")
	if err != nil {
		return nil, err
	}

	for _, item := range lst.Items {
		if item.ID() == id {
			s.conf.LocalBooks.SetBook(item.Config())
			return item, nil
		}
	}
	return nil, errors.New("content item not found")
}

func (s *LocalStorage) LastContentItemID() (string, error) {
	book, err := s.conf.LocalBooks.LastBook()
	if err != nil {
		return "", err
	}
	return book.ID, nil
}
