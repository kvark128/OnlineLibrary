package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/manager"
	"github.com/kvark128/OnlineLibrary/internal/msg"
	"github.com/kvark128/OnlineLibrary/internal/waveout"
)

func main() {
	userDataDir := config.UserData()
	if err := os.MkdirAll(userDataDir, os.ModeDir); err != nil {
		os.Exit(1)
	}

	configFile := filepath.Join(userDataDir, config.ConfigFile)
	logFile := filepath.Join(userDataDir, config.LogFile)

	if fl, err := os.Create(logFile); err == nil {
		log.SetOutput(fl)
		defer fl.Close()
	}

	log.Info("Starting OnlineLibrary version %s", config.ProgramVersion)
	conf := config.NewConfig()
	conf.Load(configFile)

	if level, err := log.StringToLevel(conf.General.LogLevel); err == nil {
		log.SetLevel(level)
	}

	msgCH := make(chan msg.Message, config.MessageBufferSize)

	if err := gui.Initialize(msgCH); err != nil {
		log.Error("GUI initializing: %v", err)
		os.Exit(1)
	}

	// Filling in the menu with the available audio output devices
	gui.SetOutputDeviceMenu(msgCH, waveout.OutputDeviceNames(), conf.General.OutputDevice)

	// Filling in the menu with the available providers
	gui.SetProvidersMenu(msgCH, conf.Services, "")

	// Setting label for the pause timer in the menu
	gui.SetPauseTimerLabel(int(conf.General.PauseTimer.Minutes()))

	mng := new(manager.Manager)
	done := make(chan bool)
	go mng.Start(conf, msgCH, done)
	gui.RunMainWindow()
	close(msgCH)

	log.Debug("Waiting for the manager to finish")
	select {
	case <-done:
		break
	case <-time.After(time.Second * 16):
		log.Error("Manager termination timeout has expired. Forced program exit")
		os.Exit(1)
	}

	conf.Save(configFile)
	log.Info("Exiting")
}
