package manager

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type ContentItem interface {
	Label() daisy.Label
	ID() string
	Resources() ([]daisy.Resource, error)
	ContentMetadata() (*daisy.ContentMetadata, error)
	Config() config.Book
	SetConfig(config.Book)
}

type ContentList interface {
	Label() daisy.Label
	ID() string
	Len() int
	Item(int) ContentItem
}

type Returner interface {
	Return() error
}

type Issuer interface {
	Issue() error
}
