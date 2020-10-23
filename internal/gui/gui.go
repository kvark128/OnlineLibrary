package gui

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

const (
	DlgCmdOK     = walk.DlgCmdOK
	DlgCmdCancel = walk.DlgCmdCancel
	DlgCmdYes    = walk.DlgCmdYes
	DlgCmdNo     = walk.DlgCmdNo
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
	MainList                         *MainListBox
	libraryMenu                      *walk.Menu
	elapseTime, totalTime, fragments *walk.StatusBarItem
)

func Initialize(eventCH chan events.Event) error {
	if mainWindow != nil {
		panic("GUI already initialized")
	}

	var lb *walk.ListBox
	var label *walk.TextLabel

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
					Menu{
						Text: "Переход",
						Items: []MenuItem{
							Action{
								Text:        "На первый фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyBack},
								OnTriggered: func() { eventCH <- events.PLAYER_FIRST },
							},
							Action{
								Text:        "На 5 сек. вперёд",
								Shortcut:    Shortcut{0, walk.KeyRight},
								OnTriggered: func() { eventCH <- events.PLAYER_FORWARD },
							},
							Action{
								Text:        "На 5 сек. назад",
								Shortcut:    Shortcut{0, walk.KeyLeft},
								OnTriggered: func() { eventCH <- events.PLAYER_BACK },
							},
							Action{
								Text:        "На 30 сек. вперёд",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyRight},
								OnTriggered: func() { eventCH <- events.PLAYER_LONG_FORWARD },
							},
							Action{
								Text:        "На 30 сек. назад",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyLeft},
								OnTriggered: func() { eventCH <- events.PLAYER_LONG_BACK },
							},
						},
					},
					Action{
						Text:        "У&величить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyUp},
						OnTriggered: func() { eventCH <- events.PLAYER_VOLUME_UP },
					},
					Action{
						Text:        "У&меньшить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyDown},
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
					Menu{
						Text: "Высота",
						Items: []MenuItem{
							Action{
								Text:        "Увеличить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyC},
								OnTriggered: func() { eventCH <- events.PLAYER_PITCH_UP },
							},
							Action{
								Text:        "Уменьшить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyX},
								OnTriggered: func() { eventCH <- events.PLAYER_PITCH_DOWN },
							},
							Action{
								Text:        "Сбросить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyZ},
								OnTriggered: func() { eventCH <- events.PLAYER_PITCH_RESET },
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
				AssignTo: &label,
			},
			ListBox{
				AssignTo:        &lb,
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

	var err error
	MainList, err = NewMainListBox(lb, label, eventCH)
	if err != nil {
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

func SetElapsedTime(elapsed time.Duration) {
	mainWindow.Synchronize(func() {
		str := util.FmtDuration(elapsed)
		elapseTime.SetText(str)
	})
}

func SetTotalTime(total time.Duration) {
	mainWindow.Synchronize(func() {
		str := util.FmtDuration(total)
		totalTime.SetText(str)
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
		var windowTitle = config.ProgramName
		if title != "" {
			windowTitle = fmt.Sprintf("%s — %s", title, windowTitle)
		}
		mainWindow.SetTitle(windowTitle)
	})
}

func Credentials(service *config.Service) int {
	var (
		dlg                                   *walk.Dialog
		nameLE, urlLE, usernameLE, passwordLE *walk.LineEdit
		OkPB, CancelPB                        *walk.PushButton
	)

	var layout = Dialog{
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
							service.Name = nameLE.Text()
							service.URL = urlLE.Text()
							service.Credentials.Username = usernameLE.Text()
							service.Credentials.Password = passwordLE.Text()
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     "Отмена",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}

	result := make(chan int)
	mainWindow.Synchronize(func() {
		layout.Run(mainWindow)
		result <- dlg.Result()
	})
	return <-result
}

func TextEntryDialog(title, msg string, text *string) int {
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
							*text = textLE.Text()
							dlg.Close(DlgCmdOK)
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     "Отмена",
						OnClicked: func() {
							dlg.Close(DlgCmdCancel)
						},
					},
				},
			},
		},
	}

	result := make(chan int)
	mainWindow.Synchronize(func() {
		layout.Run(mainWindow)
		result <- dlg.Result()
	})
	return <-result
}

func QuestionDialog(title, msg string) int {
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
						OnClicked: func() { dlg.Close(DlgCmdYes) },
					},
					PushButton{
						AssignTo:  &noPB,
						Text:      "Нет",
						OnClicked: func() { dlg.Close(DlgCmdNo) },
					},
				},
			},
		},
	}

	result := make(chan int)
	mainWindow.Synchronize(func() {
		layout.Run(mainWindow)
		result <- dlg.Result()
	})
	return <-result
}

func MessageBox(title, text string, style walk.MsgBoxStyle) {
	done := make(chan bool)
	mainWindow.Synchronize(func() {
		walk.MsgBox(mainWindow, title, text, style)
		done <- true
	})
	<-done
}
