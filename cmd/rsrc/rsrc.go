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
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\nPossible flags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	vi := new(goversioninfo.VersionInfo)

	vi.ManifestPath = *flagManifest
	vi.StringFileInfo.FileDescription = "DAISY Online Client"
	vi.StringFileInfo.ProductName = config.ProgramName
	vi.StringFileInfo.ProductVersion = config.ProgramVersion
	vi.StringFileInfo.LegalCopyright = "Copyright (c) 2020 - 2022 Alexander Linkov"

	vi.Build()
	vi.Walk()
	err := vi.WriteSyso(*flagSysoFile, *flagArch)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
