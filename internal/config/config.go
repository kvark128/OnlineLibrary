package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	ConfigFile  = "config.json"
	ProgramName = "OnlineLibrary"
	LogFile     = "session.log"
)

var Conf Config

type Book struct {
	ID          string        `json:"id"`
	Fragment    int           `json:"fragment"`
	ElapsedTime time.Duration `json:"elapsed_time"`
}

type Service struct {
	Name        string      `json:"name"`
	URL         string      `json:"url"`
	Credentials Credentials `json:"credentials"`
	RecentBooks RecentBooks `json:"recent_books,omitempty"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	Services []Service `json:"services"`
}

func UserData() string {
	return filepath.Join(os.Getenv("USERPROFILE"), ProgramName)
}

func Initialize() {
	os.MkdirAll(UserData(), os.ModeDir)

	path := filepath.Join(UserData(), ConfigFile)
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Opening config file: %v", err)
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(&Conf); err != nil {
		log.Printf("Loading config: %v", err)
	}
}

func Save() {
	path := filepath.Join(UserData(), ConfigFile)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("Creating config file: %v", err)
		return
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t") // for readability
	if err := e.Encode(&Conf); err != nil {
		log.Printf("Saving config: %v", err)
	}
}
