package gui

import (
	"fmt"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/walk"
	"github.com/leonelquinteros/gotext"
)

type StatusBar struct {
	*walk.StatusBar
	elapseTime, totalTime, fragments, bookPercent *walk.StatusBarItem
}

func (sb *StatusBar) SetElapsedTime(elapsed time.Duration) {
	sb.Synchronize(func() {
		text := util.FmtDuration(elapsed)
		sb.elapseTime.SetText(text)
	})
}

func (sb *StatusBar) SetTotalTime(total time.Duration) {
	sb.Synchronize(func() {
		text := util.FmtDuration(total)
		sb.totalTime.SetText(text)
	})
}

func (sb *StatusBar) SetFragments(current, length int) {
	sb.Synchronize(func() {
		text := gotext.Get("Fragment %d of %d", current, length)
		sb.fragments.SetText(text)
	})
}

func (sb *StatusBar) SetBookPercent(p int) {
	sb.Synchronize(func() {
		text := fmt.Sprintf("(%v%%)", p)
		sb.bookPercent.SetText(text)
	})
}
