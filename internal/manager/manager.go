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
	return &Manager{
		client:                  daisy.NewClient("", time.Duration(0)), // Empty client
		readingSystemAttributes: readingSystemAttributes,
	}
}

func (m *Manager) Start(eventCH chan events.Event) {
	m.Add(1)
	defer m.Done()

	for evt := range eventCH {
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

		case events.LIBRARY_LOGON:
			if err := m.logon(); err != nil {
				log.Printf("logon: %v", err)
				gui.MessageBox("Ошибка", fmt.Sprintf("logon: %v", err), gui.MsgBoxOK|gui.MsgBoxIconError)
			}

		case events.LIBRARY_LOGOFF:
			m.logoff()

		case events.LIBRARY_RELOGON:
			m.logoff()
			config.Conf.Services = config.Conf.Services[:len(config.Conf.Services)-1]
			m.logon()

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

		case events.PLAYER_PAUSE:
			m.bookplayer.Pause()

		case events.PLAYER_STOP:
			m.bookplayer.Stop()

		case events.PLAYER_NEXT_TRACK:
			m.bookplayer.ChangeTrack(+1)

		case events.PLAYER_PREVIOUS_TRACK:
			m.bookplayer.ChangeTrack(-1)

		case events.DOWNLOAD_BOOK:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.downloadBook(index)
			}

		case events.PLAYER_VOLUME_UP:
			m.bookplayer.ChangeVolume(+1)

		case events.PLAYER_VOLUME_DOWN:
			m.bookplayer.ChangeVolume(-1)

		case events.PLAYER_FORWARD:
			m.bookplayer.Rewind(time.Second * +5)

		case events.PLAYER_BACK:
			m.bookplayer.Rewind(time.Second * -5)

		case events.BOOK_PROPERTIES:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.showBookProperties(index)
			}

		default:
			log.Printf("Unknown event: %v", evt)

		}
	}
}

func (m *Manager) logoff() {
	_, err := m.client.LogOff()
	if err != nil {
		log.Printf("logoff: %v", err)
	}

	m.books = nil
	m.questions = nil
	gui.SetListBoxItems([]string{}, "")
}

func (m *Manager) logon() error {
	var username, password, serviceURL string
	var err error
	var save bool

	if len(config.Conf.Services) != 0 {
		// We always use the last service in the list to log in
		service := config.Conf.Services[len(config.Conf.Services)-1]
		username = service.Credentials.Username
		password = service.Credentials.Password
		serviceURL = service.URL
	}

	if username == "" || password == "" || serviceURL == "" {
		username, password, serviceURL, save, err = gui.Credentials()
		if err != nil {
			log.Printf("Credentials: %s", err)
			return nil
		}
	}

	m.client = daisy.NewClient(serviceURL, time.Second*5)
	ok, err := m.client.LogOn(username, password)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("The LogOn operation returned false")
	}

	m.serviceAttributes, err = m.client.GetServiceAttributes()
	if err != nil {
		return err
	}

	_, err = m.client.SetReadingSystemAttributes(m.readingSystemAttributes)
	if err != nil {
		return err
	}

	if save {
		if len(config.Conf.Services) == 0 {
			config.Conf.Services = append(config.Conf.Services, config.Service{})
		}
		index := len(config.Conf.Services) - 1
		config.Conf.Services[index].Credentials.Username = username
		config.Conf.Services[index].Credentials.Password = password
		config.Conf.Services[index].URL = serviceURL
	}

	m.setQuestions(daisy.UserResponse{QuestionID: daisy.Default})
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
		text, err := gui.TextEntryDialog("Ввод текста", inputQuestion.Label.Text)
		if err != nil {
			log.Printf("userText: %s", err)
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

func (m *Manager) playBook(index int) {
	book := m.books.ContentItems[index]
	playedBook := m.bookplayer.Book()
	if playedBook != nil && playedBook.ID == book.ID {
		m.bookplayer.Pause()
		return
	}
	m.bookplayer.Stop()

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	m.bookplayer = player.NewPlayer(book, r.Resources)
	fragment, elapsedTime := config.Conf.Services[len(config.Conf.Services)-1].RecentBooks.GetPosition(book.ID)
	m.bookplayer.Play(fragment, elapsedTime)
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
	go util.DownloadBook(config.Conf.UserData, book.Label.Text, r)
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

func (m *Manager) showBookProperties(index int) {
	book := m.books.ContentItems[index]
	md, err := m.client.GetContentMetadata(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentMetadata: %v", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	var textList []string
	if len(md.Metadata.Description) != 0 {
		text := fmt.Sprintf("%v", strings.Join(md.Metadata.Description, " "))
		textList = append(textList, text)
	}

	if len(textList) == 0 {
		gui.MessageBox("Ошибка", "Нет доступной информации о книге", gui.MsgBoxOK|gui.MsgBoxIconError)
		return
	}

	gui.MessageBox("Информация о книге", strings.Join(textList, "\r\n"), gui.MsgBoxOK|gui.MsgBoxIconWarning)
}
