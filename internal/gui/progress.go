package gui

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type ProgressDialog struct {
	parent *walk.MainWindow
	dlg    *walk.Dialog
	label  *walk.TextLabel
	pb     *walk.ProgressBar
}

func NewProgressDialog(parent *MainWnd, title, label string, maxValue int, cancelFN func()) *ProgressDialog {
	pd := &ProgressDialog{parent: parent.mainWindow}
	var CancelPB *walk.PushButton

	var layout = Dialog{
		Title:        title,
		AssignTo:     &pd.dlg,
		Layout:       VBox{},
		CancelButton: &CancelPB,
		Children: []Widget{

			TextLabel{
				Text:     label,
				AssignTo: &pd.label,
			},

			ProgressBar{
				MaxValue: maxValue,
				AssignTo: &pd.pb,
			},

			PushButton{
				AssignTo: &CancelPB,
				Text:     "Отмена",
				OnClicked: func() {
					cancelFN()
					pd.dlg.Cancel()
				},
			},
		},
	}

	done := make(chan bool)
	pd.parent.Synchronize(func() {
		layout.Create(pd.parent)
		done <- true
	})
	<-done
	return pd
}

func (pd *ProgressDialog) Run() {
	pd.parent.Synchronize(func() {
		pd.dlg.Run()
	})
}

func (pd *ProgressDialog) SetLabel(label string) {
	doneCH := make(chan bool)
	pd.dlg.Synchronize(func() {
		pd.label.SetText(label)
		doneCH <- true
	})
	<-doneCH
}

func (pd *ProgressDialog) SetValue(value int) {
	doneCH := make(chan bool)
	pd.dlg.Synchronize(func() {
		pd.pb.SetValue(value)
		doneCH <- true
	})
	<-doneCH
}

func (pd *ProgressDialog) Cancel() {
	done := make(chan bool)
	pd.dlg.Synchronize(func() {
		pd.dlg.Cancel()
		done <- true
	})
	<-done
}
