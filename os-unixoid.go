// +build linux solaris dragonfly freebsd netbsd openbsd darwin

package main

// These are functions specific for Unix and Unix-like systems.

import (
	"fmt"
	"os"
	"runtime"
)

func ConfigFilePath() string {
	// Returns the path to our configuration file, whether it exists or not.
	// On non-macOS systems, this should be $XDG_CONFIG_HOME/rssfs.hcl, on
	// macOS, it is ${HOME}/Library/Application Support/rssfs.hcl.
	// If $XDG_CONFIG_HOME is not defined, non-macOS systems will fall back
	// to $HOME/.config.
	path := ""
	if runtime.GOOS == "darwin" {
		path = fmt.Sprintf("%s/Library/Application Support", os.Getenv("HOME"))
	} else {
		_, xdgDefined := os.LookupEnv("XDG_CONFIG_HOME")
		if !xdgDefined {
			// Fallback:
			path = fmt.Sprintf("%s/.config", os.Getenv("HOME"))
		} else {
			path = os.Getenv("XDG_CONFIG_HOME")
		}
	}
	return fmt.Sprintf("%s/rssfs.hcl", path)
}

const LineBreak = "\n"
