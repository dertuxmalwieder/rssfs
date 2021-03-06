// +build windows

package main

// These are Windows-specific functions.

import (
	"fmt"
	"os"
)

func ConfigFilePath() string {
	// Returns the path to our configuration file, whether it exists or not.
	// On Windows, this should be %APPDATA%\rssfs.hcl.
	return fmt.Sprintf("%s\\rssfs.hcl", os.Getenv("APPDATA"))
}

const LineBreak = "\r\n"
