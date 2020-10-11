package gui

import (
	"errors"
	"fmt"
	"time"

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
	mainWindow                       *walk.MainWindow
	textLabel                        *walk.TextLabel
	listBox                          *walk.ListBox
	libraryMenu                      *walk.Menu
	elapseTime, totalTime, fragments *walk.StatusBarItem
)

func Initialize(eventCH chan events.Event) error {
	if mainWindow != nil {
		panic("GUI already initialized")
	}

	if err := (MainWindow{
		Title:    config.ProgramName,
		Layout:   VBox{},
		AssignTo: &mainWindow,
		MenuItems: []MenuItem{

			Menu{
				Text: "&Библиотека",
				Items: []MenuItem{
					Menu{
						Text:     "Учётные записи",
						AssignTo: &libraryMenu,
						Items: []MenuItem{
							Action{
								Text:        "Добавить учётную запись",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyN},
								OnTriggered: func() { eventCH <- events.LIBRARY_ADD },
							},
						},
					},
					Action{
						Text:        "Главное меню",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyM},
						OnTriggered: func() { eventCH <- events.MAIN_MENU },
					},
					Action{
						Text:        "Книжная полка",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						OnTriggered: func() { eventCH <- events.OPEN_BOOKSHELF },
					},
					Action{
						Text:        "Поиск...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyF},
						OnTriggered: func() { eventCH <- events.SEARCH_BOOK },
					},
					Action{
						Text:        "Предыдущее меню",
						Shortcut:    Shortcut{0, walk.KeyBack},
						OnTriggered: func() { eventCH <- events.MENU_BACK },
					},
					Action{
						Text:        "Удалить учётную запись",
						OnTriggered: func() { eventCH <- events.LIBRARY_REMOVE },
					},
					Action{
						Text:        "Выйти из программы",
						Shortcut:    Shortcut{walk.ModAlt, walk.KeyF4},
						OnTriggered: func() { mainWindow.Close() },
					},
				},
			},

			Menu{
				Text: "&Книги",
				Items: []MenuItem{
					Action{
						Text:        "Загрузить книгу",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyD},
						OnTriggered: func() { eventCH <- events.DOWNLOAD_BOOK },
					},
					Action{
						Text:        "Убрать книгу с полки",
						Shortcut:    Shortcut{walk.ModShift, walk.KeyDelete},
						OnTriggered: func() { eventCH <- events.REMOVE_BOOK },
					},
					Action{
						Text:        "Поставить книгу на полку",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyA},
						OnTriggered: func() { eventCH <- events.ISSUE_BOOK },
					},
					Action{
						Text:        "Информация о книге",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyI},
						OnTriggered: func() { eventCH <- events.BOOK_DESCRIPTION },
					},
				},
			},

			Menu{
				Text: "&Воспроизведение",
				Items: []MenuItem{
					Action{
						Text:        "Первый трек",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyBack},
						OnTriggered: func() { eventCH <- events.PLAYER_FIRST },
					},
					Action{
						Text:        "Перемотка вперёд",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyOEMPeriod},
						OnTriggered: func() { eventCH <- events.PLAYER_FORWARD },
					},
					Action{
						Text:        "Перемотка назад",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyOEMComma},
						OnTriggered: func() { eventCH <- events.PLAYER_BACK },
					},
					Action{
						Text:        "У&величить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyB},
						OnTriggered: func() { eventCH <- events.PLAYER_VOLUME_UP },
					},
					Action{
						Text:        "У&меньшить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyV},
						OnTriggered: func() { eventCH <- events.PLAYER_VOLUME_DOWN },
					},
					Action{
						Text:        "Следующий фрагмент",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyL},
						OnTriggered: func() { eventCH <- events.PLAYER_NEXT_TRACK },
					},
					Action{
						Text:        "Предыдущий фрагмент",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyJ},
						OnTriggered: func() { eventCH <- events.PLAYER_PREVIOUS_TRACK },
					},
					Action{
						Text:        "Воспроизвести / Приостановить",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyK},
						OnTriggered: func() { eventCH <- events.PLAYER_PLAY_PAUSE },
					},
					Menu{
						Text: "Скорость",
						Items: []MenuItem{
							Action{
								Text:        "Увеличить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyC},
								OnTriggered: func() { eventCH <- events.PLAYER_SPEED_UP },
							},
							Action{
								Text:        "Уменьшить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyX},
								OnTriggered: func() { eventCH <- events.PLAYER_SPEED_DOWN },
							},
							Action{
								Text:        "Сбросить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyZ},
								OnTriggered: func() { eventCH <- events.PLAYER_SPEED_RESET },
							},
						},
					},
				},
			},
			Menu{
				Text: "&Справка",
				Items: []MenuItem{
					Action{
						Text: "О программе",
						OnTriggered: func() {
							text := fmt.Sprintf("%v версия %v", config.ProgramName, config.ProgramVersion)
							walk.MsgBox(mainWindow, "О программе", text, MsgBoxOK|MsgBoxIconInformation)
						},
					},
				},
			},
		},

		Children: []Widget{
			TextLabel{
				AssignTo: &textLabel,
			},
			ListBox{
				AssignTo:        &listBox,
				OnItemActivated: func() { eventCH <- events.ACTIVATE_MENU },
				OnKeyPress: func(key walk.Key) {
					if key == walk.KeySpace {
						eventCH <- events.PLAYER_PLAY_PAUSE
					}
				},
			},
		},
	}.Create()); err != nil {
		return err
	}

	statusBar, err := walk.NewStatusBar(mainWindow)
	if err != nil {
		return err
	}

	elapseTime = walk.NewStatusBarItem()
	elapseTime.SetText("00:00:00")
	totalTime = walk.NewStatusBarItem()
	totalTime.SetText("00:00:00")
	separator := walk.NewStatusBarItem()
	separator.SetText("/")
	fragments = walk.NewStatusBarItem()

	statusBar.Items().Add(elapseTime)
	statusBar.Items().Add(separator)
	statusBar.Items().Add(totalTime)
	statusBar.Items().Add(fragments)
	statusBar.SetVisible(true)
	return nil
}

func SetElapsedTime(elapse time.Duration) {
	mainWindow.Synchronize(func() {
		h := (elapse / time.Hour).Hours()
		m := (elapse % time.Hour).Minutes()
		s := (elapse % time.Minute).Seconds()
		text := fmt.Sprintf("%02d:%02d:%02d", int(h), int(m), int(s))
		elapseTime.SetText(text)
	})
}

func SetTotalTime(total time.Duration) {
	mainWindow.Synchronize(func() {
		h := (total / time.Hour).Hours()
		m := (total % time.Hour).Minutes()
		s := (total % time.Minute).Seconds()
		text := fmt.Sprintf("%02d:%02d:%02d", int(h), int(m), int(s))
		totalTime.SetText(text)
	})
}

func SetFragments(current, length int) {
	mainWindow.Synchronize(func() {
		text := fmt.Sprintf("Фрагмент %d из %d", current+1, length)
		fragments.SetText(text)
	})
}

func SetLibraryMenu(eventCH chan events.Event, services []config.Service, current int) {
	mainWindow.Synchronize(func() {
		actions := libraryMenu.Actions()
		// Delete all elements except the last one
		for i := actions.Len(); i > 1; i-- {
			actions.RemoveAt(0)
		}

		// Filling the menu with services
		for i, service := range services {
			a := walk.NewAction()
			if i == current {
				a.SetChecked(true)
			}
			a.SetText(service.Name)
			a.Triggered().Attach(func() {
				for index := 0; index < actions.Len(); index++ {
					actions.At(index).SetChecked(false)
				}
				a.SetChecked(true)
				eventCH <- events.LIBRARY_SWITCH
			})
			actions.Insert(i, a)
		}
	})
}

func GetCurrentLibrary() int {
	result := make(chan int)
	mainWindow.Synchronize(func() {
		actions := libraryMenu.Actions()
		for i := 0; i < actions.Len(); i++ {
			if actions.At(i).Checked() {
				result <- i
				return
			}
		}
		panic("no checked library")
	})
	return <-result
}

func RunMainWindow() {
	mainWindow.Run()
}

func SetMainWindowTitle(title string) {
	mainWindow.Synchronize(func() {
		title := fmt.Sprintf("%s — %s", title, config.ProgramName)
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

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
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
		Children: []Widget{

			TextLabel{Text: msg},
			LineEdit{
				Accessibility: Accessibility{Name: msg},
				AssignTo:      &textLE,
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
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

func QuestionDialog(title, msg string) bool {
	var (
		dlg         *walk.Dialog
		yesPB, noPB *walk.PushButton
	)

	var layout = Dialog{
		Title:         title,
		AssignTo:      &dlg,
		Layout:        VBox{},
		CancelButton:  &noPB,
		DefaultButton: &yesPB,
		Children: []Widget{

			TextLabel{Text: msg},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &yesPB,
						Text:      "Да",
						OnClicked: func() { dlg.Accept() },
					},
					PushButton{
						AssignTo:  &noPB,
						Text:      "Нет",
						OnClicked: func() { dlg.Cancel() },
					},
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
	if dlg.Result() == walk.DlgCmdOK {
		return true
	}
	return false
}

func MessageBox(title, text string, style walk.MsgBoxStyle) {
	done := make(chan bool)
	mainWindow.Synchronize(func() {
		walk.MsgBox(mainWindow, title, text, style)
		done <- true
	})
	<-done
}
