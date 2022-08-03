package util

import (
	"os"
	"path/filepath"
)

const tmp_suffix = ".tmp"

type SecureFile struct {
	path      string
	corrupted bool
	f         *os.File
}

func CreateSecureFile(path string) (*SecureFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir); err != nil {
		return nil, err
	}

	f, err := os.Create(path + tmp_suffix)
	if err != nil {
		return nil, err
	}

	sf := new(SecureFile)
	sf.path = path
	sf.f = f
	return sf, nil
}

func (sf *SecureFile) Write(p []byte) (int, error) {
	n, err := sf.f.Write(p)
	if err != nil {
		sf.corrupted = true
	}
	return n, err
}

func (sf *SecureFile) Corrupted() {
	sf.corrupted = true
}

func (sf *SecureFile) Close() error {
	err := sf.f.Close()
	tmp_path := sf.f.Name()
	if sf.corrupted || err != nil {
		// Temporary file is corrupted. Trying to remove it
		os.Remove(tmp_path)
		return err
	}
	// Just replace the original file with a temporary one
	return os.Rename(tmp_path, sf.path)
}
