package util

import (
	"os"
)

const tmp_suffix = ".tmp"

type FaultTolerantFile struct {
	path      string
	corrupted bool
	*os.File
}

func NewFaultTolerantFile(path string) (*FaultTolerantFile, error) {
	f, err := os.Create(path + tmp_suffix)
	if err != nil {
		return nil, err
	}

	ftf := new(FaultTolerantFile)
	ftf.path = path
	ftf.File = f
	return ftf, nil
}

func (ftf *FaultTolerantFile) Corrupted() {
	ftf.corrupted = true
}

func (ftf *FaultTolerantFile) Close() error {
	err := ftf.File.Close()
	tmp_path := ftf.File.Name()
	if ftf.corrupted || err != nil {
		// Temporary file is corrupted. Trying to remove it
		os.Remove(tmp_path)
		return err
	}
	// Just replace the original file with a temporary one
	return os.Rename(tmp_path, ftf.path)
}
