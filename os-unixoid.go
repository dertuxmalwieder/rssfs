// +build linux solaris dragonfly freebsd netbsd openbsd darwin

package main

// These are functions specific for Unix and Unix-like systems.

import (
	"fmt"
	"os"
	"runtime"
)

type RssfsConfig struct {
	// Structure of the rssfs.hcl file
	MountPoint  string      `hcl:"mountpoint"`
	Feeds       []*Feed     `hcl:"feed,block"`
	Categories  []*Category `hcl:"category,block"`
}

func ConfigFilePath() string {
	// Returns the path to our configuration file, whether it exists or not.
 	// On non-macOS systems, this should be $XDG_CONFIG_HOME/rssfs.hcl, on
	// macOS, it is ${HOME}/Library/Application Support/rssfs.hcl.
	path := ""
	if (runtime.GOOS == "darwin") {
		path = fmt.Sprintf("%s/Library/Application Support", os.Getenv("HOME"))
	} else {
		path = os.Getenv("XDG_CONFIG_HOME")
	}
	return fmt.Sprintf("%s/rssfs.hcl", path)
}

const LineBreak = "\n"
