package cli

import (
	"context"

	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"

	"../crawler/proto"
)

func main() {

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Create client for RPC service
	c := crawler.NewCrawlerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	//
	switch os.Args[1] {

	case "-start":
		root:= os.Args[2]
		reply, err := c.Start(ctx, &crawler.CrawlerRequest{Root: root})

		if err != nil {
			log.Println(err)
		}

		log.Println(reply.Message)

	case "-stop":
		root:= os.Args[2]
		reply, err := c.Stop(ctx, &crawler.CrawlerRequest{Root: root})

		if err != nil {
			log.Println(err)
		}

		log.Println(reply.Message)
	case "-list":
		stream, err := c.List(ctx, &crawler.CrawlerRequest{Root: "*"})

		if err != nil {
			log.Println(err)
		}

		for {
			tree, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("%v.List = _, %v", conn, err)
			}
			log.Println(tree)
		}

	default:
		log.Println("expected '-start', '-stop', or '-list'")
		os.Exit(1)
	}
}
