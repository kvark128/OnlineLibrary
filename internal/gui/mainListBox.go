package gui

import (
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type MainListBox struct {
	*walk.ListBox
	label   *walk.TextLabel
	eventCH chan events.Event
}

func NewMainListBox(lb *walk.ListBox, label *walk.TextLabel, eventCH chan events.Event) (*MainListBox, error) {
	mlb := &MainListBox{
		ListBox: lb,
		label:   label,
		eventCH: eventCH,
	}

	if err := walk.InitWrapperWindow(mlb); err != nil {
		return nil, err
	}

	return mlb, nil
}

func (mlb *MainListBox) SetItems(items []string, label string) {
	mlb.Synchronize(func() {
		mlb.label.SetText(label)
		mlb.Accessibility().SetName(label)
		mlb.SetModel(items)
		mlb.SetCurrentIndex(0)
	})
}

func (mlb *MainListBox) CurrentIndex() int {
	ic := make(chan int)
	mlb.Synchronize(func() {
		ic <- mlb.ListBox.CurrentIndex()
	})
	return <-ic
}

func (mlb *MainListBox) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_CHAR:
		if walk.ModifiersDown() != 0 {
			return 0
		}

	case win.WM_KEYDOWN:
		mods := walk.ModifiersDown()

		if mods == walk.ModShift {
			switch walk.Key(wParam) {
			case walk.KeyC:
				mlb.eventCH <- events.PLAYER_PITCH_UP
			case walk.KeyX:
				mlb.eventCH <- events.PLAYER_PITCH_DOWN
			case walk.KeyZ:
				mlb.eventCH <- events.PLAYER_PITCH_RESET
			default:
				return mlb.ListBox.WndProc(hwnd, msg, wParam, lParam)
			}
			return 0
		}

		if mods == walk.ModControl {
			switch walk.Key(wParam) {
			case walk.KeyRight:
				mlb.eventCH <- events.PLAYER_FORWARD
			case walk.KeyLeft:
				mlb.eventCH <- events.PLAYER_BACK
			case walk.KeyUp:
				mlb.eventCH <- events.PLAYER_VOLUME_UP
			case walk.KeyDown:
				mlb.eventCH <- events.PLAYER_VOLUME_DOWN
			default:
				return mlb.ListBox.WndProc(hwnd, msg, wParam, lParam)
			}
			return 0
		}

	}

	return mlb.ListBox.WndProc(hwnd, msg, wParam, lParam)
}
