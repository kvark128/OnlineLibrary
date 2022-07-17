package manager

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
)

type Book struct {
	ContentItem
	*player.Player
	conf       config.Book
	globalConf *config.Config
}

func NewBook(conf *config.Config, contentItem ContentItem, logger *log.Logger, statusBar *gui.StatusBar) (*Book, error) {
	book := &Book{
		ContentItem: contentItem,
		conf:        contentItem.Config(),
		globalConf:  conf,
	}

	rsrc, err := book.Resources()
	if err != nil {
		return nil, fmt.Errorf("GetContentResources: %v", err)
	}

	if book.conf.Bookmarks == nil {
		book.conf.Bookmarks = make(map[string]config.Bookmark)
	}

	bookDir := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Name()))
	book.Player = player.NewPlayer(bookDir, rsrc, book.globalConf.General.OutputDevice, logger, statusBar)
	book.SetSpeed(book.conf.Speed)
	book.SetTimerDuration(book.globalConf.General.PauseTimer)
	book.SetVolume(book.globalConf.General.Volume)
	if bookmark, err := book.Bookmark(config.ListeningPosition); err == nil {
		book.SetFragment(bookmark.Fragment)
		book.SetPosition(bookmark.Position)
	}
	return book, nil
}

func (book *Book) SetBookmarkWithName(name string) error {
	if name == "" {
		return fmt.Errorf("bookmark name is missing")
	}
	for id := 10; id <= 255; id++ {
		bookmarkID := fmt.Sprintf("bookmark%d", id)
		if _, err := book.Bookmark(bookmarkID); err != nil {
			book.setBookmark(bookmarkID, name)
			return nil
		}
	}
	return fmt.Errorf("all bookmark ids already used")
}

func (book *Book) SetBookmarkWithID(id string) {
	book.setBookmark(id, "")
}

func (book *Book) setBookmark(id, name string) {
	var bookmark config.Bookmark
	bookmark.Name = name
	bookmark.Fragment = book.Fragment()
	// For convenience, we truncate the time to the nearest tenth of a second
	bookmark.Position = book.Position().Truncate(time.Millisecond * 100)
	book.conf.Bookmarks[id] = bookmark
}

func (book *Book) RemoveBookmark(id string) {
	delete(book.conf.Bookmarks, id)
}

func (book *Book) Bookmark(id string) (config.Bookmark, error) {
	if bookmark, ok := book.conf.Bookmarks[id]; ok {
		return bookmark, nil
	}
	return config.Bookmark{}, fmt.Errorf("bookmark not found")
}

func (book *Book) ToBookmark(id string) error {
	bookmark, err := book.Bookmark(id)
	if err != nil {
		return err
	}
	if book.Fragment() == bookmark.Fragment {
		book.SetPosition(bookmark.Position)
		return nil
	}
	book.Stop()
	book.SetFragment(bookmark.Fragment)
	book.SetPosition(bookmark.Position)
	book.PlayPause()
	return nil
}

func (book *Book) Bookmarks() map[string]string {
	bookmarks := make(map[string]string)
	for id, bookmark := range book.conf.Bookmarks {
		bookmarks[id] = bookmark.Name
	}
	return bookmarks
}

func (book *Book) Close() {
	book.SetBookmarkWithID(config.ListeningPosition)
	book.globalConf.General.Volume = book.Volume()
	book.conf.Speed = book.Speed()
	book.SetConfig(book.conf)
	book.Stop()
}
