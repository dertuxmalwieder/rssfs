package main

import (
	"fmt"
	"time"
	
	"github.com/mmcdole/gofeed"
	"jaytaylor.com/html2text"
)

func PopulateFeedTree(cfg RssfsConfig) (map[string][]*IndexedFile) {
	retval := make(map[string][]*IndexedFile)
	nodeCount := uint64(1001)
	
	// Generate our file system: Populate the retval.
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

			retval["/" + fileNameClean(category.Name) + "/" + fileNameClean(feeddata.Title)] = feedFiles
		}

		retval["/" + fileNameClean(category.Name)] = catsFeeds
		
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

		retval["/" + fileNameClean(feeddata.Title)] = feedFiles
	}

	// Finalize the retval:
	retval["/"] = rootItems

	return retval
}

func GenerateOutputData(feedopts *Feed, item *gofeed.Item) (ext string, content string) {
	// Generates the output file (extension and content) for an item.
	// Takes the feed's options as the first parameter to determine
	// whether to use plain text and to add the link.
	if (feedopts.PlainText) {
		// Parse into plain text:
		outContent, _ := html2text.FromString(item.Content, html2text.Options{ PrettyTables: true, OmitLinks: false })

		// Prepend the title and the link (if wanted):
		link := ""
		if (feedopts.ShowLink && item.Link != "") {
			link = fmt.Sprintf("%s%s", LineBreak, item.Link)
		}
		content = fmt.Sprintf("%s%s%s%s%s", item.Title, link, LineBreak, LineBreak, outContent)
		ext = "txt"
	} else {
		outTitle := ""
		
		// Prepend the title and link (if wanted):
		if (feedopts.ShowLink && item.Link != "") {
			outTitle = fmt.Sprintf("<h1><a href=\"%s\">%s</a></h1>", item.Link, item.Title)
		} else {
			outTitle = fmt.Sprintf("<h1>%s</h1>", item.Title)
		}
		content = fmt.Sprintf("%s%s%s", outTitle, LineBreak, item.Content)
		ext = "html"
	}

	return ext, content
}
