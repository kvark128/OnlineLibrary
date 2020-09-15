package flag

import "sync"

type Flag struct {
	sync.RWMutex
	flag bool
}

// Reset the internal flag to false.
func (f *Flag) Clear() {
	f.Lock()
	f.flag = false
	f.Unlock()
}

// Set the internal flag to true.
func (f *Flag) Set() {
	f.Lock()
	f.flag = true
	f.Unlock()
}

// Return true if and only if the internal flag is true.
func (f *Flag) IsSet() bool {
	f.RLock()
	defer f.RUnlock()
	return f.flag
}
