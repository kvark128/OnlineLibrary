package library

import (
	"errors"
	"fmt"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/content"
	"github.com/kvark128/dodp"
)

type Library struct {
	*dodp.Client
	service           *config.Service
	serviceAttributes *dodp.ServiceAttributes
	conf              *config.Config
}

func NewLibrary(conf *config.Config, service *config.Service) (*Library, error) {
	client := dodp.NewClient(service.URL, config.HTTPTimeout)
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

func (l *Library) ContentList(id string) (*content.List, error) {
	contentList, err := l.GetContentList(id, 0, -1)
	if err != nil {
		return nil, err
	}

	lst := &content.List{
		Name: contentList.Label.Text,
		ID:   contentList.ID,
	}

	for _, contentItem := range contentList.ContentItems {
		item := NewContentItemWithLabel(l, contentItem.ID, contentItem.Label.Text)
		lst.Items = append(lst.Items, item)
	}

	l.service.OpenBookshelfOnLogin = id == dodp.Issued
	return lst, nil
}

func (l *Library) LastContentListID() (string, error) {
	if !l.service.OpenBookshelfOnLogin {
		return "", errors.New("last content list not available")
	}
	return dodp.Issued, nil
}

func (l *Library) ContentItem(id string) (content.Item, error) {
	item := NewContentItem(l, id)
	return item, nil
}

func (l *Library) LastContentItemID() (string, error) {
	book, err := l.service.RecentBooks.LastBook()
	if err != nil {
		return "", err
	}
	return book.ID, nil
}

func (l *Library) Service() *config.Service {
	return l.service
}

func (l *Library) ServiceAttributes() *dodp.ServiceAttributes {
	return l.serviceAttributes
}

func (l *Library) GetQuestions(ur *dodp.UserResponses) (*dodp.Questions, error) {
	questions, err := l.Client.GetQuestions(ur)
	if err == nil {
		l.service.OpenBookshelfOnLogin = false
	}
	return questions, err
}
