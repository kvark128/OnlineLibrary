package providers

import (
	"github.com/kvark128/OnlineLibrary/internal/content"
	"github.com/kvark128/dodp"
)

type Provider interface {
	ContentList(string) (*content.List, error)
	LastContentListID() (string, error)
	ContentItem(string) (content.Item, error)
	LastContentItemID() (string, error)
}

type Questioner interface {
	GetQuestions(*dodp.UserResponses) (*dodp.Questions, error)
}
