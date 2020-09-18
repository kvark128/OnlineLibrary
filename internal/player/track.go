package player

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/kvark128/av3715/internal/winmm"
	"github.com/kvark128/minimp3"
)

type track struct {
	sync.Mutex
	stopped     bool
	lost        time.Duration
	lostSamples int64
	start       time.Time
	dec         *minimp3.Decoder
	sampleRate  int
	channels    int
	samples     []byte
	wp          *winmm.WavePlayer
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
		trk.lostSamples += int64(n / (trk.channels*2))
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

func (trk *track) pause(pause bool) {
	if pause {
		trk.lost += time.Since(trk.start)
	} else {
		trk.start = time.Now()
	}
	trk.wp.Pause(pause)
}

func (trk *track) stop() {
	trk.Lock()
	trk.stopped = true
	trk.wp.Stop()
	trk.Unlock()
}

func (trk *track) rewind(offset time.Duration) {
	trk.pause(true)
	trk.Lock()
	lostFromDec := time.Second / time.Duration(trk.sampleRate) * time.Duration(trk.lostSamples)
	newOffset := offset - (lostFromDec - trk.lost)
	trk.lost += offset
	bytesOffset := trk.sampleRate * trk.channels * 2 * int(newOffset.Seconds())
	pos, err := trk.dec.Seek(int64(bytesOffset), io.SeekCurrent)
	if err != nil {
		log.Printf("rewind: %v", err)
		return
	}
	trk.lostSamples = pos / int64(trk.channels*2)
	trk.wp.Stop()
	trk.Unlock()
}
