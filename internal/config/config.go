package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/util"
	daisy "github.com/kvark128/daisyonline"
)

const (
	ProgramAuthor  = "Kvark <kvark128@yandex.ru>"
	ProgramName    = "OnlineLibrary"
	ProgramVersion = "2020.12.01"
	ConfigFile     = "config.json"
	LogFile        = "session.log"
	HTTPTimeout    = time.Second * 12
)

// Supported mime types of content
const (
	MP3_FORMAT = "audio/mpeg"
	LKF_FORMAT = "audio/x-lkf"
	LGK_FORMAT = "application/lgk"
)

// General client configuration of DAISY-online
var ReadingSystemAttributes = daisy.ReadingSystemAttributes{
	Manufacturer: ProgramAuthor,
	Model:        ProgramName,
	Version:      ProgramVersion,
	Config: daisy.Config{
		SupportsMultipleSelections:        false,
		PreferredUILanguage:               "ru-RU",
		SupportedContentFormats:           daisy.SupportedContentFormats{},
		SupportedContentProtectionFormats: daisy.SupportedContentProtectionFormats{},
		SupportedMimeTypes:                daisy.SupportedMimeTypes{MimeType: []daisy.MimeType{daisy.MimeType{Type: LKF_FORMAT}, daisy.MimeType{Type: LGK_FORMAT}, daisy.MimeType{Type: MP3_FORMAT}}},
		SupportedInputTypes:               daisy.SupportedInputTypes{Input: []daisy.Input{daisy.Input{Type: daisy.TEXT_ALPHANUMERIC}, daisy.Input{Type: daisy.AUDIO}}},
		RequiresAudioLabels:               false,
	},
}

var Conf Config

type Service struct {
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Credentials Credentials `json:"credentials"`
	RecentBooks BookSet     `json:"recent_books,omitempty"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type General struct {
	OutputDevice            string        `json:"output_device,omitempty"`
	Volume                  int           `json:"volume,omitempty"`
	PauseTimer              time.Duration `json:"pause_timer,omitempty"`
	LogLevel                string        `json:"log_level,omitempty"`
	OpenLocalBooksAtStartup bool          `json:"open_local_books_at_startup,omitempty"`
}

type Config struct {
	General    General    `json:"general,omitempty"`
	Services   []*Service `json:"services,omitempty"`
	LocalBooks BookSet    `json:"local_books,omitempty"`
}

func (cfg *Config) SetService(service *Service) {
	for _, srv := range cfg.Services {
		if service == srv {
			// The service already exists. Don't need to do anything
			return
		}
	}

	cfg.Services = append(cfg.Services, service)
	cfg.SetCurrentService(service)
}

func (cfg *Config) ServiceByName(name string) (*Service, error) {
	for _, srv := range cfg.Services {
		if name == srv.Name {
			return srv, nil
		}
	}
	return nil, errors.New("service with this name does not exist")
}

func (cfg *Config) RemoveService(service *Service) bool {
	for i, srv := range cfg.Services {
		if service == srv {
			copy(cfg.Services[i:], cfg.Services[i+1:])
			cfg.Services = cfg.Services[:len(cfg.Services)-1]
			return true
		}
	}
	return false
}

func (cfg *Config) SetCurrentService(service *Service) error {
	for i, srv := range cfg.Services {
		if service == srv {
			cfg.Services[0], cfg.Services[i] = cfg.Services[i], cfg.Services[0]
			return nil
		}
	}
	return errors.New("service does not exist")
}

func (cfg *Config) CurrentService() (*Service, error) {
	if len(cfg.Services) > 0 {
		// the current service is first in the list
		return cfg.Services[0], nil
	}
	return nil, errors.New("services list is empty")
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
		log.Error("Opening config file: %v", err)
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(c); err != nil {
		log.Error("Loading config: %v", err)
		return
	}

	log.Info("Loading config from %v", path)
}

func (c *Config) Save() {
	path := filepath.Join(UserData(), ConfigFile)
	f, err := util.CreateSecureFile(path)
	if err != nil {
		log.Error("Creating config file: %v", err)
		return
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t") // for readability
	if err := e.Encode(c); err != nil {
		f.Corrupted()
		log.Error("Writing to config file: %v", err)
		return
	}

	log.Info("Saving config to %v", path)
}
