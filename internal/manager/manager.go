package manager

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/connection"
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
	"github.com/lxn/walk"
)

type Manager struct {
	sync.WaitGroup
	service                 *config.Service
	client                  *daisy.Client
	readingSystemAttributes *daisy.ReadingSystemAttributes
	serviceAttributes       *daisy.ServiceAttributes
	bookplayer              *player.Player
	books                   *daisy.ContentList
	questions               *daisy.Questions
	userResponses           []daisy.UserResponse
}

func NewManager(readingSystemAttributes *daisy.ReadingSystemAttributes) *Manager {
	return &Manager{readingSystemAttributes: readingSystemAttributes}
}

func (m *Manager) Start(eventCH chan events.Event) {
	m.Add(1)
	defer m.Done()

	for evt := range eventCH {
		if m.client == nil && evt.EventCode != events.LIBRARY_LOGON && evt.EventCode != events.LIBRARY_ADD {
			// If the client is nil, we can only log in or add a new account
			log.Printf("event: %v: client is nil", evt.EventCode)
			continue
		}

		switch evt.EventCode {
		case events.ACTIVATE_MENU:
			index := gui.MainList.CurrentIndex()
			if m.books != nil {
				m.playBook(index)
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

		case events.OPEN_BOOKSHELF:
			m.setContent(daisy.Issued)

		case events.MAIN_MENU:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})

		case events.SEARCH_BOOK:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Search})

		case events.MENU_BACK:
			m.setQuestions(daisy.UserResponse{QuestionID: daisy.Back})

		case events.LIBRARY_LOGON:
			service, err := config.Conf.CurrentService()
			if err != nil {
				break
			}
			gui.SetLibraryMenu(eventCH, config.Conf.Services, service.Name)

			if evt.Data != nil {
				name, ok := evt.Data.(string)
				if !ok {
					log.Printf("logon: invalid service name")
					break
				}
				service, err = config.Conf.ServiceByName(name)
				if err != nil {
					log.Printf("logon: %v", err)
					break
				}
			}

			if err := m.logon(service); err != nil {
				log.Printf("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			config.Conf.SetCurrentService(service)
			gui.SetLibraryMenu(eventCH, config.Conf.Services, service.Name)

		case events.LIBRARY_ADD:
			service := new(config.Service)
			if gui.Credentials(service) != walk.DlgCmdOK || service.Name == "" {
				log.Printf("adding library: pressed Cancel button or len(service.Name) == 0")
				break
			}
			if err := m.logon(service); err != nil {
				log.Printf("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), walk.MsgBoxOK|walk.MsgBoxIconError)
				break
			}

			config.Conf.SetService(service)
			gui.SetLibraryMenu(eventCH, config.Conf.Services, service.Name)

		case events.LIBRARY_LOGOFF:
			m.logoff()

		case events.LIBRARY_REMOVE:
			if m.service == nil {
				break
			}
			msg := fmt.Sprintf("Вы действительно хотите удалить учётную запись %v?\nТакже будут удалены сохранённые позиции всех книг этой библиотеки.\nЭто действие не может быть отменено.", m.service.Name)
			if gui.MessageBox("Удаление учётной записи", msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
				break
			}
			config.Conf.RemoveService(m.service)
			m.logoff()

			service, err := config.Conf.CurrentService()
			if err != nil {
				break
			}
			gui.SetLibraryMenu(eventCH, config.Conf.Services, service.Name)

		case events.ISSUE_BOOK:
			if m.books != nil {
				index := gui.MainList.CurrentIndex()
				m.issueBook(index)
			}

		case events.REMOVE_BOOK:
			if m.books != nil {
				index := gui.MainList.CurrentIndex()
				m.removeBook(index)
			}

		case events.DOWNLOAD_BOOK:
			if m.books != nil {
				index := gui.MainList.CurrentIndex()
				m.downloadBook(index)
			}

		case events.BOOK_DESCRIPTION:
			if m.books != nil {
				index := gui.MainList.CurrentIndex()
				m.showBookDescription(index)
			}

		case events.PLAYER_PLAY_PAUSE:
			m.bookplayer.PlayPause()

		case events.PLAYER_STOP:
			m.saveBookPosition(m.bookplayer)
			m.bookplayer.Stop()

		case events.PLAYER_NEXT_TRACK:
			m.bookplayer.ChangeFragment(+1)

		case events.PLAYER_PREVIOUS_TRACK:
			m.bookplayer.ChangeFragment(-1)

		case events.PLAYER_SPEED_RESET:
			m.bookplayer.SetSpeed(player.DEFAULT_SPEED)

		case events.PLAYER_SPEED_UP:
			m.bookplayer.ChangeSpeed(+0.1)

		case events.PLAYER_SPEED_DOWN:
			m.bookplayer.ChangeSpeed(-0.1)

		case events.PLAYER_PITCH_RESET:
			m.bookplayer.SetPitch(player.DEFAULT_PITCH)

		case events.PLAYER_PITCH_UP:
			m.bookplayer.ChangePitch(+0.05)

		case events.PLAYER_PITCH_DOWN:
			m.bookplayer.ChangePitch(-0.05)

		case events.PLAYER_VOLUME_UP:
			m.bookplayer.ChangeVolume(+1)

		case events.PLAYER_VOLUME_DOWN:
			m.bookplayer.ChangeVolume(-1)

		case events.PLAYER_REWIND:
			offset, ok := evt.Data.(time.Duration)
			if !ok {
				log.Printf("manager.rewind: invalid evt.Data")
				break
			}
			m.bookplayer.ChangeOffset(offset)

		case events.PLAYER_FIRST:
			m.bookplayer.SetFragment(0)

		case events.PLAYER_GOTO_FRAGMENT:
			var text string
			fragment, _ := m.bookplayer.PositionInfo()
			if gui.TextEntryDialog("Переход к фрагменту", "Введите номер фрагмента:", strconv.Itoa(fragment+1), &text) != walk.DlgCmdOK {
				break
			}
			fragment, err := strconv.Atoi(text)
			if err != nil {
				break
			}
			m.bookplayer.SetFragment(fragment - 1) // Requires an index of fragment

		case events.PLAYER_GOTO_POSITION:
			var text string
			_, pos := m.bookplayer.PositionInfo()
			if gui.TextEntryDialog("Переход к позиции", "Введите позицию фрагмента:", util.FmtDuration(pos), &text) != walk.DlgCmdOK {
				break
			}
			var hours, minutes, seconds time.Duration
			_, err := fmt.Sscanf(text, "%d:%d:%d", &hours, &minutes, &seconds)
			if err != nil {
				log.Printf("goto offset: %v", err)
				break
			}
			offset := time.Hour*hours + time.Minute*minutes + time.Second*seconds
			_, pos = m.bookplayer.PositionInfo() // While requesting a new position, playback can continue. We need to get an up-to-date position
			m.bookplayer.ChangeOffset(offset - pos)

		case events.PLAYER_OUTPUT_DEVICE:
			device, ok := evt.Data.(string)
			if !ok {
				log.Printf("set output device: invalid device")
				break
			}
			config.Conf.General.OutputDevice = device
			m.bookplayer.SetOutputDevice(device)

		default:
			log.Printf("Unknown event: %v", evt.EventCode)

		}
	}
}

func (m *Manager) logoff() {
	m.saveBookPosition(m.bookplayer)
	m.bookplayer.Stop()
	gui.SetMainWindowTitle("")
	if _, err := m.client.LogOff(); err != nil {
		log.Printf("logoff: %v", err)
	}
	m.bookplayer = nil
	m.books = nil
	m.questions = nil
	m.client = nil
	m.service = nil
	m.serviceAttributes = nil
	m.userResponses = nil
	gui.MainList.SetItems([]string{}, "")
}

func (m *Manager) logon(service *config.Service) error {
	client := daisy.NewClient(service.URL, time.Second*5)
	if ok, err := client.LogOn(service.Credentials.Username, service.Credentials.Password); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("The LogOn operation returned false")
	}

	serviceAttributes, err := client.GetServiceAttributes()
	if err != nil {
		return err
	}

	_, err = client.SetReadingSystemAttributes(m.readingSystemAttributes)
	if err != nil {
		return err
	}

	if m.client != nil {
		m.logoff()
	}

	m.service = service
	m.client = client
	m.serviceAttributes = serviceAttributes
	m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})

	book := service.CurrentBook()
	if book.ID == "" {
		return nil
	}

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		log.Printf("GetContentResources: %v", err)
		return nil
	}

	gui.SetMainWindowTitle(book.Name)
	m.bookplayer = player.NewPlayer(book.ID, book.Name, r.Resources, config.Conf.General.OutputDevice)
	m.bookplayer.SetFragment(book.Fragment)
	m.bookplayer.ChangeOffset(book.ElapsedTime)
	return nil
}

func (m *Manager) setQuestions(response ...daisy.UserResponse) {
	if len(response) == 0 {
		log.Printf("Error: len(response) == 0")
		m.questions = nil
		gui.MainList.SetItems([]string{}, "")
		return
	}

	ur := daisy.UserResponses{UserResponse: response}
	questions, err := m.client.GetQuestions(&ur)
	if err != nil {
		msg := fmt.Sprintf("GetQuestions: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		m.questions = nil
		gui.MainList.SetItems([]string{}, "")
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
	m.books = nil

	var items []string
	for _, c := range choiceQuestion.Choices.Choice {
		items = append(items, c.Label.Text)
	}

	gui.MainList.SetItems(items, choiceQuestion.Label.Text)
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
	log.Printf("Content set: %s", contentID)
	contentList, err := m.client.GetContentList(contentID, 0, -1)
	if err != nil {
		msg := fmt.Sprintf("GetContentList: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		m.questions = nil
		gui.MainList.SetItems([]string{}, "")
		return
	}

	if len(contentList.ContentItems) == 0 {
		gui.MessageBox("Предупреждение", "Список книг пуст", walk.MsgBoxOK|walk.MsgBoxIconWarning)
		// Return to the main menu of the library
		m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
		return
	}

	m.books = contentList
	m.questions = nil

	var booksName []string
	for _, book := range m.books.ContentItems {
		booksName = append(booksName, book.Label.Text)
	}
	gui.MainList.SetItems(booksName, m.books.Label.Text)
}

func (m *Manager) saveBookPosition(bookplayer *player.Player) {
	bookName, bookID := bookplayer.BookInfo()
	if bookID != "" {
		fragment, elapsedTime := bookplayer.PositionInfo()
		m.service.SetBook(bookID, bookName, fragment, elapsedTime)
	}
}

func (m *Manager) playBook(index int) {
	book := m.books.ContentItems[index]
	if _, id := m.bookplayer.BookInfo(); book.ID == id {
		m.bookplayer.PlayPause()
		return
	}
	m.saveBookPosition(m.bookplayer)
	m.bookplayer.Stop()

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	b := m.service.Book(book.ID)
	gui.SetMainWindowTitle(book.Label.Text)
	m.bookplayer = player.NewPlayer(book.ID, book.Label.Text, r.Resources, config.Conf.General.OutputDevice)
	m.bookplayer.SetFragment(b.Fragment)
	m.bookplayer.ChangeOffset(b.ElapsedTime)
	m.bookplayer.PlayPause()
}

func (m *Manager) downloadBook(index int) {
	book := m.books.ContentItems[index]

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}

	// Book downloading should not block handling of other events
	go func() {
		me := &sync.Mutex{}
		var dst io.WriteCloser
		var conn, src io.ReadCloser
		var stop bool
		var err error

		cancelFN := func() {
			me.Lock()
			if src != nil {
				src.Close()
			}
			stop = true
			me.Unlock()
		}

		dlg := gui.NewProgressDialog("Загрузка книги", fmt.Sprintf("Загрузка %s", book.Label.Text), len(r.Resources), cancelFN)
		dlg.Show()

		for _, v := range r.Resources {
			path := filepath.Join(config.UserData(), util.ReplaceProhibitCharacters(book.Label.Text), v.LocalURI)
			if info, e := os.Stat(path); e == nil {
				if !info.IsDir() && info.Size() == v.Size {
					// v.LocalURI already exist
					dlg.IncreaseValue(1)
					continue
				}
			}

			conn, err = connection.NewConnection(v.URI)
			if err != nil {
				break
			}

			os.MkdirAll(filepath.Dir(path), os.ModeDir)
			dst, err = os.Create(path)
			if err != nil {
				conn.Close()
				break
			}

			me.Lock()
			src = conn
			if stop {
				src.Close()
			}
			me.Unlock()

			_, err = io.CopyBuffer(dst, src, make([]byte, 512*1024))
			dst.Close()
			src.Close()
			if err != nil {
				// Removing an unwritten file
				os.Remove(path)
				break
			}

			dlg.IncreaseValue(1)
		}

		dlg.Cancel()
		if stop {
			gui.MessageBox("Предупреждение", "Загрузка отменена пользователем", walk.MsgBoxOK|walk.MsgBoxIconWarning)
			return
		}

		if err != nil {
			gui.MessageBox("Ошибка", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconWarning)
			return
		}
		gui.MessageBox("Уведомление", "Книга успешно загружена", walk.MsgBoxOK|walk.MsgBoxIconWarning)
	}()
}

func (m *Manager) removeBook(index int) {
	book := m.books.ContentItems[index]
	_, err := m.client.ReturnContent(book.ID)
	if err != nil {
		msg := fmt.Sprintf("ReturnContent: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s удалена с книжной полки", book.Label.Text), walk.MsgBoxOK|walk.MsgBoxIconWarning)
}

func (m *Manager) issueBook(index int) {
	book := m.books.ContentItems[index]
	_, err := m.client.IssueContent(book.ID)
	if err != nil {
		msg := fmt.Sprintf("IssueContent: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, walk.MsgBoxOK|walk.MsgBoxIconError)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s добавлена на книжную полку", book.Label.Text), walk.MsgBoxOK|walk.MsgBoxIconWarning)
}

func (m *Manager) showBookDescription(index int) {
	book := m.books.ContentItems[index]
	md, err := m.client.GetContentMetadata(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentMetadata: %v", err)
		log.Printf(msg)
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
