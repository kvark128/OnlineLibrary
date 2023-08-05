package content

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/dodp"
)

type List struct {
	Name  string
	ID    string
	Items []Item
}

type Item interface {
	Name() (string, error)
	Label() string
	ID() string
	Resources() ([]dodp.Resource, error)
	ContentMetadata() (*dodp.ContentMetadata, error)
	Config() *config.Book
	SaveConfig()
}

type Issuer interface {
	Issue() error
}

type Returner interface {
	Return() error
}
