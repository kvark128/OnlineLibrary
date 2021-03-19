package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/kvark128/OnlineLibrary/internal/events"
	"github.com/kvark128/OnlineLibrary/internal/gui"
	"github.com/kvark128/OnlineLibrary/internal/manager"
	"github.com/kvark128/OnlineLibrary/internal/winmm"
)

func main() {
	if fl, err := os.Create(filepath.Join(config.UserData(), config.LogFile)); err == nil {
		log.SetOutput(fl)
		defer fl.Close()
	}
	log.SetPrefix("\n")
	log.SetFlags(log.Lmsgprefix | log.Ltime | log.Lshortfile)

	log.Printf("Starting OnlineLibrary version %s", config.ProgramVersion)
	config.Conf.Load()
	eventCH := make(chan events.Event, 16)

	if err := gui.Initialize(eventCH); err != nil {
		log.Fatal(err)
	}

	// Filling in the menu with the available audio output devices
	gui.SetOutputDeviceMenu(eventCH, winmm.OutputDeviceNames(), config.Conf.General.OutputDevice)

	// Filling in the menu with the available libraries
	gui.SetLibraryMenu(eventCH, config.Conf.Services, "")

	mng := new(manager.Manager)
	go mng.Start(eventCH)

	// Trying to log in to the current library
	if service, err := config.Conf.CurrentService(); err == nil {
		eventCH <- events.Event{events.LIBRARY_LOGON, service.Name}
	}

	gui.RunMainWindow()

	eventCH <- events.Event{events.LIBRARY_LOGOFF, nil}
	close(eventCH)

	mng.Wait()
	config.Conf.Save()
	log.Printf("Exiting")
}
