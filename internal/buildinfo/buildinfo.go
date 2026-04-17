package buildinfo

import (
	"fmt"
	"runtime"
)

var (
	Version = "v0.1.0-alpha.2"
	Commit  = "unknown"
	Date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	GoOS    string `json:"goos"`
	GoArch  string `json:"goarch"`
}

func Current() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		GoOS:    runtime.GOOS,
		GoArch:  runtime.GOARCH,
	}
}

func HumanString() string {
	info := Current()
	return fmt.Sprintf("ada %s (%s/%s, commit=%s, date=%s)", info.Version, info.GoOS, info.GoArch, info.Commit, info.Date)
}
