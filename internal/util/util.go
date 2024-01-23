package util

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

// Replaces all forbidden characters in s string with sign "_"
func ReplaceForbiddenCharacters(s string) string {
	chars := []string{"/", "\\", "?", "%", "*", ":", "|", "\"", "<", ">"}
	for _, ch := range chars {
		s = strings.ReplaceAll(s, ch, "_")
	}
	return s
}

// Formatting duration as HH:MM:SS
func FmtDuration(d time.Duration) string {
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", int(h), int(m), int(s))
}

// Parsing of duration from string HH:MM:SS
func ParseDuration(s string) (time.Duration, error) {
	var hh, mm, ss time.Duration
	_, err := fmt.Sscanf(s, "%d:%d:%d", &hh, &mm, &ss)
	if err != nil {
		return 0, err
	}
	return time.Hour*hh + time.Minute*mm + time.Second*ss, nil
}

// StringInSlice checks if a string exists in the specified slice of strings
func StringInSlice(str string, slc []string) bool {
	for _, s := range slc {
		if str == s {
			return true
		}
	}
	return false
}

// FileIsExist checks that path points to an existing file with the specified size
func FileIsExist(path string, size int64) bool {
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() && info.Size() == size {
			return true
		}
	}
	return false
}

func SaveXMLFile(path string, v any) error {
	f, err := CreateSecureFile(path)
	if err != nil {
		return err
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "\t") // for readability
	err = enc.Encode(v)
	if err == nil {
		err = enc.Close()
	}
	if err != nil {
		f.Corrupted()
		f.Close()
		return err
	}
	return f.Close()
}
