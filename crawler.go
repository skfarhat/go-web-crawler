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

// Crawl site by visiting only local pages to the domain
// Returns error if any occured, nil if none
func Crawl(urls chan string, domain string, visited *sync.Map, wg *sync.WaitGroup) error {
	
	// Get URL to crawl from channel
	defer wg.Done()
	url := <-urls
	visited.Store(url, true)
	log.Printf("Crawl: %s, domain: %s\n", url, domain)

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
		children[i] = "https://" + domain + x
	}

	// Find absolute links
	absoluteLinks := FindAbsoluteLinks(html, &domain)

	children = append(children, absoluteLinks...)

	// Place child urls on the urls channel
	for _, x := range children {
		_, present :=  visited.Load(x)
		if ! present {
			urls <- x
			wg.Add(1)
			// TODO: check for Crawlers that return error. 
			go Crawl(urls, domain, visited, wg)
		}
	}

	// No error
	return nil
}

func main() {
	var wg sync.WaitGroup
	var visited sync.Map

	urls := make(chan string, 100) // create buffered channel 
	baseSite := "https://www.monzo.com/"
	domain := "monzo.com"

	urls <- baseSite
	wg.Add(1)
	// TODO: check for error returned from Crawl
	go Crawl(urls, domain, &visited, &wg)

	wg.Wait()
	log.Printf("Crawler done. Exiting main.")
}
