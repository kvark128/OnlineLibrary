package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	ProgramAuthor  = "Kvark <kvark128@yandex.ru>"
	ProgramName    = "OnlineLibrary"
	ProgramVersion = "2020.12.01"
	ConfigFile     = "config.json"
	LogFile        = "session.log"
)

// Supported mime types of content
const (
	MP3_FORMAT = "audio/mpeg"
	LKF_FORMAT = "audio/x-lkf"
	LGK_FORMAT = "application/lgk"
)

var Conf Config

type Book struct {
	Name        string        `json:"name"`
	ID          string        `json:"id"`
	Fragment    int           `json:"fragment"`
	ElapsedTime time.Duration `json:"elapsed_time"`
}

type Service struct {
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Credentials Credentials `json:"credentials"`
	RecentBooks []Book      `json:"recent_books,omitempty"`
}

func (s *Service) SetBook(id, name string, fragment int, elapsedTime time.Duration) {
	for i := range s.RecentBooks {
		if s.RecentBooks[i].ID == id {
			s.RecentBooks[i].Name = name
			s.RecentBooks[i].Fragment = fragment
			s.RecentBooks[i].ElapsedTime = elapsedTime
			s.SetCurrentBook(id)
			return
		}
	}

	book := Book{
		Name:        name,
		ID:          id,
		Fragment:    fragment,
		ElapsedTime: elapsedTime,
	}

	s.RecentBooks = append(s.RecentBooks, book)
	s.SetCurrentBook(id)

	if len(s.RecentBooks) > 256 {
		s.RecentBooks = s.RecentBooks[:256]
	}
}

func (s *Service) Book(id string) Book {
	for _, b := range s.RecentBooks {
		if b.ID == id {
			return b
		}
	}
	return Book{}
}

func (s *Service) SetCurrentBook(id string) {
	for i, b := range s.RecentBooks {
		if b.ID == id {
			copy(s.RecentBooks[1:i+1], s.RecentBooks[0:i])
			s.RecentBooks[0] = b
			break
		}
	}
}

func (s *Service) CurrentBook() Book {
	if len(s.RecentBooks) != 0 {
		return s.RecentBooks[0]
	}
	return Book{}
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type General struct {
	OutputDevice string `json:"output_device"`
}

type Config struct {
	General  General   `json:"general,omitempty"`
	Services []Service `json:"services,omitempty"`
}

func (cfg *Config) SetService(service *Service) {
	cfg.Services = append(cfg.Services, *service)
	cfg.SetCurrentService(len(cfg.Services) - 1)
}

func (cfg *Config) Service(index int) *Service {
	return &cfg.Services[index]
}

func (cfg *Config) RemoveService(index int) {
	copy(cfg.Services[index:], cfg.Services[index+1:])
	cfg.Services = cfg.Services[:len(cfg.Services)-1]
}

func (cfg *Config) SetCurrentService(index int) {
	cfg.Services[0], cfg.Services[index] = cfg.Services[index], cfg.Services[0]
}

func (cfg *Config) CurrentService() (*Service, int, error) {
	if len(cfg.Services) > 0 {
		return &cfg.Services[0], 0, nil
	}
	return &Service{}, 0, fmt.Errorf("services list is empty")
}

func UserData() string {
	if path, err := filepath.Abs(ProgramName); err == nil {
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				return path
			}
		}
	}
	return filepath.Join(os.Getenv("USERPROFILE"), ProgramName)
}

func (c *Config) Load() {
	os.MkdirAll(UserData(), os.ModeDir)

	path := filepath.Join(UserData(), ConfigFile)
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Opening config file: %v", err)
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(c); err != nil {
		log.Printf("Loading config: %v", err)
		return
	}
	log.Printf("Loading config from %v", path)
}

func (c *Config) Save() {
	path := filepath.Join(UserData(), ConfigFile)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("Creating config file: %v", err)
		return
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t") // for readability
	if err := e.Encode(c); err != nil {
		log.Printf("Saving config: %v", err)
		return
	}
	log.Printf("Saving config to %v", path)
}
