package gui

import (
	"time"

	"github.com/kvark128/OnlineLibrary/internal/gui/msg"
	"github.com/kvark128/walk"
	"github.com/lxn/win"
)

// Messages for switching fragments
var (
	next_fragment     = msg.Message{Code: msg.PLAYER_OFFSET_FRAGMENT, Data: +1}
	previous_fragment = msg.Message{Code: msg.PLAYER_OFFSET_FRAGMENT, Data: -1}
)

// Messages for rewinding a fragment
var (
	rewind_5sec_forward  = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Second * 5}
	rewind_5sec_back     = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Second * -5}
	rewind_30sec_forward = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Second * 30}
	rewind_30sec_back    = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Second * -30}
	rewind_1min_forward  = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Minute}
	rewind_1min_back     = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: -time.Minute}
	rewind_5min_forward  = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Minute * 5}
	rewind_5min_back     = msg.Message{Code: msg.PLAYER_OFFSET_POSITION, Data: time.Minute * -5}
)

type MainListBox struct {
	*walk.ListBox
	label *walk.TextLabel
	msgCH chan msg.Message
}

func (mlb *MainListBox) SetItems(items []string, label string, contextMenu *walk.Menu) {
	mlb.Synchronize(func() {
		mlb.label.SetText(label)
		mlb.Accessibility().SetName(label)
		mlb.SetModel(items)
		mlb.ListBox.SetContextMenu(contextMenu)
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
			case walk.KeyUp:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_SPEED_UP}
				return 0
			case walk.KeyDown:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_SPEED_DOWN}
				return 0
			case walk.Key1, walk.Key2, walk.Key3, walk.Key4, walk.Key5, walk.Key6, walk.Key7, walk.Key8, walk.Key9, walk.Key0:
				mlb.msgCH <- msg.Message{Code: msg.BOOKMARK_SET, Data: key.String()}
				return 0
			case walk.KeyR:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_SPEED_RESET}
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
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_VOLUME_UP}
				return 0
			case walk.KeyDown:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_VOLUME_DOWN}
				return 0
			case walk.KeyPrior:
				mlb.msgCH <- previous_fragment
				return 0
			case walk.KeyNext:
				mlb.msgCH <- next_fragment
				return 0
			case walk.Key1, walk.Key2, walk.Key3, walk.Key4, walk.Key5, walk.Key6, walk.Key7, walk.Key8, walk.Key9, walk.Key0:
				mlb.msgCH <- msg.Message{Code: msg.BOOKMARK_FETCH, Data: key.String()}
				return 0
			case walk.KeyR:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_VOLUME_RESET}
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
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_PLAY_PAUSE}
				return 0
			case walk.KeyMediaPlayPause:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_PLAY_PAUSE}
				return 0
			case walk.KeyMediaStop:
				mlb.msgCH <- msg.Message{Code: msg.PLAYER_STOP}
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
