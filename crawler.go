package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sync"
)

// ---------------------
// Error implementations
// ---------------------

type Http404Error string
type InvalidHTMLContent string

func (e Http404Error) Error() string {
	return fmt.Sprintf("Failed to find for URL (%s).", e)
}

func (e InvalidHTMLContent) Error() string {
	return fmt.Sprintf("Could not parse HTML content for URL (%s).", e)
}

// --------------------
// Crawler
// --------------------

// Used for 'urls' buffered channel
const MAX_CHAN_URLS int = 100

type CrawlerI interface {
	New(baseSite string)
	SpawnCrawler()
	Wait()
	PrintSitemap()
}

type Crawler struct {

	// First site to crawl
	baseSite string

	// Domain name encompassing all crawls, this will extracted from 'baseSite' in New()
	domain string

	// urls channel used by all goroutines to add new URLs to parse
	urls chan string

	// Cache sites that have been visited
	// key 		--> value
	// string 	--> bool
	// "site"	--> true
	visited sync.Map

	// Cache relationship between visited sites, used to construct and print sitemap
	// key 		--> value
	// (string) --> []string{}
	// "parent" --> ["child1", "child2"]
	sitemap sync.Map

	// Used to wait for all goroutines to complete
	wg sync.WaitGroup
}

func (c *Crawler) New(baseSite string) *Crawler {
	if c == nil {
		c = new(Crawler)
	}

	// e.g. https://monzo.com/
	c.baseSite = baseSite

	// TODO: extract the domain from baseSite. Hardcoded for now.
	c.domain = "monzo.com"

	// create buffered channel
	c.urls = make(chan string, MAX_CHAN_URLS)
	return c
}

// Adds a new site to process
func (c *Crawler) addSite(site string) {
	c.visited.Store(site, true)
	c.urls <- site
	c.wg.Add(1)
}

// Begin processing sites
func (c *Crawler) Start() {
	c.addSite(c.baseSite)

	// TODO: check for error returned from Crawl
	go c.Crawl()
}

// TODO: what happens when an error is spit out by Crawl?
// the link is considered handled even though it is not traversed.

// Crawl site by visiting only local pages to the domain
// Returns error if any occured, nil if none
func (c *Crawler) Crawl() error {

	// Get URL to crawl from channel
	defer c.wg.Done()
	url := <-c.urls
	// log.Printf("Crawl: %s\n", url)

	// Fetch URL contents
	resp, err := http.Get(url)
	if err != nil {
		// TODO: add the url string to list of broken URLs
		// IMPROVEMENT: pass the err to Http404Error so that we have a reference to the original
		return Http404Error(url)
	}
	defer resp.Body.Close()

	// Read HTML from Body
	bytes, err := ioutil.ReadAll(resp.Body)
	var html = string(bytes)
	if err != nil {
		return InvalidHTMLContent(url)
	}

	// Find relative links and convert them to absolute
	children := FindRelativeLinks(html)
	for i, x := range children {
		children[i] = "https://" + c.domain + x
	}

	// Find absolute links
	absoluteLinks := FindAbsoluteLinks(html, &c.domain)

	children = append(children, absoluteLinks...)
	c.sitemap.Store(url, children)

	// sitemap.LoadOrStore(url, []string{})
	// Place child urls on the urls channel
	for _, x := range children {
		if _, present := c.visited.Load(x); !present {
			c.addSite(x)
			// TODO: check for Crawlers that return error.
			go c.Crawl()
		}
	}

	// No error
	return nil
}

// Wait for all goroutines to finish - blocking function
func (c *Crawler) Wait() {
	c.wg.Wait()
}

func (c *Crawler) PrintSitemap() {
	c.sitemap.Range(func(k, v interface{}) bool {
		v1, ok := v.([]string)
		if !ok {
			return false
		}
		fmt.Printf("\n%s\n", k)
		for _, child := range v1 {
			fmt.Printf("  --> %s\n", child)
		}
		return true
	})
}

// --------------------
// Link handling
// --------------------

func FindRelativeLinks(html string) []string {
	const relativePattern string = "href=\"(/[-\\w\\d_/\\.]+)\""
	const captureGroup int = 1
	re := regexp.MustCompile(relativePattern)
	allMatches := re.FindAllStringSubmatch(html, -1)
	b := make([]string, len(allMatches))

	// take the first capturing group from matches
	for i, x := range allMatches {
		b[i] = x[captureGroup]
	}
	return b
}

// Find aboslute links present in the given html string.
// If domain is not nil, then only links local to the domain will be returned
func FindAbsoluteLinks(html string, domain *string) []string {

	// If domain == nil, use a default domain matcher
	var defaultDomainPattern string = "([^:\\/\\s]+)"
	if domain == nil {
		domain = &defaultDomainPattern
	}

	// Absolute pattern to match
	// http[s] is required for the absolute link to match, otherwise we would match relative links as well.
	// The domain may be specified by the caller of the function, otherwise a default domain pattern matcher is used.
	var absolutePattern string = fmt.Sprintf("href=\"((http[s]?:\\/\\/)([^\\s\\/]*\\.)?%s(\\/[^\\s]*)*)\"", *domain)
	const captureGroup int = 1

	re := regexp.MustCompile(absolutePattern)
	allMatches := re.FindAllStringSubmatch(html, -1)

	b := make([]string, len(allMatches))

	// Take the first capturing group from matches
	for i, x := range allMatches {
		b[i] = x[captureGroup]
	}

	return b
}

func main() {
	var c *Crawler
	c = c.New("https://monzo.com")
	c.Start()
	c.Wait()
	c.PrintSitemap()
	log.Printf("Crawler done. Exiting main.")
}
