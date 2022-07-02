package tracker_communication

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"trackerpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const PORT = "50051"

func RequestPeers(trackerUrl string, infoHash [20]byte, peerId string) map[string]string {
	cc, err := TrackerClient(trackerUrl)
	defer cc.Close()

	if err != nil {
		log.Printf("Error while creating grpc tracker client: %v", err)
		return nil
	}

	announceResp := Announce(trackerpb.NewTrackerClient(cc), infoHash, peerId)
	return announceResp.GetPeers()
}

func TrackerClient(trackerUrl string) (*grpc.ClientConn, error) {
	fmt.Println("Hello I'm a client")

	certFile := "./SSL/ca.pem" // Certificate Authority Trust certificate
	creds, sslErr := credentials.NewClientTLSFromFile(certFile, "")
	if sslErr != nil {
		log.Fatalf("Error while loading CA trust certificate: %v\n", sslErr)
		return nil, sslErr
	}
	opts := grpc.WithTransportCredentials(creds)
	cc, err := grpc.Dial(trackerUrl, opts)
	// cc, err := grpc.Dial(trackerUrl, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	return cc, nil
}

func PublishTorrent(trackerUrl string, infoHash [20]byte, peerId string) *trackerpb.PublishResponse {
	cc, err := TrackerClient(trackerUrl)
	c := trackerpb.NewTrackerClient(cc)
	defer cc.Close()
	if err != nil {
		log.Printf("Error while creating grpc tracker client: %v", err)
		return nil
	}
	fmt.Println("Starting to do a Publish RPC...")
	port, _ := strconv.Atoi(PORT)
	req := &trackerpb.PublishQuery{
		InfoHash: infoHash[:],
		PeerID:   peerId,
		IP:       "192.168.169.14",
		Port:     int32(port),
	}

	res, err := c.Publish(context.Background(), req)

	if err != nil {
		log.Fatalf("Error while calling Publish RPC: %v", err)
	}
	log.Printf("Response from Publish: %v", res.GetStatus())

	return res
}

func Announce(c trackerpb.TrackerClient, infoHash [20]byte, peerId string) *trackerpb.AnnounceResponse {
	fmt.Println("Starting to do an Announce RPC...")
	port, _ := strconv.Atoi(PORT)
	req := &trackerpb.AnnounceQuery{
		InfoHash: infoHash[:],
		PeerID:   peerId,
		IP:       "192.168.169.14",
		Port:     int32(port),
	}
	ctx := context.Background()
	res, err := c.Announce(ctx, req)
	// if p, ok := peer.FromContext(ctx); ok {
	// 	fmt.Println(p)
	// }

	// md, ok := metadata.FromIncomingContext(ctx)
	if err != nil {
		log.Fatalf("Error while calling Announce RPC: %v", err)
	}
	log.Printf("Response from Announce: %v", res.GetInterval())

	return res
}

func Scrape(c trackerpb.TrackerClient, infoHash [20]byte) *trackerpb.ScraperResponse {
	fmt.Println("Starting to do a Scrape RPC...")

	req := &trackerpb.ScraperQuery{
		InfoHash: [][]byte{infoHash[:]},
	}

	res, err := c.Scrape(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling Scrape RPC: %v", err)
	}
	log.Printf("Response from Scrape: %v", res.GetFiles())

	return res
}
