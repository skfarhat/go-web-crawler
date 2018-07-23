package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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

func urlToHTMLContent(url string) (string, error) {
	const ROOT_FOLDER string = "test_site/"
	var fileToRead string
	switch url {
	case "/":
		fileToRead = ROOT_FOLDER + "index.html"
	case "":
		fileToRead = ROOT_FOLDER + "index.html"
	default:
		fileToRead = ROOT_FOLDER + url
	}

	content, err := ioutil.ReadFile(fileToRead)
	return string(content), err

}

// Test Crawler works for test site
func TestCrawlSampleSite(t *testing.T) {
	log.Printf("Starting TestCrawlSampleSite")

	// TEST SETUP
	// ----------

	// Spawn a test server that returns static content in test_site
	// Note that test_site must be in the same directory as this script.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Load the HTML content of the file address by the path
		content, err := urlToHTMLContent(r.URL.Path)
		if err != nil {
			w.WriteHeader(404)
		}
		io.WriteString(w, content)
	}))
	defer ts.Close()

	// Start crawler
	var c Crawler
	c.Init(ts.URL)
	c.Start()
	c.Wait()

	// c.PrintSitemapFlat()

	// RUN TESTS
	// ---------

	// Test that absent URLs are indeed absent
	t.Run("TestPages4,5AreAbsent", func(t *testing.T) {

		// Page 4
		if _, ok := c.sitemap.Load(ts.URL + "/page4.html"); ok {
			t.Errorf("Sitemap contains link (page4.html) which it shouldn't.")
		}

		// Page 5
		if _, ok := c.sitemap.Load(ts.URL + "/page5.html"); ok {
			t.Errorf("Sitemap contains link (page5.html) which it shouldn't.")
		}
	})

	// Test all present URLs are indeed present
	t.Run("TestPages1,2,3,11,22a,22bArePresent", func(t *testing.T) {

		// Page 1
		if page1Children, ok := c.sitemap.Load(ts.URL + "/page1.html"); !ok {
			t.Errorf("Sitemap contains link (page1.html) which it shouldn't.")
		} else if i := Find(page1Children.([]string), ts.URL+"/page11.html"); i < 0 {
			t.Errorf("page11.html is not child of page1.html as it should be.")
		}

		// Page 2
		if page2Children, ok := c.sitemap.Load(ts.URL + "/page2.html"); !ok {
			t.Errorf("Sitemap contains link (page2.html) which it shouldn't.")
		} else if i := Find(page2Children.([]string), ts.URL+"/page22a.html"); i < 0 {
			t.Errorf("page22a.html is not child of page2.html as it should be.")
		}

		// Page 3
		if _, ok := c.sitemap.Load(ts.URL + "/page3.html"); !ok {
			t.Errorf("Sitemap contains link (page3.html) which it shouldn't.")
		}

		// Page 11
		if _, ok := c.sitemap.Load(ts.URL + "/page11.html"); !ok {
			t.Errorf("Sitemap contains link (page11.html) which it shouldn't.")
		}

		// Page 22a
		if page22aChildren, ok := c.sitemap.Load(ts.URL + "/page22a.html"); !ok {
			t.Errorf("Sitemap contains link (page22a.html) which it shouldn't.")
		} else if i := Find(page22aChildren.([]string), ts.URL+"/page22b.html"); i < 0 {
			t.Errorf("page22b.html is not child of page22a.html as it should be.")
		}

		// Page 22b
		if _, ok := c.sitemap.Load(ts.URL + "/page22b.html"); !ok {
			t.Errorf("Sitemap contains link (page22b.html) which it shouldn't.")
		}
	})

	// TEARDOWN
	// --------

	// Teardown here..

}
