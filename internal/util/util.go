package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/kvark128/av3715/internal/connect"
	"github.com/kvark128/av3715/internal/gui"
	daisy "github.com/kvark128/daisyonline"
)

func DownloadBook(dir, book string, r *daisy.Resources) {
	me := &sync.Mutex{}
	var dst io.WriteCloser
	var conn, src io.ReadCloser
	var stop bool
	var err error

	cancelFN := func() {
		me.Lock()
		if src != nil {
			src.Close()
		}
		stop = true
		me.Unlock()
	}

	dlg := gui.NewProgressDialog("Загрузка книги", fmt.Sprintf("Загрузка %s", book), len(r.Resources), cancelFN)
	dlg.Show()

	for _, v := range r.Resources {
		path := filepath.Join(dir, book, v.LocalURI)
		if info, e := os.Stat(path); e == nil {
			if !info.IsDir() && info.Size() == v.Size {
				// v.LocalURI already exist
				dlg.IncreaseValue(1)
				continue
			}
		}

		conn, err = connect.NewConnection(v.URI)
		if err != nil {
			break
		}

		os.MkdirAll(filepath.Dir(path), os.ModeDir)
		dst, err = os.Create(path)
		if err != nil {
			conn.Close()
			break
		}

		me.Lock()
		src = conn
		if stop {
			src.Close()
		}
		me.Unlock()

		_, err = io.CopyBuffer(dst, src, make([]byte, 512*1024))
		dst.Close()
		src.Close()
		if err != nil {
			// Removing an unwritten file
			os.Remove(path)
			break
		}

		dlg.IncreaseValue(1)
	}

	dlg.Cancel()
	if stop {
		gui.MessageBox("Предупреждение", "Загрузка отменена пользователем", gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}

	if err != nil {
		gui.MessageBox("Ошибка", err.Error(), gui.MsgBoxOK|gui.MsgBoxIconWarning)
		return
	}
	gui.MessageBox("Уведомление", "Книга успешно загружена", gui.MsgBoxOK|gui.MsgBoxIconWarning)
}
