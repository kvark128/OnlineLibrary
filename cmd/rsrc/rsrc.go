package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/josephspurrier/goversioninfo"
	"github.com/kvark128/OnlineLibrary/internal/config"
)

func main() {
	flagManifest := flag.String("manifest", "", "manifest file name")
	flagArch := flag.String("arch", "", "target architecture")
	flagSysoFile := flag.String("o", "", "output file name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %v [flags]\n\nPossible flags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	vi := &goversioninfo.VersionInfo{
		ManifestPath: *flagManifest,
		StringFileInfo: goversioninfo.StringFileInfo{
			FileDescription: config.ProgramDescription,
			ProductName:     config.ProgramName,
			ProductVersion:  config.ProgramVersion,
			LegalCopyright:  config.CopyrightInfo,
		},
	}

	vi.Build()
	vi.Walk()
	err := vi.WriteSyso(*flagSysoFile, *flagArch)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
