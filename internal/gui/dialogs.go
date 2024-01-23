package gui

import (
	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/walk"
	. "github.com/kvark128/walk/declarative"
	"github.com/kvark128/win"
	"github.com/leonelquinteros/gotext"
)

type MsgBoxStyle uint

const (
	MsgBoxIconError       MsgBoxStyle = win.MB_ICONERROR
	MsgBoxIconInformation MsgBoxStyle = win.MB_ICONINFORMATION
	MsgBoxIconWarning     MsgBoxStyle = win.MB_ICONWARNING
	MsgBoxIconQuestion    MsgBoxStyle = win.MB_ICONQUESTION
	MsgBoxOK              MsgBoxStyle = win.MB_OK
	MsgBoxYesNo           MsgBoxStyle = win.MB_YESNO
)

const (
	DlgCmdOK     = win.IDOK
	DlgCmdCancel = win.IDCANCEL
	DlgCmdClose  = win.IDCLOSE
	DlgCmdYes    = win.IDYES
	DlgCmdNo     = win.IDNO
)

func MessageBox(owner Form, title, message string, style MsgBoxStyle) int {
	res := make(chan int)
	parent := owner.form()
	parent.Synchronize(func() {
		res <- walk.MsgBox(parent, title, message, walk.MsgBoxStyle(style))
	})
	return <-res
}

func BookInfoDialog(owner Form, title, description string) int {
	parent := owner.form()
	var (
		dlg              *walk.Dialog
		ClosePB          *walk.PushButton
		parentSize       = parent.Size()
		descriptionLabel = gotext.Get("Description:")
	)

	layout := Dialog{
		Title:        title,
		AssignTo:     &dlg,
		Layout:       VBox{},
		CancelButton: &ClosePB,
		MinSize:      Size{Width: parentSize.Width / 2, Height: parentSize.Height / 2},
		Children: []Widget{

			TextLabel{Text: descriptionLabel},
			TextEdit{
				Accessibility: Accessibility{Name: descriptionLabel},
				Text:          description,
				ReadOnly:      true,
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &ClosePB,
						Text:     gotext.Get("Close"),
						OnClicked: func() {
							dlg.Close(walk.DlgCmdClose)
						},
					},
				},
			},
		},
	}

	res := make(chan int)
	parent.Synchronize(func() {
		layout.Create(parent)
		dlg.Run()
		res <- dlg.Result()
	})
	return <-res
}

func TextEntryDialog(owner Form, title, msg, value string, text *string) int {
	parent := owner.form()
	var (
		dlg            *walk.Dialog
		textLE         *walk.LineEdit
		OkPB, CancelPB *walk.PushButton
	)

	layout := Dialog{
		Title:         title,
		AssignTo:      &dlg,
		Layout:        VBox{},
		CancelButton:  &CancelPB,
		DefaultButton: &OkPB,
		Children: []Widget{

			TextLabel{Text: msg},
			LineEdit{
				Accessibility: Accessibility{Name: msg},
				Text:          value,
				AssignTo:      &textLE,
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &OkPB,
						Text:     gotext.Get("OK"),
						OnClicked: func() {
							*text = textLE.Text()
							dlg.Close(walk.DlgCmdOK)
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     gotext.Get("Cancel"),
						OnClicked: func() {
							dlg.Close(walk.DlgCmdCancel)
						},
					},
				},
			},
		},
	}

	res := make(chan int)
	parent.Synchronize(func() {
		layout.Create(parent)
		NewFixedPushButton(OkPB)
		NewFixedPushButton(CancelPB)
		dlg.Run()
		res <- dlg.Result()
	})
	return <-res
}

func CredentialsEntryDialog(owner Form, service *config.Service) int {
	parent := owner.form()
	var (
		dlg                                   *walk.Dialog
		nameLE, urlLE, usernameLE, passwordLE *walk.LineEdit
		nameLabel                             = gotext.Get("Displayed name:")
		urlLabel                              = gotext.Get("Server address:")
		usernameLabel                         = gotext.Get("User name:")
		passwordLabel                         = gotext.Get("Password:")
		OkPB, CancelPB                        *walk.PushButton
	)

	layout := Dialog{
		Title:         gotext.Get("Adding a new account"),
		AssignTo:      &dlg,
		Layout:        VBox{},
		CancelButton:  &CancelPB,
		DefaultButton: &OkPB,
		Children: []Widget{
			TextLabel{Text: nameLabel},
			LineEdit{
				Accessibility: Accessibility{Name: nameLabel},
				AssignTo:      &nameLE,
			},

			TextLabel{Text: urlLabel},
			LineEdit{
				Accessibility: Accessibility{Name: urlLabel},
				AssignTo:      &urlLE,
			},

			TextLabel{Text: usernameLabel},
			LineEdit{
				Accessibility: Accessibility{Name: usernameLabel},
				AssignTo:      &usernameLE,
			},

			TextLabel{Text: passwordLabel},
			LineEdit{
				Accessibility: Accessibility{Name: passwordLabel},
				AssignTo:      &passwordLE,
				PasswordMode:  true,
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &OkPB,
						Text:     gotext.Get("OK"),
						OnClicked: func() {
							service.Name = nameLE.Text()
							service.URL = urlLE.Text()
							service.Username = usernameLE.Text()
							service.Password = passwordLE.Text()
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo: &CancelPB,
						Text:     gotext.Get("Cancel"),
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}

	res := make(chan int)
	parent.Synchronize(func() {
		layout.Create(parent)
		NewFixedPushButton(OkPB)
		NewFixedPushButton(CancelPB)
		dlg.Run()
		res <- dlg.Result()
	})
	return <-res
}

type ProgressDialog struct {
	parent walk.Form
	dlg    *walk.Dialog
	label  *walk.TextLabel
	pb     *walk.ProgressBar
}

func NewProgressDialog(owner Form, title, label string, maxValue int, cancelFN func()) *ProgressDialog {
	pd := &ProgressDialog{parent: owner.form()}
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
				Text:     gotext.Get("Cancel"),
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
