package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/kyokomi/emoji"
	"github.com/mmcdole/gofeed"
	"jaytaylor.com/html2text"
)

type ByTitle []*gofeed.Item

func (a ByTitle) Len() int      { return len(a) }
func (a ByTitle) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTitle) Less(i, j int) bool {
	return fileNameClean(a[i].Title) < fileNameClean(a[j].Title)
}

func CleanupCacheTime(key string, mins time.Duration) {
	timer := time.NewTimer(mins * time.Minute)
	go func() {
		<-timer.C
		feedcache.Erase(key)
	}()
}

func getItemTimestamp(item *gofeed.Item) time.Time {
	if item.UpdatedParsed != nil {
		return *(item.UpdatedParsed)
	}
	if item.PublishedParsed != nil {
		return *(item.PublishedParsed)
	}
	return time.Now()
}

func UpdateSingleFeed(feed *Feed, nodeCount uint64) ([]*IndexedFile, uint64, *gofeed.Feed) {
	// Updates a single feed. Returns the new list of IndexedFiles,
	// an updated nodecount and a feed data object (usually feeddata).
	var feeddata *gofeed.Feed

	if feed.Cache {
		var feedbytes bytes.Buffer

		reNonAlNum, err := regexp.Compile("[^a-zA-Z0-9]+")
		if err != nil {
			panic("Regex failed. Oops.")
		}

		cacheentry := fmt.Sprintf("feed-%s", reNonAlNum.ReplaceAllString(feed.URL, ""))
		cached, found := feedcache.Read(cacheentry)
		if found != nil {
			// Retrieve the feed and put it into our cache:
			fp := gofeed.NewParser()
			feeddata, _ = fp.ParseURL(feed.URL)

			emoji.Printf(":floppy_disk: Caching the feed from '%s'.\n", feed.URL)

			var mins int32
			if feed.CacheMins == 0 {
				mins = 60
			} else {
				mins = feed.CacheMins
			}

			// Convert to and store as bytes:
			enc := gob.NewEncoder(&feedbytes)

			store := enc.Encode(feeddata)
			if store != nil {
				emoji.Printf(":bangbang: Encoding the cache entry for '%s' failed: %v\n", feed.URL, store)
			} else {
				feedcache.Write(cacheentry, feedbytes.Bytes())
				CleanupCacheTime(cacheentry, time.Duration(mins))
			}
		} else {
			// Use the cached copy of the feed:
			feedbytes = *bytes.NewBuffer(cached)
			dec := gob.NewDecoder(&feedbytes)

			emoji.Printf(":floppy_disk: Loading '%s' from the cache.\n", feed.URL)

			decoderr := dec.Decode(&feeddata)
			if decoderr != nil {
                                emoji.Printf(":bangbang: Decoding the cache entry for '%s' failed: %v\n", feed.URL, decoderr)

				// Return to no caching:
				fp := gofeed.NewParser()
				feeddata, _ = fp.ParseURL(feed.URL)
                        }
		}
	} else {
		// No caching.
		// emoji.Printf(":arrows_counterclockwise: Updating feed contents for '%s'.\n", feed.URL)

		fp := gofeed.NewParser()
		feeddata, _ = fp.ParseURL(feed.URL)
	}

	sort.Sort(ByTitle(feeddata.Items))

	fname, prev_fname := "", ""
	// File collision counter
	col_cnt := 0

	// Add files to the feeds:
	feedFiles := make([]*IndexedFile, 0)
	for _, item := range feeddata.Items {
		itemTimestamp := getItemTimestamp(item)

		// Checking collision
		fname = fileNameClean(item.Title)
		if fname == prev_fname {
			col_cnt += 1
		} else {
			col_cnt = 0
		}
		prev_fname = fname

		extension, content := GenerateOutputData(feed, item)
		if col_cnt > 0 {
			fname = fmt.Sprintf("%s [%d].%s", fname, col_cnt, extension)
		} else {
			fname = fmt.Sprintf("%s.%s", fname, extension)
		}

		feedFiles = append(feedFiles, &IndexedFile{
			Filename:  fname,
			Timestamp: itemTimestamp,
			Inode:     nodeCount,
			Data:      []byte(content),
		})

		nodeCount++
	}

	return feedFiles, nodeCount, feeddata
}

func PopulateFeedTree(cfg RssfsConfig) map[string][]*IndexedFile {
	// Generates a file system, returns a tree of folders and files.
	// This updates each feed in the tree, it can be a relatively
	// slow operation on more than just a handful of feeds...
	retval := make(map[string][]*IndexedFile)
	nodeCount := uint64(1001)
	feedFiles := make([]*IndexedFile, 0)
	var feeddata *gofeed.Feed

	rootItems := make([]*IndexedFile, 0)
	fp := gofeed.NewParser()

	for _, category := range cfg.Categories {
		// Add each category as a subdirectory.
		catsFeeds := make([]*IndexedFile, 0)

		// Descend into them and add the feeds.
		for _, subfeed := range category.Feeds {
			feedFiles, nodeCount, feeddata = UpdateSingleFeed(subfeed, nodeCount)

			var nodeTimestamp time.Time

			// Note: Certain blogs won't send a valid time stamp. Urgh.
			if feeddata.UpdatedParsed != nil {
				nodeTimestamp = *(feeddata.UpdatedParsed)
			} else {
				if feeddata.PublishedParsed != nil {
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
				Feed:        subfeed,
			})

			nodeCount += 100

			retval["/"+fileNameClean(category.Name)+"/"+fileNameClean(feeddata.Title)] = feedFiles
		}

		retval["/"+fileNameClean(category.Name)] = catsFeeds

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
		if feeddata.UpdatedParsed != nil {
			nodeTimestamp = *(feeddata.UpdatedParsed)
		} else {
			if feeddata.PublishedParsed != nil {
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
			Feed:        feed,
		})

		nodeCount += 100

		// Add files to the feeds:
		feedFiles := make([]*IndexedFile, 0)
		feedFiles, nodeCount, feeddata = UpdateSingleFeed(feed, nodeCount)

		retval["/"+fileNameClean(feeddata.Title)] = feedFiles
	}

	// Finalize the tree:
	retval["/"] = rootItems

	return retval
}

func GenerateOutputData(feedopts *Feed, item *gofeed.Item) (ext string, content string) {
	// Generates the output file (extension and content) for an item.
	// Takes the feed's options as the first parameter to determine
	// whether to use plain text and to add the link.
	if feedopts.PlainText {
		// Parse into plain text:
		outContent, _ := html2text.FromString(item.Content, html2text.Options{PrettyTables: true, OmitLinks: false})

		// Prepend the title and the link (if wanted):
		link := ""
		if feedopts.ShowLink && item.Link != "" {
			link = fmt.Sprintf("%s%s", LineBreak, item.Link)
		}
		content = fmt.Sprintf("%s%s%s%s%s", item.Title, link, LineBreak, LineBreak, outContent)
		ext = "txt"
	} else {
		outTitle := ""

		// Prepend the title and link (if wanted):
		if feedopts.ShowLink && item.Link != "" {
			outTitle = fmt.Sprintf("<h1><a href=\"%s\">%s</a></h1>", item.Link, item.Title)
		} else {
			outTitle = fmt.Sprintf("<h1>%s</h1>", item.Title)
		}
		content = fmt.Sprintf("%s%s%s", outTitle, LineBreak, item.Content)
		ext = "html"
	}

	return ext, content
}
