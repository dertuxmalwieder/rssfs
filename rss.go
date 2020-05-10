package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/mmcdole/gofeed"
)

var (
	fileIndex = map[string]*IndexedFile{
		// Generate the root node statically:
		"/": &IndexedFile{
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
	Name     string  `hcl:"name,label"`
	Feeds    []*Feed `hcl:"feed,block"`
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
	fmt.Println("rssfs starting up.")
	
	// We need a valid configuration file for feeds and mountpoints.
	var cfg RssfsConfig
	hclError := hclsimple.DecodeFile(ConfigFilePath(), nil, &cfg)
	if hclError != nil {
		errstr := fmt.Sprintf("No valid configuration file: %s (%s)", ConfigFilePath(), hclError)
		panic(errstr)
	}

	fmt.Printf("Using configuration file: %s\n", ConfigFilePath())

	nodeCount := uint64(1001)
	
	// Generate our file system: Populate the tree.
	rootItems := make([]*IndexedFile, 0)
	fp := gofeed.NewParser()
	
	for _, category := range cfg.Categories {
		// Add each category as a subdirectory.
		catsFeeds := make([]*IndexedFile, 0)

		// Descend into them and add the feeds.
		for _, subfeed := range category.Feeds {
			feeddata, _ := fp.ParseURL(subfeed.URL)
			feedFiles := make([]*IndexedFile, 0)

			var nodeTimestamp time.Time

			// Note: Certain blogs won't send a valid time stamp. Urgh.
			if (feeddata.UpdatedParsed != nil) {
				nodeTimestamp = *(feeddata.UpdatedParsed)
			} else {
				if (feeddata.PublishedParsed != nil) {
					nodeTimestamp = *(feeddata.PublishedParsed)
				} else {
					nodeTimestamp = time.Now()
				}
			}

			catsFeeds = append(catsFeeds, &IndexedFile{
				Filename:    fileNameClean(feeddata.Title),
				IsDirectory: true,
				Timestamp:   nodeTimestamp,
				Inode:       nodeCount,
			})

			nodeCount++

			// Add files to the feeds:
			for _, item := range feeddata.Items {
				var itemTimestamp time.Time
				if (item.UpdatedParsed != nil) {
					itemTimestamp = *(item.UpdatedParsed)
				} else {
					if (item.PublishedParsed != nil) {
						itemTimestamp = *(item.PublishedParsed)
					} else {
						itemTimestamp = time.Now()
					}
				}

				extension, content := GenerateOutputData(subfeed, item)
				feedFiles = append(feedFiles, &IndexedFile{
					Filename:    fmt.Sprintf("%s.%s", fileNameClean(item.Title), extension),
					Timestamp:   itemTimestamp,
					Inode:       nodeCount,
					Data:        []byte(content),
				})

				nodeCount++
			}

			tree["/" + fileNameClean(category.Name) + "/" + fileNameClean(feeddata.Title)] = feedFiles
		}

		tree["/" + fileNameClean(category.Name)] = catsFeeds
		
		// Finally, append this category to our root structure:
		rootItems = append(rootItems, &IndexedFile{
			Filename:    fileNameClean(category.Name),
			IsDirectory: true,
			Timestamp:   time.Now(),
			Inode:       nodeCount,
		})
		
		nodeCount++
	}

	for _, feed := range cfg.Feeds {
		// Add the feeds in the root structure as well.
		feeddata, _ := fp.ParseURL(feed.URL)

		var nodeTimestamp time.Time

		// Note: Certain blogs won't send a valid time stamp. Urgh.
		if (feeddata.UpdatedParsed != nil) {
			nodeTimestamp = *(feeddata.UpdatedParsed)
		} else {
			if (feeddata.PublishedParsed != nil) {
				nodeTimestamp = *(feeddata.PublishedParsed)
			} else {
				nodeTimestamp = time.Now()
			}
		}

		rootItems = append(rootItems, &IndexedFile{
			Filename:    fileNameClean(feeddata.Title),
			IsDirectory: true,
			Timestamp:   nodeTimestamp,
			Inode:       nodeCount,
		})
		
		nodeCount++
		
		// Add files to the feeds:
		feedFiles := make([]*IndexedFile, 0)
		for _, item := range feeddata.Items {
			var itemTimestamp time.Time
			if (item.UpdatedParsed != nil) {
				itemTimestamp = *(item.UpdatedParsed)
			} else {
				if (item.PublishedParsed != nil) {
					itemTimestamp = *(item.PublishedParsed)
				} else {
					itemTimestamp = time.Now()
				}
			}
			
			extension, content := GenerateOutputData(feed, item)
			feedFiles = append(feedFiles, &IndexedFile{
				Filename:    fmt.Sprintf("%s.%s", fileNameClean(item.Title), extension),
				Timestamp:   itemTimestamp,
				Inode:       nodeCount,
				Data:        []byte(content),
			})
			
			nodeCount++
		}

		tree["/" + fileNameClean(feeddata.Title)] = feedFiles
	}

	// Finalize the tree:
	tree["/"] = rootItems

	for parentPath, children := range tree {
		for _, child := range children {
			fullPath := filepath.Join(parentPath, child.Filename)
			fileIndex[fullPath] = child
		}
	}

	if (runtime.GOOS == "windows") {
		fmt.Printf("Trying to mount rssfs into %s:...\n", cfg.DriveLetter)
	} else {
		fmt.Printf("Trying to mount rssfs into %s...\n", cfg.MountPoint)
	}

	// We're done. Mount!
	
	Mount(cfg)
}
