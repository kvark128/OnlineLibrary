package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

const config_file = "config.json"
const program_name = "OnlineLibrary"

var (
	Conf      *Config
	config_dir string
)

type Service struct {
	Credentials Credentials `json:"credentials"`
	URL string `json:"url"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	UserData    string      `json:"user_data"`
	Services  []Service      `json:"services"`
}

func Initialize(configDir string) {
	if Conf != nil {
		panic("Config already initialized")
	}

	config_dir = configDir
	Conf = &Config{
		UserData:   filepath.Join(os.Getenv("USERPROFILE"), program_name),
	}

	path := filepath.Join(config_dir, config_file)
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Opening config file: %v", err)
		return
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err := d.Decode(Conf); err != nil {
		log.Printf("Loading config: %v", err)
	}
}

func Save() {
	os.MkdirAll(config_dir, os.ModeDir)
	path := filepath.Join(config_dir, config_file)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("Creating config file: %v", err)
		return
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t") // for readability
	if err := e.Encode(Conf); err != nil {
		log.Printf("Saving config: %v", err)
	}
}
