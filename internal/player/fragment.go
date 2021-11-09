package player

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/waveout"
	"github.com/kvark128/minimp3"
	"github.com/kvark128/sonic"
)

type Fragment struct {
	sync.Mutex
	paused         bool
	beRewind       bool
	stream         *sonic.Stream
	dec            *minimp3.Decoder
	pcmBytesPerSec int
	wp             *waveout.WavePlayer
	nWrite         int64
}

const buffer_size = 1024 * 16

func NewFragment(mp3 io.Reader, speed, pitch float64, devName string) (*Fragment, int, error) {
	f := &Fragment{}
	f.dec = minimp3.NewDecoder(mp3)

	buf := make([]byte, 32)
	n, err := f.dec.Read(buf)
	if err != nil {
		return nil, 0, err
	}

	sampleRate := f.dec.SampleRate()
	channels := f.dec.Channels()
	bitrate := f.dec.Bitrate()
	f.pcmBytesPerSec = sampleRate * channels * 2

	if f.pcmBytesPerSec == 0 || bitrate == 0 {
		return nil, 0, fmt.Errorf("invalid mp3 format")
	}

	f.stream = sonic.NewStream(sampleRate, channels)
	f.stream.SetSpeed(speed)
	f.stream.SetPitch(pitch)
	f.stream.Write(buf[:n])

	wp, err := waveout.NewWavePlayer(channels, sampleRate, 16, buffer_size, devName)
	if err != nil {
		return nil, 0, err
	}
	f.wp = wp

	return f, bitrate, nil
}

func (f *Fragment) play(playing *util.Flag) {
	var p int64
	wp := bufio.NewWriterSize(f.wp, buffer_size)

	for playing.IsSet() {
		gui.SetElapsedTime(f.Position())
		n, err := io.CopyN(f.stream, f.dec, buffer_size)
		if err != nil {
			f.stream.Flush()
		}
		if _, err := wp.ReadFrom(f.stream); err != nil {
			log.Error("WavePlayer: %v", err)
		}
		f.nWrite += p
		p = n

		if err != nil {
			wp.Flush()
			break
		}
	}

	f.wp.Sync()
	f.nWrite += p
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
	return time.Second / time.Duration(f.pcmBytesPerSec) * time.Duration(f.nWrite)
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
		f.stream.Read(make([]byte, buffer_size))
	}

	offsetInBytes := int64(position / (time.Second / time.Duration(f.pcmBytesPerSec)))
	if f.nWrite == offsetInBytes {
		return nil
	}

	newPos, err := f.dec.Seek(offsetInBytes, io.SeekStart)
	if err != nil {
		return err
	}
	f.nWrite = newPos

	gui.SetElapsedTime(position)
	return nil
}
