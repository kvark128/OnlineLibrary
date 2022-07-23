package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/manager"
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

	wnd, err := gui.NewMainWindow()
	if err != nil {
		logger.Error("Creating main window: %v", err)
		os.Exit(1)
	}
	menuBar := wnd.MenuBar()

	// Filling in the menu with the available audio output devices
	menuBar.SetOutputDeviceMenu(waveout.OutputDeviceNames(), conf.General.OutputDevice)

	// Filling in the menu with the available providers
	menuBar.SetProvidersMenu(conf.Services, "")

	// Setting label for the pause timer in the menu
	menuBar.SetPauseTimerLabel(int(conf.General.PauseTimer.Minutes()))

	// Filling in the menu with the supported log levels
	menuBar.SetLogLevelMenu(logger.SupportedLevels(), logger.Level())

	mng := manager.NewManager(wnd, logger)
	done := make(chan bool)
	go mng.Start(conf, done)
	wnd.Run()

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
