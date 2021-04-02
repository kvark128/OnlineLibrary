package main

import (
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/log"
	"github.com/kvark128/OnlineLibrary/internal/manager"
	"github.com/kvark128/OnlineLibrary/internal/msg"
	"github.com/kvark128/OnlineLibrary/internal/winmm"
)

func main() {
	if fl, err := os.Create(filepath.Join(config.UserData(), config.LogFile)); err == nil {
		log.SetOutput(fl)
		defer fl.Close()
	}

	log.Info("Starting OnlineLibrary version %s", config.ProgramVersion)
	config.Conf.Load()
	msgCH := make(chan msg.Message, 16)

	if err := gui.Initialize(msgCH); err != nil {
		log.Error("GUI initializing: %v", err)
		os.Exit(1)
	}

	// Filling in the menu with the available audio output devices
	gui.SetOutputDeviceMenu(msgCH, winmm.OutputDeviceNames(), config.Conf.General.OutputDevice)

	// Filling in the menu with the available libraries
	gui.SetLibraryMenu(msgCH, config.Conf.Services, "")

	mng := new(manager.Manager)
	go mng.Start(msgCH)

	// Trying to log in to the current library
	if service, err := config.Conf.CurrentService(); err == nil {
		msgCH <- msg.Message{msg.LIBRARY_LOGON, service.Name}
	}

	gui.RunMainWindow()

	msgCH <- msg.Message{msg.LIBRARY_LOGOFF, nil}
	close(msgCH)

	mng.Wait()
	config.Conf.Save()
	log.Info("Exiting")
}
