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
	"github.com/kvark128/OnlineLibrary/internal/flag"
	"github.com/kvark128/OnlineLibrary/internal/lkf"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
)

// Supported mime types of content
const (
	MP3_FORMAT = "audio/mpeg"
	LKF_FORMAT = "audio/x-lkf"
)

type Player struct {
	sync.Mutex
	playList          []daisy.Resource
	book              daisy.ContentItem
	playing           *flag.Flag
	wg                *sync.WaitGroup
	pause             bool
	currentTrackIndex int
	trk               *track
	speed             float64
}

func NewPlayer(book daisy.ContentItem, resources []daisy.Resource) *Player {
	p := &Player{
		playing: new(flag.Flag),
		wg:      new(sync.WaitGroup),
		book:    book,
		speed:   1.0,
	}

	// The player supports only LKF and MP3 formats. Unsupported resources must not be uploaded to the player
	for _, r := range resources {
		if r.MimeType == LKF_FORMAT || r.MimeType == MP3_FORMAT {
			p.playList = append(p.playList, r)
		}
	}

	return p
}

func (p *Player) Book() *daisy.ContentItem {
	if p == nil {
		return nil
	}
	return &p.book
}

func (p *Player) ChangeSpeed(offset int) {
	if p == nil {
		return
	}
	p.Lock()
	p.speed = p.speed + (float64(offset) * 0.1)
	if p.speed < 0.5 {
		p.speed = 0.5
	}
	if p.speed > 2.0 {
		p.speed = 2.0
	}
	if p.trk != nil {
		p.trk.setSpeed(p.speed)
	}
	p.Unlock()
}

func (p *Player) ChangeTrack(offset int) {
	if p == nil {
		return
	}

	p.Lock()
	newTrackIndex := p.currentTrackIndex + offset
	p.Unlock()
	p.Play(newTrackIndex, 0)
}

func (p *Player) ChangeVolume(offset int) {
	if p == nil {
		return
	}

	p.Lock()
	defer p.Unlock()
	if p.trk == nil {
		return
	}

	l, r := p.trk.wp.GetVolume()
	newOffset := offset * 4096
	newL := int(l) + newOffset
	newR := int(r) + newOffset

	if newL < 0 {
		newL = 0
	}
	if newL > 0xffff {
		newL = 0xffff
	}

	if newR < 0 {
		newR = 0
	}
	if newR > 0xffff {
		newR = 0xffff
	}

	p.trk.wp.SetVolume(uint16(newL), uint16(newR))
}

func (p *Player) Rewind(offset time.Duration) {
	if p == nil {
		return
	}

	p.Lock()
	if p.trk != nil {
		err := p.trk.rewind(offset)
		if err != nil {
			log.Printf("rewind: %v", err)
		}
	}
	p.Unlock()
}

func (p *Player) Play(trackIndex int, offset time.Duration) {
	if p == nil {
		return
	}

	if trackIndex < 0 || trackIndex >= len(p.playList) {
		return
	}
	p.Stop()
	p.pause = false
	p.wg.Add(1)
	go p.start(trackIndex, offset)
}

func (p *Player) Pause() {
	if p == nil {
		return
	}

	p.Lock()
	p.pause = !p.pause
	if p.trk != nil {
		p.trk.pause(p.pause)
	}
	p.Unlock()
}

func (p *Player) Stop() {
	if p == nil {
		return
	}

	p.playing.Clear()
	p.Lock()
	var elapsedTime time.Duration
	if p.trk != nil {
		elapsedTime = p.trk.getElapsedTime()
		p.trk.stop()
	}
	config.Conf.Services[0].RecentBooks.Update(p.book.ID, p.currentTrackIndex, elapsedTime)
	p.Unlock()
	p.wg.Wait()
}

func (p *Player) start(trackIndex int, offset time.Duration) {
	defer p.wg.Done()
	defer p.playing.Clear()
	p.playing.Set()

	for i, track := range p.playList[trackIndex:] {
		var src io.ReadCloser
		var uri string
		var err error

		uri = filepath.Join(config.UserData(), util.ReplaceProhibitCharacters(p.book.Label.Text), track.LocalURI)
		if info, e := os.Stat(uri); e == nil {
			if !info.IsDir() && info.Size() == track.Size {
				// track already exist
				src, _ = os.Open(uri)
			}
		}

		if src == nil {
			// There is no track on the disc. Trying to get it from the network
			uri = track.URI
			src, err = connect.NewConnection(uri)
			if err != nil {
				log.Printf("Connection creating: %s\n", err)
				break
			}
		}

		var mp3 io.Reader
		switch track.MimeType {
		case LKF_FORMAT:
			mp3 = lkf.NewLKFReader(src)
		case MP3_FORMAT:
			mp3 = src
		default:
			panic("Unsupported MimeType")
		}

		p.Lock()
		if !p.playing.IsSet() {
			src.Close()
			p.Unlock()
			break
		}
		p.trk = newTrack(mp3, p.speed)
		if offset > 0 {
			p.trk.rewind(offset)
			offset = 0
		}
		p.currentTrackIndex = trackIndex + i
		p.Unlock()

		log.Printf("playing %s: %s", uri, track.MimeType)
		p.trk.play()
		src.Close()
		log.Printf("stopping %s: %s", uri, track.MimeType)

		if !p.playing.IsSet() {
			break
		}
	}
}
