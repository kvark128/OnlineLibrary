package player

//#include <sonic.h>
import "C"

import (
	"io"
	"sync"
	"time"
	"unsafe"

	"github.com/kvark128/OnlineLibrary/internal/winmm"
	"github.com/kvark128/minimp3"
)

type track struct {
	sync.Mutex
	stopped     bool
	paused      bool
	beRewind    bool
	elapsedTime time.Duration
	start       time.Time
	stream      C.sonicStream
	dec         *minimp3.Decoder
	sampleRate  int
	channels    int
	samples     []byte
	speed       float64
	wp          *winmm.WavePlayer
}

func newTrack(mp3 io.Reader, speed float64) *track {
	trk := &track{}
	trk.dec = minimp3.NewDecoder(mp3)
	trk.dec.Read([]byte{}) // Reads first frame
	trk.sampleRate, trk.channels, _, _ = trk.dec.Info()
	trk.stream = C.sonicCreateStream(C.int(trk.sampleRate), C.int(trk.channels))
	C.sonicSetSpeed(trk.stream, C.float(speed))
	trk.speed = speed
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
		if n > 0 {
			C.sonicWriteShortToStream(trk.stream, (*C.short)(unsafe.Pointer(&trk.samples[0])), C.int(n/(trk.channels*2)))
		}
		if err != nil {
			C.sonicFlushStream(trk.stream)
		}
		trk.Unlock()
		for {
			trk.Lock()
			if trk.stopped {
				trk.Unlock()
				break
			}
			nSamples := C.sonicReadShortFromStream(trk.stream, (*C.short)(unsafe.Pointer(&trk.samples[0])), C.int(4096/(trk.channels*2)))
			trk.Unlock()
			if nSamples == 0 {
				break
			}
			trk.wp.Write(trk.samples[:nSamples*C.int(trk.channels*2)])
		}
		if err != nil {
			break
		}
	}

	trk.wp.Sync()
	C.sonicDestroyStream(trk.stream)
	trk.wp.Close()
}

func (trk *track) setSpeed(speed float64) {
	trk.Lock()
	C.sonicSetSpeed(trk.stream, C.float(speed))
	trk.elapsedTime += time.Duration(float64(time.Since(trk.start)) * trk.speed)
	trk.start = time.Now()
	trk.speed = speed
	trk.Unlock()
}

func (trk *track) getElapsedTime() time.Duration {
	trk.Lock()
	defer trk.Unlock()
	if trk.paused {
		return trk.elapsedTime
	}
	return trk.elapsedTime + time.Duration(float64(time.Since(trk.start))*trk.speed)
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
			trk.elapsedTime += time.Duration(float64(time.Since(trk.start)) * trk.speed)
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

	elapsedTime := trk.elapsedTime + offset
	if elapsedTime < 0 {
		elapsedTime = time.Duration(0)
	}
	offsetInBytes := int64(float64(trk.sampleRate*trk.channels*2) * elapsedTime.Seconds())

	if _, err := trk.dec.Seek(offsetInBytes, io.SeekStart); err != nil {
		return err
	}

	for {
		nSamples := C.sonicReadShortFromStream(trk.stream, (*C.short)(unsafe.Pointer(&trk.samples[0])), C.int(len(trk.samples)/(trk.channels*2)))
		if nSamples == 0 {
			break
		}
	}

	trk.elapsedTime = elapsedTime
	trk.beRewind = true
	return nil
}
