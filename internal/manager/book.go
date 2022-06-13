package manager

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
)

type Book struct {
	ContentItem
	*player.Player
	conf config.Book
}

func NewBook(contentItem ContentItem) (*Book, error) {
	book := &Book{
		ContentItem: contentItem,
		conf:        contentItem.Config(),
	}

	rsrc, err := book.Resources()
	if err != nil {
		return nil, fmt.Errorf("GetContentResources: %v", err)
	}

	bookDir := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Name()))
	book.Player = player.NewPlayer(bookDir, rsrc, config.Conf.General.OutputDevice)
	book.SetSpeed(book.conf.Speed)
	book.SetTimerDuration(config.Conf.General.PauseTimer)
	book.SetVolume(config.Conf.General.Volume)
	if bookmark, err := book.conf.Bookmark(config.ListeningPosition); err == nil {
		book.SetFragment(bookmark.Fragment)
		book.SetPosition(bookmark.Position)
	}
	return book, nil
}

func (book *Book) SetBookmark(id string) {
	var bookmark config.Bookmark
	bookmark.Fragment = book.Fragment()
	// For convenience, we truncate the time to the nearest tenth of a second
	bookmark.Position = book.Position().Truncate(time.Millisecond * 100)
	book.conf.SetBookmark(id, bookmark)
}

func (book *Book) Close() {
	book.SetBookmark(config.ListeningPosition)
	config.Conf.General.Volume = book.Volume()
	book.conf.Speed = book.Speed()
	book.SetConfig(book.conf)
	book.Stop()
}
