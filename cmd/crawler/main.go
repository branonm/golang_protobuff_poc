package main

import (
	"context"
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"google.golang.org/grpc"

	"./proto"
)

// TODO: Check env for port before assigning default
const (
	port = ":50051"
)

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	// Iterate over token attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	return
}


type siteTree struct {
	root        string
	controlChan chan int
	links       []string
	crawlDone   bool
}

func (st *siteTree) String() string {
	// Stub for converting the links slice to something printable
	return "This is a stub"
}

type crawlerServer struct {
	crawler.CrawlerServer
	rootMap sync.Map
}

// Start crawling on a particular root URL
func (s *crawlerServer) Start(ctx context.Context, request *crawler.CrawlerRequest) (*crawler.CrawlerReply, error) {

	// TODO: Confirm request.Root is valid url
	// TODO: Handle situation where request.Root is a subdir some other url already being crawled.

	root := request.Root
	log.Printf("Received root: %v", root)

	// TODO: call to rootMap.Load is common to 3 methods; extract to common method
	if iFace, loaded := s.rootMap.Load(root); loaded {
		tree := iFace.(siteTree)
		if tree.crawlDone {
			// Maybe restart instead? requirements are clear
			return &crawler.CrawlerReply{Message: "Crawl Done!"}, nil
		} else {
			return &crawler.CrawlerReply{Message: "Already Crawling!"}, nil
		}
	} else {
		thisTree := siteTree{
			root,
			make(chan int),
			make([]string, 0),
			false,
		}
		s.rootMap.Store(root, thisTree)

		go func(tree siteTree) {

			for !tree.crawlDone {

				// TODO: refactor so that the select fires once per URL scanned to keep
				// TODO: stopping responsive
				select {
				case <-tree.controlChan:
					tree.crawlDone = true
					close(tree.controlChan)

				default:
					resp, err := http.Get(tree.root)

					if err != nil {
						log.Printf("Failed to crawl %s", tree.root)
						//tree.crawlDone = true
						close(tree.controlChan)
						return
					}

					b := resp.Body
					defer b.Close()
					z := html.NewTokenizer(b)

					for {
						tt := z.Next()

						switch {
						case tt == html.ErrorToken:
							// End of the document, we're done
							tree.crawlDone = true
							return
						case tt == html.StartTagToken:
							t := z.Token()

							isAnchor := t.Data == "a"
							if !isAnchor {
								continue
							}

							ok, url := getHref(t)
							if !ok {
								continue
							}

							hasProto := strings.Index(url, "http") == 0
							if hasProto {
								tree.links = append(tree.links, url)
							}
						}
					}

					// TODO : loop through tree.links looking for more links. Don't forget to dedup.

				}
			}
		}(thisTree)

		return &crawler.CrawlerReply{Message: "Started!"}, nil
	}
}

// Stop crawling a particular root URL
func (s *crawlerServer) Stop(ctx context.Context, request *crawler.CrawlerRequest) (reply *crawler.CrawlerReply, err error) {

	if iFace, loaded := s.rootMap.Load(request.Root); loaded {
		tree := iFace.(siteTree)

		// Stop the goroutine responsible for the crawl
		tree.controlChan <- 1
		tree.crawlDone = true

		reply = &crawler.CrawlerReply{Message: "Stopping " + request.Root}
	} else {
		reply = &crawler.CrawlerReply{Message: "Not Running: " + request.Root}
	}

	return reply, err
}

// List all site trees
func (s *crawlerServer) List(request *crawler.CrawlerRequest, listServer crawler.Crawler_ListServer) error {

	s.rootMap.Range(func(key interface{}, value interface{}) bool {
		tree := value.(siteTree)
		listServer.Send(&crawler.CrawlerReply{Message: tree.String()})
		return true
	})

	return nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	fmt.Printf("Listening on port %s", port)
	if err != nil {
		fmt.Errorf("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()

	crawler.RegisterCrawlerServer(s, &crawlerServer{})
	if err := s.Serve(lis); err != nil {
		fmt.Errorf("failed to serve: %v", err)
		return
	}

}
