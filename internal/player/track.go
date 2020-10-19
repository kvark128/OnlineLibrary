package player

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/winmm"
	"github.com/kvark128/minimp3"
	"github.com/kvark128/sonic"
)

type track struct {
	sync.Mutex
	paused      bool
	beRewind    bool
	elapsedTime time.Duration
	start       time.Time
	stream      *sonic.Stream
	dec         *minimp3.Decoder
	sampleRate  int
	channels    int
	buffer      []byte
	wp          *winmm.WavePlayer
	trackSize   int64
}

func newTrack(mp3 io.Reader, speed, pitch float64, size int64) (*track, error) {
	trk := &track{}
	trk.dec = minimp3.NewDecoder(mp3)
	trk.buffer = make([]byte, 1024*16)
	n, err := trk.dec.Read(trk.buffer)
	if err != nil {
		return nil, err
	}
	trk.sampleRate, trk.channels, _, _, _, _ = trk.dec.LastFrameInfo()
	if trk.sampleRate == 0 || trk.channels == 0 {
		return nil, fmt.Errorf("invalid mp3: sampleRate=%v, channels=%v", trk.sampleRate, trk.channels)
	}
	trk.stream = sonic.NewStream(trk.sampleRate, trk.channels)
	trk.stream.SetSpeed(speed)
	trk.stream.SetPitch(pitch)
	trk.stream.Write(trk.buffer[:n])
	trk.wp = winmm.NewWavePlayer(trk.channels, trk.sampleRate, 16, len(trk.buffer), winmm.WAVE_MAPPER)
	trk.trackSize = size
	return trk, nil
}

func (trk *track) play(playing *util.Flag) {
	sampleRate, _, _, _, frameSize, samples := trk.dec.LastFrameInfo()
	if sampleRate > 0 && frameSize > 0 {
		seconds := time.Duration(trk.trackSize) / time.Duration(frameSize) * time.Duration(samples) / time.Duration(sampleRate)
		gui.SetTotalTime(time.Second * seconds)
	}

	var n int
	var err error
	trk.start = time.Now()
	for playing.IsSet() {
		for playing.IsSet() {
			gui.SetElapsedTime(trk.getElapsedTime())
			trk.Lock()
			nSamples := trk.stream.SamplesAvailable()
			if nSamples == 0 || (nSamples*trk.channels*2 < len(trk.buffer) && err == nil) {
				trk.Unlock()
				break
			}
			n, _ := trk.stream.Read(trk.buffer)
			trk.Unlock()
			trk.wp.Write(trk.buffer[:n])
		}
		if err != nil {
			break
		}
		trk.Lock()
		n, err = trk.dec.Read(trk.buffer)
		trk.stream.Write(trk.buffer[:n])
		if err != nil {
			trk.stream.Flush()
		}
		trk.Unlock()
	}

	trk.wp.Sync()
	trk.wp.Close()
}

func (trk *track) setSpeed(speed float64) {
	trk.Lock()
	defer trk.Unlock()
	if !trk.paused {
		trk.elapsedTime += time.Duration(float64(time.Since(trk.start)) * trk.stream.Speed())
		trk.start = time.Now()
	}
	trk.stream.SetSpeed(speed)
}

func (trk *track) setPitch(pitch float64) {
	trk.Lock()
	defer trk.Unlock()
	trk.stream.SetPitch(pitch)
}

func (trk *track) getElapsedTime() time.Duration {
	trk.Lock()
	defer trk.Unlock()
	if trk.paused {
		return trk.elapsedTime
	}
	return trk.elapsedTime + time.Duration(float64(time.Since(trk.start))*trk.stream.Speed())
}

func (trk *track) pause(pause bool) bool {
	trk.Lock()
	defer trk.Unlock()

	if trk.paused == pause {
		return false
	}
	trk.paused = pause

	if trk.paused {
		if !trk.start.IsZero() {
			trk.elapsedTime += time.Duration(float64(time.Since(trk.start)) * trk.stream.Speed())
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

func (trk *track) changeVolume(offset int) {
	l, r := trk.wp.GetVolume()
	newOffset := offset * 4096
	newL := int(l) + newOffset
	newR := int(r) + newOffset

	if newL < 0 {
		newL = 0
	}
	if newL > 0xffff {
		newL = 0xffff
	}

	if newR < 0 {
		newR = 0
	}
	if newR > 0xffff {
		newR = 0xffff
	}

	trk.wp.SetVolume(uint16(newL), uint16(newR))
}

func (trk *track) stop() {
	trk.wp.Stop()
}

func (trk *track) rewind(offset time.Duration) error {
	if offset == 0 {
		return nil
	}

	if trk.pause(true) {
		defer trk.pause(false)
	}

	trk.Lock()
	defer trk.Unlock()

	elapsedTime := trk.elapsedTime + offset
	if elapsedTime < 0 {
		elapsedTime = 0
	}
	offsetInBytes := int64(float64(trk.sampleRate*trk.channels*2) * elapsedTime.Seconds())

	if _, err := trk.dec.Seek(offsetInBytes, io.SeekStart); err != nil {
		return err
	}

	trk.stream.Flush()
	for trk.stream.SamplesAvailable() > 0 {
		trk.stream.Read(trk.buffer)
	}

	trk.elapsedTime = elapsedTime
	gui.SetElapsedTime(elapsedTime)
	trk.beRewind = true
	return nil
}
