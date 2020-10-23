package util

import (
	"fmt"
	"strings"
	"time"
)

// Replaces all prohibit characters with a sign "_"
func ReplaceProhibitCharacters(s string) string {
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
