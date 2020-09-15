package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

const configfile = "config.json"

var (
	Conf      *Config
	configdir string
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	Credentials Credentials `json:"credentials"`
	UserData    string      `json:"user_data"`
	ServiceURL  string      `json:"service_url"`
}

func Initialize(configDir string) {
	if Conf != nil {
		panic("Config already initialized")
	}

	configdir = configDir
	Conf = &Config{
		UserData:   filepath.Join(os.Getenv("USERPROFILE"), "AV3715"),
		ServiceURL: "https://do.av3715.ru",
	}

	path := filepath.Join(configdir, configfile)
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Opening config file: %s\n", err)
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(Conf); err != nil {
		log.Printf("Loading config: %s\n", err)
	}
}

func Save() {
	os.MkdirAll(configdir, os.ModeDir)
	path := filepath.Join(configdir, configfile)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("Creating config file: %s\n", err)
		return
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t")
	if err := e.Encode(Conf); err != nil {
		log.Printf("Saving config: %s\n", err)
	}
}
