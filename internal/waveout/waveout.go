package waveout

import (
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
)

var winmm = windows.NewLazySystemDLL("winmm.dll")

// WaveOut functions
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
	procWaveOutGetErrorTextW   = winmm.NewProc("waveOutGetErrorTextW")
	procWaveOutClose           = winmm.NewProc("waveOutClose")
)

// Some win types
type WORD uint16
type DWORD uint32

// The WAVEFORMATEX structure defines the format of waveform-audio data
type WAVEFORMATEX struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

// The WAVEHDR structure defines the header used to identify a waveform-audio buffer
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

// The WAVEOUTCAPS structure describes the capabilities of a waveform-audio output device
type WAVEOUTCAPS struct {
	wMid           WORD
	wPid           WORD
	vDriverVersion uint32
	szPname        [MAXPNAMELEN]uint16
	dwFormats      DWORD
	wChannels      WORD
	wReserved1     WORD
	dwSupport      DWORD
}

func OutputDevices() func() (int, string, error) {
	caps := &WAVEOUTCAPS{}
	numDevs, _, _ := procWaveOutGetNumDevs.Call()
	devID := WAVE_MAPPER
	return func() (int, string, error) {
		if devID == int(numDevs) {
			return 0, "", fmt.Errorf("no output device")
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
			return 0, fmt.Errorf("no device name")
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
	return fmt.Errorf("%v: %v", p.Name, windows.UTF16ToString(buf))
}

type WavePlayer struct {
	wfx                 *WAVEFORMATEX
	preferredDeviceName string
	pause               bool
	callMutex           sync.Mutex
	currentDeviceID     int
	waveout             uintptr
	waveout_event       windows.Handle
	waitMutex           sync.Mutex
	prev_whdr           *WAVEHDR
	buffers             chan []byte
}

func NewWavePlayer(channels, samplesPerSec, bitsPerSample, bufSize int, preferredDeviceName string) (*WavePlayer, error) {
	wp := &WavePlayer{
		preferredDeviceName: preferredDeviceName,
		buffers:             make(chan []byte, 2),
	}

	wp.wfx = &WAVEFORMATEX{
		wFormatTag:      WAVE_FORMAT_PCM,
		nChannels:       uint16(channels),
		nSamplesPerSec:  uint32(samplesPerSec),
		wBitsPerSample:  uint16(bitsPerSample),
		nBlockAlign:     uint16(bitsPerSample / 8 * channels),
		nAvgBytesPerSec: uint32(bitsPerSample) / 8 * uint32(channels) * uint32(samplesPerSec),
	}

	// Completely fill the buffer channel
	for i := 0; i < cap(wp.buffers); i++ {
		wp.buffers <- make([]byte, bufSize)
	}

	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	wp.waveout_event = event

	if wp.openByName(wp.preferredDeviceName) != nil {
		if err := wp.openByID(WAVE_MAPPER); err != nil {
			return nil, err
		}
	}

	// WAVE_MAPPER cannot be the preferred device
	if wp.currentDeviceID == WAVE_MAPPER {
		wp.preferredDeviceName = ""
	}

	return wp, nil
}

func (wp *WavePlayer) openByName(devName string) error {
	devID, err := OutputDeviceNameToID(devName)
	if err != nil {
		return err
	}
	return wp.openByID(devID)
}

func (wp *WavePlayer) openByID(devID int) error {
	var waveout uintptr
	err := mmcall(procWaveOutOpen, uintptr(unsafe.Pointer(&waveout)), uintptr(devID), uintptr(unsafe.Pointer(wp.wfx)), uintptr(wp.waveout_event), 0, CALLBACK_EVENT)
	if err != nil {
		return err
	}

	if wp.waveout != 0 {
		mmcall(procWaveOutReset, wp.waveout)
		mmcall(procWaveOutClose, wp.waveout)
	}

	wp.waveout = waveout
	wp.currentDeviceID = devID

	if wp.pause {
		mmcall(procWaveOutPause, wp.waveout)
	}
	return nil
}

func (wp *WavePlayer) SetOutputDevice(devName string) error {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	if err := wp.openByName(devName); err != nil {
		return err
	}

	wp.preferredDeviceName = devName
	// WAVE_MAPPER cannot be the preferred device
	if wp.currentDeviceID == WAVE_MAPPER {
		wp.preferredDeviceName = ""
	}
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

	// Using WAVE_MAPPER instead of the preferred device means that it was previously disabled. Trying to restore it
	if wp.currentDeviceID == WAVE_MAPPER && wp.preferredDeviceName != "" {
		wp.openByName(wp.preferredDeviceName)
	}

	err := wp.feed(whdr)
	if err != nil && wp.currentDeviceID != WAVE_MAPPER {
		// Device was probably disconnected. Switch to WAVE_MAPPER and try again
		if wp.openByID(WAVE_MAPPER) == nil {
			err = wp.feed(whdr)
		}
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

	wp.pause = pauseState
	if wp.pause {
		mmcall(procWaveOutPause, wp.waveout)
	} else {
		mmcall(procWaveOutRestart, wp.waveout)
	}
}

func (wp *WavePlayer) Stop() {
	wp.callMutex.Lock()
	defer wp.callMutex.Unlock()

	// Pausing first seems to make waveOutReset respond faster on some systems.
	mmcall(procWaveOutPause, wp.waveout)
	mmcall(procWaveOutReset, wp.waveout)

	if wp.pause {
		mmcall(procWaveOutPause, wp.waveout)
	}
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
