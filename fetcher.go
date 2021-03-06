package main

import (
	"errors"
	"net/url"
	"strings"

	html "code.google.com/p/go.net/html"
	atom "code.google.com/p/go.net/html/atom"
	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
)

var (
	InvalidNode                 = errors.New("Node is not an anchor")
	InvalidNodeAttributeMissing = errors.New("Node does not contain the specified attribute")
)

type Fetcher interface {
	// Fetch returns a slice of URLs found on the target page
	// along with a slice of assets.
	Fetch(target *url.URL) (urls []*url.URL, assets []*url.URL, err error)
}

type HttpFetcher struct{}

// Fetch retrieves the page at the specified URL and extracts URLs
func (h *HttpFetcher) Fetch(target *url.URL) ([]*url.URL, []*url.URL, error) {

	doc, err := goquery.NewDocument(target.String())
	if err != nil {
		return nil, nil, err
	}

	urls, err := h.extractLinks(doc)
	if err != nil {
		return nil, nil, err
	}

	assets, err := h.extractAssets(doc)
	if err != nil {
		return nil, nil, err
	}

	log.Debugf("URLs: %+v", urls)
	log.Debugf("Assets: %+v", assets)

	return urls, assets, nil
}

// extractLinks from a document
func (h *HttpFetcher) extractLinks(doc *goquery.Document) ([]*url.URL, error) {

	// Blank slice to hold the links on this page
	urls := make([]*url.URL, 0)

	// Extract all 'a' elements from the document
	sel := doc.Find("a")
	if sel == nil {
		// Assume zero links on failure
		return nil, nil
	}

	// Range over links, and add them to the list if valid
	for _, n := range sel.Nodes {

		// Validate the node is a link, and extract the target URL
		href, err := h.extractValidHref(n)
		if err != nil || href == "" {
			continue
		}

		// Normalise the URL and add if valid
		if uri := h.normaliseUrl(doc.Url, href); uri != nil {
			urls = append(urls, uri)
		}
	}

	return h.dedupeUrls(urls), nil
}

// extractAssets from a document
// @todo break this up and add tests
func (h *HttpFetcher) extractAssets(doc *goquery.Document) ([]*url.URL, error) {

	var sel *goquery.Selection
	assets := make([]*url.URL, 0)

	// First grab all the images
	sel = doc.Find("img")
	for _, n := range sel.Nodes {
		if n == nil {
			continue
		}
		for _, a := range n.Attr {
			if a.Key == "src" && a.Val != "" {
				if uri := h.normaliseUrl(doc.Url, a.Val); uri != nil {
					assets = append(assets, uri)
					break
				}

			}
		}
	}

	// Next scripts
	sel = doc.Find("script")
	for _, n := range sel.Nodes {
		if n == nil {
			continue
		}
		for _, a := range n.Attr {
			if a.Key == "src" && a.Val != "" {
				if uri := h.normaliseUrl(doc.Url, a.Val); uri != nil {
					assets = append(assets, uri)
					break
				}

			}
		}
	}

	// Links, eg styles, shortcut icons etc
	sel = doc.Find("link")
	for _, n := range sel.Nodes {
		if n == nil {
			continue
		}

		// Pull out various fields
		var rel, linktype string
		var uri *url.URL
		for _, a := range n.Attr {
			switch a.Key {
			case "rel":
				rel = a.Val
			case "type":
				linktype = a.Val
			case "href":
				uri = h.normaliseUrl(doc.Url, a.Val)
			}
		}

		// Continue if there is no link target
		if uri == nil {
			continue
		}

		// Otherwise select specific combinations
		switch {
		case rel == "stylesheet" && linktype == "text/css":
			assets = append(assets, uri)
		case rel == "shortcut icon":
			assets = append(assets, uri)
		}
	}

	return h.dedupeUrls(assets), nil
}

// validateLink is an anchor with a href, and extract normalised url
func (h *HttpFetcher) extractValidHref(n *html.Node) (string, error) {
	var href string

	// Confirm this node is an anchor element
	if n == nil || n.Type != html.ElementNode || n.DataAtom != atom.A {
		return href, InvalidNode
	}

	// Return the value of the href attr if it exists
	for _, a := range n.Attr {
		if a.Key == "href" && a.Val != "" {
			return a.Val, nil
		}
	}

	return "", InvalidNodeAttributeMissing
}

// normaliseUrl converts relative URLs to absolute URLs
func (h *HttpFetcher) normaliseUrl(parent *url.URL, urlString string) *url.URL {

	// Strip off fragment
	i := strings.LastIndex(urlString, "#")
	if i >= 0 {
		urlString = urlString[:i]
	}

	// Parse the string into a url.URL
	uri, err := url.Parse(urlString)
	if err != nil {
		log.Debugf("Failed to parse URL: %s", urlString)
		return nil
	}

	// Resolve references to get an absolute URL
	abs := parent.ResolveReference(uri)

	return abs
}

func (h *HttpFetcher) dedupeUrls(original []*url.URL) []*url.URL {
	seen := make(map[string]bool)
	ret := make([]*url.URL, 0)

	for _, u := range original {
		if _, ok := seen[u.String()]; ok {
			continue
		}

		seen[u.String()] = true
		ret = append(ret, u)
	}

	return ret
}
