package player

import (
	"fmt"
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
	"github.com/kvark128/OnlineLibrary/internal/util/buffer"
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

// Error returned when stopping playback at user request
var PlayingIsStopped = fmt.Errorf("playing is stopped")

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
	p.Lock()
	defer p.Unlock()
	p.timerDuration = d
	log.Debug("timer of player is set to %v", d)
	if p.playing.IsSet() && p.fragment != nil && !p.fragment.IsPause() {
		p.updateTimer(p.timerDuration)
	}
}

func (p *Player) TimerDuration() time.Duration {
	p.Lock()
	defer p.Unlock()
	return p.timerDuration
}

func (p *Player) updateTimer(d time.Duration) {
	if p.pauseTimer != nil {
		p.pauseTimer.Stop()
		p.pauseTimer = nil
		log.Debug("timer of player is stopped")
	}
	if d > 0 {
		p.pauseTimer = time.AfterFunc(d, p.PlayPause)
		log.Debug("timer of player is started on %v", d)
	}
}

func (p *Player) Position() time.Duration {
	p.Lock()
	defer p.Unlock()
	if p.fragment != nil {
		return p.fragment.Position()
	}
	return p.offset
}

func (p *Player) SetPosition(position time.Duration) {
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

func (p *Player) Fragment() int {
	p.Lock()
	defer p.Unlock()
	return p.fragmentIndex
}

func (p *Player) SetFragment(fragment int) {
	p.Lock()
	defer p.Unlock()
	if fragment >= len(p.playList) {
		// Attempt was made to start a non-existent track. Do nothing
		return
	}
	if fragment < 0 {
		fragment = 0
	}
	p.fragmentIndex = fragment
	if p.playing.IsSet() {
		p.stopPlayback()
		p.startPlayback()
	}
}

// Returns the name of the preferred audio device
func (p *Player) OutputDevice() string {
	p.Lock()
	defer p.Unlock()
	return p.outputDevice
}

// Sets the name of the preferred audio device
func (p *Player) SetOutputDevice(outputDevice string) {
	p.Lock()
	defer p.Unlock()
	if p.outputDevice == outputDevice {
		// The required output device is already is set
		return
	}
	p.outputDevice = outputDevice
	if p.fragment != nil {
		p.fragment.SetOutputDevice(p.outputDevice)
	}
}

func (p *Player) Speed() float64 {
	p.Lock()
	defer p.Unlock()
	return p.speed
}

func (p *Player) SetSpeed(speed float64) {
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

func (p *Player) Volume() float64 {
	p.Lock()
	defer p.Unlock()
	return p.volume
}

func (p *Player) SetVolume(volume float64) {
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

func (p *Player) PlayPause() {
	p.Lock()
	defer p.Unlock()
	if !p.playing.IsSet() {
		p.startPlayback()
		return
	}
	if p.fragment != nil {
		p.updateTimer(0)
		if !p.fragment.pause(true) {
			p.fragment.pause(false)
			p.updateTimer(p.timerDuration)
		}
	}
}

func (p *Player) Stop() {
	p.Lock()
	defer p.Unlock()
	p.stopPlayback()
}

func (p *Player) startPlayback() {
	go p.playback(p.fragmentIndex)
}

func (p *Player) stopPlayback() {
	p.playing.Clear()
	p.offset = 0
	if p.fragment != nil {
		p.fragment.stop()
	}
}

func (p *Player) playback(startFragment int) {
	log.Debug("starting playback with fragment %v. Waiting other fragments", startFragment)
	p.wg.Wait()
	log.Debug("fragment %v is started", startFragment)

	p.wg.Add(1)
	p.playing.Set()
	defer p.wg.Done()
	defer p.playing.Clear()

	p.updateTimer(p.timerDuration)
	defer p.updateTimer(0)

	for index, r := range p.playList[startFragment:] {
		log.Debug("fetching resource %v: MimeType %v; Size %v", r.LocalURI, r.MimeType, r.Size)

		err := func(r daisy.Resource) error {
			var src io.ReadSeekCloser
			localPath := filepath.Join(p.bookDir, r.LocalURI)

			if util.FileIsExist(localPath, r.Size) {
				// The fragment already exists on the local disk
				// We must use it to avoid making network requests
				var err error
				src, err = os.Open(localPath)
				if err != nil {
					return fmt.Errorf("opening local fragment: %w", err)
				}
				log.Debug("opening the local fragment from %v", localPath)
			} else {
				// There is no fragment on the local disc. Trying to get it from the network
				var err error
				src, err = connection.NewConnection(r.URI)
				if err != nil {
					return fmt.Errorf("connection creating: %w", err)
				}
				log.Debug("fetching the fragment by network from %v", r.URI)
			}
			defer src.Close()

			fragment, err := func(src io.ReadSeeker) (*Fragment, error) {
				src = buffer.NewReader(src)
				if strings.ToLower(filepath.Ext(r.LocalURI)) == LKF_EXT {
					src = lkf.NewReader(src)
				}

				fragment, err := NewFragment(src, p.OutputDevice())
				if err != nil {
					return nil, fmt.Errorf("fragment creating: %w", err)
				}

				if err := fragment.SetPosition(p.Position()); err != nil {
					return nil, fmt.Errorf("set fragment position: %w", err)
				}

				fragment.setSpeed(p.Speed())
				fragment.setVolume(p.Volume())
				return fragment, nil
			}(src)

			if err != nil {
				return err
			}
			defer fragment.Close()

			// Fragment creation is an I/O operation and can be time consuming. We have to check that the fragment was not stopped by the user
			if !p.playing.IsSet() {
				return PlayingIsStopped
			}

			p.Lock()
			p.fragment = fragment
			p.fragmentIndex = startFragment + index
			gui.SetFragments(p.fragmentIndex, len(p.playList))
			gui.SetTotalTime(time.Second * time.Duration(r.Size/int64(p.fragment.Bitrate*1000/8)))
			p.offset = 0
			p.Unlock()

			err = fragment.play(p.playing)
			p.Lock()
			p.fragment = nil
			p.Unlock()

			if err != nil {
				return fmt.Errorf("fragment playing: %w", err)
			}

			if !p.playing.IsSet() {
				return PlayingIsStopped
			}
			return nil
		}(r)

		if err != nil {
			log.Warning("playing of %v: %v", r.LocalURI, err)
			break
		}
	}
}
