package tracker_communication

import (
	"Bit_Torrent_Project/Client/trackerpb"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	//"Bit_Torrent_Project/Client/trackerpb"
	//"context"
	//"crypto/tls"
	//"crypto/x509"
	//"errors"
	//"fmt"
	//"io/ioutil"
	//"log"
	//"strconv"
	//
	//
)

const PORT = "50051"

// Requests peers to trackers
func RequestPeers(cc *grpc.ClientConn, trackerUrl string, infoHash [20]byte, peerId string, IP string, ctx context.Context) (map[string]string, error) {
	announceResp, err := Announce(trackerpb.NewTrackerClient(cc), ctx, infoHash, IP, peerId)
	if err != nil {
		return nil, err
	}
	return announceResp.GetPeers(), nil
}

// Creates grpc connection with tracker
func TrackerClient(trackerUrl string) (*grpc.ClientConn, context.Context, error) {
	fmt.Println("Hello I'm a client")

	certFile := "./client/tracker_communication/cert.pem" // Certificate Authority Trust certificate
	keyFile := "./client/tracker_communication/key.pem"   // Certificate Authority Trust certificate

	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
	caCert, _ := ioutil.ReadFile(certFile)

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{RootCAs: caCertPool, Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	opts := grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	cc, err := grpc.Dial(trackerUrl, opts)

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
		return nil, nil, err
	}
	log.Println("Connected to tracker")
	ctx := context.Background()

	return cc, ctx, nil
}

// Publishes new torrent in tracker
func PublishTorrent(trackerUrl string, infoHash [20]byte, peerId string, IP string) *trackerpb.PublishResponse {
	cc, ctx, err := TrackerClient(trackerUrl)
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
		IP:       IP,
		Port:     int32(port),
	}

	res, err := c.Publish(ctx, req)

	if err != nil {
		log.Fatalf("Error while calling Publish RPC: %v", err)
	}
	log.Printf("Response from Publish: %v", res.GetStatus())

	return res
}

// Sends first announce to tracker
func Announce(c trackerpb.TrackerClient, ctx context.Context, infoHash [20]byte, IP string, peerId string) (*trackerpb.AnnounceResponse, error) {
	fmt.Println("Starting to do an Announce RPC...")
	port, _ := strconv.Atoi(PORT)
	req := &trackerpb.AnnounceQuery{
		InfoHash: infoHash[:],
		PeerID:   peerId,
		IP:       IP,
		Port:     int32(port),
		Event:    "request",
		Request:  true,
	}
	res, err := c.Announce(ctx, req)
	if err != nil {
		log.Fatalf("Error while calling Announce RPC: %v", err)
	}
	log.Printf("Response from Announce: %v", res.GetInterval())
	if res.FailureReason != "" {
		return nil, errors.New(res.FailureReason)
	}
	return res, nil
}

// Calls Scrape RPC of tracker
func Scrape(c trackerpb.TrackerClient, ctx context.Context, infoHash [20]byte) *trackerpb.ScraperResponse {
	fmt.Println("Starting to do a Scrape RPC...")

	req := &trackerpb.ScraperQuery{
		InfoHash: [][]byte{infoHash[:]},
	}

	res, err := c.Scrape(ctx, req)
	if err != nil {
		log.Fatalf("Error while calling Scrape RPC: %v", err)
	}
	log.Printf("Response from Scrape: %v", res.GetFiles())

	return res
}
