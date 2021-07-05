package gui

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/msg"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	mainWindow                       *walk.MainWindow
	MainList                         *MainListBox
	libraryMenu                      *walk.Menu
	outputDeviceMenu                 *walk.Menu
	BookMenu                         *walk.Menu
	logLevelMenu                     *walk.Menu
	elapseTime, totalTime, fragments *walk.StatusBarItem
	pauseTimerItem                   *walk.Action
)

func Initialize(msgCH chan msg.Message) error {
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
								OnTriggered: func() { msgCH <- msg.Message{msg.LIBRARY_ADD, nil} },
							},
						},
					},
					Action{
						Text:        "Главное меню",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyM},
						OnTriggered: func() { msgCH <- msg.Message{msg.MAIN_MENU, nil} },
					},
					Action{
						Text:        "Книжная полка",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						OnTriggered: func() { msgCH <- msg.Message{msg.OPEN_BOOKSHELF, nil} },
					},
					Action{
						Text:        "Поиск...",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyF},
						OnTriggered: func() { msgCH <- msg.Message{msg.SEARCH_BOOK, nil} },
					},
					Action{
						Text:        "Предыдущее меню",
						Shortcut:    Shortcut{0, walk.KeyBack},
						OnTriggered: func() { msgCH <- msg.Message{msg.MENU_BACK, nil} },
					},
					Action{
						Text:        "Удалить учётную запись",
						OnTriggered: func() { msgCH <- msg.Message{msg.LIBRARY_REMOVE, nil} },
					},
					Action{
						Text:        "Локальные книги",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyB},
						OnTriggered: func() { msgCH <- msg.Message{msg.OPEN_LOCALBOOKS, nil} },
					},
					Action{
						Text:        "Выйти из программы",
						Shortcut:    Shortcut{walk.ModAlt, walk.KeyF4},
						OnTriggered: func() { mainWindow.Close() },
					},
				},
			},

			Menu{
				Text:     "&Книга",
				AssignTo: &BookMenu,
				Items: []MenuItem{
					Action{
						Text:        "Загрузить книгу",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyD},
						OnTriggered: func() { msgCH <- msg.Message{msg.DOWNLOAD_BOOK, nil} },
					},
					Action{
						Text:        "Убрать книгу с полки",
						Shortcut:    Shortcut{walk.ModShift, walk.KeyDelete},
						OnTriggered: func() { msgCH <- msg.Message{msg.REMOVE_BOOK, nil} },
					},
					Action{
						Text:        "Поставить книгу на полку",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyA},
						OnTriggered: func() { msgCH <- msg.Message{msg.ISSUE_BOOK, nil} },
					},
					Action{
						Text:        "Информация о книге",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyI},
						OnTriggered: func() { msgCH <- msg.Message{msg.BOOK_DESCRIPTION, nil} },
					},
				},
			},

			Menu{
				Text: "&Воспроизведение",
				Items: []MenuItem{
					Action{
						Text:        "Воспроизвести / Приостановить",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyK},
						OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_PLAY_PAUSE, nil} },
					},
					Action{
						Text:        "Остановить",
						Shortcut:    Shortcut{walk.ModControl, walk.KeySpace},
						OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_STOP, nil} },
					},
					Action{
						Text:        "У&величить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyUp},
						OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_VOLUME_UP, nil} },
					},
					Action{
						Text:        "У&меньшить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyDown},
						OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_VOLUME_DOWN, nil} },
					},
					Menu{
						Text: "Переход по книге",
						Items: []MenuItem{
							Action{
								Text:        "На первый фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyBack},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_GOTO_FRAGMENT, 0} },
							},
							Action{
								Text:        "На указанный фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyG},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_GOTO_FRAGMENT, nil} },
							},
							Action{
								Text:        "На следующий фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyL},
								OnTriggered: func() { msgCH <- next_fragment },
							},
							Action{
								Text:        "На предыдущий фрагмент",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyJ},
								OnTriggered: func() { msgCH <- previous_fragment },
							},
						},
					},
					Menu{
						Text: "Переход по фрагменту",
						Items: []MenuItem{
							Action{
								Text:        "На указанную позицию",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyG},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_GOTO_POSITION, nil} },
							},
							Action{
								Text:        "На 5 сек. вперёд",
								Shortcut:    Shortcut{0, walk.KeyRight},
								OnTriggered: func() { msgCH <- rewind_5sec_forward },
							},
							Action{
								Text:        "На 5 сек. назад",
								Shortcut:    Shortcut{0, walk.KeyLeft},
								OnTriggered: func() { msgCH <- rewind_5sec_back },
							},
							Action{
								Text:        "На 30 сек. вперёд",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyRight},
								OnTriggered: func() { msgCH <- rewind_30sec_forward },
							},
							Action{
								Text:        "На 30 сек. назад",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyLeft},
								OnTriggered: func() { msgCH <- rewind_30sec_back },
							},
							Action{
								Text:        "На 1 мин. вперёд",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyRight},
								OnTriggered: func() { msgCH <- rewind_1min_forward },
							},
							Action{
								Text:        "На 1 мин. назад",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyLeft},
								OnTriggered: func() { msgCH <- rewind_1min_back },
							},
							Action{
								Text:        "На 5 мин. вперёд",
								Shortcut:    Shortcut{walk.ModControl | walk.ModShift, walk.KeyRight},
								OnTriggered: func() { msgCH <- rewind_5min_forward },
							},
							Action{
								Text:        "На 5 мин. назад",
								Shortcut:    Shortcut{walk.ModControl | walk.ModShift, walk.KeyLeft},
								OnTriggered: func() { msgCH <- rewind_5min_back },
							},
						},
					},
					Menu{
						Text: "Скорость",
						Items: []MenuItem{
							Action{
								Text:        "Увеличить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyC},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_SPEED_UP, nil} },
							},
							Action{
								Text:        "Уменьшить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyX},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_SPEED_DOWN, nil} },
							},
							Action{
								Text:        "Сбросить скорость",
								Shortcut:    Shortcut{walk.ModControl, walk.KeyZ},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_SPEED_RESET, nil} },
							},
						},
					},
					Menu{
						Text: "Высота",
						Items: []MenuItem{
							Action{
								Text:        "Увеличить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyC},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_PITCH_UP, nil} },
							},
							Action{
								Text:        "Уменьшить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyX},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_PITCH_DOWN, nil} },
							},
							Action{
								Text:        "Сбросить высоту",
								Shortcut:    Shortcut{walk.ModShift, walk.KeyZ},
								OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_PITCH_RESET, nil} },
							},
						},
					},
				},
			},
			Menu{
				Text: "&Настройки",
				Items: []MenuItem{
					Menu{
						Text:     "Устройство вывода звука",
						AssignTo: &outputDeviceMenu,
					},
					Action{
						Text:        "Таймер паузы",
						AssignTo:    &pauseTimerItem,
						Shortcut:    Shortcut{walk.ModControl, walk.KeyP},
						OnTriggered: func() { msgCH <- msg.Message{msg.PLAYER_SET_TIMER, nil} },
					},
					Menu{
						Text:     "Уровень ведения журнала",
						AssignTo: &logLevelMenu,
					},
				},
			},
			Menu{
				Text: "&Справка",
				Items: []MenuItem{
					Action{
						Text: "О программе",
						OnTriggered: func() {
							text := fmt.Sprintf("%v версия %v\nКаталог приложения: %s", config.ProgramName, config.ProgramVersion, config.UserData())
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
				OnItemActivated: func() { msgCH <- msg.Message{msg.ACTIVATE_MENU, nil} },
			},
		},
		StatusBarItems: []StatusBarItem{
			StatusBarItem{
				AssignTo: &elapseTime,
				Text:     "00:00:00",
			},
			StatusBarItem{
				Text: "/",
			},
			StatusBarItem{
				AssignTo: &totalTime,
				Text:     "00:00:00",
			},
			StatusBarItem{
				AssignTo: &fragments,
			},
		},
	}.Create()); err != nil {
		return err
	}

	logLevels := []log.Level{log.ErrorLevel, log.InfoLevel, log.WarningLevel, log.DebugLevel}
	logActions := logLevelMenu.Actions()
	currentLogLevel := log.GetLevel()
	for _, level := range logLevels {
		level := level // Avoid capturing the iteration variable
		a := walk.NewAction()
		a.SetText(level.String())
		if level == currentLogLevel {
			a.SetChecked(true)
		}
		a.Triggered().Attach(func() {
			actions := logLevelMenu.Actions()
			for i := 0; i < actions.Len(); i++ {
				actions.At(i).SetChecked(false)
			}
			a.SetChecked(true)
			msgCH <- msg.Message{Code: msg.LOG_SET_LEVEL, Data: level}
		})
		logActions.Add(a)
	}

	var err error
	MainList, err = NewMainListBox(lb, label, msgCH)
	if err != nil {
		return err
	}

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

func SetLibraryMenu(msgCH chan msg.Message, services []*config.Service, current string) {
	mainWindow.Synchronize(func() {
		actions := libraryMenu.Actions()
		// Delete all elements except the last one
		for i := actions.Len(); i > 1; i-- {
			actions.RemoveAt(0)
		}

		// Filling the menu with services
		for i, service := range services {
			a := walk.NewAction()
			if service.Name == current {
				a.SetChecked(true)
			}
			a.SetText(service.Name)
			a.Triggered().Attach(func() {
				for index := 0; index < actions.Len(); index++ {
					actions.At(index).SetChecked(false)
				}
				a.SetChecked(true)
				msgCH <- msg.Message{msg.LIBRARY_LOGON, a.Text()}
			})
			actions.Insert(i, a)
		}
	})
}

func SetOutputDeviceMenu(msgCH chan msg.Message, deviceNames []string, current string) {
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
				msgCH <- msg.Message{msg.PLAYER_OUTPUT_DEVICE, a.Text()}
			})
			actions.Insert(i, a)
		}
	})
}

func SetPauseTimerLabel(minutes int) {
	label := "Таймер паузы (Нет)"
	if minutes > 0 {
		label = fmt.Sprintf("Таймер паузы (%d мин.)", minutes)
	}
	mainWindow.Synchronize(func() {
		pauseTimerItem.SetText(label)
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

func TextEntryDialog(title, msg, value string, text *string) int {
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
				Text:          value,
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
