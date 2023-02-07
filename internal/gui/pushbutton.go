package gui

import (
	"github.com/kvark128/walk"
	"github.com/lxn/win"
)

type FixedPushButton struct {
	*walk.PushButton
}

func NewFixedPushButton(pb *walk.PushButton) (*FixedPushButton, error) {
	fpb := &FixedPushButton{
		PushButton: pb,
	}

	if err := walk.InitWrapperWindow(fpb); err != nil {
		return nil, err
	}

	return fpb, nil
}

func (fpb *FixedPushButton) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_GETDLGCODE:
		return win.DLGC_BUTTON
	}
	return fpb.PushButton.WndProc(hwnd, msg, wParam, lParam)
}
