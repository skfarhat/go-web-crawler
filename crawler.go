package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
)

// Custom errors
type Http404Error string
type InvalidHTMLContent string

// Error implementations

func (e Http404Error) Error() string {
	return fmt.Sprintf("Failed to find for URL (%s).", e)
}

func (e InvalidHTMLContent) Error() string {
	return fmt.Sprintf("Could not parse HTML content for URL (%s).", e)
}

func FindRelativeLinks(html string) []string {
	const relativePattern string = "href=\"(/[-\\w\\d_/\\.]+)\""
	re := regexp.MustCompile(relativePattern)
	allMatches := re.FindAllStringSubmatch(html, -1)
	b := make([]string, len(allMatches))

	// take the first capturing group from matches
	for i, x := range allMatches {
		b[i] = x[1]
	}
	return b
}

func FindAbsoluteLinks(html string) []string {
	const aboslutePattern string = "href=\"((http(s)?):\\/\\/[-\\w\\d_/\\.]+)\""
	re := regexp.MustCompile(aboslutePattern)
	allMatches := re.FindAllStringSubmatch(html, -1)
	b := make([]string, len(allMatches))

	// take the first capturing group from matches
	for i, x := range allMatches {
		b[i] = x[1]
	}
	return b
}

// Crawl site by visiting only local pages. Base site is specified using 'baseSite' parameter 
func Crawl(url string, urls chan string, baseSite string) error {
	log.Printf("crawl: %s, baseSite: %s\n", url, baseSite)

	// Fetch URL contents
	resp, err := http.Get(url)
	if err != nil {
		// TODO: add the url string to list of broken URLs
		// IMPROVEMENT: pass the err to Http404Error so that we have a reference to the original
		return Http404Error(url)
	}
	defer resp.Body.Close()

	var bytes []byte
	bytes, err = ioutil.ReadAll(resp.Body)
	var html = string(bytes)
	if err != nil {
		return InvalidHTMLContent(url)
	}

	// Parse contents for list of href=
	relativeLinks := FindRelativeLinks(html)
	// absoluteLinks := FindAbsoluteLinks(html)

	// Filter absolute links that don't match base site
	fmt.Printf("Type of relativeLinks is %T\n", relativeLinks)
	for i, x := range relativeLinks {
		relativeLinks[i] = baseSite	+ x
		fmt.Println(relativeLinks[i])
	}
	// Change: relative url -> absolute url

	// Add normalised urls to (urls) channel
	return nil
}

func main() {
	urls := make(chan string)
	Crawl("https://www.monzo.com/", urls, "https://monzo.com/")
}
