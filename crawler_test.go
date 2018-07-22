package main

import (
	// "fmt"
	"io/ioutil"
	"sync"
	"testing"
)

// TODO: test that TestFindAbsoluteLinks returns only monzo.com when provided the right domain
//

// IMPROVEMENT: move this to utils package
// Find returns the smallest index i at which x == a[i],
// or -1 if there is no such index.
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

// Checks that the two string arrays have the same unordered contents
// Returns 0 if they match, 1 lengths don't match, 2 if lengths match but not the contents
func checkArraysMatch(t *testing.T, arr1 []string, arr2 []string) int {

	// 1. Check that len(arr1) = len(arr2)
	if len(arr1) != len(arr2) {
		return 1
	}

	// 2. Check that the arrays have the same content (unordered)
	for _, res := range arr1 {
		i := Find(arr2, res)
		if i < 0 {
			return 2
		}
	}
	return 0
}

// Test that FindRelativeLinks finds all links that we expect it to find
// from the html file monzo-html.txt
func TestFindRelativeLinks_findsAllCorrectly(t *testing.T) {
	// Filename of Monzo's main html page
	const monzoHTMLFilename string = "monzo.html"
	// Relative links we expect to find in the file
	RELATIVE_LINKS := [...]string{
		"/static/images/favicon.png",
		"/static/images/mondo-mark-01.png",
		"/feed.xml",
		"/about",
		"/blog",
		"/community",
		"/faq",
		"/download",
		"/-play-store-redirect",
		"/features/apple-pay",
		"/features/travel",
		"/features/switch",
		"/features/overdrafts",
		"/-play-store-redirect",
		"/-play-store-redirect",
		"/about",
		"/blog",
		"/press",
		"/careers",
		"/community",
		"/transparency",
		"/blog/how-money-works",
		"/tone-of-voice",
		"/faq",
		"/legal/terms-and-conditions",
		"/legal/fscs-information",
		"/legal/privacy-policy",
		"/legal/cookie-policy",
		"/-play-store-redirect",
	}

	// Read HTML file
	data, err := ioutil.ReadFile(monzoHTMLFilename)
	if err != nil {
		t.Errorf("Failed to open file %s", monzoHTMLFilename)
	}

	// Get relative links and test function
	results := FindRelativeLinks(string(data))
	res := checkArraysMatch(t, RELATIVE_LINKS[:], results)

	if res == 1 {
		t.Errorf("Not all relative links were found. Expecting (%d), found (%d)\n",
			len(RELATIVE_LINKS), len(results))
	} else if res == 2 {
		t.Errorf("Relative links found don't match those expected.")
	}
}

func TestFindAbsoluteLinks_findsAllCorrectly(t *testing.T) {
	// Filename of Monzo's main html page
	const monzoHTMLFilename string = "monzo.html"
	// Absolute links we expect to find in the file
	ABSOLUTE_LINKS := [...]string{
		"https://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.7.0/css/font-awesome.min.css",
		"https://cdnjs.cloudflare.com/ajax/libs/sweetalert/1.1.3/sweetalert.min.css",
		"https://itunes.apple.com/gb/app/mondo/id1052238659",
		"https://www.theguardian.com/technology/2017/dec/17/monzo-facebook-of-banking",
		"https://www.telegraph.co.uk/personal-banking/current-accounts/monzo-atom-revolut-starling-everything-need-know-digital-banks/",
		"https://www.thetimes.co.uk/article/tom-blomfield-the-man-who-made-monzo-g8z59dr8n",
		"https://www.standard.co.uk/tech/monzo-prepaid-card-current-accounts-challenger-bank-a3805761.html",
		"https://www.fscs.org.uk/",
		"https://itunes.apple.com/gb/app/mondo/id1052238659",
		"https://monzo.com/community",
		"https://itunes.apple.com/gb/app/mondo/id1052238659",
		"https://web.monzo.com",
		"https://itunes.apple.com/gb/app/mondo/id1052238659",
		"https://twitter.com/monzo",
		"https://www.facebook.com/monzobank",
		"https://www.linkedin.com/company/monzo-bank",
		"https://www.youtube.com/monzobank",
	}

	// Read HTML file
	data, err := ioutil.ReadFile(monzoHTMLFilename)
	if err != nil {
		t.Errorf("Failed to open file %s", monzoHTMLFilename)
	}

	// Get absolute links and test function
	results := FindAbsoluteLinks(string(data), nil)
	res := checkArraysMatch(t, ABSOLUTE_LINKS[:], results)
	if res == 1 {
		t.Errorf("Not all absolute links were found. Expecting (%d), found (%d)\n",
			len(ABSOLUTE_LINKS), len(results))
	} else if res == 2 {
		t.Errorf("Absolute links found don't match those expected.")
	}
}

func TestCrawlerNew(t *testing.T) {
	var crawler Crawler
	crawler.New("https://monzo.com")
}

func TestCrawlerNew_fromNullPointer(t *testing.T) {
	var crawler *Crawler
	crawler = crawler.New("https://monzo.com")
	if crawler == nil {
		t.Errorf("Failed to create Crawler from nil pointer")
	}
}

// Test that PrintSitemap doesn't fail horribly in simple cases
func TestPrintSitemap_noFailOnEmpty(t *testing.T) {
	var crawler Crawler
	crawler.New("https://monzo.com")
	crawler.PrintSitemap()
}
