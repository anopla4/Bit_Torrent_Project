package tracker_communication

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"trackerpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const PORT = "50051"

func RequestPeers(cc *grpc.ClientConn, trackerUrl string, infoHash [20]byte, peerId string, IP net.IP, ctx context.Context) map[string]string {
	announceResp := Announce(trackerpb.NewTrackerClient(cc), ctx, infoHash, IP, peerId)
	return announceResp.GetPeers()
}

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

	ctx := context.Background()

	return cc, ctx, nil
}

func PublishTorrent(trackerUrl string, infoHash [20]byte, peerId string, IP net.IP) *trackerpb.PublishResponse {
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
		IP:       IP.String(),
		Port:     int32(port),
	}

	res, err := c.Publish(ctx, req)

	if err != nil {
		log.Fatalf("Error while calling Publish RPC: %v", err)
	}
	log.Printf("Response from Publish: %v", res.GetStatus())

	return res
}

func Announce(c trackerpb.TrackerClient, ctx context.Context, infoHash [20]byte, IP net.IP, peerId string) *trackerpb.AnnounceResponse {
	fmt.Println("Starting to do an Announce RPC...")
	port, _ := strconv.Atoi(PORT)
	req := &trackerpb.AnnounceQuery{
		InfoHash: infoHash[:],
		PeerID:   peerId,
		IP:       IP.String(),
		Port:     int32(port),
		Event:    "request",
		Request:  true,
	}
	res, err := c.Announce(ctx, req)
	if err != nil {
		log.Fatalf("Error while calling Announce RPC: %v", err)
	}
	log.Printf("Response from Announce: %v", res.GetInterval())

	return res
}

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
