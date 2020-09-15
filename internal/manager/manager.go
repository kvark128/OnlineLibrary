package manager

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/kvark128/av3715/internal/config"
	"github.com/kvark128/av3715/internal/connect"
	"github.com/kvark128/av3715/internal/events"
	"github.com/kvark128/av3715/internal/gui"
	"github.com/kvark128/av3715/internal/player"
	daisy "github.com/kvark128/daisyonline"
)

type Manager struct {
	sync.WaitGroup
	client                  *daisy.Client
	readingSystemAttributes *daisy.ReadingSystemAttributes
	bookplayer              *player.Player
	books                   *daisy.ContentList
	questions               *daisy.Questions
	userResponses           []daisy.UserResponse
}

func NewManager(client *daisy.Client, readingSystemAttributes *daisy.ReadingSystemAttributes) *Manager {
	return &Manager{
		client:                  client,
		readingSystemAttributes: readingSystemAttributes,
	}
}

func (m *Manager) Listen(eventCH chan events.Event) {
	m.Add(1)
	defer m.Done()

	for evt := range eventCH {
		switch evt {

		case events.ACTIVATE_MENU:
			index := gui.CurrentListBoxIndex()
			if m.books != nil {
				m.PlayBook(index)
			} else if m.questions != nil {
				questionIndex := len(m.userResponses)
				questionID := m.questions.MultipleChoiceQuestion[questionIndex].ID
				value := m.questions.MultipleChoiceQuestion[questionIndex].Choices.Choice[index].ID
				m.userResponses = append(m.userResponses, daisy.UserResponse{QuestionID: questionID, Value: value})
				questionIndex++
				if questionIndex < len(m.questions.MultipleChoiceQuestion) {
					m.SetMultipleChoiceQuestion(questionIndex)
					break
				}
				m.SetInputQuestion()
			}

		case events.OPEN_BOOKSHELF:
			m.SetContent(daisy.Issued)

		case events.MAIN_MENU:
			m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Default})

		case events.SEARCH_BOOK:
			m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Search})

		case events.MENU_BACK:
			m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Back})

		case events.LIBRARY_LOGON:
			username := config.Conf.Credentials.Username
			password := config.Conf.Credentials.Password
			var err error
			var save bool

			if username == "" || password == "" {
				username, password, save, err = gui.Credentials()
				if err != nil {
					log.Printf("Credentials: %s", err)
					break
				}
			}

			_, err = daisy.Authentication(m.client, m.readingSystemAttributes, username, password)
			if err != nil {
				msg := fmt.Sprintf("Authorization: %s", err)
				log.Printf(msg)
				gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
				eventCH <- events.LIBRARY_LOGON
				break
			}

			if save {
				config.Conf.Credentials.Username = username
				config.Conf.Credentials.Password = password
			}

			m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Default})

		case events.LIBRARY_LOGOFF:
			ok, err := m.client.LogOff()
			log.Printf("LogOff: %v, error: %v", ok, err)
			m.books = nil
			m.questions = nil
			config.Conf.Credentials.Username = ""
			config.Conf.Credentials.Password = ""
			gui.SetListBoxItems([]string{}, "")
			eventCH <- events.LIBRARY_LOGON

		case events.ISSUE_BOOK:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.IssueBook(index)
			}

		case events.REMOVE_BOOK:
			if m.books != nil {
				index := gui.CurrentListBoxIndex()
				m.RemoveBook(index)
			}

		case events.PLAYER_PAUSE:
			if m.bookplayer != nil {
				m.bookplayer.Pause()
			}

		case events.PLAYER_STOP:
			if m.bookplayer != nil {
				m.bookplayer.Stop()
			}

		case events.PLAYER_NEXT_TRACK:
			if m.bookplayer != nil {
				m.bookplayer.ChangeTrack(+1)
			}

		case events.PLAYER_PREVIOUS_TRACK:
			if m.bookplayer != nil {
				m.bookplayer.ChangeTrack(-1)
			}

		case events.DOWNLOAD_BOOK:
			m.DownloadBook(gui.CurrentListBoxIndex())

		case events.PLAYER_VOLUME_UP:
			if m.bookplayer != nil {
				m.bookplayer.ChangeVolume(+1)
			}

		case events.PLAYER_VOLUME_DOWN:
			if m.bookplayer != nil {
				m.bookplayer.ChangeVolume(-1)
			}

		default:
			log.Printf("Unknown event: %v\n", evt)

		}
	}
}

func (m *Manager) SetQuestions(response ...daisy.UserResponse) {
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
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	if questions.Label.Text != "" {
		gui.MessageBox("Предупреждение", questions.Label.Text, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		// Return to the main menu of the library
		m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Default})
		return
	}

	if questions.ContentListRef != "" {
		m.SetContent(questions.ContentListRef)
		return
	}

	m.questions = questions
	m.userResponses = make([]daisy.UserResponse, 0)

	if len(m.questions.MultipleChoiceQuestion) > 0 {
		m.SetMultipleChoiceQuestion(0)
		return
	}

	m.SetInputQuestion()
}

func (m *Manager) SetMultipleChoiceQuestion(index int) {
	choiceQuestion := m.questions.MultipleChoiceQuestion[index]
	m.books = nil

	var items []string
	for _, c := range choiceQuestion.Choices.Choice {
		items = append(items, c.Label.Text)
	}

	gui.SetListBoxItems(items, choiceQuestion.Label.Text)
}

func (m *Manager) SetInputQuestion() {
	for _, inputQuestion := range m.questions.InputQuestion {
		text, err := gui.TextEntryDialog("Ввод текста", inputQuestion.Label.Text)
		if err != nil {
			log.Printf("userText: %s", err)
			// Return to the main menu of the library
			m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Default})
			return
		}
		m.userResponses = append(m.userResponses, daisy.UserResponse{QuestionID: inputQuestion.ID, Value: text})
	}

	m.SetQuestions(m.userResponses...)
}

func (m *Manager) SetContent(contentID string) {
	log.Printf("Content set: %s", contentID)
	contentList, err := m.client.GetContentList(contentID, 0, -1)
	if err != nil {
		msg := fmt.Sprintf("GetContentList: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	if len(contentList.ContentItems) == 0 {
		gui.MessageBox("Предупреждение", "Список книг пуст", gui.MsgBoxOK|gui.MsgBoxIconWarning)
		// Return to the main menu of the library
		m.SetQuestions(daisy.UserResponse{QuestionID: daisy.Default})
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

func (m *Manager) PlayBook(index int) {
	if m.bookplayer != nil {
		m.bookplayer.Stop()
	}
	book := m.books.ContentItems[index]

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	m.bookplayer = player.NewPlayer(book.Label.Text, r)
	m.bookplayer.Play(0)
}

func (m *Manager) DownloadBook(index int) {
	book := m.books.ContentItems[index]

	r, err := m.client.GetContentResources(book.ID)
	if err != nil {
		msg := fmt.Sprintf("GetContentResources: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	go DownloadResources(book.Label.Text, r)
}

func (m *Manager) RemoveBook(index int) {
	book := m.books.ContentItems[index]
	_, err := m.client.ReturnContent(book.ID)
	if err != nil {
		msg := fmt.Sprintf("ReturnContent: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s удалена с книжной полки", book.Label.Text), gui.MsgBoxOK|gui.MsgBoxIconWarning)
}

func (m *Manager) IssueBook(index int) {
	book := m.books.ContentItems[index]
	_, err := m.client.IssueContent(book.ID)
	if err != nil {
		msg := fmt.Sprintf("IssueContent: %s", err)
		log.Printf(msg)
		gui.MessageBox("Ошибка", msg, gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}
	gui.MessageBox("Уведомление", fmt.Sprintf("%s добавлена на книжную полку", book.Label.Text), gui.MsgBoxOK|gui.MsgBoxIconWarning)
}

func DownloadResources(book string, r *daisy.Resources) {
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

	dlg := gui.NewProgressDialog("Загрузка книги", fmt.Sprintf("Загрузка %s", book), len(r.Resources), cancelFN)
	dlg.Show()

	for _, v := range r.Resources {
		path := filepath.Join(config.Conf.UserData, book, v.LocalURI)
		if info, e := os.Stat(path); e == nil {
			if !info.IsDir() && info.Size() == int64(v.Size) {
				// v.LocalURI already exist
				dlg.IncreaseValue(1)
				continue
			}
		}

		conn, err = connect.NewConnection(v.URI)
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
		gui.MessageBox("Предупреждение", "Загрузка отменена пользователем", gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	if err != nil {
		gui.MessageBox("Ошибка", err.Error(), gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}
	gui.MessageBox("Уведомление", "Книга успешно загружена", gui.MsgBoxOK|gui.MsgBoxIconWarning)
}
