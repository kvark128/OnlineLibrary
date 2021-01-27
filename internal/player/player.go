package player

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/connection"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/lkf"
	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/OnlineLibrary/internal/winmm"
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
	playList       []daisy.Resource
	bookID         string
	bookName       string
	playing        *util.Flag
	wg             *sync.WaitGroup
	trk            *track
	outputDeviceID int
	speed          float64
	pitch          float64
	fragment       int
	offset         time.Duration
}

func NewPlayer(bookID, bookName string, resources []daisy.Resource, outputDevice string) *Player {
	p := &Player{
		playing:        new(util.Flag),
		wg:             new(sync.WaitGroup),
		bookID:         bookID,
		bookName:       bookName,
		speed:          DEFAULT_SPEED,
		pitch:          DEFAULT_PITCH,
		outputDeviceID: winmm.WAVE_MAPPER,
	}

	// The player supports only LKF and MP3 formats. Unsupported resources must not be uploaded to the player
	// Some services specify an incorrect r.MimeType value, so we check the resource type by extension from the r.LocalURI field
	for _, r := range resources {
		ext := strings.ToLower(filepath.Ext(r.LocalURI))
		if ext == LKF_EXT || ext == MP3_EXT {
			p.playList = append(p.playList, r)
		}
	}

	if devID, err := winmm.OutputDeviceNameToID(outputDevice); err == nil {
		p.outputDeviceID = devID
	}

	return p
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
	if p.trk != nil {
		elapsedTime = p.trk.getElapsedTime()
	}
	return p.fragment, elapsedTime
}

func (p *Player) SetOutputDevice(outputDevice string) {
	if p == nil {
		return
	}

	devID, err := winmm.OutputDeviceNameToID(outputDevice)
	if err != nil || p.outputDeviceID == devID {
		return
	}

	_, offset := p.PositionInfo()
	p.Stop()
	p.Lock()
	p.outputDeviceID = devID
	p.offset = offset
	p.Unlock()
	p.PlayPause()
}

func (p *Player) ChangeSpeed(offset float64) {
	if p == nil {
		return
	}
	p.Lock()
	newSpeed := p.speed + offset
	p.Unlock()
	p.SetSpeed(newSpeed)
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
	if p.trk != nil {
		p.trk.setSpeed(p.speed)
	}
}

func (p *Player) ChangePitch(offset float64) {
	if p == nil {
		return
	}
	p.Lock()
	newPitch := p.pitch + offset
	p.Unlock()
	p.SetPitch(newPitch)
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
	if p.trk != nil {
		p.trk.setPitch(p.pitch)
	}
}

func (p *Player) SetFragment(fragment int) {
	if p == nil {
		return
	}
	p.Lock()
	switch {
	case fragment < 0:
		p.fragment = 0
	case fragment >= len(p.playList):
		p.Unlock()
		return
	default:
		p.fragment = fragment
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
	if p.trk != nil {
		p.trk.changeVolume(offset)
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
	if p.trk != nil {
		if err := p.trk.setPosition(position); err != nil {
			log.Printf("set fragment position: %v", err)
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
		go p.start(p.fragment)
	} else if p.trk != nil {
		if !p.trk.pause(true) {
			p.trk.pause(false)
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
	if p.trk != nil {
		p.trk.stop()
	}
}

func (p *Player) start(startFragment int) {
	defer p.wg.Done()
	defer p.playing.Clear()

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
				log.Printf("Connection creating: %v", err)
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
		outputDeviceID := p.outputDeviceID
		p.Unlock()

		trk, kbps, err := newTrack(src, speed, pitch, outputDeviceID)
		if err != nil {
			log.Printf("new track for %v: %v", uri, err)
			src.Close()
			continue
		}

		if err := trk.setPosition(offset); err != nil {
			log.Printf("set fragment position: %v", err)
			src.Close()
			continue
		}

		if !p.playing.IsSet() {
			src.Close()
			break
		}

		p.Lock()
		p.trk = trk
		p.fragment = startFragment + index
		gui.SetFragments(p.fragment, len(p.playList))
		gui.SetTotalTime(time.Second * time.Duration(r.Size/int64(kbps*1000/8)))
		p.offset = 0
		p.Unlock()

		trk.play(p.playing)
		src.Close()

		p.Lock()
		p.trk = nil
		p.Unlock()

		if !p.playing.IsSet() {
			break
		}
	}
}
