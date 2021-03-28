package gui

import (
	"time"

	"github.com/kvark128/OnlineLibrary/internal/msg"
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// Messages for switching fragments
var (
	next_fragment     = msg.Message{msg.PLAYER_OFFSET_FRAGMENT, +1}
	previous_fragment = msg.Message{msg.PLAYER_OFFSET_FRAGMENT, -1}
)

// Messages for rewinding a fragment
var (
	rewind_5sec_forward  = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Second * 5}
	rewind_5sec_back     = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Second * -5}
	rewind_30sec_forward = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Second * 30}
	rewind_30sec_back    = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Second * -30}
	rewind_1min_forward  = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Minute}
	rewind_1min_back     = msg.Message{msg.PLAYER_OFFSET_POSITION, -time.Minute}
	rewind_5min_forward  = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Minute * 5}
	rewind_5min_back     = msg.Message{msg.PLAYER_OFFSET_POSITION, time.Minute * -5}
)

type MainListBox struct {
	*walk.ListBox
	label *walk.TextLabel
	msgCH chan msg.Message
}

func NewMainListBox(lb *walk.ListBox, label *walk.TextLabel, msgCH chan msg.Message) (*MainListBox, error) {
	mlb := &MainListBox{
		ListBox: lb,
		label:   label,
		msgCH:   msgCH,
	}

	if err := walk.InitWrapperWindow(mlb); err != nil {
		return nil, err
	}

	return mlb, nil
}

func (mlb *MainListBox) SetItems(items []string, label string, menu *walk.Menu) {
	mlb.Synchronize(func() {
		mlb.label.SetText(label)
		mlb.Accessibility().SetName(label)
		mlb.SetModel(items)
		mlb.ListBox.SetContextMenu(menu)
		mlb.SetCurrentIndex(0)
	})
}

func (mlb *MainListBox) Clear() {
	mlb.SetItems([]string{}, "", nil)
}

func (mlb *MainListBox) CurrentIndex() int {
	ic := make(chan int)
	mlb.Synchronize(func() {
		ic <- mlb.ListBox.CurrentIndex()
	})
	return <-ic
}

func (mlb *MainListBox) WndProc(hwnd win.HWND, winmsg uint32, wParam, lParam uintptr) uintptr {
	switch winmsg {
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
				mlb.msgCH <- rewind_5min_back
				return 0
			case walk.KeyRight:
				mlb.msgCH <- rewind_5min_forward
				return 0
			}
		}

		if mods == walk.ModShift {
			switch key {
			case walk.KeyLeft:
				mlb.msgCH <- rewind_1min_back
				return 0
			case walk.KeyRight:
				mlb.msgCH <- rewind_1min_forward
				return 0
			}
		}

		if mods == walk.ModControl {
			switch key {
			case walk.KeyLeft:
				mlb.msgCH <- rewind_30sec_back
				return 0
			case walk.KeyRight:
				mlb.msgCH <- rewind_30sec_forward
				return 0
			case walk.KeyUp:
				mlb.msgCH <- msg.Message{msg.PLAYER_VOLUME_UP, nil}
				return 0
			case walk.KeyDown:
				mlb.msgCH <- msg.Message{msg.PLAYER_VOLUME_DOWN, nil}
				return 0
			}
		}

		if mods == 0 {
			switch key {
			case walk.KeyRight:
				mlb.msgCH <- rewind_5sec_forward
				return 0
			case walk.KeyLeft:
				mlb.msgCH <- rewind_5sec_back
				return 0
			case walk.KeySpace:
				mlb.msgCH <- msg.Message{msg.PLAYER_PLAY_PAUSE, nil}
				return 0
			case walk.KeyMediaPlayPause:
				mlb.msgCH <- msg.Message{msg.PLAYER_PLAY_PAUSE, nil}
				return 0
			case walk.KeyMediaStop:
				mlb.msgCH <- msg.Message{msg.PLAYER_STOP, nil}
				return 0
			case walk.KeyMediaNextTrack:
				mlb.msgCH <- next_fragment
				return 0
			case walk.KeyMediaPrevTrack:
				mlb.msgCH <- previous_fragment
				return 0
			}
		}

	}

	return mlb.ListBox.WndProc(hwnd, winmsg, wParam, lParam)
}
