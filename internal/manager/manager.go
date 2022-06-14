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
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/msg"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/util/buffer"
	daisy "github.com/kvark128/daisyonline"
	"github.com/lxn/walk"
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
	book          *Book
	contentList   *ContentList
	questions     *daisy.Questions
	userResponses []daisy.UserResponse
	lastInputText string
}

func (m *Manager) Start(msgCH chan msg.Message, done chan<- bool) {
	log.Debug("Entering to Manager Loop")
	defer func() {
		if p := recover(); p != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			log.Error("Manager panic: %v:\n\n%v", p, string(buf[:n]))
			os.Exit(1)
		}
		m.cleaning()
		log.Debug("Exiting from Manager Loop")
		done <- true
	}()

	if config.Conf.General.Provider != "" {
		msgCH <- msg.Message{Code: msg.SET_PROVIDER, Data: config.Conf.General.Provider}
	}

	for message := range msgCH {
		switch message.Code {
		case msg.ACTIVATE_MENU:
			index := gui.MainList.CurrentIndex()
			if m.contentList != nil {
				book := m.contentList.Items[index]
				if m.book == nil || m.book.ID() != book.ID() {
					if err := m.setBook(msgCH, book); err != nil {
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
				provider = NewLocalStorage()
			} else {
				service, err := config.Conf.ServiceByID(id)
				if err != nil {
					break
				}
				provider, err = NewLibrary(service)
			}

			if err != nil {
				log.Error("Provider creating: %v", err)
				break
			}
			m.setProvider(provider, msgCH, id)

		case msg.LIBRARY_ADD:
			service := new(config.Service)
			if gui.Credentials(service) != walk.DlgCmdOK || service.Name == "" {
				log.Warning("Library adding: pressed Cancel button or len(service.Name) == 0")
				break
			}

			if _, err := config.Conf.ServiceByName(service.Name); err == nil {
				gui.MessageBox("Ошибка", fmt.Sprintf("Учётная запись «%v» уже существует", service.Name), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			// Service id creation. Maximum index value of 256 is intended to protect against an infinite loop
			for i := 0; i < 256; i++ {
				id := fmt.Sprintf("library%d", i)
				if _, err := config.Conf.ServiceByID(id); err != nil {
					service.ID = id
					log.Debug("Created new service id: %v", id)
					break
				}
			}

			provider, err := NewLibrary(service)
			if err != nil {
				m.messageBoxError(fmt.Errorf("library creating: %w", err))
				break
			}
			config.Conf.SetService(service)
			m.setProvider(provider, msgCH, service.ID)

		case msg.LIBRARY_REMOVE:
			lib, ok := m.provider.(*Library)
			if !ok {
				break
			}
			msg := fmt.Sprintf("Вы действительно хотите удалить учётную запись %v?%sТакже будут удалены сохранённые позиции всех книг этой библиотеки.%sЭто действие не может быть отменено.", lib.service.Name, CRLF, CRLF)
			if gui.MessageBox("Удаление учётной записи", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
				break
			}
			config.Conf.RemoveService(lib.service)
			m.cleaning()
			gui.SetProvidersMenu(msgCH, config.Conf.Services, "")

		case msg.ISSUE_BOOK:
			if m.contentList == nil {
				log.Warning("Attempt to add book to bookshelf when there is no content list")
				break
			}
			book := m.contentList.Items[gui.MainList.CurrentIndex()]
			if err := m.issueBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Issue book: %w", err))
				break
			}
			gui.MessageBox("Уведомление", fmt.Sprintf("«%s» добавлена на книжную полку", book.Name()), walk.MsgBoxOK|walk.MsgBoxIconWarning)

		case msg.REMOVE_BOOK:
			if m.contentList == nil {
				log.Warning("Attempt to remove book from bookshelf when there is no content list")
				break
			}
			book := m.contentList.Items[gui.MainList.CurrentIndex()]
			if err := m.removeBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Removing book: %w", err))
				break
			}
			gui.MessageBox("Уведомление", fmt.Sprintf("«%s» удалена с книжной полки", book.Name()), walk.MsgBoxOK|walk.MsgBoxIconWarning)
			// If a bookshelf is open, it must be updated to reflect the changes made
			if m.contentList.ID == daisy.Issued {
				m.setContentList(daisy.Issued)
			}

		case msg.DOWNLOAD_BOOK:
			if m.contentList == nil {
				break
			}
			index := gui.MainList.CurrentIndex()
			book := m.contentList.Items[index]
			if err := m.downloadBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Book downloading: %w", err))
			}

		case msg.BOOK_DESCRIPTION:
			if m.contentList == nil {
				break
			}
			index := gui.MainList.CurrentIndex()
			book := m.contentList.Items[index]
			text, err := m.bookDescription(book)
			if err != nil {
				m.messageBoxError(err)
				break
			}
			gui.MessageBox("Информация о книге", text, walk.MsgBoxOK|walk.MsgBoxIconWarning)

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
				log.Error("Invalid fragment offset")
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
				log.Error("Invalid position offset")
				break
			}
			if m.book != nil {
				pos := m.book.Position()
				m.book.SetPosition(pos + offset)
			}

		case msg.PLAYER_GOTO_FRAGMENT:
			fragment, ok := message.Data.(int)
			if !ok {
				var text string
				var curFragment int
				if m.book != nil {
					curFragment = m.book.Fragment()
				}
				if gui.TextEntryDialog("Переход к фрагменту", "Введите номер фрагмента:", strconv.Itoa(curFragment+1), &text) != walk.DlgCmdOK {
					break
				}
				newFragment, err := strconv.Atoi(text)
				if err != nil {
					break
				}
				fragment = newFragment - 1 // Requires an index of fragment
			}
			if m.book != nil {
				m.book.SetFragment(fragment)
			}

		case msg.PLAYER_GOTO_POSITION:
			var text string
			var pos time.Duration
			if m.book != nil {
				pos = m.book.Position()
			}
			if gui.TextEntryDialog("Переход к позиции", "Введите позицию фрагмента:", util.FmtDuration(pos), &text) != walk.DlgCmdOK {
				break
			}
			position, err := util.ParseDuration(text)
			if err != nil {
				log.Error("Goto position: %v", err)
				break
			}
			if m.book != nil {
				m.book.SetPosition(position)
			}

		case msg.PLAYER_OUTPUT_DEVICE:
			device, ok := message.Data.(string)
			if !ok {
				log.Error("Invalid output device")
				break
			}
			config.Conf.General.OutputDevice = device
			if m.book != nil {
				m.book.SetOutputDevice(device)
			}

		case msg.PLAYER_SET_TIMER:
			var text string
			var d int
			if m.book != nil {
				d = int(m.book.TimerDuration().Minutes())
			}

			if gui.TextEntryDialog("Установка таймера паузы", "Введите время таймера в минутах:", strconv.Itoa(d), &text) != walk.DlgCmdOK {
				break
			}

			n, err := strconv.Atoi(text)
			if err != nil {
				break
			}

			config.Conf.General.PauseTimer = time.Minute * time.Duration(n)
			gui.SetPauseTimerLabel(n)
			if m.book != nil {
				m.book.SetTimerDuration(config.Conf.General.PauseTimer)
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
			if gui.TextEntryDialog("Создание новой закладки", "Имя закладки:", "", &bookmarkName) != walk.DlgCmdOK {
				break
			}
			if err := m.book.SetBookmarkWithName(bookmarkName); err != nil {
				log.Warning("Set bookmark with name: %v", err)
			}
			gui.SetBookmarksMenu(msgCH, m.book.Bookmarks())

		case msg.BOOKMARK_FETCH:
			if m.book == nil {
				break
			}
			if bookmarkID, ok := message.Data.(string); ok {
				if err := m.book.ToBookmark(bookmarkID); err != nil {
					log.Warning("Bookmark fetching: %v", err)
				}
			}

		case msg.BOOKMARK_REMOVE:
			if m.book == nil {
				break
			}
			if bookmarkID, ok := message.Data.(string); ok {
				bookmark, err := m.book.Bookmark(bookmarkID)
				if err != nil {
					log.Warning("Bookmark removing: %v", err)
					break
				}
				msg := fmt.Sprintf("Вы действительно хотите удалить закладку «%v»?", bookmark.Name)
				if gui.MessageBox("Удаление закладки", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
					break
				}
				m.book.RemoveBookmark(bookmarkID)
				gui.SetBookmarksMenu(msgCH, m.book.Bookmarks())
			}

		case msg.LOG_SET_LEVEL:
			level, ok := message.Data.(log.Level)
			if !ok {
				log.Error("Invalid log level")
				break
			}
			config.Conf.General.LogLevel = level.String()
			log.SetLevel(level)
			log.Info("Set log level to %s", level)

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
			gui.MessageBox("Информация о библиотеке", strings.Join(lines, CRLF), walk.MsgBoxOK|walk.MsgBoxIconWarning)

		default:
			log.Warning("Unknown message: %v", message.Code)

		}
	}
}

func (m *Manager) cleaning() {
	if m.book != nil {
		m.book.Close()
		m.book = nil
	}
	gui.MainList.Clear()
	gui.SetMainWindowTitle("")
	m.contentList = nil
	m.questions = nil
	m.userResponses = nil

	if m.provider != nil {
		if lib, ok := m.provider.(*Library); ok {
			_, err := lib.LogOff()
			if err != nil {
				log.Warning("Library logoff: %v", err)
			}
		}
		m.provider = nil
	}
}

func (m *Manager) setProvider(provider Provider, msgCH chan msg.Message, id string) {
	log.Info("Set provider: %v", id)
	m.cleaning()
	m.provider = provider
	config.Conf.General.Provider = id
	if id, err := m.provider.LastContentListID(); err == nil {
		m.setContentList(id)
	} else if _, ok := m.provider.(Questioner); ok {
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
	}
	if id, err := m.provider.LastContentItemID(); err == nil {
		if contentItem, err := m.provider.ContentItem(id); err == nil {
			if err := m.setBook(msgCH, contentItem); err != nil {
				log.Error("Set book: %v", err)
			}
		}
	}
	gui.SetProvidersMenu(msgCH, config.Conf.Services, id)
}

func (m *Manager) setQuestions(response ...daisy.UserResponse) {
	lib, ok := m.provider.(*Library)
	if !ok {
		m.messageBoxError(OperationNotSupported)
		return
	}

	if len(response) == 0 {
		log.Error("len(response) == 0")
		m.questions = nil
		gui.MainList.Clear()
		return
	}

	ur := daisy.UserResponses{UserResponse: response}
	questions, err := lib.GetQuestions(&ur)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetQuestions: %w", err))
		m.questions = nil
		gui.MainList.Clear()
		return
	}

	if questions.Label.Text != "" {
		// We have received a notification from the library. Show it to the user
		gui.MessageBox("Предупреждение", questions.Label.Text, walk.MsgBoxOK|walk.MsgBoxIconWarning)
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
	var labels []string
	for _, c := range choiceQuestion.Choices.Choice {
		labels = append(labels, c.Label.Text)
	}

	m.contentList = nil
	gui.MainList.SetItems(labels, choiceQuestion.Label.Text, false)
}

func (m *Manager) setInputQuestion() {
	for _, inputQuestion := range m.questions.InputQuestion {
		var text string
		if gui.TextEntryDialog("Ввод текста", inputQuestion.Label.Text, m.lastInputText, &text) != walk.DlgCmdOK {
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
	log.Debug("Set content list: %v", contentID)
	contentList, err := m.provider.ContentList(contentID)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetContentList: %w", err))
		m.questions = nil
		gui.MainList.Clear()
		return
	}

	if len(contentList.Items) == 0 {
		gui.MessageBox("Предупреждение", "Список книг пуст", walk.MsgBoxOK|walk.MsgBoxIconWarning)
		// Return to the main menu of the library
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
		return
	}

	if contentID == daisy.Issued {
		ids := make([]string, len(contentList.Items))
		for i := range ids {
			book := contentList.Items[i]
			ids[i] = book.ID()
		}
		if m.book != nil && !util.StringInSlice(m.book.ID(), ids) {
			ids = append(ids, m.book.ID())
		}
		if lib, ok := m.provider.(*Library); ok {
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
	m.questions = nil
	gui.MainList.SetItems(labels, contentList.Name, true)
}

func (m *Manager) setBook(msgCH chan msg.Message, contentItem ContentItem) error {
	book, err := NewBook(contentItem)
	if err != nil {
		return err
	}

	if m.book != nil {
		m.book.Close()
	}

	gui.SetMainWindowTitle(book.Name())
	gui.SetBookmarksMenu(msgCH, book.Bookmarks())
	m.book = book
	log.Debug("Set book: %v", book.ID())
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
			log.Warning("Creating %v: %v", MetadataFileName, err)
		} else {
			defer f.Close()
			e := xml.NewEncoder(f)
			e.Indent("", "\t") // for readability
			if err := e.Encode(md); err != nil {
				f.Corrupted()
				log.Error("Writing to %v: %v", MetadataFileName, err)
			}
		}
	}

	dlFunc := func(rsrc []daisy.Resource, bookDir, bookID string) {
		var err error
		var totalSize, downloadedSize int64
		ctx, cancelFunc := context.WithCancel(context.TODO())
		label := "Загрузка «%s»\nСкорость: %d Кб/с"
		dlg := gui.NewProgressDialog("Загрузка книги", fmt.Sprintf(label, book.Name(), 0), 100, cancelFunc)
		dlg.Show()

		for _, r := range rsrc {
			totalSize += r.Size
		}

		log.Debug("Downloading book %v started", bookID)
		for _, r := range rsrc {
			err = func() error {
				path := filepath.Join(bookDir, r.LocalURI)
				if util.FileIsExist(path, r.Size) {
					// This fragment already exists on disk
					downloadedSize += r.Size
					dlg.SetValue(int(downloadedSize / (totalSize / 100)))
					return nil
				}

				conn, err := connection.NewConnectionWithContext(ctx, r.URI)
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
			gui.MessageBox("Предупреждение", "Загрузка отменена пользователем", walk.MsgBoxOK|walk.MsgBoxIconWarning)
		case err != nil:
			gui.MessageBox("Ошибка", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
		default:
			gui.MessageBox("Уведомление", "Книга успешно загружена", walk.MsgBoxOK|walk.MsgBoxIconWarning)
			log.Debug("Book %v has been successfully downloaded. Total size: %v", bookID, totalSize)
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
	log.Error(msg)
	switch {
	case errors.As(err, new(net.Error)):
		msg = "Ошибка сети. Пожалуйста, проверьте ваше подключение к интернету."
	case errors.Is(err, OperationNotSupported):
		msg = "Операция не поддерживается"
	case errors.Is(err, BookDescriptionNotAvailable):
		msg = "Описание книги недоступно"
	}
	gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
}
