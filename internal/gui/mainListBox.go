package gui

import (
	"time"

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
		if wParam <= 32 || walk.ModifiersDown() != 0 {
			return 0
		}

	case win.WM_KEYDOWN:
		mods := walk.ModifiersDown()
		key := walk.Key(wParam)

		if mods == walk.ModControl|walk.ModShift {
			switch key {
			case walk.KeyLeft:
				mlb.eventCH <- events.Event{events.PLAYER_REWIND, -time.Minute}
				return 0
			case walk.KeyRight:
				mlb.eventCH <- events.Event{events.PLAYER_REWIND, time.Minute}
				return 0
			}
		}

		if mods == walk.ModShift {
			switch key {
			case walk.KeyC:
				mlb.eventCH <- events.Event{events.PLAYER_PITCH_UP, nil}
				return 0
			case walk.KeyX:
				mlb.eventCH <- events.Event{events.PLAYER_PITCH_DOWN, nil}
				return 0
			case walk.KeyZ:
				mlb.eventCH <- events.Event{events.PLAYER_PITCH_RESET, nil}
				return 0
			}
		}

		if mods == walk.ModControl {
			switch key {
			case walk.KeyLeft:
				mlb.eventCH <- events.Event{events.PLAYER_REWIND, time.Second * -30}
				return 0
			case walk.KeyRight:
				mlb.eventCH <- events.Event{events.PLAYER_REWIND, time.Second * 30}
				return 0
			case walk.KeyUp:
				mlb.eventCH <- events.Event{events.PLAYER_VOLUME_UP, nil}
				return 0
			case walk.KeyDown:
				mlb.eventCH <- events.Event{events.PLAYER_VOLUME_DOWN, nil}
				return 0
			}
		}

		if mods == 0 {
			switch key {
			case walk.KeyRight:
				mlb.eventCH <- events.Event{events.PLAYER_REWIND, time.Second * 5}
				return 0
			case walk.KeyLeft:
				mlb.eventCH <- events.Event{events.PLAYER_REWIND, time.Second * -5}
				return 0
			}
		}

	}

	return mlb.ListBox.WndProc(hwnd, msg, wParam, lParam)
}
