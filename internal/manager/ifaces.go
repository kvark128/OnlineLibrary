package manager

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type ContentItem interface {
	Name() string
	ID() string
	Resources() ([]daisy.Resource, error)
	ContentMetadata() (*daisy.ContentMetadata, error)
	Config() config.Book
	SetConfig(config.Book)
}

type Questioner interface {
	GetQuestions(*daisy.UserResponses) (*daisy.Questions, error)
}

type Returner interface {
	Return() error
}

type Issuer interface {
	Issue() error
}

type Provider interface {
	ContentList(string) (*ContentList, error)
	ContentItem(string) (ContentItem, error)
	LastItem() (ContentItem, error)
	SetLastItem(ContentItem)
}
