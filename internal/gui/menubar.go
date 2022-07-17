package gui

import (
	"fmt"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/lxn/walk"
)

type MenuBar struct {
	wnd                                  *walk.MainWindow
	libraryMenu, outputDeviceMenu        *walk.Menu
	libraryLogon                         *walk.MutableCondition
	bookMenu, bookmarkMenu, logLevelMenu *walk.Menu
	bookMenuEnabled                      *walk.MutableCondition
	pauseTimerItem                       *walk.Action
	msgCH                                chan msg.Message
}

func (mb *MenuBar) SetProvidersMenu(services []*config.Service, currentID string) {
	mb.wnd.Synchronize(func() {
		mb.libraryLogon.SetSatisfied(false)
		actions := mb.libraryMenu.Actions()
		// Delete all elements except the last one
		for i := actions.Len(); i > 1; i-- {
			actions.RemoveAt(0)
		}

		// Filling the menu with services
		for i, service := range services {
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
			actions.Insert(i, a)
		}
	})
}

func (mb *MenuBar) SetBookmarksMenu(bookmarks map[string]string) {
	mb.wnd.Synchronize(func() {
		actions := mb.bookmarkMenu.Actions()
		// Delete all elements except the last one
		for i := actions.Len(); i > 1; i-- {
			actions.RemoveAt(0)
		}

		// Filling the menu with bookmarks
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
	})
}

func (mb *MenuBar) SetBookMenuEnabled(enabled bool) {
	done := make(chan bool)
	mb.wnd.Synchronize(func() {
		mb.bookMenuEnabled.SetSatisfied(enabled)
		done <- true
	})
	<-done
}

func (mb *MenuBar) SetOutputDeviceMenu(deviceNames []string, current string) {
	mb.wnd.Synchronize(func() {
		actions := mb.outputDeviceMenu.Actions()
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
				mb.msgCH <- msg.Message{msg.PLAYER_OUTPUT_DEVICE, a.Text()}
			})
			actions.Insert(i, a)
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

func (mb *MenuBar) BookMenu() *walk.Menu {
	return mb.bookMenu
}
