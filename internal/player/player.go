package player

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/connect"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/lkf"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
	"github.com/kvark128/sonic"
)

const (
	DEFAULT_SPEED = sonic.DEFAULT_SPEED
	MIN_SPEED     = sonic.DEFAULT_SPEED / 2
	MAX_SPEED     = sonic.DEFAULT_SPEED * 3
)

type Player struct {
	sync.Mutex
	playList []daisy.Resource
	bookID   string
	bookName string
	playing  *util.Flag
	wg       *sync.WaitGroup
	trk      *track
	speed    float64
	fragment int
	offset   time.Duration
}

func NewPlayer(bookID, bookName string, resources []daisy.Resource) *Player {
	p := &Player{
		playing:  new(util.Flag),
		wg:       new(sync.WaitGroup),
		bookID:   bookID,
		bookName: bookName,
		speed:    DEFAULT_SPEED,
	}

	// The player supports only LKF and MP3 formats. Unsupported resources must not be uploaded to the player
	for _, r := range resources {
		if r.MimeType == config.LKF_FORMAT || r.MimeType == config.MP3_FORMAT {
			p.playList = append(p.playList, r)
		}
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

func (p *Player) ChangeTrack(offset int) {
	if p == nil {
		return
	}
	p.Lock()
	newFragment := p.fragment + offset
	p.Unlock()
	p.SetTrack(newFragment)
}

func (p *Player) SetTrack(fragment int) {
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

func (p *Player) ChangeOffset(offset time.Duration) {
	if p == nil {
		return
	}
	p.Lock()
	defer p.Unlock()
	if !p.playing.IsSet() {
		p.offset += offset
		return
	}
	if p.trk != nil {
		if err := p.trk.rewind(offset); err != nil {
			log.Printf("rewind: %v", err)
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

	for i, r := range p.playList[startFragment:] {
		var src io.ReadCloser
		var uri string
		var err error

		uri = filepath.Join(config.UserData(), util.ReplaceProhibitCharacters(p.bookName), r.LocalURI)
		if info, e := os.Stat(uri); e == nil {
			if !info.IsDir() && info.Size() == r.Size {
				// track already exist
				src, _ = os.Open(uri)
			}
		}

		if src == nil {
			// There is no track on the disc. Trying to get it from the network
			uri = r.URI
			src, err = connect.NewConnection(uri)
			if err != nil {
				log.Printf("Connection creating: %s\n", err)
				break
			}
		}

		var mp3 io.Reader
		switch r.MimeType {
		case config.LKF_FORMAT:
			mp3 = lkf.NewLKFReader(src)
		case config.MP3_FORMAT:
			mp3 = src
		default:
			src.Close()
			panic("Unsupported MimeType")
		}

		p.Lock()
		speed := p.speed
		offset := p.offset
		p.Unlock()

		trk, err := newTrack(mp3, speed, r.Size)
		if err != nil {
			log.Printf("new track for %v: %v", uri, err)
			src.Close()
			continue
		}

		if err := trk.rewind(offset); err != nil {
			log.Printf("track rewind: %v", err)
			src.Close()
			continue
		}

		if !p.playing.IsSet() {
			src.Close()
			break
		}

		p.Lock()
		p.trk = trk
		p.fragment += i
		currentFragment := p.fragment // copy for gui.SetFragments
		p.offset = 0
		p.Unlock()

		log.Printf("playing %s: %s", uri, r.MimeType)
		gui.SetFragments(currentFragment, len(p.playList))
		trk.play(p.playing)
		src.Close()
		log.Printf("stopping %s: %s", uri, r.MimeType)

		p.Lock()
		p.trk = nil
		p.Unlock()

		if !p.playing.IsSet() {
			break
		}
	}
}
