package gui

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type ProgressDialog struct {
	dlg *walk.Dialog
	pb  *walk.ProgressBar
}

func NewProgressDialog(title, msg string, maxValue int, cancelFN func()) *ProgressDialog {
	pd := &ProgressDialog{}
	var CancelPB *walk.PushButton

	var layout = Dialog{
		Title:        title,
		AssignTo:     &pd.dlg,
		Layout:       VBox{},
		CancelButton: &CancelPB,
		Children: []Widget{

			TextLabel{Text: msg},

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
	mainWindow.Synchronize(func() {
		layout.Create(nil)
		done <- true
	})
	<-done
	return pd
}

func (pd *ProgressDialog) Show() {
	mainWindow.Synchronize(func() {
		pd.dlg.Run()
	})
}

func (pd *ProgressDialog) IncreaseValue(value int) int {
	valueCH := make(chan int)
	pd.dlg.Synchronize(func() {
		newValue := pd.pb.Value() + value
		pd.pb.SetValue(newValue)
		valueCH <- newValue
	})
	return <-valueCH
}

func (pd *ProgressDialog) Cancel() {
	done := make(chan bool)
	pd.dlg.Synchronize(func() {
		pd.dlg.Cancel()
		done <- true
	})
	<-done
}
