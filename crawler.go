/* Task:
 simple web crawler
 Given a URL, it should output a simple textual sitemap, showing the links between pages.
- The crawler should be limited to one subdomain
- not follow external links

show links
*/

/* Method:

- struct to contain a pages url and sub-pages
- main func sets recursive crawler off on base domain with given depth
- walk function does the following:
	1) check if depth == 0, and return if true
	2) get links in page
	3) for each link, set recursive call to parse its link with decremented depth
	4) send link struct pointer to recursive call so when they all fold back down the walked structure is preserved in struct
- after all walking routines finished, output structure to text file

*/

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"golang.org/x/net/html"
)

// Link struct represents a page and all links found on the page
type Link struct {
	URL   *url.URL
	links []*Link
}

// Concurrency safe set implementation of seen links. saves us from traversing previously traversed links
// although doesn't guarantee max depth is adheered to
type Processed struct {
	links map[string]*Link
	mux   sync.Mutex
}

var processed = Processed{links: make(map[string]*Link)}
var domain string
var globalWait sync.WaitGroup

func main() {
	// Setup logger
	f, err := os.OpenFile("run.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// Define and save command line args
	baseURLStr := flag.String("b", "monzo.com", "Starting URL to crawl from")
	var searchDepth = *flag.Int("d", 7, "Number of levels you want to traverse (depth)")
	flag.Parse()

	// Start the crawl
	log.Print(fmt.Sprintf("------- STARTING NEW CRAWL FOR: %s -------", *baseURLStr))

	baseURL, err := parseURL(*baseURLStr)
	if err != nil {
		logger("e", "Base url could not be parsed")
		panic(err)
	}

	// create link and set domain
	domain = baseURL.Hostname()
	baseLink := &Link{URL: baseURL}
	processed.links[baseURL.String()] = baseLink

	// start crawl from base link
	globalWait.Add(1)
	go crawl(baseLink, searchDepth)
	globalWait.Wait()

}

/*
Crawls one page:
1) check if depth reached, and return if it has
2) gets all links on a page
3) for each link
	. recursively set a new go routine running to crawl
*/
func crawl(link *Link, depth int) {
	// handle wait group at start
	defer globalWait.Done()

	// check depth
	if depth == 0 {
		return
	}

	// get all links on the given page
	newLinks, err := getLinksFromURL(link.URL)
	if err != nil {
		logger("e", fmt.Sprintf("Error getting links for: %s \n", link.URL.String()))
		return
	}

	for _, newL := range newLinks {
		logger("i", fmt.Sprintf("URL found: %s", newL.URL.String()))
	}
}

func getLinksFromURL(link *url.URL) ([]*Link, error) {
	// get response
	resp, err := http.Get(link.String())
	if err != nil {
		logger("e", fmt.Sprintf("Error getting response from: %s", link.String()))
		return nil, err
	}

	// parse for <a> tags
	var newLinks []*Link
	z := html.NewTokenizer(resp.Body)

	for {
		token := z.Next()

		switch {
		case token == html.ErrorToken:
			// End of page, return
			return newLinks, nil
		case token == html.StartTagToken:
			// check if anchor tag found
			tag := z.Token()
			// worth noting  here thatthis does not guarantee 100% of links, could be stuff in javascript somewhere
			isAnchor := tag.Data == "a"
			if isAnchor {
				// get href attribute
				for _, a := range tag.Attr {
					if a.Key == "href" {

						// link found, lets check if it belongs to current subdomain
						l, err := parseURL(a.Val)
						if err != nil {
							logger("e", fmt.Sprintf("Error parsing link: %s", a.Val))
							return nil, err
						}

						if l.Hostname() == domain {
							// we will use this link
							lLink := &Link{URL: l}
							newLinks = append(newLinks, lLink)
							break
						}
					}
				}
			}
		}
	}
}

// Takes a url string and retruns a url.URL type if valid
func parseURL(URLString string) (*url.URL, error) {
	resultURL, err := url.Parse(URLString)
	if err != nil {
		logger("e", fmt.Sprintf("Could not parse URL: %s", URLString))
	}
	return resultURL, err
}

// Allows for levels of severity in our logger
func logger(severity string, message string) {
	switch severity {
	case "e":
		log.Print(fmt.Sprintf("[ERROR] %s", message))
	case "i":
		log.Print(fmt.Sprintf("[INFO] %s", message))
	}
}
