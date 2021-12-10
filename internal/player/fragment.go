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
	stream         *sonic.Stream
	dec            *minimp3.Decoder
	pcmBytesPerSec int
	Bitrate        int
	wp             *waveout.WavePlayer
	nWrite         int64
}

const buffer_size = 1024 * 16

func NewFragment(mp3 io.Reader, devName string) (*Fragment, error) {
	dec := minimp3.NewDecoder(mp3)
	// Reading into an empty buffer will fill the internal buffer of the decoder, so you can get the audio data parameters
	if _, err := dec.Read([]byte{}); err != nil {
		return nil, err
	}

	sampleRate := dec.SampleRate()
	channels := dec.Channels()
	bitrate := dec.Bitrate()
	pcmBytesPerSec := sampleRate * channels * 2

	if pcmBytesPerSec == 0 || bitrate == 0 {
		return nil, fmt.Errorf("invalid mp3 format")
	}

	wp, err := waveout.NewWavePlayer(channels, sampleRate, 16, buffer_size, devName)
	if err != nil {
		return nil, err
	}

	f := &Fragment{
		pcmBytesPerSec: pcmBytesPerSec,
		Bitrate:        bitrate,
		stream:         sonic.NewStream(sampleRate, channels),
		dec:            dec,
		wp:             wp,
	}

	return f, nil
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

func (f *Fragment) setVolume(volume float64) {
	f.Lock()
	defer f.Unlock()
	f.stream.SetVolume(volume)
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

	f.wp.Pause(f.paused)
	return true
}

func (f *Fragment) IsPause() bool {
	f.Lock()
	defer f.Unlock()
	return f.paused
}

func (f *Fragment) SetOutputDevice(devName string) error {
	return f.wp.SetOutputDevice(devName)
}

func (f *Fragment) stop() {
	f.wp.Stop()
}

func (f *Fragment) SetPosition(position time.Duration) error {
	f.Lock()
	defer f.Unlock()

	if position < 0 {
		position = 0
	}

	f.wp.Stop()
	f.stream.Flush()
	io.ReadAll(f.stream)

	offset := int64(position / (time.Second / time.Duration(f.pcmBytesPerSec)))
	if f.nWrite == offset {
		return nil
	}

	newPos, err := f.dec.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	f.nWrite = newPos

	gui.SetElapsedTime(position)
	return nil
}
