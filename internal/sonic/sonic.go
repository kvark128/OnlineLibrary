package sonic

//#cgo LDFLAGS: -lsonic
//#include <sonic.h>
import "C"

import (
	"errors"
	"io"
	"runtime"
	"unsafe"
)

type Stream struct {
	sampleSize int
	stream     C.sonicStream
}

func NewStream(sampleRate, numChannels int) *Stream {
	sonicStream := C.sonicCreateStream(C.int(sampleRate), C.int(numChannels))
	if sonicStream == nil {
		panic("sonicCreateStream returned NULL")
	}

	s := &Stream{
		sampleSize: numChannels * C.sizeof_short,
		stream:     sonicStream,
	}

	runtime.SetFinalizer(s, func(s *Stream) { C.sonicDestroyStream(s.stream) })
	return s
}

func (s *Stream) Write(data []byte) (int, error) {
	nSamples := len(data) / s.sampleSize
	if nSamples == 0 {
		return 0, nil
	}
	ok := C.sonicWriteShortToStream(s.stream, (*C.short)(unsafe.Pointer(&data[0])), C.int(nSamples))
	if ok == 0 {
		return 0, errors.New("memory realloc failed")
	}
	return nSamples * s.sampleSize, nil
}

func (s *Stream) Read(data []byte) (int, error) {
	nSamples := len(data) / s.sampleSize
	if nSamples == 0 {
		return 0, io.ErrShortBuffer
	}
	readSamples := C.sonicReadShortFromStream(s.stream, (*C.short)(unsafe.Pointer(&data[0])), C.int(nSamples))
	if readSamples == 0 {
		return 0, io.EOF
	}
	return int(readSamples) * s.sampleSize, nil
}

func (s *Stream) SamplesAvailable() int {
	nSamples := C.sonicSamplesAvailable(s.stream)
	return int(nSamples)
}

func (s *Stream) Speed() float64 {
	speed := C.sonicGetSpeed(s.stream)
	return float64(speed)
}

func (s *Stream) SetSpeed(speed float64) {
	C.sonicSetSpeed(s.stream, C.float(speed))
}

func (s *Stream) Pitch() float64 {
	pitch := C.sonicGetPitch(s.stream)
	return float64(pitch)
}

func (s *Stream) SetPitch(pitch float64) {
	C.sonicSetPitch(s.stream, C.float(pitch))
}

func (s *Stream) Rate() float64 {
	rate := C.sonicGetRate(s.stream)
	return float64(rate)
}

func (s *Stream) SetRate(rate float64) {
	C.sonicSetRate(s.stream, C.float(rate))
}

func (s *Stream) Volume() float64 {
	volume := C.sonicGetVolume(s.stream)
	return float64(volume)
}

func (s *Stream) SetVolume(volume float64) {
	C.sonicSetVolume(s.stream, C.float(volume))
}

func (s *Stream) NumChannels() int {
	numChannels := C.sonicGetNumChannels(s.stream)
	return int(numChannels)
}

func (s *Stream) SetNumChannels(numChannels int) {
	C.sonicSetNumChannels(s.stream, C.int(numChannels))
}

func (s *Stream) ChordPitch() bool {
	if C.sonicGetChordPitch(s.stream) == 0 {
		return false
	}
	return true
}

func (s *Stream) SetChordPitch(useChordPitch bool) {
	if useChordPitch {
		C.sonicSetChordPitch(s.stream, C.int(1))
	} else {
		C.sonicSetChordPitch(s.stream, C.int(0))
	}
}

func (s *Stream) Flush() int {
	return int(C.sonicFlushStream(s.stream))
}
