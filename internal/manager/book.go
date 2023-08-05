package manager

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/content"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/player"
)

type Book struct {
	content.Item
	*player.Player
	title string
	conf  *config.Book
}

func NewBook(outputDevice string, contentItem content.Item, logger *log.Logger, statusBar *gui.StatusBar) (*Book, error) {
	book := &Book{
		Item: contentItem,
		conf: contentItem.Config(),
	}

	name, err := book.Name()
	if err != nil {
		return nil, err
	}

	dir, err := config.BookDir(name)
	if err != nil {
		return nil, err
	}

	rsrc, err := book.Resources()
	if err != nil {
		return nil, fmt.Errorf("GetContentResources: %v", err)
	}

	if book.conf.Bookmarks == nil {
		book.conf.Bookmarks = make(map[string]config.Bookmark)
	}

	book.title = name
	book.Player = player.NewPlayer(dir, rsrc, outputDevice, logger, statusBar)
	book.SetSpeed(book.conf.Speed)
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

func (book *Book) Save() {
	book.SetBookmarkWithID(config.ListeningPosition)
	book.conf.Speed = book.Speed()
	book.SaveConfig()
}

func (book *Book) Title() string {
	return book.title
}
