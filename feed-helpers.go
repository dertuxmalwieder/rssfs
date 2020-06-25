package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/mmcdole/gofeed"
	"jaytaylor.com/html2text"
)

type ByTitle []*gofeed.Item

func (a ByTitle) Len() int      { return len(a) }
func (a ByTitle) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTitle) Less(i, j int) bool {
	return fileNameClean(a[i].Title) < fileNameClean(a[j].Title)
}

func getTimestamp(updated *time.Time, published *time.Time) time.Time {
	if updated != nil {
		return *updated
	}
	if published != nil {
		return *published
	}
	return time.Now()
}

func UpdateSingleFeed(feed *Feed, nodeCount uint64, cfg *RssfsConfig) ([]*IndexedFile, uint64, *gofeed.Feed) {
	// Updates a single feed. Returns the new list of IndexedFiles,
	// an updated nodecount and a feed data object (usually feeddata).
	fp := gofeed.NewParser()
	feeddata, _ := fp.ParseURL(feed.URL)
	feedFiles := make([]*IndexedFile, 0)

	sort.Sort(ByTitle(feeddata.Items))

	fname, prev_fname := "", ""
	// File collision counter
	col_cnt := 0

	// Add files to the feeds:
	for _, item := range feeddata.Items {
		itemTimestamp := getTimestamp(item.UpdatedParsed, item.PublishedParsed)

		// Checking collision
		fname = fileNameClean(item.Title)
		if fname == prev_fname {
			col_cnt += 1
		} else {
			col_cnt = 0
		}
		prev_fname = fname

		extension, content := GenerateOutputData(feed, item, cfg.Style)
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
			feedFiles, nodeCount, feeddata = UpdateSingleFeed(subfeed, nodeCount, &cfg)
			// Note: Certain blogs won't send a valid time stamp. Urgh.
			nodeTimestamp := getTimestamp(feeddata.UpdatedParsed, feeddata.PublishedParsed)

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
		feedFiles, nodeCount, feeddata = UpdateSingleFeed(feed, nodeCount, &cfg)

		retval["/"+fileNameClean(feeddata.Title)] = feedFiles
	}

	// Finalize the tree:
	retval["/"] = rootItems

	return retval
}

func GenerateOutputData(feedopts *Feed, item *gofeed.Item, style string) (ext string, content string) {
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
		content = fmt.Sprintf("%s%s%s%s", style, outTitle, LineBreak, item.Content)
		ext = "html"
	}

	return ext, content
}
