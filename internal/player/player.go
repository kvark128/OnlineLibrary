package player

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/connection"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/lkf"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
)

const (
	DEFAULT_SPEED = 1.0
	MIN_SPEED     = 0.5
	MAX_SPEED     = 3.0

	DEFAULT_VOLUME = 1.0
	MIN_VOLUME     = 0.05
	MAX_VOLUME     = 1.5
)

// Extensions of supported formats
const (
	LKF_EXT = ".lkf"
	MP3_EXT = ".mp3"
)

type Player struct {
	sync.Mutex
	playList      []daisy.Resource
	bookDir       string
	playing       *util.Flag
	wg            *sync.WaitGroup
	fragment      *Fragment
	outputDevice  string
	speed         float64
	volume        float64
	fragmentIndex int
	offset        time.Duration
	timerDuration time.Duration
	pauseTimer    *time.Timer
}

func NewPlayer(bookDir string, resources []daisy.Resource, outputDevice string) *Player {
	p := &Player{
		playing:      new(util.Flag),
		wg:           new(sync.WaitGroup),
		bookDir:      bookDir,
		speed:        DEFAULT_SPEED,
		volume:       DEFAULT_VOLUME,
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
		speed = MIN_SPEED
	case speed > MAX_SPEED:
		speed = MAX_SPEED
	}
	p.speed = speed
	if p.fragment != nil {
		p.fragment.setSpeed(p.speed)
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

func (p *Player) SetVolume(volume float64) {
	if p == nil {
		return
	}
	p.Lock()
	defer p.Unlock()
	switch {
	case volume < MIN_VOLUME:
		volume = MIN_VOLUME
	case volume > MAX_VOLUME:
		volume = MAX_VOLUME
	}
	p.volume = volume
	if p.fragment != nil {
		p.fragment.setVolume(p.volume)
	}
}

func (p *Player) Volume() float64 {
	if p == nil {
		return 0.0
	}
	p.Lock()
	defer p.Unlock()
	return p.volume
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
			log.Error("Set fragment position: %v", err)
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
		log.Debug("Fetching resource: %s; URI: %s; MimeType: %s; Size: %d", r.LocalURI, r.URI, r.MimeType, r.Size)

		uri = filepath.Join(p.bookDir, r.LocalURI)
		if util.FileIsExist(uri, r.Size) {
			// fragment already exists on disk
			// we use it to reduce the load on the server
			src, _ = os.Open(uri)
		}

		if src == nil {
			// There is no fragment on the disc. Trying to get it from the network
			uri = r.URI
			src, err = connection.NewConnection(uri)
			if err != nil {
				log.Error("Connection creating: %v", err)
				break
			}
		}

		if strings.ToLower(filepath.Ext(r.LocalURI)) == LKF_EXT {
			src = lkf.NewReader(src)
		}

		p.Lock()
		speed := p.speed
		volume := p.volume
		offset := p.offset
		outputDevice := p.outputDevice
		p.Unlock()

		fragment, err := NewFragment(src, outputDevice)
		if err != nil {
			log.Error("New fragment for %v: %v", uri, err)
			src.Close()
			continue
		}

		fragment.setSpeed(speed)
		fragment.setVolume(volume)

		if err := fragment.SetPosition(offset); err != nil {
			log.Error("Set fragment position: %v", err)
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
		gui.SetTotalTime(time.Second * time.Duration(r.Size/int64(p.fragment.Bitrate*1000/8)))
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
