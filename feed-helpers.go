package main

import (
	"fmt"
	
	"github.com/mmcdole/gofeed"
	"jaytaylor.com/html2text"
)

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
