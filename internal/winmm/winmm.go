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

func mmcall(p *windows.LazyProc, args ...uintptr) error {
	mmrError, _, _ := p.Call(args...)
	if mmrError == MMSYSERR_NOERROR {
		return nil
	}

	// Buffer for description the error that occurred
	buf := make([]uint16, 256)

	r, _, _ := procWaveOutGetErrorTextW.Call(mmrError, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r != MMSYSERR_NOERROR {
		return fmt.Errorf("%v: unknown error: code %v", p.Name, mmrError)
	}
	return fmt.Errorf("%v: %v", p.Name, windows.UTF16PtrToString(&buf[0]))
}

type WavePlayer struct {
	wfx                           *WAVEFORMATEX
	preferredDevice               int
	preferredDeviceIsNotAvailable bool
	callMutex                     sync.Mutex
	waveout                       uintptr
	waveout_event                 windows.Handle
	waitMutex                     sync.Mutex
	prev_whdr                     *WAVEHDR
	buffers                       chan []byte
}

func NewWavePlayer(channels, samplesPerSec, bitsPerSample, buffSize, preferredDevice int) *WavePlayer {
	wp := &WavePlayer{
		preferredDevice: preferredDevice,
		buffers:         make(chan []byte, 2),
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
	if wp.open(wp.preferredDevice) != nil {
		wp.preferredDeviceIsNotAvailable = true
		wp.open(WAVE_MAPPER)
	}

	return wp
}

func (wp *WavePlayer) open(outputDevice int) error {
	var waveout uintptr
	err := mmcall(procWaveOutOpen, uintptr(unsafe.Pointer(&waveout)), uintptr(outputDevice), uintptr(unsafe.Pointer(wp.wfx)), uintptr(wp.waveout_event), 0, CALLBACK_EVENT)
	if err != nil {
		return err
	}

	if wp.waveout != 0 {
		mmcall(procWaveOutReset, wp.waveout)
		mmcall(procWaveOutClose, wp.waveout)
	}

	wp.waveout = waveout
	return nil
}

func (wp *WavePlayer) SetOutputDevice(outputDevice int) error {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	if err := wp.open(outputDevice); err != nil {
		return err
	}

	wp.preferredDevice = outputDevice
	wp.preferredDeviceIsNotAvailable = false
	return nil
}

func (wp *WavePlayer) feed(whdr *WAVEHDR) error {
	err := mmcall(procWaveOutPrepareHeader, wp.waveout, uintptr(unsafe.Pointer(whdr)), unsafe.Sizeof(*whdr))
	if err != nil {
		return err
	}
	return mmcall(procWaveOutWrite, wp.waveout, uintptr(unsafe.Pointer(whdr)), unsafe.Sizeof(*whdr))
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

	if wp.preferredDeviceIsNotAvailable {
		if wp.open(wp.preferredDevice) == nil {
			wp.preferredDeviceIsNotAvailable = false
		}
	}

	err := wp.feed(whdr)
	if err != nil && !wp.preferredDeviceIsNotAvailable && wp.preferredDevice != WAVE_MAPPER {
		wp.preferredDeviceIsNotAvailable = true
		wp.open(WAVE_MAPPER)
		err = wp.feed(whdr)
	}

	wp.callMutex.Unlock()

	if err != nil {
		return 0, err
	}

	wp.wait(whdr)
	return length, nil
}

func (wp *WavePlayer) Sync() {
	wp.wait(nil)
}

func (wp *WavePlayer) wait(whdr *WAVEHDR) {
	wp.waitMutex.Lock()
	defer wp.waitMutex.Unlock()

	if wp.prev_whdr != nil {
		for wp.prev_whdr.dwFlags&WHDR_DONE == 0 {
			windows.WaitForSingleObject(wp.waveout_event, windows.INFINITE)
		}

		wp.callMutex.Lock()
		mmcall(procWaveOutUnprepareHeader, wp.waveout, uintptr(unsafe.Pointer(wp.prev_whdr)), unsafe.Sizeof(*wp.prev_whdr))
		wp.callMutex.Unlock()
	}
	wp.prev_whdr = whdr
}

func (wp *WavePlayer) Pause(pauseState bool) {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	if pauseState {
		mmcall(procWaveOutPause, wp.waveout)
	} else {
		mmcall(procWaveOutRestart, wp.waveout)
	}
}

func (wp *WavePlayer) Position() (uint32, error) {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	pmmt := &MMTIME{WType: TIME_BYTES}
	err := mmcall(procWaveOutGetPosition, wp.waveout, uintptr(unsafe.Pointer(pmmt)), unsafe.Sizeof(*pmmt))
	if err != nil {
		return 0, err
	}

	if pmmt.WType != TIME_BYTES {
		return 0, errors.New("waveOutGetPosition: TIME_BYTES is not supported")
	}

	return pmmt.Cb, nil
}

func (wp *WavePlayer) Stop() {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	// Pausing first seems to make waveOutReset respond faster on some systems.
	mmcall(procWaveOutPause, wp.waveout)
	mmcall(procWaveOutReset, wp.waveout)
}

func (wp *WavePlayer) GetVolume() (uint16, uint16) {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	var volume uint32
	mmcall(procWaveOutGetVolume, wp.waveout, uintptr(unsafe.Pointer(&volume)))
	return uint16(volume), uint16(volume >> 16)
}

func (wp *WavePlayer) SetVolume(l, r uint16) {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	volume := uint32(r)<<16 + uint32(l)
	mmcall(procWaveOutSetVolume, wp.waveout, uintptr(volume))
}

func (wp *WavePlayer) Close() error {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	err := mmcall(procWaveOutClose, wp.waveout)
	windows.CloseHandle(wp.waveout_event)
	wp.waveout = 0
	wp.waveout_event = 0
	return err
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
