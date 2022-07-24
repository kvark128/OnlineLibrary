package lang

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/kvark128/OnlineLibrary/internal/config"
	"github.com/leonelquinteros/gotext"
	"golang.org/x/sys/windows"
)

const (
	LOCALE_SLOCALIZEDLANGUAGENAME = 0x0000006f
)

var kernel32 = windows.NewLazySystemDLL("kernel32.dll")

var (
	procGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
	procGetLocaleInfoW           = kernel32.NewProc("GetLocaleInfoW")
	procLocaleNameToLCID         = kernel32.NewProc("LocaleNameToLCID")
	procLCIDToLocaleName         = kernel32.NewProc("LCIDToLocaleName")
)

type Language struct {
	ID          string
	Description string
}

type LCID uint

func GetUserDefaultUILanguage() LCID {
	lcid, _, _ := procGetUserDefaultUILanguage.Call()
	return LCID(lcid)
}

func GetLocaleDescription(lcid LCID) (string, error) {
	bufSize := 1024
	buf := make([]uint16, bufSize)
	r, _, _ := procGetLocaleInfoW.Call(uintptr(lcid), LOCALE_SLOCALIZEDLANGUAGENAME, uintptr(unsafe.Pointer(&buf[0])), uintptr(bufSize))
	if r == 0 {
		return "", errors.New("invalid lcid")
	}
	return windows.UTF16PtrToString(&buf[0]), nil
}

func LocaleNameToLCID(localeName string) (LCID, error) {
	str, err := windows.UTF16PtrFromString(localeName)
	if err != nil {
		return 0, err
	}
	lcid, _, _ := procLocaleNameToLCID.Call(uintptr(unsafe.Pointer(str)), 0)
	runtime.KeepAlive(localeName)
	runtime.KeepAlive(str)
	return LCID(lcid), nil
}

func LCIDToLocaleName(lcid LCID) (string, error) {
	bufSize := 128
	buf := make([]uint16, bufSize)
	r, _, _ := procLCIDToLocaleName.Call(uintptr(lcid), uintptr(unsafe.Pointer(&buf[0])), uintptr(bufSize), 0)
	if r == 0 {
		return "", errors.New("invalid lcid")
	}
	return windows.UTF16PtrToString(&buf[0]), nil
}

func availableLanguages(lib string) []Language {
	langs := make([]Language, 0)
	path, err := filepath.Abs(lib)
	if err != nil {
		return langs
	}

	entrys, err := os.ReadDir(path)
	if err != nil {
		return langs
	}

	for _, e := range entrys {
		if !e.IsDir() {
			continue
		}
		locale := e.Name()
		lcid, err := LocaleNameToLCID(locale)
		if err != nil {
			continue
		}
		description, err := GetLocaleDescription(lcid)
		if err != nil {
			continue
		}
		l := Language{ID: locale, Description: description}
		langs = append(langs, l)
	}
	return langs
}

func Init(lang string) ([]Language, bool) {
	lib := "locales"
	langs := availableLanguages(lib)

	langIsAvailable := false
	for _, l := range langs {
		if lang == l.ID {
			langIsAvailable = true
			break
		}
	}

	if !langIsAvailable {
		lang = "en"
		lcid := GetUserDefaultUILanguage()
		if locale, err := LCIDToLocaleName(lcid); err == nil {
			lang = locale
		}
	}

	gotext.Configure(lib, lang, config.ProgramName)
	return langs, langIsAvailable
}
