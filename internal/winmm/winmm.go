package winmm

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	WAVE_FORMAT_PCM  = 1
	WAVE_MAPPER      = -1
	WHDR_DONE        = 1
	CALLBACK_EVENT   = 0x50000
	MAXPNAMELEN      = 32
	MMSYSERR_NOERROR = 0
	TIME_BYTES       = 4
)

var winmm = windows.NewLazySystemDLL("winmm.dll")
var ErrClosed = errors.New("wave player already closed")

// Output
var (
	procWaveOutGetNumDevs      = winmm.NewProc("waveOutGetNumDevs")
	procWaveOutGetDevCapsW     = winmm.NewProc("waveOutGetDevCapsW")
	procWaveOutOpen            = winmm.NewProc("waveOutOpen")
	procWaveOutPrepareHeader   = winmm.NewProc("waveOutPrepareHeader")
	procWaveOutUnprepareHeader = winmm.NewProc("waveOutUnprepareHeader")
	procWaveOutWrite           = winmm.NewProc("waveOutWrite")
	procWaveOutPause           = winmm.NewProc("waveOutPause")
	procWaveOutRestart         = winmm.NewProc("waveOutRestart")
	procWaveOutReset           = winmm.NewProc("waveOutReset")
	procWaveOutGetVolume       = winmm.NewProc("waveOutGetVolume")
	procWaveOutSetVolume       = winmm.NewProc("waveOutSetVolume")
	procWaveOutGetPosition     = winmm.NewProc("waveOutGetPosition")
	procWaveOutGetErrorTextW   = winmm.NewProc("waveOutGetErrorTextW")
	procWaveOutClose           = winmm.NewProc("waveOutClose")
)

// Input
var (
	procWaveInOpen            = winmm.NewProc("waveInOpen")
	procWaveInPrepareHeader   = winmm.NewProc("waveInPrepareHeader")
	procWaveInUnprepareHeader = winmm.NewProc("waveInUnprepareHeader")
	procWaveInAddBuffer       = winmm.NewProc("waveInAddBuffer")
	procWaveInStart           = winmm.NewProc("waveInStart")
	procWaveInStop            = winmm.NewProc("waveInStop")
	procWaveInClose           = winmm.NewProc("waveInClose")
)

type WAVEFORMATEX struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

type WAVEHDR struct {
	lpData          uintptr
	dwBufferLength  uint32
	dwBytesRecorded uint32
	dwUser          uintptr
	dwFlags         uint32
	dwLoops         uint32
	lpNext          uintptr
	reserved        uintptr
}

type WAVEOUTCAPS struct {
	wMid           uint16
	wPid           uint16
	vDriverVersion uint
	szPname        [MAXPNAMELEN]uint16
	dwFormats      uint32
	wChannels      uint16
	wReserved1     uint16
	dwSupport      uint32
}

type MMTIME struct {
	WType uint
	Cb    uint32
	pad   uint32
}

func OutputDevices() func() (int, string, error) {
	caps := &WAVEOUTCAPS{}
	numDevs, _, _ := procWaveOutGetNumDevs.Call()
	devID := WAVE_MAPPER
	return func() (int, string, error) {
		if devID == int(numDevs) {
			return 0, "", errors.New("no output device")
		}
		procWaveOutGetDevCapsW.Call(uintptr(devID), uintptr(unsafe.Pointer(caps)), unsafe.Sizeof(*caps))
		devID++
		return devID - 1, windows.UTF16ToString(caps.szPname[:MAXPNAMELEN]), nil
	}
}

func OutputDeviceNames() []string {
	names := make([]string, 0)
	device := OutputDevices()
	for {
		_, name, err := device()
		if err != nil {
			return names
		}
		names = append(names, name)
	}
}

func OutputDeviceNameToID(devName string) (int, error) {
	device := OutputDevices()
	for {
		id, name, err := device()
		if err != nil {
			return 0, errors.New("no device name")
		}
		if devName == name {
			return id, nil
		}
	}
}

type WavePlayer struct {
	wfx           *WAVEFORMATEX
	outputDevice  int
	callMutex     sync.Mutex
	waveout       uintptr
	waveout_event windows.Handle
	prev_whdr     *WAVEHDR
	buffers       chan []byte
}

func NewWavePlayer(channels, samplesPerSec, bitsPerSample, buffSize, outputDevice int) *WavePlayer {
	wp := &WavePlayer{
		outputDevice: outputDevice,
		buffers:      make(chan []byte, 2),
	}

	wp.wfx = &WAVEFORMATEX{
		wFormatTag:      WAVE_FORMAT_PCM,
		nChannels:       uint16(channels),
		nSamplesPerSec:  uint32(samplesPerSec),
		wBitsPerSample:  uint16(bitsPerSample),
		nBlockAlign:     uint16(bitsPerSample / 8 * channels),
		nAvgBytesPerSec: uint32(bitsPerSample) / 8 * uint32(channels) * uint32(samplesPerSec),
	}

	wp.buffers <- make([]byte, buffSize)
	wp.buffers <- make([]byte, buffSize)

	wp.waveout_event, _ = windows.CreateEvent(nil, 0, 0, nil)
	procWaveOutOpen.Call(uintptr(unsafe.Pointer(&wp.waveout)), uintptr(wp.outputDevice), uintptr(unsafe.Pointer(wp.wfx)), uintptr(wp.waveout_event), 0, CALLBACK_EVENT)
	return wp
}

func (wp *WavePlayer) SetOutputDevice(outputDevice int) error {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	if wp.waveout != 0 {
		procWaveOutClose.Call(wp.waveout)
		wp.waveout = 0
	}

	wp.outputDevice = outputDevice
	procWaveOutOpen.Call(uintptr(unsafe.Pointer(&wp.waveout)), uintptr(wp.outputDevice), uintptr(unsafe.Pointer(wp.wfx)), uintptr(wp.waveout_event), 0, CALLBACK_EVENT)
	return nil
}

func (wp *WavePlayer) error(mmrError uintptr) error {
	buf := make([]uint16, 256)
	wp.callMutex.Lock()
	r, _, _ := procWaveOutGetErrorTextW.Call(mmrError, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	wp.callMutex.Unlock()
	if r != MMSYSERR_NOERROR {
		return fmt.Errorf("unknown error: %v", mmrError)
	}
	return errors.New(windows.UTF16PtrToString(&buf[0]))
}

func (wp *WavePlayer) Write(data []byte) (int, error) {
	buffer := <-wp.buffers
	defer func() { wp.buffers <- buffer }()
	length := copy(buffer, data)
	if length == 0 {
		return 0, nil
	}

	whdr := &WAVEHDR{
		lpData:         uintptr(unsafe.Pointer(&buffer[0])),
		dwBufferLength: uint32(length),
	}

	wp.callMutex.Lock()
	r, _, _ := procWaveOutPrepareHeader.Call(wp.waveout, uintptr(unsafe.Pointer(whdr)), unsafe.Sizeof(*whdr))
	if r == MMSYSERR_NOERROR {
		r, _, _ = procWaveOutWrite.Call(wp.waveout, uintptr(unsafe.Pointer(whdr)), unsafe.Sizeof(*whdr))
	}
	wp.callMutex.Unlock()

	wp.Sync()

	if r != MMSYSERR_NOERROR {
		return 0, wp.error(r)
	}

	wp.prev_whdr = whdr
	return length, nil
}

func (wp *WavePlayer) Sync() {
	if wp.prev_whdr == nil {
		return
	}

	for wp.prev_whdr.dwFlags&WHDR_DONE == 0 {
		windows.WaitForSingleObject(wp.waveout_event, windows.INFINITE)
	}

	wp.callMutex.Lock()
	procWaveOutUnprepareHeader.Call(wp.waveout, uintptr(unsafe.Pointer(wp.prev_whdr)), unsafe.Sizeof(*wp.prev_whdr))
	wp.callMutex.Unlock()
	wp.prev_whdr = nil
}

func (wp *WavePlayer) Pause(pauseState bool) {
	wp.callMutex.Lock()
	if pauseState {
		procWaveOutPause.Call(wp.waveout)
	} else {
		procWaveOutRestart.Call(wp.waveout)
	}
	wp.callMutex.Unlock()
}

func (wp *WavePlayer) Position() (uint32, error) {
	pmmt := &MMTIME{WType: TIME_BYTES}
	wp.callMutex.Lock()
	r, _, _ := procWaveOutGetPosition.Call(wp.waveout, uintptr(unsafe.Pointer(pmmt)), unsafe.Sizeof(*pmmt))
	wp.callMutex.Unlock()

	if r != MMSYSERR_NOERROR {
		return 0, wp.error(r)
	}

	if pmmt.WType != TIME_BYTES {
		return 0, errors.New("waveOutGetPosition: TIME_BYTES is not supported")
	}

	return pmmt.Cb, nil
}

func (wp *WavePlayer) Stop() {
	// Pausing first seems to make waveOutReset respond faster on some systems.
	wp.callMutex.Lock()
	procWaveOutPause.Call(wp.waveout)
	procWaveOutReset.Call(wp.waveout)
	wp.callMutex.Unlock()
}

func (wp *WavePlayer) GetVolume() (uint16, uint16) {
	var volume uint32
	wp.callMutex.Lock()
	procWaveOutGetVolume.Call(wp.waveout, uintptr(unsafe.Pointer(&volume)))
	wp.callMutex.Unlock()
	return uint16(volume), uint16(volume >> 16)
}

func (wp *WavePlayer) SetVolume(l, r uint16) {
	volume := uint32(r)<<16 + uint32(l)
	wp.callMutex.Lock()
	procWaveOutSetVolume.Call(wp.waveout, uintptr(volume))
	wp.callMutex.Unlock()
}

func (wp *WavePlayer) Close() error {
	wp.callMutex.Lock()
	procWaveOutClose.Call(wp.waveout)
	wp.callMutex.Unlock()

	windows.CloseHandle(wp.waveout_event)
	wp.waveout = 0
	wp.waveout_event = 0
	return nil
}

type WaveRecorder struct {
	wfx          *WAVEFORMATEX
	inputDevice  int
	wavein       uintptr
	wavein_event windows.Handle
	prev_whdr    *WAVEHDR
	buffer       []byte
	chanBuffers  chan []byte
}

func NewWaveRecorder(channels, samplesPerSec, bitsPerSample, bufSize, inputDevice int) *WaveRecorder {
	wr := &WaveRecorder{
		inputDevice: inputDevice,
		chanBuffers: make(chan []byte, 2),
	}

	wr.wfx = &WAVEFORMATEX{
		wFormatTag:      WAVE_FORMAT_PCM,
		nChannels:       uint16(channels),
		nSamplesPerSec:  uint32(samplesPerSec),
		wBitsPerSample:  uint16(bitsPerSample),
		nBlockAlign:     uint16(bitsPerSample / 8 * channels),
		nAvgBytesPerSec: uint32(bitsPerSample) / 8 * uint32(channels) * uint32(samplesPerSec),
	}

	wr.chanBuffers <- make([]byte, bufSize)
	wr.chanBuffers <- make([]byte, bufSize)

	wr.wavein_event, _ = windows.CreateEvent(nil, 0, 0, nil)
	wr.Open()
	return wr
}

func (wr *WaveRecorder) Open() {
	if wr.wavein == 0 {
		procWaveInOpen.Call(uintptr(unsafe.Pointer(&wr.wavein)), uintptr(wr.inputDevice), uintptr(unsafe.Pointer(wr.wfx)), uintptr(wr.wavein_event), 0, CALLBACK_EVENT)
	}
}

func (wr *WaveRecorder) Read(data []byte) (int, error) {
	if wr.wavein == 0 {
		return 0, ErrClosed
	}

	var n = 0
	for buffer := range wr.chanBuffers {
		wr.chanBuffers <- buffer

		whdr := &WAVEHDR{
			lpData:         uintptr(unsafe.Pointer(&buffer[0])),
			dwBufferLength: uint32(len(buffer)),
		}

		procWaveInPrepareHeader.Call(wr.wavein, uintptr(unsafe.Pointer(whdr)), unsafe.Sizeof(*whdr))
		procWaveInAddBuffer.Call(wr.wavein, uintptr(unsafe.Pointer(whdr)), unsafe.Sizeof(*whdr))

		if wr.prev_whdr != nil {
			for wr.prev_whdr.dwFlags&WHDR_DONE == 0 {
				windows.WaitForSingleObject(wr.wavein_event, windows.INFINITE)
			}
			procWaveInUnprepareHeader.Call(wr.wavein, uintptr(unsafe.Pointer(wr.prev_whdr)), unsafe.Sizeof(*wr.prev_whdr))
			n += copy(data[n:], wr.buffer)
		} else {
			procWaveInStart.Call(wr.wavein)
		}

		wr.prev_whdr = whdr
		wr.buffer = buffer

		if n == len(data) {
			break
		}
	}
	return n, nil
}

func (wr *WaveRecorder) Close() error {
	if wr.wavein == 0 {
		return ErrClosed
	}

	procWaveInClose.Call(wr.wavein)
	windows.CloseHandle(wr.wavein_event)
	wr.wavein = 0
	wr.wavein_event = 0
	return nil
}
