package player

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kvark128/av3715/internal/config"
	"github.com/kvark128/av3715/internal/connect"
	"github.com/kvark128/av3715/internal/flag"
	"github.com/kvark128/av3715/internal/lkf"
	"github.com/kvark128/av3715/internal/winmm"
	daisy "github.com/kvark128/daisyonline"
	"github.com/kvark128/minimp3"
)

// Supported mime types of content
const (
	MP3_FORMAT = "audio/mpeg"
	LKF_FORMAT = "audio/x-lkf"
	LGK_FORMAT = "application/lgk"
)

type Player struct {
	sync.Mutex
	playList          []daisy.Resource
	book              string
	playing           *flag.Flag
	wg                *sync.WaitGroup
	pause             bool
	src               io.ReadCloser
	dec               io.Seeker
	wp                *winmm.WavePlayer
	currentTrackIndex int
}

func NewPlayer(book string, r *daisy.Resources) *Player {
	p := &Player{
		playing:  new(flag.Flag),
		wg:       new(sync.WaitGroup),
		playList: r.Resources,
		book:     book,
	}
	return p
}

func (p *Player) ChangeTrack(offset int) {
	if p == nil {
		return
	}

	p.Lock()
	newTrackIndex := p.currentTrackIndex + offset
	p.Unlock()
	p.Play(newTrackIndex)
}

func (p *Player) ChangeVolume(offset int) {
	if p == nil {
		return
	}

	p.Lock()
	defer p.Unlock()
	if p.wp == nil {
		return
	}

	l, r := p.wp.GetVolume()
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

	p.wp.SetVolume(uint16(newL), uint16(newR))
}

func (p *Player) Rewind(offset time.Duration) {
	if p == nil {
		return
	}

	p.Lock()
	if p.dec != nil {
		t := 22050.0 * 2.0 * offset.Seconds()
		p.dec.Seek(int64(t), io.SeekCurrent)
	}
	p.Unlock()
}

func (p *Player) Play(trackIndex int) {
	if p == nil {
		return
	}

	if trackIndex < 0 || trackIndex >= len(p.playList) {
		return
	}
	p.Stop()
	p.pause = false
	p.wg.Add(1)
	go p.start(trackIndex)
}

func (p *Player) Pause() {
	if p == nil {
		return
	}

	p.Lock()
	p.pause = !p.pause
	if p.wp != nil {
		p.wp.Pause(p.pause)
	}
	p.Unlock()
}

func (p *Player) Stop() {
	if p == nil {
		return
	}

	p.playing.Clear()
	p.Lock()
	if p.src != nil {
		p.src.Close()
	}
	if p.wp != nil {
		p.wp.Stop()
	}
	p.Unlock()
	p.wg.Wait()
}

func (p *Player) start(trackIndex int) {
	defer p.wg.Done()
	defer p.playing.Clear()
	for i, track := range p.playList[trackIndex:] {
		var src io.ReadCloser
		var uri string
		var err error

		uri = filepath.Join(config.Conf.UserData, p.book, track.LocalURI)
		if info, e := os.Stat(uri); e == nil {
			if !info.IsDir() && info.Size() == int64(track.Size) {
				// track already exist
				src, _ = os.Open(uri)
			}
		}

		if src == nil {
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
			log.Printf("Pass %s: %s", uri, track.MimeType)
			continue
		}

		log.Printf("Playing %s: %s", uri, track.MimeType)
		dec := minimp3.NewDecoder(mp3)
		dec.Read([]byte{}) // Reads first frame
		SampleRate, Channels, _, _ := dec.Info()
		samples := make([]byte, SampleRate*Channels*2) // buffer for 1 second

		p.Lock()
		p.src = src
		p.dec = dec
		p.wp = winmm.NewWavePlayer(Channels, SampleRate, 16, len(samples), winmm.WAVE_MAPPER)
		p.currentTrackIndex = trackIndex + i
		p.Unlock()

		p.playing.Set()
		for p.playing.IsSet() {
			n, err := dec.Read(samples)
			if n > 0 {
				p.wp.Write(samples[:n])
			}
			if err != nil {
				break
			}
		}

		p.wp.Sync()

		p.Lock()
		p.src.Close()
		p.wp.Close()
		p.src = nil
		p.dec = nil
		p.wp = nil
		p.Unlock()

		if !p.playing.IsSet() {
			break
		}
	}
}
