package main

import (
	html "code.google.com/p/go.net/html"
	atom "code.google.com/p/go.net/html/atom"
	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
)

func main() {

	// Flush logs before exit
	defer log.Flush()

	// Crawl the specified site
	Crawl("http://golang.org/")
}

// Crawl takes a URL and recursively crawls pages
func Crawl(url string) {

	_, urls, err := fetch(url)
	if err != nil {
		log.Errorf("Error:", err)
		return
	}

	log.Infof("URLs found: %+v", urls)
}

// fetch retrieves the page at the specified URL and extracts URLs
func fetch(url string) (string, []string, error) {

	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", nil, err
	}

	urls, err := extractLinks(doc)
	if err != nil {
		return "", urls, err
	}

	log.Debugf("URLs: %+v", urls)

	return "", urls, nil
}

// extractLinks from a document
func extractLinks(doc *goquery.Document) ([]string, error) {

	// Blank slice to hold our links on this page
	urls := make([]string, 0)

	// Extract all 'a' elements from the document
	sel := doc.Find("a")
	if sel == nil {
		// Assume zero links on failure
		return nil, nil
	}

	// Range over links, and add them to our list if valid
	for i, n := range sel.Nodes {
		if n.Type != html.ElementNode || n.DataAtom != atom.A {
			log.Debugf("Node is not an anchor: %v", n.Type)
			continue
		}

		var href string

		for _, a := range n.Attr {
			if a.Key != "href" {
				continue
			}
			href = a.Val
		}

		if href == "" {
			continue
		}

		log.Debugf("Node %v: %s", i, href)
		urls = append(urls, href)
	}

	return urls, nil
}
