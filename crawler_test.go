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

// --------------
// Test Globals
// --------------

// Filename of Monzo's main html page
const MONZO_HTML_FILENAME string = "test-files/1/monzo.html"

// --------------
// Test Helpers
// --------------

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
func testArraysMatch(t *testing.T, arr1 []string, arr2 []string) int {

        // 1. Check that len(arr1) = len(arr2)
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


// --------------------
// Test Link handling
// --------------------

// Test that FindRelativeLinks finds all links that we expect it to find
// from the html file monzo-html.txt
func TestFindRelativeLinks_findsAllCorrectly(t *testing.T) {
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
        data, err := ioutil.ReadFile(MONZO_HTML_FILENAME)
        if err != nil {
                t.Errorf("Failed to open file %s", MONZO_HTML_FILENAME)
        }

        // Get relative links and test function
        results := FindRelativeLinks(string(data))
        res := testArraysMatch(t, RELATIVE_LINKS[:], results)

        if res == 1 {
                t.Errorf("Not all relative links were found. Expecting (%d), found (%d)\n",
                        len(RELATIVE_LINKS), len(results))
        } else if res == 2 {
                t.Errorf("Relative links found don't match those expected.")
        }
}

func TestFindAbsoluteLinks_findsAllCorrectly(t *testing.T) {
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
        data, err := ioutil.ReadFile(MONZO_HTML_FILENAME)
        if err != nil {
                t.Errorf("Failed to open file %s", MONZO_HTML_FILENAME)
        }

        // Get absolute links and test function
        results := FindAbsoluteLinks(string(data), nil)
        res := testArraysMatch(t, ABSOLUTE_LINKS[:], results)
        if res == 1 {
                t.Errorf("Not all absolute links were found. Expecting (%d), found (%d)\n",
                        len(ABSOLUTE_LINKS), len(results))
        } else if res == 2 {
                t.Errorf("Absolute links found don't match those expected.")
        }
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
	const ROOT_FOLDER string = "test-files/test_site/"
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
			t.Errorf("Sitemap does not contain (page1.html) as it should.")
		} else if i := Find(page1Children.([]string), ts.URL+"/page11.html"); i < 0 {
			t.Errorf("page11.html is not child of page1.html as it should be.")
		}

		// Page 2
		if page2Children, ok := c.sitemap.Load(ts.URL + "/page2.html"); !ok {
			t.Errorf("Sitemap does not contain (page2.html) as it should.")
		} else if i := Find(page2Children.([]string), ts.URL+"/page22a.html"); i < 0 {
			t.Errorf("page22a.html is not child of page2.html as it should be.")
		}

		// Page 3
		if _, ok := c.sitemap.Load(ts.URL + "/page3.html"); !ok {
			t.Errorf("Sitemap does not contain (page3.html) as it should.")
		}

		// Page 11
		if _, ok := c.sitemap.Load(ts.URL + "/page11.html"); !ok {
			t.Errorf("Sitemap does not contain (page11.html) as it should.")
		}

		// Page 22a
		if page22aChildren, ok := c.sitemap.Load(ts.URL + "/page22a.html"); !ok {
			t.Errorf("Sitemap does not contain (page22a.html) as it should.")
		} else if i := Find(page22aChildren.([]string), ts.URL+"/page22b.html"); i < 0 {
			t.Errorf("page22b.html is not child of page22a.html as it should be.")
		}

		// Page 22b
		if _, ok := c.sitemap.Load(ts.URL + "/page22b.html"); !ok {
			t.Errorf("Sitemap does not contain (page22b.html) as it should.")
		}
	})

	// TEARDOWN
	// --------

	// Teardown here..

}
