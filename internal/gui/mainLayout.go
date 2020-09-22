package gui

import (
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func CreateWND(actionQueue chan events.Event) (*walk.MainWindow, error) {
	var wnd *walk.MainWindow

	var layout = MainWindow{
		Title:    "DAISY Online Client",
		Layout:   VBox{},
		AssignTo: &wnd,

		MenuItems: []MenuItem{

			Menu{
				Text: "&Библиотека",
				Items: []MenuItem{
					Action{
						Text:        "Загрузить книгу",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyD},
						OnTriggered: func() { actionQueue <- events.DOWNLOAD_BOOK },
					},
					Action{
						Text:        "Убрать книгу с полки",
						Shortcut:    Shortcut{walk.ModShift, walk.KeyDelete},
						OnTriggered: func() { actionQueue <- events.REMOVE_BOOK },
					},
					Action{
						Text:        "Поставить книгу на полку",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyA},
						OnTriggered: func() { actionQueue <- events.ISSUE_BOOK },
					},
					Action{
						Text:        "Информация о книге",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyG},
						OnTriggered: func() { actionQueue <- events.BOOK_PROPERTIES },
					},
					Action{
						Text:        "Поиск",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyF},
						OnTriggered: func() { actionQueue <- events.SEARCH_BOOK },
					},
					Action{
						Text:        "Главное меню библиотеки",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyM},
						OnTriggered: func() { actionQueue <- events.MAIN_MENU },
					},
					Action{
						Text:        "Книжная полка",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyE},
						OnTriggered: func() { actionQueue <- events.OPEN_BOOKSHELF },
					},
					Action{
						Text:        "Назад по меню",
						Shortcut:    Shortcut{0, walk.KeyBack},
						OnTriggered: func() { actionQueue <- events.MENU_BACK },
					},
					Action{
						Text:        "Выйти из учётной записи",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyQ},
						OnTriggered: func() { actionQueue <- events.LIBRARY_RELOGON },
					},
				},
			},

			Menu{
				Text: "&Воспроизведение",
				Items: []MenuItem{
					Action{
						Text:        "Перемотка вперёд",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyO},
						OnTriggered: func() { actionQueue <- events.PLAYER_FORWARD },
					},
					Action{
						Text:        "Перемотка назад",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyI},
						OnTriggered: func() { actionQueue <- events.PLAYER_BACK },
					},
					Action{
						Text:        "У&величить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyB},
						OnTriggered: func() { actionQueue <- events.PLAYER_VOLUME_UP },
					},
					Action{
						Text:        "У&меньшить громкость",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyV},
						OnTriggered: func() { actionQueue <- events.PLAYER_VOLUME_DOWN },
					},
					Action{
						Text:        "Следующий фрагмент",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyL},
						OnTriggered: func() { actionQueue <- events.PLAYER_NEXT_TRACK },
					},
					Action{
						Text:        "Предыдущий фрагмент",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyJ},
						OnTriggered: func() { actionQueue <- events.PLAYER_PREVIOUS_TRACK },
					},
					Action{
						Text:        "Пауза",
						Shortcut:    Shortcut{walk.ModControl, walk.KeyK},
						OnTriggered: func() { actionQueue <- events.PLAYER_PAUSE },
					},
				},
			},
		},
	}

	err := layout.Create()
	return wnd, err
}
