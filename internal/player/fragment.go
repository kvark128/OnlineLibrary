package player

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/winmm"
	"github.com/kvark128/minimp3"
	"github.com/kvark128/sonic"
)

type Fragment struct {
	sync.Mutex
	paused     bool
	beRewind   bool
	stream     *sonic.Stream
	dec        *minimp3.Decoder
	sampleRate int
	channels   int
	buffer     []byte
	wp         *winmm.WavePlayer
	nWrite     int64
}

func NewFragment(mp3 io.Reader, speed, pitch float64, devName string) (*Fragment, int, error) {
	f := &Fragment{}
	f.dec = minimp3.NewDecoder(mp3)

	f.buffer = make([]byte, 1024*16)
	n, err := f.dec.Read(f.buffer)
	if err != nil {
		return nil, 0, err
	}

	var kbps int
	f.sampleRate, f.channels, kbps, _, _, _ = f.dec.LastFrameInfo()
	if f.sampleRate == 0 || f.channels == 0 || kbps == 0 {
		return nil, 0, fmt.Errorf("invalid mp3 format")
	}

	f.stream = sonic.NewStream(f.sampleRate, f.channels)
	f.stream.SetSpeed(speed)
	f.stream.SetPitch(pitch)
	f.stream.Write(f.buffer[:n])

	wp, err := winmm.NewWavePlayer(f.channels, f.sampleRate, 16, len(f.buffer), devName)
	if err != nil {
		return nil, 0, err
	}
	f.wp = wp

	return f, kbps, nil
}

func (f *Fragment) fillBuf(buf []byte) (int, error) {
	f.Lock()
	defer f.Unlock()

	var err error
	var n int

	for {
		nSamples := f.stream.SamplesAvailable()
		if err != nil || nSamples*f.channels*2 >= len(buf) {
			break
		}
		n, err = f.dec.Read(buf)
		f.stream.Write(buf[:n])
		if err != nil {
			f.stream.Flush()
		}
	}

	nRead, _ := f.stream.Read(buf)
	return nRead, err
}

func (f *Fragment) play(playing *util.Flag) {
	var p int
	for playing.IsSet() {
		gui.SetElapsedTime(f.Position())
		n, err := f.fillBuf(f.buffer)
		if _, err := f.wp.Write(f.buffer[:n]); err != nil {
			log.Printf("wavePlayer: %v", err)
		}
		f.nWrite += int64(float64(p) * f.stream.Speed())
		p = n

		if err != nil {
			break
		}
	}

	f.wp.Sync()
	f.nWrite += int64(float64(p) * f.stream.Speed())
	f.wp.Close()
	gui.SetElapsedTime(0)
}

func (f *Fragment) setSpeed(speed float64) {
	f.Lock()
	defer f.Unlock()
	f.stream.SetSpeed(speed)
}

func (f *Fragment) setPitch(pitch float64) {
	f.Lock()
	defer f.Unlock()
	f.stream.SetPitch(pitch)
}

func (f *Fragment) Position() time.Duration {
	f.Lock()
	defer f.Unlock()
	return time.Second / time.Duration(f.sampleRate*f.channels*2) * time.Duration(f.nWrite)
}

func (f *Fragment) pause(pause bool) bool {
	f.Lock()
	defer f.Unlock()

	if f.paused == pause {
		return false
	}
	f.paused = pause

	if f.beRewind {
		f.beRewind = false
		f.wp.Stop()
		return true
	}
	f.wp.Pause(f.paused)
	return true
}

func (f *Fragment) IsPause() bool {
	f.Lock()
	defer f.Unlock()
	return f.paused
}

func (f *Fragment) changeVolume(offset int) {
	l, r := f.wp.GetVolume()
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

	f.wp.SetVolume(uint16(newL), uint16(newR))
}

func (f *Fragment) SetOutputDevice(devName string) error {
	return f.wp.SetOutputDevice(devName)
}

func (f *Fragment) stop() {
	f.wp.Stop()
}

func (f *Fragment) SetPosition(position time.Duration) error {
	if f.pause(true) {
		defer f.pause(false)
	}

	f.Lock()
	defer f.Unlock()

	if position < 0 {
		position = 0
	}

	f.beRewind = true
	f.stream.Flush()
	for f.stream.SamplesAvailable() > 0 {
		f.stream.Read(f.buffer)
	}

	offsetInBytes := int64(position / (time.Second / time.Duration(f.sampleRate*f.channels*2)))
	if f.nWrite == offsetInBytes {
		return nil
	}
	f.nWrite = offsetInBytes

	if _, err := f.dec.Seek(offsetInBytes, io.SeekStart); err != nil {
		return err
	}

	gui.SetElapsedTime(position)
	return nil
}
