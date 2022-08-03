package player

import (
	"bufio"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/util/syncio"
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
	wpBufSize      int
	Bitrate        int
	wp             *waveout.WavePlayer
	willBeStopped  bool
	pos            time.Duration
}

const BufferDuration = time.Millisecond * 400

func NewFragment(mp3Source io.Reader, devName string) (*Fragment, error) {
	dec := minimp3.NewDecoder(mp3Source)
	// Reading into an empty buffer will fill the internal buffer of the decoder, so you can get the audio data parameters
	if _, err := dec.Read(nil); err != nil {
		return nil, err
	}

	sampleRate := dec.SampleRate()
	channels := dec.Channels()
	bitrate := dec.Bitrate()
	pcmBytesPerSec := sampleRate * channels * 2

	if pcmBytesPerSec == 0 || bitrate == 0 {
		return nil, fmt.Errorf("invalid mp3")
	}

	wpBufSize := int(time.Duration(pcmBytesPerSec) * BufferDuration / time.Second)
	wp, err := waveout.NewWavePlayer(channels, sampleRate, 16, wpBufSize, devName)
	if err != nil {
		return nil, err
	}

	f := &Fragment{
		pcmBytesPerSec: pcmBytesPerSec,
		wpBufSize:      wpBufSize,
		Bitrate:        bitrate,
		stream:         sonic.NewStream(sampleRate, channels),
		dec:            dec,
		wp:             wp,
	}

	return f, nil
}

func (f *Fragment) play(playing *atomic.Bool, elapsedTimeCallback func(time.Duration)) error {
	var p time.Duration
	wp := bufio.NewWriterSize(f.wp, f.wpBufSize)
	stream := syncio.NewReadWriter(f.stream, f)
	dec := syncio.NewReader(f.dec, f)

	for playing.Load() {
		elapsedTimeCallback(f.Position())
		_, err := io.CopyN(stream, dec, int64(f.wpBufSize))
		if err != nil {
			if err != io.EOF {
				f.wp.Stop()
				return fmt.Errorf("copying from mp3 decoder to sonic stream: %w", err)
			}
			f.Lock()
			f.stream.Flush()
			f.Unlock()
		}
		if _, err := wp.ReadFrom(stream); err != nil {
			return fmt.Errorf("copying from sonic stream to wave player: %w", err)
		}
		f.Lock()
		if f.willBeStopped {
			p = 0
			f.willBeStopped = false
		} else {
			f.pos += p
			p = BufferDuration
		}
		f.Unlock()

		if err != nil {
			// Here err is always io.EOF
			wp.Flush()
			break
		}
	}

	f.wp.Sync()
	f.Lock()
	f.pos += p
	f.Unlock()
	elapsedTimeCallback(0)
	return nil
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

func (f *Fragment) SetPosition(pos time.Duration) error {
	f.Lock()
	defer f.Unlock()

	if pos < 0 {
		// Negative position means the beginning of the fragment
		pos = 0
	}

	if pos == f.pos {
		// Requested position has already been set
		return nil
	}

	f.wp.Stop()
	f.willBeStopped = true
	f.stream.Flush()
	io.ReadAll(f.stream)

	offset := int64(pos / (time.Second / time.Duration(f.pcmBytesPerSec)))
	_, err := f.dec.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	f.pos = pos
	return nil
}

func (f *Fragment) Position() time.Duration {
	f.Lock()
	defer f.Unlock()
	return f.pos
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

func (f *Fragment) Close() error {
	f.Lock()
	defer f.Unlock()
	f.wp.Stop()
	err := f.wp.Close()
	f.stream.Flush()
	io.ReadAll(f.stream)
	f.stream = nil
	f.dec = nil
	return err
}
