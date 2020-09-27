package gui

import (
	"errors"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const (
	MsgBoxOK              = walk.MsgBoxOK
	MsgBoxIconInformation = walk.MsgBoxIconInformation
	MsgBoxIconError       = walk.MsgBoxIconError
	MsgBoxIconWarning     = walk.MsgBoxIconWarning
	MsgBoxIconExclamation = walk.MsgBoxIconExclamation
	MsgBoxIconAsterisk    = walk.MsgBoxIconAsterisk
	MsgBoxUserIcon        = walk.MsgBoxUserIcon
)

var (
	mainWindow  *walk.MainWindow
	statusBar   *walk.StatusBar
	textLabel   *walk.TextLabel
	listBox     *walk.ListBox
	libraryMenu *walk.Menu
)

func Initialize(eventCH chan events.Event) (err error) {
	if mainWindow != nil {
		panic("GUI already initialized")
	}

	mainWindow, err = CreateWND(eventCH)
	if err != nil {
		return err
	}

	statusBar, err = walk.NewStatusBar(mainWindow)
	if err != nil {
		return err
	}
	statusBar.SetVisible(true)

	textLabel, err = walk.NewTextLabel(mainWindow)
	if err != nil {
		return err
	}

	listBox, err = walk.NewListBox(mainWindow)
	if err != nil {
		return err
	}

	listBox.ItemActivated().Attach(func() { eventCH <- events.ACTIVATE_MENU })
	listBox.KeyPress().Attach(func(key walk.Key) {
		if key == walk.KeySpace {
			eventCH <- events.ACTIVATE_MENU
		}
	})
	return nil
}

func UpdateLibraryMenu(eventCH chan events.Event) {
	mainWindow.Synchronize(func() {
		actions := libraryMenu.Actions()
		// Delete all elements except the last one
		for i := actions.Len(); i > 1; i-- {
			actions.RemoveAt(0)
		}

		// Filling the menu with services from the config
		for i, service := range config.Conf.Services {
			index := i
			a := walk.NewAction()
			if i == 0 {
				a.SetChecked(true)
			}
			a.SetText(service.Name)
			a.Triggered().Attach(func() {
				eventCH <- events.LIBRARY_LOGOFF
				config.SetMainLibrary(index)
				eventCH <- events.LIBRARY_LOGON
			})
			actions.Insert(i, a)
		}
	})
}

func RunMainWindow() {
	mainWindow.Run()
}

func SetMainWindowTitle(title string) {
	mainWindow.Synchronize(func() {
		mainWindow.SetTitle(title)
	})
}

func SetListBoxItems(items []string, label string) {
	mainWindow.Synchronize(func() {
		textLabel.SetText(label)
		listBox.Accessibility().SetName(label)
		listBox.SetModel(items)
		listBox.SetCurrentIndex(0)
	})
}

func CurrentListBoxIndex() int {
	ic := make(chan int)
	mainWindow.Synchronize(func() {
		ic <- listBox.CurrentIndex()
	})
	return <-ic
}

func Credentials() (name, url, username, password string, err error) {
	var (
		dlg                                   *walk.Dialog
		nameLE, urlLE, usernameLE, passwordLE *walk.LineEdit
		OkPB, CancelPB                        *walk.PushButton
	)

	Dialog{
		Title:         "Добавление новой учётной записи",
		AssignTo:      &dlg,
		Layout:        VBox{},
		CancelButton:  &CancelPB,
		DefaultButton: &OkPB,
		Children: []Widget{
			TextLabel{Text: "Отображаемое имя:"},
			LineEdit{
				Accessibility: Accessibility{Name: "Отображаемое имя:"},
				AssignTo:      &nameLE,
			},

			TextLabel{Text: "Адрес сервера:"},
			LineEdit{
				Accessibility: Accessibility{Name: "Адрес сервера:"},
				AssignTo:      &urlLE,
				Text:          "https://",
			},

			TextLabel{Text: "Имя пользователя:"},
			LineEdit{
				Accessibility: Accessibility{Name: "Имя пользователя:"},
				AssignTo:      &usernameLE,
			},

			TextLabel{Text: "Пароль:"},
			LineEdit{
				Accessibility: Accessibility{Name: "Пароль:"},
				AssignTo:      &passwordLE,
				PasswordMode:  true,
			},

			PushButton{
				AssignTo: &OkPB,
				Text:     "OK",
				OnClicked: func() {
					name = nameLE.Text()
					url = urlLE.Text()
					username = usernameLE.Text()
					password = passwordLE.Text()
					if name == "" || url == "" || username == "" || password == "" {
						err = errors.New("There is an empty field")
						return
					}
					dlg.Cancel()
				},
			},

			PushButton{
				AssignTo: &CancelPB,
				Text:     "Отмена",
				OnClicked: func() {
					err = errors.New("Cancel")
					dlg.Cancel()
				},
			},
		},
	}.Run(mainWindow)

	return
}

func TextEntryDialog(title, msg string) (text string, err error) {
	var (
		dlg            *walk.Dialog
		textLE         *walk.LineEdit
		OkPB, CancelPB *walk.PushButton
	)

	var layout = Dialog{
		Title:         title,
		AssignTo:      &dlg,
		Layout:        VBox{},
		CancelButton:  &CancelPB,
		DefaultButton: &OkPB,
		Accessibility: Accessibility{Role: AccRoleClock},
		Children: []Widget{

			TextLabel{Text: msg},
			LineEdit{
				Accessibility: Accessibility{Name: msg},
				AssignTo:      &textLE,
			},

			PushButton{
				AssignTo: &OkPB,
				Text:     "OK",
				OnClicked: func() {
					text = textLE.Text()
					if text == "" {
						err = errors.New("Text is empty")
					}
					dlg.Cancel()
				},
			},

			PushButton{
				AssignTo: &CancelPB,
				Text:     "Отмена",
				OnClicked: func() {
					err = errors.New("Cancel")
					dlg.Cancel()
				},
			},
		},
	}

	done := make(chan bool)
	mainWindow.Synchronize(func() {
		layout.Run(mainWindow)
		done <- true
	})

	<-done
	return
}

func MessageBox(title, text string, style walk.MsgBoxStyle) {
	done := make(chan bool)
	mainWindow.Synchronize(func() {
		walk.MsgBox(mainWindow, title, text, style)
		done <- true
	})
	<-done
}
