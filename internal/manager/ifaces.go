package manager

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/dodp"
)

type ContentItem interface {
	Name() string
	ID() string
	Resources() ([]dodp.Resource, error)
	ContentMetadata() (*dodp.ContentMetadata, error)
	Config() config.Book
	SetConfig(config.Book)
}

type Questioner interface {
	GetQuestions(*dodp.UserResponses) (*dodp.Questions, error)
}

type Returner interface {
	Return() error
}

type Issuer interface {
	Issue() error
}

type Provider interface {
	ContentList(string) (*ContentList, error)
	LastContentListID() (string, error)
	ContentItem(string) (ContentItem, error)
	LastContentItemID() (string, error)
}
