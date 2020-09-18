package player

import (
	"io"
	"sync"
	"time"

	"github.com/kvark128/av3715/internal/winmm"
	"github.com/kvark128/minimp3"
)

type track struct {
	sync.Mutex
	stopped    bool
	paused     bool
	beRewind   bool
	lost       time.Duration
	start      time.Time
	dec        *minimp3.Decoder
	sampleRate int
	channels   int
	samples    []byte
	wp         *winmm.WavePlayer
}

func newTrack(mp3 io.Reader) *track {
	trk := &track{}
	trk.dec = minimp3.NewDecoder(mp3)
	trk.dec.Read([]byte{}) // Reads first frame
	trk.sampleRate, trk.channels, _, _ = trk.dec.Info()
	trk.samples = make([]byte, trk.sampleRate*trk.channels*2) // buffer for 1 second
	trk.wp = winmm.NewWavePlayer(trk.channels, trk.sampleRate, 16, len(trk.samples), winmm.WAVE_MAPPER)
	return trk
}

func (trk *track) play() {
	trk.start = time.Now()
	for {
		trk.Lock()
		if trk.stopped {
			trk.Unlock()
			break
		}
		n, err := trk.dec.Read(trk.samples)
		trk.Unlock()
		if n > 0 {
			trk.wp.Write(trk.samples[:n])
		}
		if err != nil {
			break
		}
	}
	trk.wp.Sync()
	trk.wp.Close()
}

func (trk *track) pause(pause bool) bool {
	if trk.paused == pause {
		return false
	}
	trk.paused = pause

	if trk.paused {
		if !trk.start.IsZero() {
			trk.lost += time.Since(trk.start)
		}
	} else {
		trk.start = time.Now()
	}
	if trk.beRewind {
		trk.beRewind = false
		trk.wp.Stop()
		return true
	}
	trk.wp.Pause(trk.paused)
	return true
}

func (trk *track) stop() {
	trk.Lock()
	trk.stopped = true
	trk.wp.Stop()
	trk.Unlock()
}

func (trk *track) rewind(offset time.Duration) error {
	if trk.pause(true) {
		defer trk.pause(false)
	}

	trk.Lock()
	defer trk.Unlock()

	lost := trk.lost + offset
	if lost < 0 {
		lost = time.Duration(0)
	}
	lostInBytes := int64(float64(trk.sampleRate*trk.channels*2) * lost.Seconds())

	if _, err := trk.dec.Seek(lostInBytes, io.SeekStart); err != nil {
		return err
	}

	trk.lost = lost
	trk.beRewind = true
	return nil
}
