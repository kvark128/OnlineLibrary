package manager

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
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
	daisy "github.com/kvark128/daisyonline"
	"github.com/lxn/walk"
)

const (
	MetadataFileName = "metadata.xml"
	CRLF             = "\r\n"
)

var (
	NoBookDescription     = errors.New("no book description")
	OperationNotSupported = errors.New("operation not supported")
)

type Manager struct {
	library       *Library
	bookplayer    *player.Player
	currentBook   ContentItem
	contentList   ContentList
	questions     *daisy.Questions
	userResponses []daisy.UserResponse
	lastInputText string
}

func (m *Manager) Start(msgCH chan msg.Message, done chan<- bool) {
	defer func() { done <- true }()
	defer m.cleaning()

	if config.Conf.General.OpenLocalBooksAtStartup {
		msgCH <- msg.Message{Code: msg.OPEN_LOCALBOOKS}
	} else if service, err := config.Conf.CurrentService(); err == nil {
		msgCH <- msg.Message{Code: msg.LIBRARY_LOGON, Data: service.Name}
	}

	for message := range msgCH {
		switch message.Code {
		case msg.ACTIVATE_MENU:
			index := gui.MainList.CurrentIndex()
			if m.contentList != nil {
				book := m.contentList.Item(index)
				if m.currentBook == nil || m.currentBook.ID() != book.ID() {
					if err := m.setBookplayer(book); err != nil {
						m.messageBoxError(fmt.Errorf("Set book player: %w", err))
						break
					}
				}
				if m.bookplayer != nil {
					m.bookplayer.PlayPause()
				}
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
			m.setContent(daisy.Issued)

		case msg.MAIN_MENU:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})

		case msg.SEARCH_BOOK:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Search})

		case msg.MENU_BACK:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Back})

		case msg.LIBRARY_LOGON:
			name, ok := message.Data.(string)
			if !ok {
				break
			}

			service, err := config.Conf.ServiceByName(name)
			if err != nil {
				log.Error("logon: %v", err)
				break
			}

			if err := m.setLibrary(service); err != nil {
				m.messageBoxError(fmt.Errorf("Set library: %w", err))
				break
			}

			config.Conf.SetCurrentService(service)
			gui.SetLibraryMenu(msgCH, config.Conf.Services, service.Name)

		case msg.LIBRARY_ADD:
			service := new(config.Service)
			if gui.Credentials(service) != walk.DlgCmdOK || service.Name == "" {
				log.Warning("Adding library: pressed Cancel button or len(service.Name) == 0")
				break
			}

			if _, err := config.Conf.ServiceByName(service.Name); err == nil {
				gui.MessageBox("Ошибка", fmt.Sprintf("Учётная запись «%v» уже существует", service.Name), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			if err := m.setLibrary(service); err != nil {
				m.messageBoxError(fmt.Errorf("setLibrary: %w", err))
				break
			}

			config.Conf.SetService(service)
			gui.SetLibraryMenu(msgCH, config.Conf.Services, service.Name)

		case msg.LIBRARY_REMOVE:
			if m.library == nil {
				break
			}
			msg := fmt.Sprintf("Вы действительно хотите удалить учётную запись %v?%sТакже будут удалены сохранённые позиции всех книг этой библиотеки.%sЭто действие не может быть отменено.", m.library.service.Name, CRLF, CRLF)
			if gui.MessageBox("Удаление учётной записи", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
				break
			}
			config.Conf.RemoveService(m.library.service)
			m.cleaning()
			gui.SetLibraryMenu(msgCH, config.Conf.Services, "")

		case msg.ISSUE_BOOK:
			if m.contentList == nil {
				log.Warning("Attempt to add a book to a bookshelf when there is no content list")
				break
			}
			book := m.contentList.Item(gui.MainList.CurrentIndex())
			if err := m.issueBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Issue book: %w", err))
				break
			}
			gui.MessageBox("Уведомление", fmt.Sprintf("«%s» добавлена на книжную полку", book.Label().Text), walk.MsgBoxOK|walk.MsgBoxIconWarning)

		case msg.REMOVE_BOOK:
			if m.contentList == nil {
				log.Warning("Attempt to remove a book from a bookshelf when there is no content list")
				break
			}
			book := m.contentList.Item(gui.MainList.CurrentIndex())
			if err := m.removeBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Removing book: %w", err))
				break
			}
			gui.MessageBox("Уведомление", fmt.Sprintf("«%s» удалена с книжной полки", book.Label().Text), walk.MsgBoxOK|walk.MsgBoxIconWarning)
			// If a bookshelf is open, it must be updated to reflect the changes made
			if m.contentList.ID() == daisy.Issued {
				m.setContent(daisy.Issued)
			}

		case msg.DOWNLOAD_BOOK:
			if m.contentList == nil {
				break
			}
			index := gui.MainList.CurrentIndex()
			book := m.contentList.Item(index)
			if err := m.downloadBook(book); err != nil {
				m.messageBoxError(fmt.Errorf("Book downloading: %w", err))
			}

		case msg.BOOK_DESCRIPTION:
			if m.contentList == nil {
				break
			}
			index := gui.MainList.CurrentIndex()
			book := m.contentList.Item(index)
			text, err := m.bookDescription(book)
			if err != nil {
				m.messageBoxError(fmt.Errorf("book description: %w", err))
				break
			}
			gui.MessageBox("Информация о книге", text, walk.MsgBoxOK|walk.MsgBoxIconWarning)

		case msg.PLAYER_PLAY_PAUSE:
			if m.bookplayer != nil {
				m.bookplayer.PlayPause()
			}

		case msg.PLAYER_STOP:
			m.setBookmark(config.ListeningPosition)
			if m.bookplayer != nil {
				m.bookplayer.Stop()
			}

		case msg.PLAYER_OFFSET_FRAGMENT:
			offset, ok := message.Data.(int)
			if !ok {
				log.Error("Invalid offset fragment")
				break
			}
			if m.bookplayer != nil {
				fragment := m.bookplayer.Fragment()
				m.bookplayer.SetFragment(fragment + offset)
			}

		case msg.PLAYER_SPEED_RESET:
			if m.bookplayer != nil {
				m.bookplayer.SetSpeed(player.DEFAULT_SPEED)
			}

		case msg.PLAYER_SPEED_UP:
			if m.bookplayer != nil {
				value := m.bookplayer.Speed()
				m.bookplayer.SetSpeed(value + 0.1)
			}

		case msg.PLAYER_SPEED_DOWN:
			if m.bookplayer != nil {
				value := m.bookplayer.Speed()
				m.bookplayer.SetSpeed(value - 0.1)
			}

		case msg.PLAYER_VOLUME_RESET:
			if m.bookplayer != nil {
				m.bookplayer.SetVolume(player.DEFAULT_VOLUME)
			}

		case msg.PLAYER_VOLUME_UP:
			if m.bookplayer != nil {
				volume := m.bookplayer.Volume()
				m.bookplayer.SetVolume(volume + 0.05)
			}

		case msg.PLAYER_VOLUME_DOWN:
			if m.bookplayer != nil {
				volume := m.bookplayer.Volume()
				m.bookplayer.SetVolume(volume - 0.05)
			}

		case msg.PLAYER_OFFSET_POSITION:
			offset, ok := message.Data.(time.Duration)
			if !ok {
				log.Error("Invalid offset position")
				break
			}
			if m.bookplayer != nil {
				pos := m.bookplayer.Position()
				m.bookplayer.SetPosition(pos + offset)
			}

		case msg.PLAYER_GOTO_FRAGMENT:
			fragment, ok := message.Data.(int)
			if !ok {
				var text string
				var curFragment int
				if m.bookplayer != nil {
					curFragment = m.bookplayer.Fragment()
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
			if m.bookplayer != nil {
				m.bookplayer.SetFragment(fragment)
			}

		case msg.PLAYER_GOTO_POSITION:
			var text string
			var pos time.Duration
			if m.bookplayer != nil {
				pos = m.bookplayer.Position()
			}
			if gui.TextEntryDialog("Переход к позиции", "Введите позицию фрагмента:", util.FmtDuration(pos), &text) != walk.DlgCmdOK {
				break
			}
			position, err := util.ParseDuration(text)
			if err != nil {
				log.Error("goto position: %v", err)
				break
			}
			if m.bookplayer != nil {
				m.bookplayer.SetPosition(position)
			}

		case msg.PLAYER_OUTPUT_DEVICE:
			device, ok := message.Data.(string)
			if !ok {
				log.Error("set output device: invalid device")
				break
			}
			config.Conf.General.OutputDevice = device
			if m.bookplayer != nil {
				m.bookplayer.SetOutputDevice(device)
			}

		case msg.PLAYER_SET_TIMER:
			var text string
			var d int
			if m.bookplayer != nil {
				d = int(m.bookplayer.TimerDuration().Minutes())
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
			if m.bookplayer != nil {
				m.bookplayer.SetTimerDuration(config.Conf.General.PauseTimer)
			}

		case msg.BOOKMARK_SET:
			bookmarkID, ok := message.Data.(string)
			if !ok {
				break
			}
			m.setBookmark(bookmarkID)

		case msg.BOOKMARK_FETCH:
			bookmarkID, ok := message.Data.(string)
			if !ok {
				break
			}
			if m.currentBook != nil {
				conf := m.currentBook.Config()
				bookmark, err := conf.Bookmark(bookmarkID)
				if err != nil {
					break
				}
				if m.bookplayer != nil {
					if f := m.bookplayer.Fragment(); f == bookmark.Fragment {
						m.bookplayer.SetPosition(bookmark.Position)
						break
					}
					m.bookplayer.Stop()
					m.bookplayer.SetFragment(bookmark.Fragment)
					m.bookplayer.SetPosition(bookmark.Position)
					m.bookplayer.PlayPause()
				}
			}

		case msg.LOG_SET_LEVEL:
			level, ok := message.Data.(log.Level)
			if !ok {
				log.Error("Set log level: invalid level")
				break
			}
			config.Conf.General.LogLevel = level.String()
			log.SetLevel(level)
			log.Info("Set log level to %s", level)

		case msg.OPEN_LOCALBOOKS:
			contentList, err := NewLocalContentList()
			if err != nil {
				m.messageBoxError(err)
				break
			}
			m.cleaning()
			m.updateContentList(contentList)
			config.Conf.General.OpenLocalBooksAtStartup = true
			book, err := config.Conf.LocalBooks.LastBook()
			if err != nil {
				break
			}
			for i := contentList.Len() - 1; i >= 0; i-- {
				item := contentList.Item(i)
				if item.ID() == book.ID {
					m.setBookplayer(item)
					break
				}
			}

		case msg.LIBRARY_INFO:
			if m.library == nil {
				break
			}
			var lines []string
			attrs := m.library.serviceAttributes
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
	m.setBookmark(config.ListeningPosition)
	if m.bookplayer != nil {
		config.Conf.General.Volume = int(m.bookplayer.Volume()/0.05) - 20
		book := m.currentBook.Config()
		book.Speed = int((m.bookplayer.Speed()-player.MIN_SPEED)/0.1) - 5
		m.currentBook.SetConfig(book)
		m.bookplayer.Stop()
		m.bookplayer = nil
	}
	gui.MainList.Clear()
	gui.SetMainWindowTitle("")
	m.currentBook = nil
	m.contentList = nil
	m.questions = nil
	m.userResponses = nil

	if m.library != nil {
		_, err := m.library.LogOff()
		if err != nil {
			log.Warning("library logoff: %v", err)
		}
		m.library = nil
	}
}

func (m *Manager) setLibrary(service *config.Service) error {
	library, err := NewLibrary(service)
	if err != nil {
		return fmt.Errorf("creating library: %w", err)
	}

	m.cleaning()
	m.library = library
	m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
	config.Conf.General.OpenLocalBooksAtStartup = false

	if book, err := m.library.service.RecentBooks.LastBook(); err == nil {
		if err := m.setBookplayer(NewLibraryContentItem(m.library, book.ID, book.Name)); err != nil {
			log.Error("Set book player: %v", err)
		}
	}
	return nil
}

func (m *Manager) setQuestions(response ...daisy.UserResponse) {
	if m.library == nil {
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
	questions, err := m.library.GetQuestions(&ur)
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
		m.setContent(questions.ContentListRef)
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
	gui.MainList.SetItems(labels, choiceQuestion.Label.Text, nil)
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

func (m *Manager) setContent(contentID string) {
	if m.library == nil {
		m.messageBoxError(OperationNotSupported)
		return
	}

	log.Info("Content set: %s", contentID)
	contentList, err := NewLibraryContentList(m.library, contentID)
	if err != nil {
		m.messageBoxError(fmt.Errorf("GetContentList: %w", err))
		m.questions = nil
		gui.MainList.Clear()
		return
	}

	if contentList.Len() == 0 {
		gui.MessageBox("Предупреждение", "Список книг пуст", walk.MsgBoxOK|walk.MsgBoxIconWarning)
		// Return to the main menu of the library
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
		return
	}

	if contentID == daisy.Issued {
		ids := make([]string, contentList.Len())
		for i := range ids {
			book := contentList.Item(i)
			ids[i] = book.ID()
		}
		if m.currentBook != nil && !util.StringInSlice(m.currentBook.ID(), ids) {
			ids = append(ids, m.currentBook.ID())
		}
		m.library.service.RecentBooks.Tidy(ids)
	}

	m.updateContentList(contentList)
}

func (m *Manager) updateContentList(contentList ContentList) {
	labels := make([]string, contentList.Len())
	for i := range labels {
		book := contentList.Item(i)
		labels[i] = book.Label().Text
	}

	m.contentList = contentList
	m.questions = nil
	gui.MainList.SetItems(labels, contentList.Label().Text, gui.BookMenu)
}

func (m *Manager) setBookmark(bookmarkID string) {
	if m.currentBook != nil {
		var bookmark config.Bookmark
		if m.bookplayer != nil {
			bookmark.Fragment = m.bookplayer.Fragment()
			bookmark.Position = m.bookplayer.Position()
		}
		conf := m.currentBook.Config()
		conf.SetBookmark(bookmarkID, bookmark)
		m.currentBook.SetConfig(conf)
	}
}

func (m *Manager) setBookplayer(book ContentItem) error {
	m.setBookmark(config.ListeningPosition)
	if m.bookplayer != nil {
		config.Conf.General.Volume = int(m.bookplayer.Volume()/0.05) - 20
		book := m.currentBook.Config()
		book.Speed = int((m.bookplayer.Speed()-player.MIN_SPEED)/0.1) - 5
		m.currentBook.SetConfig(book)
		m.bookplayer.Stop()
	}

	rsrc, err := book.Resources()
	if err != nil {
		return fmt.Errorf("GetContentResources: %v", err)
	}

	gui.SetMainWindowTitle(book.Label().Text)
	bookDir := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Label().Text))
	conf := book.Config()
	m.bookplayer = player.NewPlayer(bookDir, rsrc, config.Conf.General.OutputDevice)
	m.currentBook = book
	m.bookplayer.SetSpeed(float64(conf.Speed+5)*0.1 + player.MIN_SPEED)
	m.bookplayer.SetTimerDuration(config.Conf.General.PauseTimer)
	m.bookplayer.SetVolume(float64(config.Conf.General.Volume+20) * 0.05)
	if bookmark, err := conf.Bookmark(config.ListeningPosition); err == nil {
		m.bookplayer.SetFragment(bookmark.Fragment)
		m.bookplayer.SetPosition(bookmark.Position)
	}
	return nil
}

func (m *Manager) downloadBook(book ContentItem) error {
	if m.library == nil {
		return OperationNotSupported
	}

	rsrc, err := book.Resources()
	if err != nil {
		return fmt.Errorf("getContentResources: %w", err)
	}

	// Path to the directory where we will download a resources. Make sure that it does not contain prohibited characters
	bookDir := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Label().Text))

	if md, err := book.ContentMetadata(); err == nil {
		path := filepath.Join(bookDir, MetadataFileName)
		f, err := util.CreateSecureFile(path)
		if err != nil {
			log.Warning("creating %v: %v", MetadataFileName, err)
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

	// Book downloading should not block handling of other messages
	go func() {
		var err error
		var totalSize, downloadedSize int64
		ctx, cancelFunc := context.WithCancel(context.TODO())
		label := "Загрузка «%s»\nСкорость: %d Кб/с"
		dlg := gui.NewProgressDialog("Загрузка книги", fmt.Sprintf(label, book.Label().Text, 0), 100, cancelFunc)
		dlg.Show()

		for _, r := range rsrc {
			totalSize += r.Size
		}

		for _, r := range rsrc {
			err = func() error {
				path := filepath.Join(bookDir, r.LocalURI)
				if util.FileIsExist(path, r.Size) {
					// This fragment already exists on disk
					downloadedSize += r.Size
					dlg.SetValue(int(downloadedSize / (totalSize / 100)))
					return nil
				}

				src, err := connection.NewConnectionWithContext(ctx, r.URI)
				if err != nil {
					return err
				}
				defer src.Close()

				dst, err := util.CreateSecureFile(path)
				if err != nil {
					return err
				}
				defer dst.Close()

				var fragmentSize int64
				for err == nil {
					var n int64
					t := time.Now()
					n, err = io.CopyN(dst, src, 512*1024)
					sec := time.Since(t).Seconds()
					speed := int(float64(n) / 1024 / sec)
					dlg.SetLabel(fmt.Sprintf(label, book.Label().Text, speed))
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
		}
	}()
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
		return "", err
	}

	text := fmt.Sprintf("%v", strings.Join(md.Metadata.Description, CRLF))
	if text == "" {
		return "", NoBookDescription
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
	}
	gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
}
