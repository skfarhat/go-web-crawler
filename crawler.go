package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
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
	var domainPattern string = "([^:\\/\\s]+)"
	if domain == nil { 
		domain = &domainPattern
	}

	// Absolute pattern to match 
	// http[s] is required for the absolute link to match, otherwise we would match relative links as well.
	// The domain may be specified by the caller of the function, otherwise a default domain pattern matcher is used.
	var absolutePattern string = fmt.Sprintf("href=\"((http[s]?:\\/\\/)([^\\s\\/]*\\.)?%s(\\/[^\\s]*)*)\"", *domain)
	const captureGroup int = 1

	re := regexp.MustCompile(absolutePattern)
	allMatches := re.FindAllStringSubmatch(html, -1)
	
	b := make([]string, len(allMatches))

	// take the first capturing group from matches
	for i, x := range allMatches {
		b[i] = x[captureGroup]
	}

	return b
}

// Crawl site by visiting only local pages to the domain
func Crawl(url string, urls chan string, domain string) error {
	log.Printf("crawl: %s, domain: %s\n", url, domain)

	// Fetch URL contents
	resp, err := http.Get(url)
	if err != nil {
		// TODO: add the url string to list of broken URLs
		// IMPROVEMENT: pass the err to Http404Error so that we have a reference to the original
		return Http404Error(url)
	}
	defer resp.Body.Close()

	// Read HTML from Body
	var bytes []byte
	bytes, err = ioutil.ReadAll(resp.Body)
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

	// Debug 
	for i, x := range children { 
		fmt.Println(i, x)
	}
	
	// Add normalised urls to (urls) channel
	return nil
}

func main() {
	urls := make(chan string)
	baseSite := "https://www.monzo.com/"
	domain := "monzo.com"

	Crawl(baseSite, urls, domain)
}
