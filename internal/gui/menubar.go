package gui

import (
	"fmt"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/kvark128/OnlineLibrary/internal/lang"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/lxn/walk"
)

type MenuBar struct {
	wnd                                  *walk.MainWindow
	libraryMenu, outputDeviceMenu        *walk.Menu
	libraryLogon                         *walk.MutableCondition
	bookMenu, bookmarkMenu, logLevelMenu *walk.Menu
	bookMenuEnabled                      *walk.MutableCondition
	languageMenu                         *walk.Menu
	pauseTimerItem                       *walk.Action
	msgCH                                chan msg.Message
}

func (mb *MenuBar) SetProvidersMenu(services []*config.Service, currentID string) {
	mb.wnd.Synchronize(func() {
		mb.libraryLogon.SetSatisfied(false)
		actions := mb.libraryMenu.Actions()
		newLibraryAction := actions.At(actions.Len() - 1)
		actions.Clear()

		for _, service := range services {
			a := walk.NewAction()
			id := service.ID
			if id == currentID {
				a.SetChecked(true)
				mb.libraryLogon.SetSatisfied(true)
			}
			a.SetText(service.Name)
			a.Triggered().Attach(func() {
				mb.msgCH <- msg.Message{msg.SET_PROVIDER, id}
			})
			actions.Add(a)
		}
		actions.Add(newLibraryAction)
	})
}

func (mb *MenuBar) SetBookmarksMenu(bookmarks map[string]string) {
	mb.wnd.Synchronize(func() {
		actions := mb.bookmarkMenu.Actions()
		newBookmarkAction := actions.At(actions.Len() - 1)
		actions.Clear()

		for id, name := range bookmarks {
			if name == "" {
				continue
			}
			id := id
			subMenu, err := walk.NewMenu()
			if err != nil {
				panic(err)
			}
			a, err := actions.InsertMenu(0, subMenu)
			if err != nil {
				panic(err)
			}
			a.SetText(name)
			bookmarkActions := subMenu.Actions()
			moveAction := walk.NewAction()
			moveAction.SetText("Перейти...")
			moveAction.Triggered().Attach(func() {
				mb.msgCH <- msg.Message{msg.BOOKMARK_FETCH, id}
			})
			bookmarkActions.Add(moveAction)
			removeAction := walk.NewAction()
			removeAction.SetText("Удалить...")
			removeAction.Triggered().Attach(func() {
				mb.msgCH <- msg.Message{msg.BOOKMARK_REMOVE, id}
			})
			bookmarkActions.Add(removeAction)
		}
		actions.Add(newBookmarkAction)
	})
}

func (mb *MenuBar) SetBookMenuEnabled(enabled bool) {
	mb.wnd.Synchronize(func() {
		mb.bookMenuEnabled.SetSatisfied(enabled)
	})
}

func (mb *MenuBar) SetOutputDeviceMenu(deviceNames []string, current string) {
	mb.wnd.Synchronize(func() {
		actions := mb.outputDeviceMenu.Actions()
		actions.Clear()

		for i, name := range deviceNames {
			a := walk.NewAction()
			a.SetText(name)
			if name == current || (current == "" && i == 0) {
				a.SetChecked(true)
			}
			a.Triggered().Attach(func() {
				actions := mb.outputDeviceMenu.Actions()
				for k := 0; k < actions.Len(); k++ {
					actions.At(k).SetChecked(false)
				}
				a.SetChecked(true)
				mb.msgCH <- msg.Message{msg.PLAYER_OUTPUT_DEVICE, a.Text()}
			})
			actions.Add(a)
		}
	})
}

func (mb *MenuBar) SetLanguageMenu(langs []lang.Language, current string) {
	mb.wnd.Synchronize(func() {
		actions := mb.languageMenu.Actions()
		actions.Clear()

		langsWithDefaultLang := make([]lang.Language, len(langs)+1)
		langsWithDefaultLang[0] = lang.Language{Description: "По умолчанию"}
		copy(langsWithDefaultLang[1:], langs)

		for _, lang := range langsWithDefaultLang {
			a := walk.NewAction()
			a.SetText(lang.Description)
			id := lang.ID
			if id == current {
				a.SetChecked(true)
			}
			a.Triggered().Attach(func() {
				actions := mb.languageMenu.Actions()
				for k := 0; k < actions.Len(); k++ {
					actions.At(k).SetChecked(false)
				}
				a.SetChecked(true)
				mb.msgCH <- msg.Message{msg.SET_LANGUAGE, id}
			})
			actions.Add(a)
		}
	})
}

func (mb *MenuBar) SetPauseTimerLabel(minutes int) {
	label := "Таймер паузы (Нет)"
	if minutes > 0 {
		label = fmt.Sprintf("Таймер паузы (%d мин.)", minutes)
	}
	mb.wnd.Synchronize(func() {
		mb.pauseTimerItem.SetText(label)
	})
}

func (mb *MenuBar) SetLogLevelMenu(levels []log.Level, current log.Level) {
	mb.wnd.Synchronize(func() {
		actions := mb.logLevelMenu.Actions()
		actions.Clear()

		for _, level := range levels {
			level := level // Avoid capturing the iteration variable
			a := walk.NewAction()
			a.SetText(level.String())
			if level == current {
				a.SetChecked(true)
			}
			a.Triggered().Attach(func() {
				actions := mb.logLevelMenu.Actions()
				for k := 0; k < actions.Len(); k++ {
					actions.At(k).SetChecked(false)
				}
				a.SetChecked(true)
				mb.msgCH <- msg.Message{Code: msg.LOG_SET_LEVEL, Data: level}
			})
			actions.Add(a)
		}
	})
}

func (mb *MenuBar) BookMenu() *walk.Menu {
	return mb.bookMenu
}
