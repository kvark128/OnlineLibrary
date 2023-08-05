package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/util"
	"github.com/kvark128/dodp"
	"gopkg.in/yaml.v3"
)

var (
	ServiceNotFound = errors.New("service not found")
)

const (
	ProgramAuthor      = "Alexander Linkov <kvark128@yandex.ru>"
	ProgramName        = "OnlineLibrary"
	ProgramVersion     = "2023.06.12"
	ProgramDescription = "DAISY Online Client"
	CopyrightInfo      = "Copyright (C) 2020 - 2022 Alexander Linkov"
	ConfigFile         = "config.yaml"
	LogFile            = "session.log"
	MessageBufferSize  = 16
	HTTPTimeout        = time.Second * 12
	LocalStorageID     = "localstorage"
	MetadataFileName   = "metadata.xml"
)

// Supported mime types of content
const (
	MP3_FORMAT = "audio/mpeg"
	LKF_FORMAT = "audio/x-lkf"
	LGK_FORMAT = "application/lgk"
)

// General client configuration of DAISY-online
var ReadingSystemAttributes = dodp.ReadingSystemAttributes{
	Manufacturer: ProgramAuthor,
	Model:        ProgramName,
	Version:      ProgramVersion,
	Config: dodp.Config{
		SupportsMultipleSelections:        false,
		PreferredUILanguage:               "ru-RU",
		SupportedContentFormats:           dodp.SupportedContentFormats{},
		SupportedContentProtectionFormats: dodp.SupportedContentProtectionFormats{},
		SupportedMimeTypes:                dodp.SupportedMimeTypes{MimeType: []dodp.MimeType{dodp.MimeType{Type: LKF_FORMAT}, dodp.MimeType{Type: LGK_FORMAT}, dodp.MimeType{Type: MP3_FORMAT}}},
		SupportedInputTypes:               dodp.SupportedInputTypes{Input: []dodp.Input{dodp.Input{Type: dodp.TEXT_ALPHANUMERIC}, dodp.Input{Type: dodp.AUDIO}}},
		RequiresAudioLabels:               false,
	},
}

// Cached path to the user data directory
var userDataPath string

func UserData() string {
	if userDataPath != "" {
		return userDataPath
	}
	if path, err := filepath.Abs(ProgramName); err == nil {
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				userDataPath = path
			}
		}
	}
	if userDataPath == "" {
		userDataPath = filepath.Join(os.Getenv("USERPROFILE"), ProgramName)
	}
	return userDataPath
}

// BookDir returns the full path to the book directory by it name
func BookDir(name string) (string, error) {
	name = util.ReplaceForbiddenCharacters(name)
	// Windows does not allow trailing dots and spaces in a directory name
	name = strings.TrimRight(name, ". ")
	// Whitespace around the edges of a directory name is a very bad thing
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return "", fmt.Errorf("book directory is invalid")
	}
	return filepath.Join(UserData(), name), nil
}

type Service struct {
	ID                   string  `yaml:"id"`
	Name                 string  `yaml:"name"`
	URL                  string  `yaml:"url"`
	Username             string  `yaml:"username"`
	Password             string  `yaml:"password"`
	OpenBookshelfOnLogin bool    `yaml:"open_bookshelf_on_login"`
	RecentBooks          BookSet `yaml:"books,omitempty"`
}

type General struct {
	OutputDevice string        `yaml:"output_device,omitempty"`
	Language     string        `yaml:"language,omitempty"`
	Volume       float64       `yaml:"volume,omitempty"`
	PauseTimer   time.Duration `yaml:"pause_timer,omitempty"`
	LogLevel     string        `yaml:"log_level,omitempty"`
	Provider     string        `yaml:"provider,omitempty"`
}

type Config struct {
	General    General    `yaml:"general,omitempty"`
	Services   []*Service `yaml:"services,omitempty"`
	LocalBooks BookSet    `yaml:"local_books,omitempty"`
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

func (cfg *Config) ServiceByID(id string) (*Service, error) {
	for _, srv := range cfg.Services {
		if srv.ID == id {
			return srv, nil
		}
	}
	return nil, ServiceNotFound
}

func (cfg *Config) ServiceByName(name string) (*Service, error) {
	for _, srv := range cfg.Services {
		if name == srv.Name {
			return srv, nil
		}
	}
	return nil, ServiceNotFound
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

func NewConfig() *Config {
	cfg := new(Config)
	cfg.General.Volume = 1.0
	return cfg
}

func (cfg *Config) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	return d.Decode(cfg)
}

func (cfg *Config) Save(path string) error {
	f, err := util.CreateSecureFile(path)
	if err != nil {
		return err
	}
	defer f.Close()
	e := yaml.NewEncoder(f)
	if err := e.Encode(cfg); err != nil {
		f.Corrupted()
		return err
	}
	return nil
}
