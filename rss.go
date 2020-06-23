package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/kyokomi/emoji"
)

var (
	fileIndex = map[string]*IndexedFile{
		// Generate the root node statically:
		"/": {
			Filename:    "",
			IsDirectory: true,
			Timestamp:   time.Now(),
			Inode:       1001,
		},
	}
	tree = make(map[string][]*IndexedFile)
)

type Feed struct {
	// A feed has an URL and two optional switches:
	// 1. PlainText: Determines whether to parse the feed contents
	//               as plain text (creates .txt files). Defaults to
	//               false.
	// 2. ShowLink:  Determines whether to add a link to the original
	//               article to the generated files where applicable.
	//               Defaults to false.
	URL       string `hcl:"url"`
	PlainText bool   `hcl:"plaintext,optional"`
	ShowLink  bool   `hcl:"showlink,optional"`
}

type Category struct {
	// A category has a name in its title and zero or more feeds.
	Name  string  `hcl:"name,label"`
	Feeds []*Feed `hcl:"feed,block"`
}

// RssConfig is implemented by platform.

type IndexedFile struct {
	// A file.
	Filename    string
	IsDirectory bool
	Timestamp   time.Time
	Inode       uint64
	Size        uint64

	Data []byte
	Feed *Feed
}

func fileNameClean(in string) string {
	// Returns a valid file name for <in>.
	invalidFileNameCharacters := [3]string{"/", "\\", ":"}
	ret := in
	for _, character := range invalidFileNameCharacters {
		r := strings.NewReplacer(character, "-")
		ret = r.Replace(ret)
	}
	return strings.TrimSpace(ret)
}

func main() {
	emoji.Println(":stopwatch: rssfs starting up.")

	// We need a valid configuration file for feeds and mountpoints.
	var cfg RssfsConfig
	hclError := hclsimple.DecodeFile(ConfigFilePath(), nil, &cfg)
	if hclError != nil {
		emoji.Printf(":bangbang: No valid configuration file: %s (%s)", ConfigFilePath(), hclError)
		os.Exit(1)
	}

	emoji.Printf(":wrench: Using configuration file: %s\n", ConfigFilePath())

	tree = PopulateFeedTree(cfg)
	for parentPath, children := range tree {
		for _, child := range children {
			fullPath := filepath.Join(parentPath, child.Filename)
			fileIndex[fullPath] = child
		}
	}

	if runtime.GOOS == "windows" {
		emoji.Println(":gear: Trying to mount rssfs...")
	} else {
		emoji.Printf(":gear: Trying to mount rssfs into %s...\n", cfg.MountPoint)
	}

	// We're done. Mount!

	Mount(cfg)
}
