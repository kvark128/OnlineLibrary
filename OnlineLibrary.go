package main

import (
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/manager"
	"github.com/kvark128/OnlineLibrary/internal/msg"
	"github.com/kvark128/OnlineLibrary/internal/waveout"
)

func main() {
	if fl, err := os.Create(filepath.Join(config.UserData(), config.LogFile)); err == nil {
		log.SetOutput(fl)
		defer fl.Close()
	}

	log.Info("Starting OnlineLibrary version %s", config.ProgramVersion)
	config.Conf.Load()

	if level, err := log.StringToLevel(config.Conf.General.LogLevel); err == nil {
		log.SetLevel(level)
	}

	msgCH := make(chan msg.Message, 16)

	if err := gui.Initialize(msgCH); err != nil {
		log.Error("GUI initializing: %v", err)
		os.Exit(1)
	}

	// Filling in the menu with the available audio output devices
	gui.SetOutputDeviceMenu(msgCH, waveout.OutputDeviceNames(), config.Conf.General.OutputDevice)

	// Filling in the menu with the available libraries
	gui.SetLibraryMenu(msgCH, config.Conf.Services, "")

	// Setting label for the pause timer in the menu
	gui.SetPauseTimerLabel(int(config.Conf.General.PauseTimer.Minutes()))

	mng := new(manager.Manager)
	done := make(chan bool)
	go mng.Start(msgCH, done)
	gui.RunMainWindow()
	close(msgCH)
	<-done

	config.Conf.Save()
	log.Info("Exiting")
}
