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

var (
	mainWindow                       *walk.MainWindow
	MainList                         *MainListBox
	libraryMenu                      *walk.Menu
	outputDeviceMenu                 *walk.Menu
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
								OnTriggered: func() { eventCH <- events.Event{events.LIBRARY_ADD, nil} },
							},
						},
					},
					Action{
						Text:        "Главное меню",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyM},
						OnTriggered: func() { eventCH <- events.Event{events.MAIN_MENU, nil} },
					},
					Action{
						Text:        "Книжная полка",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						OnTriggered: func() { eventCH <- events.Event{events.OPEN_BOOKSHELF, nil} },
					},
					Action{
						Text:        "Поиск...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyF},
						OnTriggered: func() { eventCH <- events.Event{events.SEARCH_BOOK, nil} },
					},
					Action{
						Text:        "Предыдущее меню",
						Shortcut:    Shortcut{0, walk.KeyBack},
						OnTriggered: func() { eventCH <- events.Event{events.MENU_BACK, nil} },
					},
					Action{
						Text:        "Удалить учётную запись",
						OnTriggered: func() { eventCH <- events.Event{events.LIBRARY_REMOVE, nil} },
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
						OnTriggered: func() { eventCH <- events.Event{events.DOWNLOAD_BOOK, nil} },
					},
					Action{
						Text:        "Убрать книгу с полки",
						Shortcut:    Shortcut{walk.ModShift, walk.KeyDelete},
						OnTriggered: func() { eventCH <- events.Event{events.REMOVE_BOOK, nil} },
					},
					Action{
						Text:        "Поставить книгу на полку",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyA},
						OnTriggered: func() { eventCH <- events.Event{events.ISSUE_BOOK, nil} },
					},
					Action{
						Text:        "Информация о книге",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyI},
						OnTriggered: func() { eventCH <- events.Event{events.BOOK_DESCRIPTION, nil} },
					},
				},
			},

			Menu{
				Text: "&Воспроизведение",
				Items: []MenuItem{
					Action{
						Text:        "Воспроизвести / Приостановить",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyK},
						OnTriggered: func() { eventCH <- events.Event{events.PLAYER_PLAY_PAUSE, nil} },
					},
					Action{
						Text:        "Остановить",
						Shortcut:    Shortcut{walk.ModControl, walk.KeySpace},
						OnTriggered: func() { eventCH <- events.Event{events.PLAYER_STOP, nil} },
					},
					Action{
						Text:        "У&величить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyUp},
						OnTriggered: func() { eventCH <- events.Event{events.PLAYER_VOLUME_UP, nil} },
					},
					Action{
						Text:        "У&меньшить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyDown},
						OnTriggered: func() { eventCH <- events.Event{events.PLAYER_VOLUME_DOWN, nil} },
					},
					Menu{
						Text: "Переход по книге",
						Items: []MenuItem{
							Action{
								Text:        "На первый фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyBack},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_FIRST, nil} },
							},
							Action{
								Text:        "На указанный фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyG},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_GOTO, nil} },
							},
							Action{
								Text:        "На следующий фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyL},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_NEXT_TRACK, nil} },
							},
							Action{
								Text:        "На предыдущий фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyJ},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_PREVIOUS_TRACK, nil} },
							},
						},
					},
					Menu{
						Text: "Переход по фрагменту",
						Items: []MenuItem{
							Action{
								Text:        "На 5 сек. вперёд",
								Shortcut:    Shortcut{0, walk.KeyRight},
								OnTriggered: func() { eventCH <- rewind_5sec_forward },
							},
							Action{
								Text:        "На 5 сек. назад",
								Shortcut:    Shortcut{0, walk.KeyLeft},
								OnTriggered: func() { eventCH <- rewind_5sec_back },
							},
							Action{
								Text:        "На 30 сек. вперёд",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyRight},
								OnTriggered: func() { eventCH <- rewind_30sec_forward },
							},
							Action{
								Text:        "На 30 сек. назад",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyLeft},
								OnTriggered: func() { eventCH <- rewind_30sec_back },
							},
							Action{
								Text:        "На 1 мин. вперёд",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyRight},
								OnTriggered: func() { eventCH <- rewind_1min_forward },
							},
							Action{
								Text:        "На 1 мин. назад",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyLeft},
								OnTriggered: func() { eventCH <- rewind_1min_back },
							},
							Action{
								Text:        "На 5 мин. вперёд",
								Shortcut:    Shortcut{walk.ModControl | walk.ModShift, walk.KeyRight},
								OnTriggered: func() { eventCH <- rewind_5min_forward },
							},
							Action{
								Text:        "На 5 мин. назад",
								Shortcut:    Shortcut{walk.ModControl | walk.ModShift, walk.KeyLeft},
								OnTriggered: func() { eventCH <- rewind_5min_back },
							},
						},
					},
					Menu{
						Text: "Скорость",
						Items: []MenuItem{
							Action{
								Text:        "Увеличить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyC},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_SPEED_UP, nil} },
							},
							Action{
								Text:        "Уменьшить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyX},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_SPEED_DOWN, nil} },
							},
							Action{
								Text:        "Сбросить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyZ},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_SPEED_RESET, nil} },
							},
						},
					},
					Menu{
						Text: "Высота",
						Items: []MenuItem{
							Action{
								Text:        "Увеличить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyC},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_PITCH_UP, nil} },
							},
							Action{
								Text:        "Уменьшить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyX},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_PITCH_DOWN, nil} },
							},
							Action{
								Text:        "Сбросить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyZ},
								OnTriggered: func() { eventCH <- events.Event{events.PLAYER_PITCH_RESET, nil} },
							},
						},
					},
				},
			},
			Menu{
				Text: "&Настройки",
				Items: []MenuItem{
					Menu{
						Text:     "Устройство вывода",
						AssignTo: &outputDeviceMenu,
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
							walk.MsgBox(mainWindow, "О программе", text, walk.MsgBoxOK|walk.MsgBoxIconInformation)
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
				OnItemActivated: func() { eventCH <- events.Event{events.ACTIVATE_MENU, nil} },
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
				eventCH <- events.Event{events.LIBRARY_LOGON, actions.Index(a)}
			})
			actions.Insert(i, a)
		}
	})
}

func SetOutputDeviceMenu(eventCH chan events.Event, deviceNames []string, current string) {
	mainWindow.Synchronize(func() {
		actions := outputDeviceMenu.Actions()
		// Delete all elements
		actions.Clear()

		// Filling the menu with devices
		for i, name := range deviceNames {
			a := walk.NewAction()
			if name == current || (current == "" && i == 0) {
				a.SetChecked(true)
			}
			a.SetText(name)
			a.Triggered().Attach(func() {
				for index := 0; index < actions.Len(); index++ {
					actions.At(index).SetChecked(false)
				}
				a.SetChecked(true)
				eventCH <- events.Event{events.PLAYER_OUTPUT_DEVICE, a.Text()}
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
		layout.Create(mainWindow)
		NewFixedPushButton(OkPB)
		NewFixedPushButton(CancelPB)
		dlg.Run()
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
							dlg.Close(walk.DlgCmdOK)
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     "Отмена",
						OnClicked: func() {
							dlg.Close(walk.DlgCmdCancel)
						},
					},
				},
			},
		},
	}

	result := make(chan int)
	mainWindow.Synchronize(func() {
		layout.Create(mainWindow)
		NewFixedPushButton(OkPB)
		NewFixedPushButton(CancelPB)
		dlg.Run()
		result <- dlg.Result()
	})
	return <-result
}

func MessageBox(title, text string, style walk.MsgBoxStyle) int {
	result := make(chan int)
	mainWindow.Synchronize(func() {
		result <- walk.MsgBox(mainWindow, title, text, style)
	})
	return <-result
}
