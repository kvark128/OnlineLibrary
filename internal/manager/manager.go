package manager

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/player"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
)

type Manager struct {
	sync.WaitGroup
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
		if m.client == nil && evt != events.LIBRARY_LOGON && evt != events.LIBRARY_ADD && evt != events.LIBRARY_SWITCH {
			// If the client is nil, we can only log in or add a new account
			log.Printf("event: %v: client is nil", evt)
			continue
		}

		switch evt {
		case events.ACTIVATE_MENU:
			index := gui.CurrentListBoxIndex()
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

		case events.LIBRARY_SWITCH:
			index := gui.GetCurrentLibrary()
			service := config.Conf.Services.Service(index)
			if err := m.logon(service); err != nil {
				log.Printf("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), gui.MsgBoxOK|gui.MsgBoxIconError)
				break
			}
			config.Conf.Services.SetCurrentService(index)
			_, index, _ = config.Conf.Services.CurrentService()
			gui.SetLibraryMenu(eventCH, config.Conf.Services, index)

		case events.LIBRARY_ADD:
			var service config.Service
			if gui.Credentials(&service) != gui.DlgCmdOK || service.Name == "" {
				log.Printf("adding library: pressed Cancel button or len(service.Name) == 0")
				break
			}
			if err := m.logon(service); err != nil {
				log.Printf("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), gui.MsgBoxOK|gui.MsgBoxIconError)
				break
			}
			config.Conf.Services.SetService(service)
			_, index, _ := config.Conf.Services.CurrentService()
			gui.SetLibraryMenu(eventCH, config.Conf.Services, index)

		case events.LIBRARY_LOGON:
			service, index, err := config.Conf.Services.CurrentService()
			if err != nil {
				break
			}
			gui.SetLibraryMenu(eventCH, config.Conf.Services, index)
			if err := m.logon(service); err != nil {
				log.Printf("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), gui.MsgBoxOK|gui.MsgBoxIconError)
			}

		case events.LIBRARY_LOGOFF:
			m.logoff()

		case events.LIBRARY_REMOVE:
			service, index, err := config.Conf.Services.CurrentService()
			if err != nil {
				log.Printf("removing of service: %v", err)
				break
			}
			msg := fmt.Sprintf("Вы действительно хотите удалить учётную запись %v?\nТакже будут удалены сохранённые позиции всех книг этой библиотеки.\nЭто действие не может быть отменено.", service.Name)
			if gui.QuestionDialog("Удаление учётной записи", msg) != gui.DlgCmdYes {
				break
			}
			m.logoff()
			config.Conf.Services.RemoveService(index)
			_, index, _ = config.Conf.Services.CurrentService()
			gui.SetLibraryMenu(eventCH, config.Conf.Services, index)

		case events.ISSUE_BOOK:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.issueBook(index)
			}

		case events.REMOVE_BOOK:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.removeBook(index)
			}

		case events.DOWNLOAD_BOOK:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.downloadBook(index)
			}

		case events.BOOK_DESCRIPTION:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.showBookDescription(index)
			}

		case events.PLAYER_PLAY_PAUSE:
			m.bookplayer.PlayPause()

		case events.PLAYER_STOP:
			m.saveBookPosition(m.bookplayer)
			m.bookplayer.Stop()

		case events.PLAYER_NEXT_TRACK:
			m.bookplayer.ChangeTrack(+1)

		case events.PLAYER_PREVIOUS_TRACK:
			m.bookplayer.ChangeTrack(-1)

		case events.PLAYER_SPEED_RESET:
			m.bookplayer.SetSpeed(player.DEFAULT_SPEED)

		case events.PLAYER_SPEED_UP:
			m.bookplayer.ChangeSpeed(+0.1)

		case events.PLAYER_SPEED_DOWN:
			m.bookplayer.ChangeSpeed(-0.1)

		case events.PLAYER_VOLUME_UP:
			m.bookplayer.ChangeVolume(+1)

		case events.PLAYER_VOLUME_DOWN:
			m.bookplayer.ChangeVolume(-1)

		case events.PLAYER_FORWARD:
			m.bookplayer.ChangeOffset(time.Second * +5)

		case events.PLAYER_BACK:
			m.bookplayer.ChangeOffset(time.Second * -5)

		case events.PLAYER_FIRST:
			m.bookplayer.SetTrack(0)

		default:
			log.Printf("Unknown event: %v", evt)

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
	m.serviceAttributes = nil
	m.userResponses = nil
	gui.SetListBoxItems([]string{}, "")
}

func (m *Manager) logon(service config.Service) error {
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

	m.client = client
	m.serviceAttributes = serviceAttributes
	m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})

	book := service.RecentBooks.CurrentBook()
	if book.ID == "" {
		return nil
	}

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		log.Printf("GetContentResources: %v", err)
		return nil
	}

	gui.SetMainWindowTitle(book.Name)
	m.bookplayer = player.NewPlayer(book.ID, book.Name, r.Resources)
	m.bookplayer.SetTrack(book.Fragment)
	m.bookplayer.ChangeOffset(book.ElapsedTime)
	return nil
}

func (m *Manager) setQuestions(response ...daisy.UserResponse) {
	if len(response) == 0 {
		log.Printf("Error: len(response) == 0")
		m.questions = nil
		gui.SetListBoxItems([]string{}, "")
		return
	}

	ur := daisy.UserResponses{UserResponse: response}
	questions, err := m.client.GetQuestions(&ur)
	if err != nil {
		msg := fmt.Sprintf("GetQuestions: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		m.questions = nil
		gui.SetListBoxItems([]string{}, "")
		return
	}

	if questions.Label.Text != "" {
		gui.MessageBox("Предупреждение", questions.Label.Text, gui.MsgBoxOK|gui.MsgBoxIconWarning)
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

	gui.SetListBoxItems(items, choiceQuestion.Label.Text)
}

func (m *Manager) setInputQuestion() {
	for _, inputQuestion := range m.questions.InputQuestion {
		var text string
		if gui.TextEntryDialog("Поиск", inputQuestion.Label.Text, &text) != gui.DlgCmdOK {
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
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		m.questions = nil
		gui.SetListBoxItems([]string{}, "")
		return
	}

	if len(contentList.ContentItems) == 0 {
		gui.MessageBox("Предупреждение", "Список книг пуст", gui.MsgBoxOK|gui.MsgBoxIconWarning)
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
	gui.SetListBoxItems(booksName, m.books.Label.Text)
}

func (m *Manager) saveBookPosition(bookplayer *player.Player) {
	bookName, bookID := bookplayer.BookInfo()
	if bookID != "" {
		fragment, elapsedTime := bookplayer.PositionInfo()
		service, _, _ := config.Conf.Services.CurrentService()
		service.RecentBooks.SetBook(bookID, bookName, fragment, elapsedTime)
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
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	service, _, _ := config.Conf.Services.CurrentService()
	b := service.RecentBooks.Book(book.ID)
	gui.SetMainWindowTitle(book.Label.Text)
	m.bookplayer = player.NewPlayer(book.ID, book.Label.Text, r.Resources)
	m.bookplayer.SetTrack(b.Fragment)
	m.bookplayer.ChangeOffset(b.ElapsedTime)
	m.bookplayer.PlayPause()
}

func (m *Manager) downloadBook(index int) {
	book := m.books.ContentItems[index]

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	// Downloading book should not block handling of other events
	go util.DownloadBook(config.UserData(), book.Label.Text, r)
}

func (m *Manager) removeBook(index int) {
	book := m.books.ContentItems[index]
	_, err := m.client.ReturnContent(book.ID)
	if err != nil {
		msg := fmt.Sprintf("ReturnContent: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s удалена с книжной полки", book.Label.Text), gui.MsgBoxOK|gui.MsgBoxIconWarning)
}

func (m *Manager) issueBook(index int) {
	book := m.books.ContentItems[index]
	_, err := m.client.IssueContent(book.ID)
	if err != nil {
		msg := fmt.Sprintf("IssueContent: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s добавлена на книжную полку", book.Label.Text), gui.MsgBoxOK|gui.MsgBoxIconWarning)
}

func (m *Manager) showBookDescription(index int) {
	book := m.books.ContentItems[index]
	md, err := m.client.GetContentMetadata(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentMetadata: %v", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	text := fmt.Sprintf("%v", strings.Join(md.Metadata.Description, "\r\n"))
	if text == "" {
		gui.MessageBox("Ошибка", "Нет доступной информации о книге", gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	gui.MessageBox("Информация о книге", text, gui.MsgBoxOK|gui.MsgBoxIconWarning)
}
