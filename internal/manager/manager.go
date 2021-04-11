package manager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

type ContentItem interface {
	Label() daisy.Label
	ID() string
	Resources() ([]daisy.Resource, error)
}

type ContentList interface {
	Label() daisy.Label
	Len() int
	Item(int) ContentItem
}

type Manager struct {
	sync.WaitGroup
	library       *Library
	bookplayer    *player.Player
	currentBook   *config.Book
	contentList   ContentList
	questions     *daisy.Questions
	userResponses []daisy.UserResponse
}

func (m *Manager) Start(msgCH chan msg.Message) {
	m.Add(1)
	defer m.Done()

	for evt := range msgCH {
		if m.library == nil && evt.Code != msg.LIBRARY_LOGON && evt.Code != msg.LIBRARY_ADD && evt.Code != msg.LOG_SET_LEVEL {
			// If the library is nil, we can only log in or add a new account
			log.Info("message: %v: library is nil", evt.Code)
			continue
		}

		switch evt.Code {
		case msg.ACTIVATE_MENU:
			index := gui.MainList.CurrentIndex()
			if m.contentList != nil {
				book := m.contentList.Item(index)
				if m.currentBook == nil || book.ID() != m.currentBook.ID {
					if err := m.setBookplayer(book); err != nil {
						log.Info(err.Error())
						gui.MessageBox("Ошибка", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
						break
					}
				}
				m.bookplayer.PlayPause()
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
			name, ok := evt.Data.(string)
			if !ok {
				break
			}

			service, err := config.Conf.ServiceByName(name)
			if err != nil {
				log.Info("logon: %v", err)
				break
			}

			if err := m.logon(service); err != nil {
				log.Info("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			config.Conf.SetCurrentService(service)
			gui.SetLibraryMenu(msgCH, config.Conf.Services, service.Name)

		case msg.LIBRARY_ADD:
			service := new(config.Service)
			if gui.Credentials(service) != walk.DlgCmdOK || service.Name == "" {
				log.Info("adding library: pressed Cancel button or len(service.Name) == 0")
				break
			}

			if _, err := config.Conf.ServiceByName(service.Name); err == nil {
				gui.MessageBox("Ошибка", fmt.Sprintf("Учётная запись «%v» уже существует", service.Name), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			if err := m.logon(service); err != nil {
				log.Info("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			config.Conf.SetService(service)
			gui.SetLibraryMenu(msgCH, config.Conf.Services, service.Name)

		case msg.LIBRARY_LOGOFF:
			m.logoff()

		case msg.LIBRARY_REMOVE:
			msg := fmt.Sprintf("Вы действительно хотите удалить учётную запись %v?\nТакже будут удалены сохранённые позиции всех книг этой библиотеки.\nЭто действие не может быть отменено.", m.library.service.Name)
			if gui.MessageBox("Удаление учётной записи", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
				break
			}
			config.Conf.RemoveService(m.library.service)
			m.logoff()
			gui.SetLibraryMenu(msgCH, config.Conf.Services, "")

		case msg.ISSUE_BOOK:
			if m.contentList != nil {
				index := gui.MainList.CurrentIndex()
				book := m.contentList.Item(index)
				m.issueBook(book)
			}

		case msg.REMOVE_BOOK:
			if m.contentList != nil {
				index := gui.MainList.CurrentIndex()
				book := m.contentList.Item(index)
				m.removeBook(book)
			}

		case msg.DOWNLOAD_BOOK:
			if m.contentList != nil {
				index := gui.MainList.CurrentIndex()
				book := m.contentList.Item(index)
				m.downloadBook(book)
			}

		case msg.BOOK_DESCRIPTION:
			if m.contentList != nil {
				index := gui.MainList.CurrentIndex()
				book := m.contentList.Item(index)
				m.showBookDescription(book)
			}

		case msg.PLAYER_PLAY_PAUSE:
			m.bookplayer.PlayPause()

		case msg.PLAYER_STOP:
			m.saveBookPosition(m.bookplayer)
			m.bookplayer.Stop()

		case msg.PLAYER_OFFSET_FRAGMENT:
			offset, ok := evt.Data.(int)
			if !ok {
				log.Info("invalid offset fragment")
				break
			}
			fragment, _ := m.bookplayer.PositionInfo()
			m.bookplayer.SetFragment(fragment + offset)

		case msg.PLAYER_SPEED_RESET:
			m.bookplayer.SetSpeed(player.DEFAULT_SPEED)

		case msg.PLAYER_SPEED_UP:
			value := m.bookplayer.Speed()
			m.bookplayer.SetSpeed(value + 0.1)

		case msg.PLAYER_SPEED_DOWN:
			value := m.bookplayer.Speed()
			m.bookplayer.SetSpeed(value - 0.1)

		case msg.PLAYER_PITCH_RESET:
			m.bookplayer.SetPitch(player.DEFAULT_PITCH)

		case msg.PLAYER_PITCH_UP:
			value := m.bookplayer.Pitch()
			m.bookplayer.SetPitch(value + 0.05)

		case msg.PLAYER_PITCH_DOWN:
			value := m.bookplayer.Pitch()
			m.bookplayer.SetPitch(value - 0.05)

		case msg.PLAYER_VOLUME_UP:
			m.bookplayer.ChangeVolume(+1)

		case msg.PLAYER_VOLUME_DOWN:
			m.bookplayer.ChangeVolume(-1)

		case msg.PLAYER_OFFSET_POSITION:
			offset, ok := evt.Data.(time.Duration)
			if !ok {
				log.Info("invalid offset position")
				break
			}
			_, pos := m.bookplayer.PositionInfo()
			m.bookplayer.SetPosition(pos + offset)

		case msg.PLAYER_GOTO_FRAGMENT:
			fragment, ok := evt.Data.(int)
			if !ok {
				var text string
				curFragment, _ := m.bookplayer.PositionInfo()
				if gui.TextEntryDialog("Переход к фрагменту", "Введите номер фрагмента:", strconv.Itoa(curFragment+1), &text) != walk.DlgCmdOK {
					break
				}
				newFragment, err := strconv.Atoi(text)
				if err != nil {
					break
				}
				fragment = newFragment - 1 // Requires an index of fragment
			}
			m.bookplayer.SetFragment(fragment)

		case msg.PLAYER_GOTO_POSITION:
			var text string
			_, pos := m.bookplayer.PositionInfo()
			if gui.TextEntryDialog("Переход к позиции", "Введите позицию фрагмента:", util.FmtDuration(pos), &text) != walk.DlgCmdOK {
				break
			}
			position, err := util.ParseDuration(text)
			if err != nil {
				log.Info("goto position: %v", err)
				break
			}
			m.bookplayer.SetPosition(position)

		case msg.PLAYER_OUTPUT_DEVICE:
			device, ok := evt.Data.(string)
			if !ok {
				log.Info("set output device: invalid device")
				break
			}
			config.Conf.General.OutputDevice = device
			m.bookplayer.SetOutputDevice(device)

		case msg.PLAYER_SET_TIMER:
			var text string
			d := int(m.bookplayer.TimerDuration().Minutes())

			if gui.TextEntryDialog("Установка таймера паузы", "Введите время таймера в минутах:", strconv.Itoa(d), &text) != walk.DlgCmdOK {
				break
			}

			n, err := strconv.Atoi(text)
			if err != nil {
				break
			}

			config.Conf.General.PauseTimer = time.Minute * time.Duration(n)
			gui.SetPauseTimerLabel(n)
			m.bookplayer.SetTimerDuration(config.Conf.General.PauseTimer)

		case msg.BOOKMARK_SET:
			bookmarkID, ok := evt.Data.(string)
			if !ok {
				break
			}
			if m.currentBook != nil {
				bookmark := config.Bookmark{}
				bookmark.Fragment, bookmark.Position = m.bookplayer.PositionInfo()
				m.currentBook.SetBookmark(bookmarkID, bookmark)
			}

		case msg.BOOKMARK_FETCH:
			bookmarkID, ok := evt.Data.(string)
			if !ok {
				break
			}
			if m.currentBook != nil {
				bookmark, err := m.currentBook.Bookmark(bookmarkID)
				if err != nil {
					break
				}
				if f, _ := m.bookplayer.PositionInfo(); f == bookmark.Fragment {
					m.bookplayer.SetPosition(bookmark.Position)
					break
				}
				m.bookplayer.Stop()
				m.bookplayer.SetFragment(bookmark.Fragment)
				m.bookplayer.SetPosition(bookmark.Position)
				m.bookplayer.PlayPause()
			}

		case msg.LOG_SET_LEVEL:
			level, ok := evt.Data.(log.Level)
			if !ok {
				log.Error("Set log level: invalid level")
				break
			}
			config.Conf.General.LogLevel = level
			log.SetLevel(level)
			log.Info("Set log level to %s", level)

		default:
			log.Info("Unknown message: %v", evt.Code)

		}
	}
}

func (m *Manager) logoff() {
	m.saveBookPosition(m.bookplayer)
	m.bookplayer.Stop()
	gui.MainList.Clear()
	gui.SetMainWindowTitle("")

	if _, err := m.library.LogOff(); err != nil {
		log.Info("logoff: %v", err)
	}

	m.bookplayer = nil
	m.currentBook = nil
	m.contentList = nil
	m.questions = nil
	m.library = nil
	m.userResponses = nil
}

func (m *Manager) logon(service *config.Service) error {
	library, err := NewLibrary(service)
	if err != nil {
		return err
	}

	if m.library != nil {
		m.logoff()
	}

	m.library = library
	m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})

	id := m.library.service.CurrentBook()
	if id == "" {
		return nil
	}

	book, _ := m.library.service.Book(id)
	if err := m.setBookplayer(NewLibraryContentItem(m.library, book.ID, book.Name)); err != nil {
		log.Info(err.Error())
	}
	return nil
}

func (m *Manager) setQuestions(response ...daisy.UserResponse) {
	if len(response) == 0 {
		log.Info("Error: len(response) == 0")
		m.questions = nil
		gui.MainList.Clear()
		return
	}

	ur := daisy.UserResponses{UserResponse: response}
	questions, err := m.library.GetQuestions(&ur)
	if err != nil {
		msg := fmt.Sprintf("GetQuestions: %s", err)
		log.Info(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		m.questions = nil
		gui.MainList.Clear()
		return
	}

	if questions.Label.Text != "" {
		gui.MessageBox("Предупреждение", questions.Label.Text, walk.MsgBoxOK|walk.MsgBoxIconWarning)
		// Return to the main menu of the library
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
		return
	}

	if questions.ContentListRef != "" {
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
	m.contentList = nil

	var labels []string
	for _, c := range choiceQuestion.Choices.Choice {
		labels = append(labels, c.Label.Text)
	}

	gui.MainList.SetItems(labels, choiceQuestion.Label.Text, nil)
}

func (m *Manager) setInputQuestion() {
	for _, inputQuestion := range m.questions.InputQuestion {
		var text string
		if gui.TextEntryDialog("Поиск", inputQuestion.Label.Text, "", &text) != walk.DlgCmdOK {
			// Return to the main menu of the library
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
			return
		}
		m.userResponses = append(m.userResponses, daisy.UserResponse{QuestionID: inputQuestion.ID, Value: text})
	}

	m.setQuestions(m.userResponses...)
}

func (m *Manager) setContent(contentID string) {
	log.Info("Content set: %s", contentID)
	contentList, err := NewLibraryContentList(m.library, contentID)
	if err != nil {
		msg := fmt.Sprintf("GetContentList: %s", err)
		log.Info(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
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

	m.contentList = contentList
	m.questions = nil

	labels := make([]string, m.contentList.Len())
	ids := make([]string, m.contentList.Len())
	for i := range labels {
		book := m.contentList.Item(i)
		labels[i] = book.Label().Text
		ids[i] = book.ID()
	}

	if contentID == daisy.Issued {
		if m.currentBook != nil && !util.StringInSlice(m.currentBook.ID, ids) {
			ids = append(ids, m.currentBook.ID)
		}
		m.library.service.Tidy(ids)
	}

	gui.MainList.SetItems(labels, m.contentList.Label().Text, gui.BookMenu)
}

func (m *Manager) saveBookPosition(bookplayer *player.Player) {
	if m.currentBook != nil {
		m.currentBook.Fragment, m.currentBook.ElapsedTime = bookplayer.PositionInfo()
		m.library.service.SetBook(*m.currentBook)
	}
}

func (m *Manager) setBookplayer(book ContentItem) error {
	m.saveBookPosition(m.bookplayer)
	m.bookplayer.Stop()

	rsrc, err := book.Resources()
	if err != nil {
		return fmt.Errorf("GetContentResources: %v", err)
	}

	gui.SetMainWindowTitle(book.Label().Text)
	bookDir := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Label().Text))
	m.bookplayer = player.NewPlayer(bookDir, rsrc, config.Conf.General.OutputDevice)
	m.currentBook = &config.Book{
		Name: book.Label().Text,
		ID:   book.ID(),
	}
	m.bookplayer.SetTimerDuration(config.Conf.General.PauseTimer)
	if book, err := m.library.service.Book(book.ID()); err == nil {
		m.bookplayer.SetFragment(book.Fragment)
		m.bookplayer.SetPosition(book.ElapsedTime)
		m.currentBook = book
	}
	return nil
}

func (m *Manager) downloadBook(book ContentItem) {
	rsrc, err := book.Resources()
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %v", err)
		log.Info(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	// Book downloading should not block handling of other messages
	go func() {
		var err error
		ctx, cancelFunc := context.WithCancel(context.TODO())
		dlg := gui.NewProgressDialog("Загрузка книги", fmt.Sprintf("Загрузка %s", book.Label().Text), len(rsrc), cancelFunc)
		dlg.Show()

		for _, r := range rsrc {
			var n int64
			path := filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(book.Label().Text), r.LocalURI)
			n, err = func() (int64, error) {
				if info, e := os.Stat(path); e == nil {
					if !info.IsDir() && info.Size() == r.Size {
						// r.LocalURI already exist
						return info.Size(), nil
					}
				}

				src, err := connection.NewConnectionWithContext(ctx, r.URI)
				if err != nil {
					return 0, err
				}
				defer src.Close()

				os.MkdirAll(filepath.Dir(path), os.ModeDir)
				dst, err := os.Create(path)
				if err != nil {
					return 0, err
				}
				defer dst.Close()

				return io.CopyBuffer(dst, src, make([]byte, 512*1024))
			}()

			if err == nil && r.Size != n {
				err = errors.New("resource size mismatch")
			}

			if err != nil {
				// Removing an unwritten file
				os.Remove(path)
				break
			}

			dlg.IncreaseValue(1)
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
}

func (m *Manager) removeBook(book ContentItem) {
	_, err := m.library.ReturnContent(book.ID())
	if err != nil {
		msg := fmt.Sprintf("ReturnContent: %s", err)
		log.Info(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s удалена с книжной полки", book.Label().Text), walk.MsgBoxOK|walk.MsgBoxIconWarning)
}

func (m *Manager) issueBook(book ContentItem) {
	_, err := m.library.IssueContent(book.ID())
	if err != nil {
		msg := fmt.Sprintf("IssueContent: %s", err)
		log.Info(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s добавлена на книжную полку", book.Label().Text), walk.MsgBoxOK|walk.MsgBoxIconWarning)
}

func (m *Manager) showBookDescription(book ContentItem) {
	md, err := m.library.GetContentMetadata(book.ID())
	if err != nil {
		msg := fmt.Sprintf("GetContentMetadata: %v", err)
		log.Info(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	text := fmt.Sprintf("%v", strings.Join(md.Metadata.Description, "\r\n"))
	if text == "" {
		gui.MessageBox("Ошибка", "Нет доступной информации о книге", walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	gui.MessageBox("Информация о книге", text, walk.MsgBoxOK|walk.MsgBoxIconWarning)
}
