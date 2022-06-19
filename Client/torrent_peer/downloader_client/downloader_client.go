package downloader_client

import (
	"context"
	"fmt"
	"io"
	"log"
	"torrent_peerpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func StartClient(port string) (*grpc.ClientConn, error) {
	fmt.Println("Hello I'm a client")

	tls := false
	opts := grpc.WithInsecure()

	// SSL credentials
	if tls {
		certFile := "ssl/ca.crt" // Certificate Authority Trust certificate
		creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
		if sslErr != nil {
			log.Fatalf("Error while loading CA trust certificate: %v\n", sslErr)
			return nil, sslErr
		}
		opts = grpc.WithTransportCredentials(creds)
	}

	// Start connection with server

	return grpc.Dial("localhost:"+port, opts)
}

func MessageStream(c torrent_peerpb.DownloadServiceClient, requests []*torrent_peerpb.Message) {

	// Create a stream by invoking the client
	stream, err := c.SendMessage(context.Background())

	if err != nil {
		log.Fatalf("Error while creating stream: %v\n", err)
		return
	}

	waitc := make(chan struct{})

	// Send messages to server
	go func() {
		for _, msg := range requests {
			fmt.Printf("Sending message:  %v\n", msg)
			sendErr := stream.Send(&torrent_peerpb.SendMessageRequest{Message: msg})

			if sendErr != nil {
				log.Fatalf("Error while sending request to client: %v\n", err)
				break
			}
		}
		err := stream.CloseSend()
		if err != nil {
			log.Fatalf("Error while closing send stream: %v\n", err)
		}
	}()

	// Receive messages from server

	go func() {
		for {
			res, streamErr := stream.Recv()
			if streamErr == io.EOF {
				break
			}
			if streamErr != nil {
				log.Fatalf("Error while receiving data: %v\n", err)
				break
			}
			fmt.Printf("Response: %v\n", res)
		}
		close(waitc)
	}()

	<-waitc
}
