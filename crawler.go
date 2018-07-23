package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ---------------------
// Globals
// ---------------------

var verbose *bool

// ---------------------
// Error implementations
// ---------------------

type Http404Error string
type InvalidHTMLContent string
type InvalidURL string

func (e Http404Error) Error() string {
	return fmt.Sprintf("Failed to find for URL (%s).", e)
}

func (e InvalidHTMLContent) Error() string {
	return fmt.Sprintf("Could not parse HTML content for URL (%s).", e)
}

func (e InvalidURL) Error() string {
	return fmt.Sprintf("Failed to parse URL (%s)", e)
}

// --------------------
// Crawler
// --------------------

// Stat struct storing the elapsed time of the total crawl operation and that of HTTP.GET
type CrawlStat struct {
	getTime   time.Duration
	totalTime time.Duration
}

// Used for 'urls' buffered channel
const MAX_CHAN_URLS int = 100

// Crawler has not been tested with successive crawls yet (TODO)
// Safest is to create a new Crawler and operate with it
type Crawler struct {

	// First site to crawl
	baseSite string

	// Domain name encompassing all crawls, this will extracted from 'baseSite' in New()
	// and used for the regex FindAbsoluteLinks
	domain string

	// urls channel used by all goroutines to add new URLs to parse
	urls chan string

	// Cache sites that have been visited
	// string 	--> bool
	// "site"	--> true
	visited sync.Map

	// Cache relationship between visited sites, used to construct and print sitemap
	// string --> []string{}
	// "parent" --> ["child1", "child2"]
	sitemap sync.Map

	// Used to wait for all goroutines to complete
	wg sync.WaitGroup

	// Counts the number of websites that have been crawled
	totalCrawls int

	// Slice initialised in New with the list of suffixes that the crawler should ignore
	ignoreSuffixes []string

	// Stores statistics about each URL crawled
	stats sync.Map
}

// Initialise the Crawler
// Must be called before other functions
// Returns error if any occured, otherwise returns nil
func (c *Crawler) Init(baseSite string) error {

	// Parse baseSite URL
	u, e := url.Parse(baseSite)

	if e != nil || len(u.Host) == 0 {
		return InvalidURL(baseSite)
	}

	// Reset the map
	c.stats = sync.Map{}

	// e.g. https://monzo.com/
	c.baseSite = baseSite

	// Create buffered channel
	c.urls = make(chan string, MAX_CHAN_URLS)

	// Extract the domain from the parsed URL
	c.domain = u.Host

	// Initialise list of ignore suffixes
	c.ignoreSuffixes = []string{"pdf", "png", "jpeg"}

	return nil
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
	go c.Crawl()
}

// Checks if the provided URL ends with any of the suffixes defined in ignoreSuffixes.
// Returns true if it does, otherwise false.
func (c *Crawler) matchesIgnoreSuffix(url string) bool {
	for _, ext := range c.ignoreSuffixes {
		if strings.HasSuffix(url, ext) {
			return true
		}
	}
	return false
}

// Crawl site by visiting only local pages to the domain
// Returns error if any occured, nil if none
func (c *Crawler) Crawl() error {
	start1 := time.Now()

	// Get URL to crawl from channel
	defer c.wg.Done()
	url := <-c.urls

	// If URL matches any of the 'ignore' suffixes, return.
	// We don't want to crawl it.
	if c.matchesIgnoreSuffix(url) {
		return nil
	}

	// Fetch URL contents
	startHTTPGET := time.Now()
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode >= 300 {
		// TODO LATER: add the url string to list of broken URLs
		c.visited.Delete(url)
		return Http404Error(url)
	}
	defer resp.Body.Close()
	elapsedHTTPGET := time.Since(startHTTPGET)

	// Read HTML from Body
	bytes, err := ioutil.ReadAll(resp.Body)
	var html = string(bytes)
	if err != nil {
		return InvalidHTMLContent(url)
	}

	// Find relative links and convert them to absolute
	children := FindRelativeLinks(html)
	for i, x := range children {
		children[i] = c.baseSite + x
	}

	// Find absolute links
	absoluteLinks := FindAbsoluteLinks(html, &c.domain)

	// Concatenate relative and absolute children together
	children = append(children, absoluteLinks...)

	// Store URL in sitemap along with its children
	// Storing the children helps reconstruct the hierarchy if needed
	c.sitemap.Store(url, children)

	// Place child urls on the urls channel
	for _, x := range children {
		if _, present := c.visited.Load(x); !present {
			c.addSite(x)
			go c.Crawl()
		}
	}

	// Increment number of pages crawled
	c.totalCrawls++

	// Compute total time taken and store stats
	totalTime := time.Since(start1)
	c.stats.Store(url, CrawlStat{totalTime: totalTime, getTime: elapsedHTTPGET})

	// No error
	return nil
}

// Wait for all goroutines to finish - blocking function
func (c *Crawler) Wait() {
	c.wg.Wait()
}

// Print all crawled URLs and print them without any hierarchical relationship to their children
func (c *Crawler) PrintSitemapFlattest() {
	c.sitemap.Range(func(k, v interface{}) bool {
		fmt.Println(k)
		return true
	})
}

// Print all sites that have been crawled along with their children.
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

// TODO LATER: look into using 'net/url' package instead

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
	// Parse command line
	verbose = flag.Bool("verbose", false, "Provides versbose output.")
	printMode := flag.String("printmode", "mode1", "options: mode1 (flattest), mode2 (flat)")
	flag.Parse()

	// Crawl and measure time taken
	var c *Crawler = new(Crawler)
	start := time.Now()
	c.Init("https://monzo.com")
	c.Start()
	c.Wait()
	elapsed := time.Since(start)

	if *verbose {
		c.stats.Range(func(url, stats interface{}) bool {
			stat := stats.(CrawlStat)
			log.Printf("Crawl (%s)  took %s. GET(%s)\n", url, stat.totalTime, stat.getTime)
			return true
		})
		log.Printf("%d Crawls took %s\n", c.totalCrawls, elapsed)
	}

	switch *printMode {
	case "mode1":
		c.PrintSitemapFlattest()
	case "mode2":
		c.PrintSitemapFlat()
	default:
		log.Fatalf("Unknown printmode (%s). Not printing.\n", *printMode)
	}

	log.Printf("Crawler done. Exiting main.")
}
