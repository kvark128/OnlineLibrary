package manager

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/books"
	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/connection"
	"github.com/kvark128/OnlineLibrary/internal/content"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/util/buffer"
	"github.com/kvark128/dodp"
	"github.com/leonelquinteros/gotext"

	"github.com/kvark128/OnlineLibrary/internal/providers"
	"github.com/kvark128/OnlineLibrary/internal/providers/library"
	"github.com/kvark128/OnlineLibrary/internal/providers/localstorage"
)

const (
	CRLF = "\r\n"
)

var (
	BookDescriptionNotAvailable = errors.New("book description not available")
	OperationNotSupported       = errors.New("operation not supported")
)

type ChoiceItem struct {
	dodp.Choice
}

func (c ChoiceItem) Label() string {
	return c.Choice.Label.Text
}

type Manager struct {
	provider      providers.Provider
	mainWnd       *gui.MainWnd
	logger        *log.Logger
	book          *books.Book
	contentList   *content.List
	questions     *dodp.Questions
	userResponses []dodp.UserResponse
	lastInputText string
}

func NewManager(mainWnd *gui.MainWnd, logger *log.Logger) *Manager {
	return &Manager{mainWnd: mainWnd, logger: logger}
}

func (m *Manager) Start(conf *config.Config, done chan<- bool) {
	m.logger.Debug("Entering to Manager Loop")
	defer func() {
		if p := recover(); p != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			m.logger.Error("Manager panic: %v:\n\n%v", p, string(buf[:n]))
			os.Exit(1)
		}
		m.cleaning(conf)
		m.logger.Debug("Exiting from Manager Loop")
		done <- true
	}()

	for message := range m.mainWnd.MsgChan() {
		switch message.Code {
		case msg.ACTIVATE_MENU:
			if m.contentList != nil {
				book := m.mainWnd.MainListBox().CurrentItem().(content.Item)
				if m.book == nil || m.book.ID() != book.ID() {
					if err := m.setBook(conf, book); err != nil {
						m.messageBoxError(fmt.Errorf("Set book: %w", err))
						break
					}
				}
				m.book.PlayPause()
			} else if m.questions != nil {
				questionIndex := len(m.userResponses)
				questionID := m.questions.MultipleChoiceQuestion[questionIndex].ID
				value := m.mainWnd.MainListBox().CurrentItem().(ChoiceItem).ID
				m.userResponses = append(m.userResponses, dodp.UserResponse{QuestionID: questionID, Value: value})
				questionIndex++
				if questionIndex < len(m.questions.MultipleChoiceQuestion) {
					m.setMultipleChoiceQuestion(questionIndex)
					break
				}
				m.setInputQuestion()
			}

		case msg.OPEN_BOOKSHELF:
			m.setContentList(dodp.Issued)

		case msg.OPEN_NEWBOOKS:
			m.setContentList(dodp.New)

		case msg.MAIN_MENU:
			m.setQuestions(dodp.UserResponse{QuestionID: dodp.Default})

		case msg.SEARCH_BOOK:
			m.setQuestions(dodp.UserResponse{QuestionID: dodp.Search})

		case msg.MENU_BACK:
			m.setQuestions(dodp.UserResponse{QuestionID: dodp.Back})

		case msg.SET_PROVIDER:
			id, ok := message.Data.(string)
			if !ok {
				if conf.General.Provider == "" {
					break
				}
				id = conf.General.Provider
			}

			if m.book != nil {
				m.book.Pause(true)
			}

			var provider providers.Provider
			var err error
			if id == config.LocalStorageID {
				provider = localstorage.NewLocalStorage(conf)
			} else {
				var service *config.Service
				service, err = conf.ServiceByID(id)
				if err != nil {
					m.logger.Debug("Get service %v: %v", id, err)
					break
				}
				provider, err = library.NewLibrary(conf, service)
			}

			if err != nil {
				m.messageBoxError(fmt.Errorf("Provider creating %v: %w", id, err))
				break
			}
			m.setProvider(provider, conf, id)

		case msg.LIBRARY_ADD:
			service := new(config.Service)
			if gui.CredentialsEntryDialog(m.mainWnd, service) != gui.DlgCmdOK || service.Name == "" {
				m.logger.Warning("Library adding: pressed Cancel button or len(service.Name) == 0")
				break
			}

			if _, err := conf.ServiceByName(service.Name); err == nil {
				gui.MessageBox(m.mainWnd, gotext.Get("Error"), gotext.Get("Account \"%v\" already exists", service.Name), gui.MsgBoxOK|gui.MsgBoxIconError)
				break
			}

			// Service id creation. Maximum index value of 256 is intended to protect against an infinite loop
			for i := 0; i < 256; i++ {
				id := fmt.Sprintf("library%d", i)
				if _, err := conf.ServiceByID(id); err != nil {
					service.ID = id
					m.logger.Debug("Created new service id: %v", id)
					break
				}
			}

			provider, err := library.NewLibrary(conf, service)
			if err != nil {
				m.messageBoxError(fmt.Errorf("library creating: %w", err))
				break
			}
			conf.SetService(service)
			m.setProvider(provider, conf, service.ID)

		case msg.LIBRARY_REMOVE:
			lib, ok := m.provider.(*library.Library)
			if !ok {
				break
			}
			msg := gotext.Get("Are you sure you want to delete the account \"%v\"?\nAll saved bookmarks of all books in this library will also be deleted.\nThis action cannot be undone.", lib.Service().Name)
			if gui.MessageBox(m.mainWnd, gotext.Get("Deleting an account"), msg, gui.MsgBoxYesNo|gui.MsgBoxIconQuestion) != gui.DlgCmdYes {
				break
			}
			conf.RemoveService(lib.Service())
			m.cleaning(conf)
			m.mainWnd.MenuBar().SetProvidersMenu(conf.Services, "")

		case msg.ISSUE_BOOK:
			if m.contentList == nil {
				break
			}
			book := m.mainWnd.MainListBox().CurrentItem().(content.Item)
			if err := m.issueBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Adding book: %w", err))
				break
			}
			title := gotext.Get("Warning")
			msg := gotext.Get("Selected book has been added to the bookshelf")
			gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconInformation)

		case msg.REMOVE_BOOK:
			if m.contentList == nil {
				break
			}
			book := m.mainWnd.MainListBox().CurrentItem().(content.Item)
			if err := m.removeBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Removing book: %w", err))
				break
			}
			title := gotext.Get("Warning")
			msg := gotext.Get("Selected book has been removed from the bookshelf")
			gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconInformation)
			// If a bookshelf is open, it must be updated to reflect the changes made
			if m.contentList.ID == dodp.Issued {
				m.setContentList(dodp.Issued)
			}

		case msg.DOWNLOAD_BOOK:
			if m.contentList == nil {
				break
			}
			book := m.mainWnd.MainListBox().CurrentItem().(content.Item)
			if err := m.downloadBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Book downloading: %w", err))
			}

		case msg.BOOK_DESCRIPTION:
			if m.contentList == nil {
				break
			}
			book := m.mainWnd.MainListBox().CurrentItem().(content.Item)
			text, err := m.bookDescription(book)
			if err != nil {
				m.messageBoxError(err)
				break
			}
			gui.BookInfoDialog(m.mainWnd, gotext.Get("Book information"), text)

		case msg.PLAYER_PLAY_PAUSE:
			if m.book != nil {
				m.book.PlayPause()
			}

		case msg.PLAYER_STOP:
			if m.book != nil {
				m.book.Stop()
			}

		case msg.PLAYER_OFFSET_FRAGMENT:
			offset, ok := message.Data.(int)
			if !ok {
				m.logger.Error("Invalid fragment offset")
				break
			}
			if m.book != nil {
				fragment := m.book.Fragment()
				m.book.SetFragment(fragment + offset)
			}

		case msg.PLAYER_SPEED_RESET:
			if m.book != nil {
				m.book.SetSpeed(player.DEFAULT_SPEED)
			}

		case msg.PLAYER_SPEED_UP:
			if m.book != nil {
				value := m.book.Speed()
				m.book.SetSpeed(value + player.STEP_SPEED)
			}

		case msg.PLAYER_SPEED_DOWN:
			if m.book != nil {
				value := m.book.Speed()
				m.book.SetSpeed(value - player.STEP_SPEED)
			}

		case msg.PLAYER_VOLUME_RESET:
			if m.book != nil {
				m.book.SetVolume(player.DEFAULT_VOLUME)
			}

		case msg.PLAYER_VOLUME_UP:
			if m.book != nil {
				volume := m.book.Volume()
				m.book.SetVolume(volume + player.STEP_VOLUME)
			}

		case msg.PLAYER_VOLUME_DOWN:
			if m.book != nil {
				volume := m.book.Volume()
				m.book.SetVolume(volume - player.STEP_VOLUME)
			}

		case msg.PLAYER_OFFSET_POSITION:
			offset, ok := message.Data.(time.Duration)
			if !ok {
				m.logger.Error("Invalid position offset")
				break
			}
			if m.book != nil {
				pos := m.book.Position()
				m.book.SetPosition(pos + offset)
			}

		case msg.PLAYER_GOTO_FRAGMENT:
			if m.book == nil {
				break
			}
			fragment, ok := message.Data.(int)
			if !ok {
				var text string
				var err error
				fragment = m.book.Fragment()
				fragment++ // User needs a fragment number instead of an index
				if gui.TextEntryDialog(m.mainWnd, gotext.Get("Go to fragment"), gotext.Get("Enter fragment number:"), strconv.Itoa(fragment), &text) != gui.DlgCmdOK {
					break
				}
				fragment, err = strconv.Atoi(text)
				if err != nil {
					m.logger.Error("Goto fragment: %v", err)
					break
				}
				fragment-- // Player needs the fragment index instead of its number
			}
			m.book.SetFragment(fragment)

		case msg.PLAYER_GOTO_POSITION:
			if m.book == nil {
				break
			}
			pos, ok := message.Data.(time.Duration)
			if !ok {
				var text string
				var err error
				pos = m.book.Position()
				if gui.TextEntryDialog(m.mainWnd, gotext.Get("Go to position"), gotext.Get("Enter fragment position:"), util.FmtDuration(pos), &text) != gui.DlgCmdOK {
					break
				}
				pos, err = util.ParseDuration(text)
				if err != nil {
					m.logger.Error("Goto position: %v", err)
					break
				}
			}
			m.book.SetPosition(pos)

		case msg.PLAYER_OUTPUT_DEVICE:
			device, ok := message.Data.(string)
			if !ok {
				m.logger.Error("Invalid output device")
				break
			}
			conf.General.OutputDevice = device
			if m.book != nil {
				m.book.SetOutputDevice(device)
			}

		case msg.PLAYER_SET_TIMER:
			var text string
			var d int
			if m.book != nil {
				d = int(m.book.TimerDuration().Minutes())
			}

			if gui.TextEntryDialog(m.mainWnd, gotext.Get("Setting the pause timer"), gotext.Get("Enter the timer value in minutes:"), strconv.Itoa(d), &text) != gui.DlgCmdOK {
				break
			}

			n, err := strconv.Atoi(text)
			if err != nil {
				break
			}

			conf.General.PauseTimer = time.Minute * time.Duration(n)
			m.mainWnd.MenuBar().SetPauseTimerLabel(n)
			if m.book != nil {
				m.book.SetTimerDuration(conf.General.PauseTimer)
			}

		case msg.BOOKMARK_SET:
			if m.book == nil {
				// To set a bookmark, need a book
				break
			}
			if bookmarkID, ok := message.Data.(string); ok {
				m.book.SetBookmarkWithID(bookmarkID)
				break
			}
			var bookmarkName string
			if gui.TextEntryDialog(m.mainWnd, gotext.Get("Adding a new bookmark"), gotext.Get("Bookmark name:"), "", &bookmarkName) != gui.DlgCmdOK {
				break
			}
			if err := m.book.SetBookmarkWithName(bookmarkName); err != nil {
				m.logger.Warning("Set bookmark with name: %v", err)
			}
			m.mainWnd.MenuBar().SetBookmarksMenu(m.book.Bookmarks())

		case msg.BOOKMARK_FETCH:
			if m.book == nil {
				break
			}
			if bookmarkID, ok := message.Data.(string); ok {
				if err := m.book.ToBookmark(bookmarkID); err != nil {
					m.logger.Warning("Bookmark fetching: %v", err)
				}
			}

		case msg.BOOKMARK_REMOVE:
			if m.book == nil {
				break
			}
			if bookmarkID, ok := message.Data.(string); ok {
				bookmark, err := m.book.Bookmark(bookmarkID)
				if err != nil {
					m.logger.Warning("Bookmark removing: %v", err)
					break
				}
				msg := gotext.Get("Are you sure you want to delete the bookmark \"%v\"?", bookmark.Name)
				if gui.MessageBox(m.mainWnd, gotext.Get("Deleting a bookmark"), msg, gui.MsgBoxYesNo|gui.MsgBoxIconQuestion) != gui.DlgCmdYes {
					break
				}
				m.book.RemoveBookmark(bookmarkID)
				m.mainWnd.MenuBar().SetBookmarksMenu(m.book.Bookmarks())
			}

		case msg.LOG_SET_LEVEL:
			level, ok := message.Data.(log.Level)
			if !ok {
				m.logger.Error("Invalid log level")
				break
			}
			conf.General.LogLevel = level.String()
			m.logger.SetLevel(level)
			m.logger.Info("Set log level to %v", level)

		case msg.LIBRARY_INFO:
			if m.provider == nil {
				break
			}
			lib, ok := m.provider.(*library.Library)
			if !ok {
				break
			}
			var lines []string
			attrs := lib.ServiceAttributes()
			lines = append(lines, gotext.Get("Name: %v (%v)", attrs.Service.Label.Text, attrs.Service.ID))
			lines = append(lines, gotext.Get("Back command support: %v", attrs.SupportsServerSideBack))
			lines = append(lines, gotext.Get("Search command support: %v", attrs.SupportsSearch))
			lines = append(lines, gotext.Get("Audio labels support: %v", attrs.SupportsAudioLabels))
			lines = append(lines, gotext.Get("Supported Optional Operations: %v", attrs.SupportedOptionalOperations.Operation))
			title := gotext.Get("Library information")
			msg := strings.Join(lines, CRLF)
			gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconInformation)

		case msg.SET_LANGUAGE:
			lang, ok := message.Data.(string)
			if !ok {
				break
			}
			conf.General.Language = lang
			title := gotext.Get("Warning")
			msg := gotext.Get("Changes will be reflected upon restarting the program")
			gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)

		default:
			m.logger.Warning("Unknown message: %v", message.Code)

		}
	}
}

func (m *Manager) cleaning(conf *config.Config) {
	m.setBook(conf, nil)
	m.mainWnd.MainListBox().Clear()
	m.mainWnd.MenuBar().SetBookMenuEnabled(false)
	m.contentList = nil
	m.questions = nil
	m.userResponses = nil

	if m.provider != nil {
		if term, ok := m.provider.(providers.Terminator); ok {
			m.logger.Warning("Terminating current provider")
			if err := term.Terminate(); err != nil {
				m.logger.Warning("Provider terminating: %v", err)
			}
		}
		m.provider = nil
	}
}

func (m *Manager) setProvider(provider providers.Provider, conf *config.Config, id string) {
	m.logger.Info("Set provider: %v", id)
	m.cleaning(conf)
	m.provider = provider
	conf.General.Provider = id
	if id, err := m.provider.LastContentListID(); err == nil {
		m.setContentList(id)
	} else if _, ok := m.provider.(providers.Questioner); ok {
		m.setQuestions(dodp.UserResponse{QuestionID: dodp.Default})
	}
	if id, err := m.provider.LastContentItemID(); err == nil {
		if contentItem, err := m.provider.ContentItem(id); err == nil {
			if err := m.setBook(conf, contentItem); err != nil {
				m.logger.Error("Set book: %v", err)
			}
		}
	}
	m.mainWnd.MenuBar().SetProvidersMenu(conf.Services, id)
}

func (m *Manager) setQuestions(response ...dodp.UserResponse) {
	qst, ok := m.provider.(providers.Questioner)
	if !ok {
		m.messageBoxError(OperationNotSupported)
		return
	}

	if len(response) == 0 {
		m.logger.Error("len(response) == 0")
		return
	}

	m.questions = nil
	m.userResponses = nil
	m.mainWnd.MainListBox().Clear()
	m.mainWnd.MenuBar().SetBookMenuEnabled(false)

	ur := dodp.UserResponses{UserResponse: response}
	questions, err := qst.GetQuestions(&ur)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetQuestions: %w", err))
		return
	}

	if questions.Label.Text != "" {
		// We have received a notification from the library. Show it to the user
		gui.MessageBox(m.mainWnd, gotext.Get("Warning"), questions.Label.Text, gui.MsgBoxOK|gui.MsgBoxIconInformation)
		// Return to the main menu of the library
		m.setQuestions(dodp.UserResponse{QuestionID: dodp.Default})
		return
	}

	if questions.ContentListRef != "" {
		// We got a list of content. Show it to the user
		m.setContentList(questions.ContentListRef)
		return
	}

	m.questions = questions
	m.userResponses = make([]dodp.UserResponse, 0)

	if len(m.questions.MultipleChoiceQuestion) > 0 {
		m.setMultipleChoiceQuestion(0)
		return
	}
	m.setInputQuestion()
}

func (m *Manager) setMultipleChoiceQuestion(index int) {
	choiceQuestion := m.questions.MultipleChoiceQuestion[index]
	items := make([]gui.ListItem, len(choiceQuestion.Choices.Choice))
	for i, c := range choiceQuestion.Choices.Choice {
		items[i] = ChoiceItem{Choice: c}
	}
	m.contentList = nil
	m.mainWnd.MainListBox().SetItems(items, choiceQuestion.Label.Text, nil)
	m.mainWnd.MenuBar().SetBookMenuEnabled(false)
}

func (m *Manager) setInputQuestion() {
	for _, inputQuestion := range m.questions.InputQuestion {
		var text string
		if gui.TextEntryDialog(m.mainWnd, gotext.Get("Entering text"), inputQuestion.Label.Text, m.lastInputText, &text) != gui.DlgCmdOK {
			// Return to the main menu of the library
			m.setQuestions(dodp.UserResponse{QuestionID: dodp.Default})
			return
		}
		m.lastInputText = text
		m.userResponses = append(m.userResponses, dodp.UserResponse{QuestionID: inputQuestion.ID, Value: text})
	}
	m.setQuestions(m.userResponses...)
}

func (m *Manager) setContentList(contentID string) {
	m.questions = nil
	m.mainWnd.MainListBox().Clear()
	m.mainWnd.MenuBar().SetBookMenuEnabled(false)
	m.logger.Debug("Set content list: %v", contentID)

	contentList, err := m.provider.ContentList(contentID)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetContentList: %w", err))
		return
	}

	if len(contentList.Items) == 0 {
		title := gotext.Get("Warning")
		msg := gotext.Get("List of books is empty")
		gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	if contentList.ID == dodp.Issued {
		ids := make([]string, len(contentList.Items))
		for i := range ids {
			book := contentList.Items[i]
			ids[i] = book.ID()
		}
		if m.book != nil && !util.StringInSlice(m.book.ID(), ids) {
			ids = append(ids, m.book.ID())
		}
		m.provider.Tidy(ids)
	}

	m.updateContentList(contentList)
}

func (m *Manager) updateContentList(contentList *content.List) {
	m.contentList = contentList
	items := make([]gui.ListItem, len(m.contentList.Items))
	for i, v := range m.contentList.Items {
		items[i] = v
	}
	m.mainWnd.MainListBox().SetItems(items, contentList.Name, m.mainWnd.MenuBar().BookMenu())
	m.mainWnd.MenuBar().SetBookMenuEnabled(true)
}

func (m *Manager) setBook(conf *config.Config, contentItem content.Item) error {
	if m.book != nil {
		m.book.Pause(true)
	}

	if contentItem != nil {
		book, err := books.NewBook(conf.General.OutputDevice, contentItem, m.logger, m.mainWnd.StatusBar())
		if err != nil {
			return err
		}
		defer func() {
			book.SetTimerDuration(conf.General.PauseTimer)
			book.SetVolume(conf.General.Volume)
			m.mainWnd.SetTitle(book.Title)
			m.mainWnd.MenuBar().SetBookmarksMenu(book.Bookmarks())
			m.book = book
			m.logger.Debug("Set book: %v", book.ID())
		}()
	}

	if m.book != nil {
		conf.General.Volume = m.book.Volume()
		m.book.Save()
		m.book.Stop()
		m.mainWnd.SetTitle("")
		m.mainWnd.MenuBar().SetBookmarksMenu(nil)
		m.book = nil
	}
	return nil
}

func (m *Manager) downloadBook(book content.Item) error {
	if _, ok := m.provider.(*library.Library); !ok {
		return OperationNotSupported
	}

	name, err := book.Name()
	if err != nil {
		return err
	}

	dir, err := config.BookDir(name)
	if err != nil {
		return err
	}

	rsrc, err := book.Resources()
	if err != nil {
		return fmt.Errorf("getContentResources: %w", err)
	}

	if md, err := book.ContentMetadata(); err == nil {
		path := filepath.Join(dir, config.MetadataFileName)
		f, err := util.CreateSecureFile(path)
		if err != nil {
			m.logger.Warning("Creating %v: %v", config.MetadataFileName, err)
		} else {
			defer f.Close()
			e := xml.NewEncoder(f)
			e.Indent("", "\t") // for readability
			if err := e.Encode(md); err != nil {
				f.Corrupted()
				m.logger.Error("Writing to %v: %v", config.MetadataFileName, err)
			}
		}
	}

	dlFunc := func(rsrc []dodp.Resource, dir string, id string) {
		var err error
		var totalSize, downloadedSize int64
		ctx, cancelFunc := context.WithCancel(context.TODO())
		dlg := gui.NewProgressDialog(m.mainWnd, gotext.Get("Book downloading"), gotext.Get("Downloading \"%v\"\nSpeed: %d KB/s", name, 0), 100, cancelFunc)
		dlg.Run()

		for _, r := range rsrc {
			totalSize += r.Size
		}

		m.logger.Debug("Downloading book %v started", id)
		for _, r := range rsrc {
			err = func() error {
				path := filepath.Join(dir, r.LocalURI)
				if util.FileIsExist(path, r.Size) {
					// This fragment already exists on disk
					downloadedSize += r.Size
					dlg.SetValue(int(downloadedSize / (totalSize / 100)))
					return nil
				}

				conn, err := connection.NewConnectionWithContext(ctx, r.URI, m.logger)
				if err != nil {
					return err
				}
				defer conn.Close()

				dst, err := util.CreateSecureFile(path)
				if err != nil {
					return err
				}
				defer dst.Close()

				var fragmentSize int64
				var src io.Reader = buffer.NewReader(conn)
				for err == nil {
					var n int64
					t := time.Now()
					n, err = io.CopyN(dst, src, 512*1024)
					sec := time.Since(t).Seconds()
					speed := int(float64(n) / 1024 / sec)
					dlg.SetLabel(gotext.Get("Downloading \"%v\"\nSpeed: %d KB/s", name, speed))
					downloadedSize += n
					fragmentSize += n
					dlg.SetValue(int(downloadedSize / (totalSize / 100)))
				}

				if err == io.EOF {
					err = nil
				}

				if err == nil && r.Size != fragmentSize {
					err = errors.New("resource size mismatch")
				}

				if err != nil {
					dst.Corrupted()
				}
				return err
			}()

			if err != nil {
				break
			}
		}

		dlg.Cancel()

		switch {
		case errors.Is(err, context.Canceled):
			title := gotext.Get("Warning")
			msg := gotext.Get("Download canceled by user")
			gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		case err != nil:
			gui.MessageBox(m.mainWnd, gotext.Get("Error"), err.Error(), gui.MsgBoxOK|gui.MsgBoxIconError)
		default:
			title := gotext.Get("Warning")
			msg := gotext.Get("Book successfully downloaded")
			gui.MessageBox(m.mainWnd, title, msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
			m.logger.Debug("Book %v has been successfully downloaded. Total size: %v", id, totalSize)
		}
	}

	// Book downloading should not block handling of other messages
	go dlFunc(rsrc, dir, book.ID())
	return nil
}

func (m *Manager) removeBook(book content.Item) error {
	returner, ok := book.(content.Returner)
	if !ok {
		return OperationNotSupported
	}
	return returner.Return()
}

func (m *Manager) issueBook(book content.Item) error {
	issuer, ok := book.(content.Issuer)
	if !ok {
		return OperationNotSupported
	}
	return issuer.Issue()
}

func (m *Manager) bookDescription(book content.Item) (string, error) {
	md, err := book.ContentMetadata()
	if err != nil {
		return "", BookDescriptionNotAvailable
	}

	text := strings.Join(md.Metadata.Description, CRLF)
	if text == "" {
		return "", BookDescriptionNotAvailable
	}
	return text, nil
}

func (m *Manager) messageBoxError(err error) {
	msg := err.Error()
	m.logger.Error(msg)
	switch {
	case errors.As(err, new(net.Error)):
		msg = gotext.Get("Network error. Please check your internet connection")
	case errors.Is(err, OperationNotSupported):
		msg = gotext.Get("Operation not supported")
	case errors.Is(err, BookDescriptionNotAvailable):
		msg = gotext.Get("Book description not available")
	}
	gui.MessageBox(m.mainWnd, gotext.Get("Error"), msg, gui.MsgBoxOK|gui.MsgBoxIconError)
}
