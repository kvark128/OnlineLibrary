package player

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/connection"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/lkf"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
	"github.com/kvark128/sonic"
)

const (
	DEFAULT_SPEED = sonic.DEFAULT_SPEED
	MIN_SPEED     = sonic.DEFAULT_SPEED / 2
	MAX_SPEED     = sonic.DEFAULT_SPEED * 3

	DEFAULT_PITCH = sonic.DEFAULT_PITCH
	MIN_PITCH     = sonic.DEFAULT_PITCH / 2
	MAX_PITCH     = sonic.DEFAULT_PITCH * 3
)

// Extensions of supported formats
const (
	LKF_EXT = ".lkf"
	MP3_EXT = ".mp3"
)

type Player struct {
	sync.Mutex
	playList      []daisy.Resource
	bookID        string
	bookName      string
	playing       *util.Flag
	wg            *sync.WaitGroup
	fragment      *Fragment
	outputDevice  string
	speed         float64
	pitch         float64
	fragmentIndex int
	offset        time.Duration
	timerDuration time.Duration
	pauseTimer    *time.Timer
}

func NewPlayer(bookID, bookName string, resources []daisy.Resource, outputDevice string) *Player {
	p := &Player{
		playing:      new(util.Flag),
		wg:           new(sync.WaitGroup),
		bookID:       bookID,
		bookName:     bookName,
		speed:        DEFAULT_SPEED,
		pitch:        DEFAULT_PITCH,
		outputDevice: outputDevice,
	}

	// The player supports only LKF and MP3 formats. Unsupported resources must not be uploaded to the player
	// Some services specify an incorrect r.MimeType value, so we check the resource type by extension from the r.LocalURI field
	for _, r := range resources {
		ext := strings.ToLower(filepath.Ext(r.LocalURI))
		if ext == LKF_EXT || ext == MP3_EXT {
			p.playList = append(p.playList, r)
		}
	}

	return p
}

func (p *Player) SetTimerDuration(d time.Duration) {
	if p == nil {
		return
	}

	p.Lock()
	defer p.Unlock()
	p.timerDuration = d
	if p.playing.IsSet() && p.fragment != nil && !p.fragment.IsPause() {
		p.updateTimer(p.timerDuration)
	}
}

func (p *Player) TimerDuration() time.Duration {
	if p == nil {
		return 0
	}

	p.Lock()
	defer p.Unlock()
	return p.timerDuration
}

func (p *Player) updateTimer(d time.Duration) {
	if p.pauseTimer != nil {
		p.pauseTimer.Stop()
		p.pauseTimer = nil
	}
	if d > 0 {
		p.pauseTimer = time.AfterFunc(d, p.PlayPause)
	}
}

func (p *Player) BookInfo() (string, string) {
	if p == nil {
		return "", ""
	}
	p.Lock()
	defer p.Unlock()
	return p.bookName, p.bookID
}

func (p *Player) PositionInfo() (int, time.Duration) {
	if p == nil {
		return 0, 0
	}
	p.Lock()
	defer p.Unlock()
	elapsedTime := p.offset
	if p.fragment != nil {
		elapsedTime = p.fragment.Position()
	}
	return p.fragmentIndex, elapsedTime
}

func (p *Player) SetOutputDevice(outputDevice string) {
	if p == nil {
		return
	}

	if p.outputDevice == outputDevice {
		return
	}

	p.Lock()
	p.outputDevice = outputDevice
	if p.fragment != nil {
		p.fragment.SetOutputDevice(p.outputDevice)
	}
	p.Unlock()
}

func (p *Player) Speed() float64 {
	if p == nil {
		return 0.0
	}

	p.Lock()
	defer p.Unlock()
	return p.speed
}

func (p *Player) SetSpeed(speed float64) {
	if p == nil {
		return
	}
	p.Lock()
	defer p.Unlock()
	switch {
	case speed < MIN_SPEED:
		p.speed = MIN_SPEED
	case speed > MAX_SPEED:
		p.speed = MAX_SPEED
	default:
		p.speed = speed
	}
	if p.fragment != nil {
		p.fragment.setSpeed(p.speed)
	}
}

func (p *Player) Pitch() float64 {
	if p == nil {
		return 0.0
	}

	p.Lock()
	defer p.Unlock()
	return p.pitch
}

func (p *Player) SetPitch(pitch float64) {
	if p == nil {
		return
	}
	p.Lock()
	defer p.Unlock()
	switch {
	case pitch < MIN_PITCH:
		p.pitch = MIN_PITCH
	case pitch > MAX_PITCH:
		p.pitch = MAX_PITCH
	default:
		p.pitch = pitch
	}
	if p.fragment != nil {
		p.fragment.setPitch(p.pitch)
	}
}

func (p *Player) SetFragment(fragment int) {
	if p == nil {
		return
	}
	p.Lock()
	switch {
	case fragment < 0:
		p.fragmentIndex = 0
	case fragment >= len(p.playList):
		p.Unlock()
		return
	default:
		p.fragmentIndex = fragment
	}
	p.Unlock()
	if p.playing.IsSet() {
		p.Stop()
		p.PlayPause()
	}
}

func (p *Player) ChangeVolume(offset int) {
	if p == nil {
		return
	}
	p.Lock()
	defer p.Unlock()
	if p.fragment != nil {
		p.fragment.changeVolume(offset)
	}
}

func (p *Player) SetPosition(position time.Duration) {
	if p == nil {
		return
	}

	p.Lock()
	defer p.Unlock()
	if !p.playing.IsSet() {
		p.offset = position
		return
	}
	if p.fragment != nil {
		if err := p.fragment.SetPosition(position); err != nil {
			log.Info("set fragment position: %v", err)
		}
	}
}

func (p *Player) PlayPause() {
	if p == nil {
		return
	}
	p.Lock()
	defer p.Unlock()
	if !p.playing.IsSet() {
		p.playing.Set()
		p.wg.Add(1)
		go p.start(p.fragmentIndex)
	} else if p.fragment != nil {
		p.updateTimer(0)
		if !p.fragment.pause(true) {
			p.fragment.pause(false)
			p.updateTimer(p.timerDuration)
		}
	}
}

func (p *Player) Stop() {
	if p == nil {
		return
	}
	defer p.wg.Wait()
	p.Lock()
	defer p.Unlock()
	p.playing.Clear()
	p.offset = 0
	if p.fragment != nil {
		p.fragment.stop()
	}
}

func (p *Player) start(startFragment int) {
	defer p.wg.Done()
	defer p.playing.Clear()

	p.updateTimer(p.timerDuration)
	defer p.updateTimer(0)

	for index, r := range p.playList[startFragment:] {
		var src io.ReadCloser
		var uri string
		var err error

		uri = filepath.Join(config.UserData(), util.ReplaceForbiddenCharacters(p.bookName), r.LocalURI)
		if info, e := os.Stat(uri); e == nil {
			if !info.IsDir() && info.Size() == r.Size {
				// fragment already exists on disk
				// we use it to reduce the load on the server
				src, _ = os.Open(uri)
			}
		}

		if src == nil {
			// There is no fragment on the disc. Trying to get it from the network
			uri = r.URI
			src, err = connection.NewConnection(uri)
			if err != nil {
				log.Info("Connection creating: %v", err)
				break
			}
		}

		if strings.ToLower(filepath.Ext(r.LocalURI)) == LKF_EXT {
			src = lkf.NewReader(src)
		}

		p.Lock()
		speed := p.speed
		pitch := p.pitch
		offset := p.offset
		outputDevice := p.outputDevice
		p.Unlock()

		fragment, kbps, err := NewFragment(src, speed, pitch, outputDevice)
		if err != nil {
			log.Info("new fragment for %v: %v", uri, err)
			src.Close()
			continue
		}

		if err := fragment.SetPosition(offset); err != nil {
			log.Info("set fragment position: %v", err)
			src.Close()
			continue
		}

		if !p.playing.IsSet() {
			src.Close()
			break
		}

		p.Lock()
		p.fragment = fragment
		p.fragmentIndex = startFragment + index
		gui.SetFragments(p.fragmentIndex, len(p.playList))
		gui.SetTotalTime(time.Second * time.Duration(r.Size/int64(kbps*1000/8)))
		p.offset = 0
		p.Unlock()

		fragment.play(p.playing)
		src.Close()

		p.Lock()
		p.fragment = nil
		p.Unlock()

		if !p.playing.IsSet() {
			break
		}
	}
}
