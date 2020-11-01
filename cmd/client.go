package main

import (
	"context"
	"log"
	"os"
	"time"

	pb "github.com/jasmaa/misaka-net/internal/grpc"
	"google.golang.org/grpc"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:8001", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewProgramClient(conn)

	// Contact the server and print out its response.
	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	switch cmd {
	case "run":
		r, err := c.Run(ctx, &pb.RunRequest{})
		if err != nil {
			log.Fatalf("could not run: %v", err)
		}
		log.Printf("Resp: %v", r)
	case "pause":
		r, err := c.Pause(ctx, &pb.PauseRequest{})
		if err != nil {
			log.Fatalf("could not run: %v", err)
		}
		log.Printf("Resp: %v", r)
	case "reset":
		r, err := c.Reset(ctx, &pb.ResetRequest{})
		if err != nil {
			log.Fatalf("could not run: %v", err)
		}
		log.Printf("Resp: %v", r)
	}
}
