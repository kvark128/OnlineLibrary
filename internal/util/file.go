package util

import (
	"os"
)

const tmp_suffix = ".tmp"

type FaultTolerantFile struct {
	path      string
	f         *os.File
	corrupted bool
}

func NewFaultTolerantFile(path string) (*FaultTolerantFile, error) {
	f, err := os.Create(path + tmp_suffix)
	if err != nil {
		return nil, err
	}

	ftf := new(FaultTolerantFile)
	ftf.path = path
	ftf.f = f
	return ftf, nil
}

func (ftf *FaultTolerantFile) Write(p []byte) (int, error) {
	n, err := ftf.f.Write(p)
	if err != nil {
		ftf.corrupted = true
	}
	return n, err
}

func (ftf *FaultTolerantFile) Close() error {
	err := ftf.f.Close()
	tmp_path := ftf.f.Name()
	if ftf.corrupted || err != nil {
		// Temporary file is corrupted. Trying to remove it
		os.Remove(tmp_path)
		return err
	}
	// Just replace the original file with a temporary one
	return os.Rename(tmp_path, ftf.path)
}
