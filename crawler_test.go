package main

import (
	"io/ioutil"
	"log"
	// "net/url"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------
// Test Error implementations
// ---------------------------

func TestHttp404Error(t *testing.T) {
	_ = Http404Error("Some error message")
}

func TestInvalidHTMLContent(t *testing.T) {
	_ = InvalidHTMLContent("Some error message")
}

func TestInvalidURL(t *testing.T) {
	_ = InvalidURL("Some error message")
}

// --------------
// Test Crawler
// --------------

// Test that an error is retuned whne the baseSite is not a valid URL
func TestInit_returnsErrorWhenSiteIsNotAValidURL(t *testing.T) {
	const baseSite string = "hello"
	var c Crawler
	e := c.Init(baseSite)
	if e == nil {
		t.Errorf("Expecting error but nothing\n")
	}
}

// Test that no error comes from Init when the provided baseSite is a valid URL
func TestInit_noErrorWhenSiteIsValidURL(t *testing.T) {
	const baseSite string = "https://monzo.com/abcde"
	var c Crawler
	e := c.Init(baseSite)
	if e != nil {
		t.Errorf("Go an unexpected error %s\n", e)
	}
}

// Test that Crawler.Wait() does not block when wg.Add(1) has not been called
func TestWait_doesNotBlockWGIsZero(t *testing.T) {
	var c Crawler
	start := time.Now()
	c.Wait()
	elapsed := time.Since(start)
	if elapsed > 3*time.Second {
		t.Errorf("Blocked for too long on c.Wait() - something is fishy.")
	}
}

// Test that Crawler.Wait() works by ensuring it blocks for at least x seconds
// before getting to the end of the function. We start a goroutine that calls Crawler.wg.Done()
// after x seconds.
func TestWait_blocksWhenWGNotZero(t *testing.T) {
	delay := 1 * time.Second
	var c Crawler
	c.wg.Add(1)
	go func() {
		time.Sleep(delay)
		c.wg.Done()
	}()

	start := time.Now()
	c.Wait()
	elapsed := time.Since(start)

	if elapsed < delay {
		t.Errorf("Crawler.Wait() failed to block")
	}
}

// Test that we can call PrintSitemapFlat without having called Start()
func TestPrintSitemapFlat_doesNotCrashIfEmpty(t *testing.T) {
	var c Crawler
	c.Init("https://monzo.com")
	c.PrintSitemapFlat()
}

// Test that we can call PrintSitemapFlattest without having called Start()
func TestPrintSitemapFlattest_doesNotCrashIfEmpty(t *testing.T) {
	var c Crawler
	c.Init("https://monzo.com")
	c.PrintSitemapFlattest()
}

// Test Crawler works for test site
func TestCrawlSampleSite(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("path is %s\n", r.URL.Path)
		// fmt.FPrintln(w, "Hello, client")
	}))
	defer ts.Close()

	// TODO: construct the URL here 
	res, err := http.Get(ts.URL)
	if err != nil {
		log.Fatal(err)
	}
	greeting, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%s", greeting)
}
