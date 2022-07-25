package gui

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/leonelquinteros/gotext"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type MainWnd struct {
	mainWindow  *walk.MainWindow
	msgChan     chan msg.Message
	menuBar     *MenuBar
	mainListBox *MainListBox
	statusBar   *StatusBar
}

func NewMainWindow() (*MainWnd, error) {
	wnd := &MainWnd{
		msgChan:     make(chan msg.Message, config.MessageBufferSize),
		menuBar:     new(MenuBar),
		mainListBox: new(MainListBox),
		statusBar:   new(StatusBar),
	}

	wnd.menuBar.libraryLogon = walk.NewMutableCondition()
	MustRegisterCondition("libraryLogon", wnd.menuBar.libraryLogon)
	wnd.menuBar.bookMenuEnabled = walk.NewMutableCondition()
	MustRegisterCondition("bookMenuEnabled", wnd.menuBar.bookMenuEnabled)

	wndLayout := MainWindow{
		Title:    config.ProgramName,
		Layout:   VBox{},
		AssignTo: &wnd.mainWindow,

		MenuItems: []MenuItem{
			Menu{
				Text: gotext.Get("&Library"),
				Items: []MenuItem{
					Menu{
						Text:     gotext.Get("Accounts"),
						AssignTo: &wnd.menuBar.libraryMenu,
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Add account"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyN},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.LIBRARY_ADD, nil} },
							},
						},
					},
					Action{
						Text:        gotext.Get("Main menu"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyM},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.MAIN_MENU, nil} },
					},
					Action{
						Text:        gotext.Get("Bookshelf"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.OPEN_BOOKSHELF, nil} },
					},
					Action{
						Text:        gotext.Get("New receipts"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyK},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.OPEN_NEWBOOKS, nil} },
					},
					Action{
						Text:        gotext.Get("Search..."),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyF},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.SEARCH_BOOK, nil} },
					},
					Action{
						Text:        gotext.Get("Previous menu"),
						Shortcut:    Shortcut{0, walk.KeyBack},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.MENU_BACK, nil} },
					},
					Action{
						Text:        gotext.Get("Local books"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyL},
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.SET_PROVIDER, config.LocalStorageID} },
					},
					Action{
						Text:        gotext.Get("Library information"),
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.LIBRARY_INFO} },
					},
					Action{
						Text:        gotext.Get("Delete account"),
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.LIBRARY_REMOVE, nil} },
					},
					Action{
						Text:        gotext.Get("Exit"),
						Shortcut:    Shortcut{walk.ModAlt, walk.KeyF4},
						OnTriggered: func() { wnd.mainWindow.Close() },
					},
				},
			},

			Menu{
				Text:     gotext.Get("&Book"),
				AssignTo: &wnd.menuBar.bookMenu,
				Enabled:  Bind("bookMenuEnabled"),
				Items: []MenuItem{
					Action{
						Text:        gotext.Get("Download book"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyD},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.DOWNLOAD_BOOK, nil} },
					},
					Action{
						Text:        gotext.Get("Remove book from bookshelf"),
						Shortcut:    Shortcut{walk.ModShift, walk.KeyDelete},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.REMOVE_BOOK, nil} },
					},
					Action{
						Text:        gotext.Get("Put book on bookshelf"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyA},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.ISSUE_BOOK, nil} },
					},
					Action{
						Text:        gotext.Get("Book information"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeyI},
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.BOOK_DESCRIPTION, nil} },
					},
				},
			},

			Menu{
				Text: gotext.Get("&Playback"),
				Items: []MenuItem{
					Menu{
						Text:     gotext.Get("Bookmarks"),
						AssignTo: &wnd.menuBar.bookmarkMenu,
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Add bookmark"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyB},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.BOOKMARK_SET, nil} },
							},
						},
					},
					Action{
						Text:        gotext.Get("Play / Pause"),
						Shortcut:    Shortcut{Key: walk.KeySpace},
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_PLAY_PAUSE, nil} },
					},
					Action{
						Text:        gotext.Get("Stop"),
						Shortcut:    Shortcut{walk.ModControl, walk.KeySpace},
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_STOP, nil} },
					},

					Menu{
						Text: gotext.Get("Book navigation"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("First fragment"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyBack},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_GOTO_FRAGMENT, 0} },
							},
							Action{
								Text:        gotext.Get("Go to fragment"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyG},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_GOTO_FRAGMENT, nil} },
							},
							Action{
								Text:        gotext.Get("Next fragment"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyNext},
								OnTriggered: func() { wnd.msgChan <- next_fragment },
							},
							Action{
								Text:        gotext.Get("Previous fragment"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyPrior},
								OnTriggered: func() { wnd.msgChan <- previous_fragment },
							},
						},
					},

					Menu{
						Text: gotext.Get("Fragment navigation"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Beginning of the fragment"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyBack},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_GOTO_POSITION, time.Duration(0)} },
							},
							Action{
								Text:        gotext.Get("Go to position"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyG},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_GOTO_POSITION, nil} },
							},
							Action{
								Text:        gotext.Get("5 sec. forward"),
								Shortcut:    Shortcut{0, walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_5sec_forward },
							},
							Action{
								Text:        gotext.Get("5 sec. backward"),
								Shortcut:    Shortcut{0, walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_5sec_back },
							},
							Action{
								Text:        gotext.Get("30 sec. forward"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_30sec_forward },
							},
							Action{
								Text:        gotext.Get("30 sec. backward"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_30sec_back },
							},
							Action{
								Text:        gotext.Get("1 min. forward"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_1min_forward },
							},
							Action{
								Text:        gotext.Get("1 min. backward"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_1min_back },
							},
							Action{
								Text:        gotext.Get("5 min. forward"),
								Shortcut:    Shortcut{walk.ModControl | walk.ModShift, walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_5min_forward },
							},
							Action{
								Text:        gotext.Get("5 min. backward"),
								Shortcut:    Shortcut{walk.ModControl | walk.ModShift, walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_5min_back },
							},
						},
					},

					Menu{
						Text: gotext.Get("Volume"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Increase volume"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyUp},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_VOLUME_UP, nil} },
							},
							Action{
								Text:        gotext.Get("Decrease volume"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyDown},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_VOLUME_DOWN, nil} },
							},
							Action{
								Text:        gotext.Get("Reset volume"),
								Shortcut:    Shortcut{walk.ModControl, walk.KeyR},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_VOLUME_RESET, nil} },
							},
						},
					},

					Menu{
						Text: gotext.Get("Speed"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Increase speed"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyUp},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_SPEED_UP, nil} },
							},
							Action{
								Text:        gotext.Get("Decrease speed"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyDown},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_SPEED_DOWN, nil} },
							},
							Action{
								Text:        gotext.Get("Reset speed"),
								Shortcut:    Shortcut{walk.ModShift, walk.KeyR},
								OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_SPEED_RESET, nil} },
							},
						},
					},
				},
			},
			Menu{
				Text: gotext.Get("&Settings"),
				Items: []MenuItem{
					Menu{
						Text:     gotext.Get("Audio output device"),
						AssignTo: &wnd.menuBar.outputDeviceMenu,
					},
					Menu{
						Text:     gotext.Get("Language"),
						AssignTo: &wnd.menuBar.languageMenu,
					},
					Action{
						Text:        gotext.Get("Pause timer"),
						AssignTo:    &wnd.menuBar.pauseTimerItem,
						Shortcut:    Shortcut{walk.ModControl, walk.KeyP},
						OnTriggered: func() { wnd.msgChan <- msg.Message{msg.PLAYER_SET_TIMER, nil} },
					},
					Menu{
						Text:     gotext.Get("Logging level"),
						AssignTo: &wnd.menuBar.logLevelMenu,
					},
				},
			},
			Menu{
				Text: gotext.Get("&Help"),
				Items: []MenuItem{
					Action{
						Text: gotext.Get("About"),
						OnTriggered: func() {
							msg := gotext.Get("%v version %v\nWorking directory: %v\nAuthor: %v", config.ProgramName, config.ProgramVersion, config.UserData(), config.CopyrightInfo)
							walk.MsgBox(wnd.mainWindow, gotext.Get("About"), msg, walk.MsgBoxOK|walk.MsgBoxIconInformation)
						},
					},
				},
			},
		},

		Children: []Widget{
			TextLabel{
				AssignTo: &wnd.mainListBox.label,
			},
			ListBox{
				AssignTo:        &wnd.mainListBox.ListBox,
				OnItemActivated: func() { wnd.msgChan <- msg.Message{msg.ACTIVATE_MENU, wnd.mainListBox.ListBox.CurrentIndex()} },
			},
		},

		StatusBarItems: []StatusBarItem{
			StatusBarItem{
				AssignTo: &wnd.statusBar.elapseTime,
				Text:     "00:00:00",
			},
			StatusBarItem{
				Text: "/",
			},
			StatusBarItem{
				AssignTo: &wnd.statusBar.totalTime,
				Text:     "00:00:00",
			},
			StatusBarItem{
				AssignTo: &wnd.statusBar.fragments,
			},
			StatusBarItem{
				AssignTo: &wnd.statusBar.bookPercent,
			},
		},
	}

	if err := wndLayout.Create(); err != nil {
		return nil, err
	}

	wnd.menuBar.wnd = wnd.mainWindow
	wnd.menuBar.msgCH = wnd.msgChan
	wnd.mainListBox.msgCH = wnd.msgChan
	wnd.statusBar.StatusBar = wnd.mainWindow.StatusBar()

	if err := walk.InitWrapperWindow(wnd.mainListBox); err != nil {
		return nil, err
	}
	return wnd, nil
}

func (mw *MainWnd) MsgChan() chan msg.Message {
	return mw.msgChan
}

func (mw *MainWnd) Run() {
	mw.mainWindow.Run()
	close(mw.msgChan)
}

func (mw *MainWnd) MenuBar() *MenuBar {
	return mw.menuBar
}

func (mw *MainWnd) MainListBox() *MainListBox {
	return mw.mainListBox
}

func (mw *MainWnd) StatusBar() *StatusBar {
	return mw.statusBar
}

func (mw *MainWnd) SetTitle(title string) {
	mw.mainWindow.Synchronize(func() {
		var windowTitle = config.ProgramName
		if title != "" {
			windowTitle = fmt.Sprintf("%s — %s", title, windowTitle)
		}
		mw.mainWindow.SetTitle(windowTitle)
	})
}

func (mw *MainWnd) CredentialsEntryDialog(service *config.Service) bool {
	var (
		dlg                                   *walk.Dialog
		nameLE, urlLE, usernameLE, passwordLE *walk.LineEdit
		OkPB, CancelPB                        *walk.PushButton
	)

	layout := Dialog{
		Title:         gotext.Get("Adding a new account"),
		AssignTo:      &dlg,
		Layout:        VBox{},
		CancelButton:  &CancelPB,
		DefaultButton: &OkPB,
		Children: []Widget{
			TextLabel{Text: gotext.Get("Displayed name:")},
			LineEdit{
				Accessibility: Accessibility{Name: gotext.Get("Displayed name:")},
				AssignTo:      &nameLE,
			},

			TextLabel{Text: gotext.Get("Server address:")},
			LineEdit{
				Accessibility: Accessibility{Name: gotext.Get("Server address:")},
				AssignTo:      &urlLE,
			},

			TextLabel{Text: gotext.Get("User name:")},
			LineEdit{
				Accessibility: Accessibility{Name: gotext.Get("User name:")},
				AssignTo:      &usernameLE,
			},

			TextLabel{Text: gotext.Get("Password:")},
			LineEdit{
				Accessibility: Accessibility{Name: gotext.Get("Password:")},
				AssignTo:      &passwordLE,
				PasswordMode:  true,
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &OkPB,
						Text:     gotext.Get("OK"),
						OnClicked: func() {
							service.Name = nameLE.Text()
							service.URL = urlLE.Text()
							service.Username = usernameLE.Text()
							service.Password = passwordLE.Text()
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     gotext.Get("Cancel"),
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}

	res := make(chan bool)
	mw.mainWindow.Synchronize(func() {
		layout.Create(mw.mainWindow)
		NewFixedPushButton(OkPB)
		NewFixedPushButton(CancelPB)
		dlg.Run()
		res <- dlg.Result() == walk.DlgCmdOK
	})
	return <-res
}

func (mw *MainWnd) TextEntryDialog(title, msg, value string, text *string) bool {
	var (
		dlg            *walk.Dialog
		textLE         *walk.LineEdit
		OkPB, CancelPB *walk.PushButton
	)

	layout := Dialog{
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
						Text:     gotext.Get("OK"),
						OnClicked: func() {
							*text = textLE.Text()
							dlg.Close(walk.DlgCmdOK)
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     gotext.Get("Cancel"),
						OnClicked: func() {
							dlg.Close(walk.DlgCmdCancel)
						},
					},
				},
			},
		},
	}

	res := make(chan bool)
	mw.mainWindow.Synchronize(func() {
		layout.Create(mw.mainWindow)
		NewFixedPushButton(OkPB)
		NewFixedPushButton(CancelPB)
		dlg.Run()
		res <- dlg.Result() == walk.DlgCmdOK
	})
	return <-res
}

func (mw *MainWnd) messageBox(title, message string, style walk.MsgBoxStyle) int {
	res := make(chan int)
	mw.mainWindow.Synchronize(func() {
		res <- walk.MsgBox(mw.mainWindow, title, message, style)
	})
	return <-res
}

func (mw *MainWnd) MessageBoxError(title, message string) {
	mw.messageBox(title, message, walk.MsgBoxOK|walk.MsgBoxIconError)
}

func (mw *MainWnd) MessageBoxWarning(title, message string) {
	mw.messageBox(title, message, walk.MsgBoxOK|walk.MsgBoxIconWarning)
}

func (mw *MainWnd) MessageBoxQuestion(title, message string) bool {
	return mw.messageBox(title, message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes
}
