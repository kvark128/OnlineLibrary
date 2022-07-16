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

	logger := log.New(os.Stdout, log.Info, "\t")
	configFile := filepath.Join(userDataDir, config.ConfigFile)
	logFile := filepath.Join(userDataDir, config.LogFile)

	if fl, err := os.Create(logFile); err == nil {
		logger.SetOutput(fl)
		defer fl.Close()
	}

	logger.Info("Starting OnlineLibrary version %s", config.ProgramVersion)
	conf := config.NewConfig()
	logger.Info("Loading config file from %v", configFile)
	if err := conf.Load(configFile); err != nil {
		logger.Error("Loading config file: %v", err)
	}

	if level, err := log.StringToLevel(conf.General.LogLevel); err == nil {
		logger.SetLevel(level)
	}

	msgCH := make(chan msg.Message, config.MessageBufferSize)

	if err := gui.Initialize(msgCH, logger.Level()); err != nil {
		logger.Error("GUI initializing: %v", err)
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
	go mng.Start(conf, msgCH, logger, done)
	gui.RunMainWindow()
	close(msgCH)

	logger.Debug("Waiting for the manager to finish")
	select {
	case <-done:
		break
	case <-time.After(time.Second * 16):
		logger.Error("Manager termination timeout has expired. Forced program exit")
		os.Exit(1)
	}

	logger.Info("Saving config file to %v", configFile)
	if err := conf.Save(configFile); err != nil {
		logger.Error("Saving config file: %v", err)
	}
	logger.Info("Exiting")
}
