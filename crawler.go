package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
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

// TODO: what happens if we raise this?
// Used for 'urls' buffered channel
const MAX_CHAN_URLS int = 100

// Crawler has not been tested with successive crawls yet,
// safest is to create a new Crawler and operate with it
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
	// string --> []string{}
	// "parent" --> ["child1", "child2"]
	sitemap sync.Map

	// Used to wait for all goroutines to complete
	wg sync.WaitGroup

	// Counts the number of websites that have been crawled
	totalCrawls int

	// Slice initialised in New with the list of suffixes that the crawler should ignore
	ignoreSuffixes []string
}

func (c *Crawler) New(baseSite string) *Crawler {
	if c == nil {
		c = new(Crawler)
	}

	// e.g. https://monzo.com/
	c.baseSite = baseSite

	// TODO: extract the domain from baseSite. Hardcoded for now.
	c.domain = "monzo.com"

	// Create buffered channel
	c.urls = make(chan string, MAX_CHAN_URLS)

	// Initialise list of ignore suffixes
	c.ignoreSuffixes = []string{"pdf", "png", "jpeg"}

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

func (c *Crawler) matchesIgnoreSuffix(url string) bool {

	for _, ext := range c.ignoreSuffixes {
		if strings.HasSuffix(url, ext) {
			return true
		}
	}
	return false
}

// TODO: what happens when an error is spit out by Crawl?
// the link is considered handled even though it is not traversed.

// Crawl site by visiting only local pages to the domain
// Returns error if any occured, nil if none
func (c *Crawler) Crawl() error {
	start1 := time.Now()

	// Get URL to crawl from channel
	defer c.wg.Done()
	url := <-c.urls

	// if URL matches any of the 'ignore' suffixes, return.
	// We don't want to crawl it.
	if c.matchesIgnoreSuffix(url) {
		return nil
	}

	// Fetch URL contents
	startHTTPGET := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		// TODO: add the url string to list of broken URLs
		// IMPROVEMENT: pass the err to Http404Error so that we have a reference to the original
		return Http404Error(url)
	}
	elapsedHTTPGET := time.Since(startHTTPGET)
	defer resp.Body.Close()

	// Read HTML from Body
	startReadBody := time.Now()
	bytes, err := ioutil.ReadAll(resp.Body)
	var html = string(bytes)
	if err != nil {
		return InvalidHTMLContent(url)
	}
	elapsedReadBody := time.Since(startReadBody)

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

	c.totalCrawls++

	defer log.Printf("Crawl #%d (%s)  took %s. GET(%s), HTML.Read(%s)\n",
		c.totalCrawls, url, time.Since(start1), elapsedHTTPGET, elapsedReadBody)
	// No error
	return nil
}

// Wait for all goroutines to finish - blocking function
func (c *Crawler) Wait() {
	c.wg.Wait()
}

func (c *Crawler) printRecurse(site string, indent int, visited map[string]bool) {
	visited[site] = true

	children, ok := c.sitemap.Load(site)

	if !ok {
		return
	}

	for _, child := range children.([]string) {

		// Print child
		for i := 0; i < indent; i++ {
			fmt.Printf(" ")
		}
		fmt.Printf("%s\n", child)

		// If child has not been visited do it
		if _, ok := visited[child]; !ok {
			c.printRecurse(child, indent+1, visited)
		}
	}
}

func (c *Crawler) PrintSitemapHierarchy() {
	visited := make(map[string]bool)

	fmt.Println(c.baseSite)
	c.printRecurse(c.baseSite, 1, visited)
}

func (c *Crawler) PrintSitemapFlat() {
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

	// Crawl and measure time taken
	start1 := time.Now()
	c = c.New("https://monzo.com")
	c.Start()
	c.Wait()
	elapsed1 := time.Since(start1)
	log.Printf("%d Crawls took %s\n", c.totalCrawls, elapsed1)

	// Print Sitemap Hierarchy
	// start2 := time.Now()
	// c.PrintSitemapHierarchy()
	// elapsed2 := time.Since(start2)
	// log.Printf("PrintSitemapHierarchy took %s\n", elapsed2)

	// Print Sitemap Flat
	// start3 := time.Now()
	// c.PrintSitemapFlat()
	// elapsed3 := time.Since(start3)
	// log.Printf("PrintSitemapHierarchy took %s\n", elapsed3)

	log.Printf("Crawler done. Exiting main.")
}
