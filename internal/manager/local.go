package manager

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	daisy "github.com/kvark128/daisyonline"
)

type LocalContentItem struct {
	id    string
	label daisy.Label
	path  string
}

func NewLocalContentItem(path string) *LocalContentItem {
	ci := &LocalContentItem{}
	ci.path = path
	ci.label.Text = filepath.Base(path)
	return ci
}

func (ci *LocalContentItem) Label() daisy.Label {
	return ci.label
}

func (ci *LocalContentItem) ID() string {
	return ci.label.Text
}

func (ci *LocalContentItem) Resources() ([]daisy.Resource, error) {
	rsrc := make([]daisy.Resource, 0)

	walker := func(targpath string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		localURI, err := filepath.Rel(ci.path, targpath)
		if err != nil {
			return err
		}
		r := daisy.Resource{
			LocalURI: localURI,
			Size:     info.Size(),
		}
		rsrc = append(rsrc, r)
		return nil
	}

	if err := filepath.Walk(ci.path, walker); err != nil {
		return nil, err
	}
	return rsrc, nil
}

type LocalContentList struct {
	books []string
	id    string
	label daisy.Label
}

func NewLocalContentList() (*LocalContentList, error) {
	cl := &LocalContentList{}
	userData := config.UserData()
	entrys, err := os.ReadDir(userData)
	if err != nil {
		return nil, err
	}
	for _, e := range entrys {
		if e.IsDir() {
			path := filepath.Join(userData, e.Name())
			cl.books = append(cl.books, path)
		}
	}
	return cl, nil
}

func (cl *LocalContentList) Label() daisy.Label {
	return cl.label
}

func (cl *LocalContentList) ID() string {
	return cl.id
}

func (cl *LocalContentList) Len() int {
	return len(cl.books)
}

func (cl *LocalContentList) Item(index int) ContentItem {
	path := cl.books[index]
	return NewLocalContentItem(path)
}
