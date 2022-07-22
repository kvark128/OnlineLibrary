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

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/connection"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/util/buffer"
	daisy "github.com/kvark128/daisyonline"
)

const (
	MetadataFileName = "metadata.xml"
	CRLF             = "\r\n"
)

var (
	BookDescriptionNotAvailable = errors.New("book description not available")
	OperationNotSupported       = errors.New("operation not supported")
)

type Manager struct {
	provider      Provider
	mainWnd       *gui.MainWnd
	logger        *log.Logger
	book          *Book
	contentList   *ContentList
	questions     *daisy.Questions
	userResponses []daisy.UserResponse
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
		m.cleaning()
		m.logger.Debug("Exiting from Manager Loop")
		done <- true
	}()

	msgCH := m.mainWnd.MsgChan()
	if conf.General.Provider != "" {
		msgCH <- msg.Message{Code: msg.SET_PROVIDER, Data: conf.General.Provider}
	}

	for message := range msgCH {
		switch message.Code {
		case msg.ACTIVATE_MENU:
			index := message.Data.(int)
			if m.contentList != nil {
				book := m.contentList.Items[index]
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
				value := m.questions.MultipleChoiceQuestion[questionIndex].Choices.Choice[index].ID
				m.userResponses = append(m.userResponses, daisy.UserResponse{QuestionID: questionID, Value: value})
				questionIndex++
				if questionIndex < len(m.questions.MultipleChoiceQuestion) {
					m.setMultipleChoiceQuestion(questionIndex)
					break
				}
				m.setInputQuestion()
			}

		case msg.OPEN_BOOKSHELF:
			m.setContentList(daisy.Issued)

		case msg.OPEN_NEWBOOKS:
			m.setContentList(daisy.New)

		case msg.MAIN_MENU:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})

		case msg.SEARCH_BOOK:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Search})

		case msg.MENU_BACK:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Back})

		case msg.SET_PROVIDER:
			id, ok := message.Data.(string)
			if !ok {
				break
			}

			var provider Provider
			var err error
			if id == config.LocalStorageID {
				provider = NewLocalStorage(conf)
			} else {
				var service *config.Service
				service, err = conf.ServiceByID(id)
				if err != nil {
					m.logger.Debug("Get service %v: %v", id, err)
					break
				}
				provider, err = NewLibrary(conf, service)
			}

			if err != nil {
				m.messageBoxError(fmt.Errorf("Provider creating %v: %w", id, err))
				break
			}
			m.setProvider(provider, conf, id)

		case msg.LIBRARY_ADD:
			service := new(config.Service)
			if !m.mainWnd.CredentialsEntryDialog(service) || service.Name == "" {
				m.logger.Warning("Library adding: pressed Cancel button or len(service.Name) == 0")
				break
			}

			if _, err := conf.ServiceByName(service.Name); err == nil {
				m.mainWnd.MessageBoxError("Ошибка", fmt.Sprintf("Учётная запись «%v» уже существует", service.Name))
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

			provider, err := NewLibrary(conf, service)
			if err != nil {
				m.messageBoxError(fmt.Errorf("library creating: %w", err))
				break
			}
			conf.SetService(service)
			m.setProvider(provider, conf, service.ID)

		case msg.LIBRARY_REMOVE:
			lib, ok := m.provider.(*Library)
			if !ok {
				break
			}
			msg := fmt.Sprintf("Вы действительно хотите удалить учётную запись %v?%sТакже будут удалены сохранённые позиции всех книг этой библиотеки.%sЭто действие не может быть отменено.", lib.service.Name, CRLF, CRLF)
			if !m.mainWnd.MessageBoxQuestion("Удаление учётной записи", msg) {
				break
			}
			conf.RemoveService(lib.service)
			m.cleaning()
			m.mainWnd.MenuBar().SetProvidersMenu(conf.Services, "")

		case msg.ISSUE_BOOK:
			if m.contentList == nil {
				m.logger.Warning("Attempt to add book to bookshelf when there is no content list")
				break
			}
			book := m.contentList.Items[m.mainWnd.MainListBox().CurrentIndex()]
			if err := m.issueBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Issue book: %w", err))
				break
			}
			m.mainWnd.MessageBoxWarning("Уведомление", fmt.Sprintf("«%s» добавлена на книжную полку", book.Name()))

		case msg.REMOVE_BOOK:
			if m.contentList == nil {
				m.logger.Warning("Attempt to remove book from bookshelf when there is no content list")
				break
			}
			book := m.contentList.Items[m.mainWnd.MainListBox().CurrentIndex()]
			if err := m.removeBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Removing book: %w", err))
				break
			}
			m.mainWnd.MessageBoxWarning("Уведомление", fmt.Sprintf("«%s» удалена с книжной полки", book.Name()))
			// If a bookshelf is open, it must be updated to reflect the changes made
			if m.contentList.ID == daisy.Issued {
				m.setContentList(daisy.Issued)
			}

		case msg.DOWNLOAD_BOOK:
			if m.contentList == nil {
				break
			}
			index := m.mainWnd.MainListBox().CurrentIndex()
			book := m.contentList.Items[index]
			if err := m.downloadBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Book downloading: %w", err))
			}

		case msg.BOOK_DESCRIPTION:
			if m.contentList == nil {
				break
			}
			index := m.mainWnd.MainListBox().CurrentIndex()
			book := m.contentList.Items[index]
			text, err := m.bookDescription(book)
			if err != nil {
				m.messageBoxError(err)
				break
			}
			m.mainWnd.MessageBoxWarning("Информация о книге", text)

		case msg.PLAYER_PLAY_PAUSE:
			if m.book != nil {
				m.book.PlayPause()
			}

		case msg.PLAYER_STOP:
			if m.book != nil {
				m.book.SetBookmarkWithID(config.ListeningPosition)
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
				if !m.mainWnd.TextEntryDialog("Переход к фрагменту", "Введите номер фрагмента:", strconv.Itoa(fragment), &text) {
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
				if !m.mainWnd.TextEntryDialog("Переход к позиции", "Введите позицию фрагмента:", util.FmtDuration(pos), &text) {
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

			if !m.mainWnd.TextEntryDialog("Установка таймера паузы", "Введите время таймера в минутах:", strconv.Itoa(d), &text) {
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
			if !m.mainWnd.TextEntryDialog("Добавление новой закладки", "Имя закладки:", "", &bookmarkName) {
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
				msg := fmt.Sprintf("Вы действительно хотите удалить закладку «%v»?", bookmark.Name)
				if !m.mainWnd.MessageBoxQuestion("Удаление закладки", msg) {
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
			m.logger.Info("Set log level to %s", level)

		case msg.LIBRARY_INFO:
			if m.provider == nil {
				break
			}
			lib, ok := m.provider.(*Library)
			if !ok {
				break
			}
			var lines []string
			attrs := lib.serviceAttributes
			lines = append(lines, fmt.Sprintf("Имя: «%v» (%v)", attrs.Service.Label.Text, attrs.Service.ID))
			lines = append(lines, fmt.Sprintf("Поддержка команды back: %v", attrs.SupportsServerSideBack))
			lines = append(lines, fmt.Sprintf("Поддержка команды search: %v", attrs.SupportsSearch))
			lines = append(lines, fmt.Sprintf("Поддержка аудиометок: %v", attrs.SupportsAudioLabels))
			lines = append(lines, fmt.Sprintf("Поддерживаемые опциональные операции: %v", attrs.SupportedOptionalOperations.Operation))
			m.mainWnd.MessageBoxWarning("Информация о библиотеке", strings.Join(lines, CRLF))

		default:
			m.logger.Warning("Unknown message: %v", message.Code)

		}
	}
}

func (m *Manager) cleaning() {
	if m.book != nil {
		m.book.Close()
		m.book = nil
	}
	m.mainWnd.MainListBox().Clear()
	m.mainWnd.SetTitle("")
	m.contentList = nil
	m.questions = nil
	m.userResponses = nil

	if m.provider != nil {
		if lib, ok := m.provider.(*Library); ok {
			_, err := lib.LogOff()
			if err != nil {
				m.logger.Warning("Library logoff: %v", err)
			}
		}
		m.provider = nil
	}
}

func (m *Manager) setProvider(provider Provider, conf *config.Config, id string) {
	m.logger.Info("Set provider: %v", id)
	m.cleaning()
	m.provider = provider
	conf.General.Provider = id
	if id, err := m.provider.LastContentListID(); err == nil {
		m.setContentList(id)
	} else if _, ok := m.provider.(Questioner); ok {
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
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

func (m *Manager) setQuestions(response ...daisy.UserResponse) {
	qst, ok := m.provider.(Questioner)
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

	ur := daisy.UserResponses{UserResponse: response}
	questions, err := qst.GetQuestions(&ur)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetQuestions: %w", err))
		return
	}

	if questions.Label.Text != "" {
		// We have received a notification from the library. Show it to the user
		m.mainWnd.MessageBoxWarning("Предупреждение", questions.Label.Text)
		// Return to the main menu of the library
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
		return
	}

	if questions.ContentListRef != "" {
		// We got a list of content. Show it to the user
		m.setContentList(questions.ContentListRef)
		return
	}

	m.questions = questions
	m.userResponses = make([]daisy.UserResponse, 0)

	if len(m.questions.MultipleChoiceQuestion) > 0 {
		m.setMultipleChoiceQuestion(0)
		return
	}
	m.setInputQuestion()
}

func (m *Manager) setMultipleChoiceQuestion(index int) {
	choiceQuestion := m.questions.MultipleChoiceQuestion[index]
	labels := make([]string, len(choiceQuestion.Choices.Choice))
	for i, c := range choiceQuestion.Choices.Choice {
		labels[i] = c.Label.Text
	}
	m.contentList = nil
	m.mainWnd.MainListBox().SetItems(labels, choiceQuestion.Label.Text, nil)
}

func (m *Manager) setInputQuestion() {
	for _, inputQuestion := range m.questions.InputQuestion {
		var text string
		if !m.mainWnd.TextEntryDialog("Ввод текста", inputQuestion.Label.Text, m.lastInputText, &text) {
			// Return to the main menu of the library
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
			return
		}
		m.lastInputText = text
		m.userResponses = append(m.userResponses, daisy.UserResponse{QuestionID: inputQuestion.ID, Value: text})
	}
	m.setQuestions(m.userResponses...)
}

func (m *Manager) setContentList(contentID string) {
	m.questions = nil
	m.mainWnd.MainListBox().Clear()
	m.logger.Debug("Set content list: %v", contentID)

	contentList, err := m.provider.ContentList(contentID)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetContentList: %w", err))
		return
	}

	if len(contentList.Items) == 0 {
		m.mainWnd.MessageBoxWarning("Предупреждение", "Список книг пуст")
		return
	}

	if contentID == daisy.Issued {
		if lib, ok := m.provider.(*Library); ok {
			ids := make([]string, len(contentList.Items))
			for i := range ids {
				book := contentList.Items[i]
				ids[i] = book.ID()
			}
			if m.book != nil && !util.StringInSlice(m.book.ID(), ids) {
				ids = append(ids, m.book.ID())
			}
			lib.service.RecentBooks.Tidy(ids)
		}
	}

	m.updateContentList(contentList)
}

func (m *Manager) updateContentList(contentList *ContentList) {
	labels := make([]string, len(contentList.Items))
	for i := range labels {
		book := contentList.Items[i]
		labels[i] = book.Name()
	}
	m.contentList = contentList
	m.mainWnd.MainListBox().SetItems(labels, contentList.Name, m.mainWnd.MenuBar().BookMenu())
}

func (m *Manager) setBook(conf *config.Config, contentItem ContentItem) error {
	book, err := NewBook(conf, contentItem, m.logger, m.mainWnd.StatusBar())
	if err != nil {
		return err
	}

	if m.book != nil {
		m.book.Close()
	}

	m.mainWnd.SetTitle(book.Name())
	m.mainWnd.MenuBar().SetBookmarksMenu(book.Bookmarks())
	m.book = book
	m.logger.Debug("Set book: %v", book.ID())
	return nil
}

func (m *Manager) downloadBook(book ContentItem) error {
	if _, ok := m.provider.(*Library); !ok {
		return OperationNotSupported
	}

	rsrc, err := book.Resources()
	if err != nil {
		return fmt.Errorf("getContentResources: %w", err)
	}

	// Path to the directory where we will download a resources. Make sure that it does not contain prohibited characters
	bookDir := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Name()))

	if md, err := book.ContentMetadata(); err == nil {
		path := filepath.Join(bookDir, MetadataFileName)
		f, err := util.CreateSecureFile(path)
		if err != nil {
			m.logger.Warning("Creating %v: %v", MetadataFileName, err)
		} else {
			defer f.Close()
			e := xml.NewEncoder(f)
			e.Indent("", "\t") // for readability
			if err := e.Encode(md); err != nil {
				f.Corrupted()
				m.logger.Error("Writing to %v: %v", MetadataFileName, err)
			}
		}
	}

	dlFunc := func(rsrc []daisy.Resource, bookDir, bookID string) {
		var err error
		var totalSize, downloadedSize int64
		ctx, cancelFunc := context.WithCancel(context.TODO())
		label := "Загрузка «%s»\nСкорость: %d Кб/с"
		dlg := gui.NewProgressDialog(m.mainWnd, "Загрузка книги", fmt.Sprintf(label, book.Name(), 0), 100, cancelFunc)
		dlg.Run()

		for _, r := range rsrc {
			totalSize += r.Size
		}

		m.logger.Debug("Downloading book %v started", bookID)
		for _, r := range rsrc {
			err = func() error {
				path := filepath.Join(bookDir, r.LocalURI)
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
					dlg.SetLabel(fmt.Sprintf(label, book.Name(), speed))
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
			m.mainWnd.MessageBoxWarning("Предупреждение", "Загрузка отменена пользователем")
		case err != nil:
			m.mainWnd.MessageBoxError("Ошибка", err.Error())
		default:
			m.mainWnd.MessageBoxWarning("Уведомление", "Книга успешно загружена")
			m.logger.Debug("Book %v has been successfully downloaded. Total size: %v", bookID, totalSize)
		}
	}

	// Book downloading should not block handling of other messages
	go dlFunc(rsrc, bookDir, book.ID())
	return nil
}

func (m *Manager) removeBook(book ContentItem) error {
	returner, ok := book.(Returner)
	if !ok {
		return OperationNotSupported
	}
	return returner.Return()
}

func (m *Manager) issueBook(book ContentItem) error {
	issuer, ok := book.(Issuer)
	if !ok {
		return OperationNotSupported
	}
	return issuer.Issue()
}

func (m *Manager) bookDescription(book ContentItem) (string, error) {
	md, err := book.ContentMetadata()
	if err != nil {
		return "", BookDescriptionNotAvailable
	}

	text := fmt.Sprintf("%v", strings.Join(md.Metadata.Description, CRLF))
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
		msg = "Ошибка сети. Пожалуйста, проверьте ваше подключение к интернету."
	case errors.Is(err, OperationNotSupported):
		msg = "Операция не поддерживается"
	case errors.Is(err, BookDescriptionNotAvailable):
		msg = "Описание книги недоступно"
	}
	m.mainWnd.MessageBoxError("Ошибка", msg)
}
