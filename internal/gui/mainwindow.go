package gui

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/kvark128/walk"
	. "github.com/kvark128/walk/declarative"
	"github.com/leonelquinteros/gotext"
)

type Form interface {
	form() walk.Form
}

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
								Text:        gotext.Get("New account"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyN},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.LIBRARY_ADD} },
							},
						},
					},
					Action{
						Text:        gotext.Get("Bookshelf"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyE},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.OPEN_BOOKSHELF} },
					},
					Action{
						Text:        gotext.Get("New books"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyK},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.OPEN_NEWBOOKS} },
					},
					Action{
						Text:        gotext.Get("Find..."),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyF},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.SEARCH_BOOK} },
					},
					Action{
						Text:        gotext.Get("Main menu"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyM},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.MAIN_MENU} },
					},
					Action{
						Text:        gotext.Get("Previous menu"),
						Shortcut:    Shortcut{Modifiers: 0, Key: walk.KeyBack},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.MENU_BACK} },
					},
					Action{
						Text:        gotext.Get("Local books"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyL},
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.SET_PROVIDER, Data: config.LocalStorageID} },
					},
					Action{
						Text:        gotext.Get("Library information"),
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.LIBRARY_INFO} },
					},
					Action{
						Text:        gotext.Get("Delete account"),
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.LIBRARY_REMOVE} },
					},
					Action{
						Text:        gotext.Get("Exit"),
						Shortcut:    Shortcut{Modifiers: walk.ModAlt, Key: walk.KeyF4},
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
						Text:        gotext.Get("Add book to bookshelf"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyA},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.ISSUE_BOOK} },
					},
					Action{
						Text:        gotext.Get("Download book"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyD},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.DOWNLOAD_BOOK} },
					},
					Action{
						Text:        gotext.Get("Remove book from bookshelf"),
						Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyDelete},
						Enabled:     Bind("libraryLogon"),
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.REMOVE_BOOK} },
					},
					Action{
						Text:        gotext.Get("Book information"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyI},
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.BOOK_DESCRIPTION} },
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
								Text:        gotext.Get("New bookmark"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyB},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.BOOKMARK_SET} },
							},
						},
					},
					Action{
						Text:        gotext.Get("Play / Pause"),
						Shortcut:    Shortcut{Key: walk.KeySpace},
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_PLAY_PAUSE} },
					},
					Action{
						Text:        gotext.Get("Stop"),
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeySpace},
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_STOP} },
					},

					Menu{
						Text: gotext.Get("Book navigation"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Beginning of the book"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyBack},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_GOTO_FRAGMENT, Data: 0} },
							},
							Action{
								Text:        gotext.Get("Go to fragment..."),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyG},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_GOTO_FRAGMENT} },
							},
							Action{
								Text:        gotext.Get("Next fragment"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyNext},
								OnTriggered: func() { wnd.msgChan <- next_fragment },
							},
							Action{
								Text:        gotext.Get("Previous fragment"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyPrior},
								OnTriggered: func() { wnd.msgChan <- previous_fragment },
							},
						},
					},

					Menu{
						Text: gotext.Get("Fragment navigation"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Beginning of the fragment"),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyBack},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_GOTO_POSITION, Data: time.Duration(0)} },
							},
							Action{
								Text:        gotext.Get("Go to position..."),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyG},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_GOTO_POSITION} },
							},
							Action{
								Text:        gotext.Get("5 sec. forward"),
								Shortcut:    Shortcut{Modifiers: 0, Key: walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_5sec_forward },
							},
							Action{
								Text:        gotext.Get("5 sec. backward"),
								Shortcut:    Shortcut{Modifiers: 0, Key: walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_5sec_back },
							},
							Action{
								Text:        gotext.Get("30 sec. forward"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_30sec_forward },
							},
							Action{
								Text:        gotext.Get("30 sec. backward"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_30sec_back },
							},
							Action{
								Text:        gotext.Get("1 min. forward"),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_1min_forward },
							},
							Action{
								Text:        gotext.Get("1 min. backward"),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_1min_back },
							},
							Action{
								Text:        gotext.Get("5 min. forward"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyRight},
								OnTriggered: func() { wnd.msgChan <- rewind_5min_forward },
							},
							Action{
								Text:        gotext.Get("5 min. backward"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyLeft},
								OnTriggered: func() { wnd.msgChan <- rewind_5min_back },
							},
						},
					},

					Menu{
						Text: gotext.Get("Volume"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Increase volume"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyUp},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_VOLUME_UP} },
							},
							Action{
								Text:        gotext.Get("Decrease volume"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyDown},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_VOLUME_DOWN} },
							},
							Action{
								Text:        gotext.Get("Reset volume"),
								Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyR},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_VOLUME_RESET} },
							},
						},
					},

					Menu{
						Text: gotext.Get("Speed"),
						Items: []MenuItem{
							Action{
								Text:        gotext.Get("Increase speed"),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyUp},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_SPEED_UP} },
							},
							Action{
								Text:        gotext.Get("Decrease speed"),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyDown},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_SPEED_DOWN} },
							},
							Action{
								Text:        gotext.Get("Reset speed"),
								Shortcut:    Shortcut{Modifiers: walk.ModShift, Key: walk.KeyR},
								OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_SPEED_RESET} },
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
						Shortcut:    Shortcut{Modifiers: walk.ModControl, Key: walk.KeyP},
						OnTriggered: func() { wnd.msgChan <- msg.Message{Code: msg.PLAYER_SET_TIMER} },
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
				AssignTo: &wnd.mainListBox.ListBox,
				OnItemActivated: func() {
					wnd.msgChan <- msg.Message{Code: msg.ACTIVATE_MENU}
				},
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

func (mw *MainWnd) form() walk.Form {
	return mw.mainWindow
}

func (mw *MainWnd) Run() {
	mw.msgChan <- msg.Message{Code: msg.SET_PROVIDER}
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
			windowTitle = fmt.Sprintf("%v \u2014 %v", title, windowTitle)
		}
		mw.mainWindow.SetTitle(windowTitle)
	})
}
