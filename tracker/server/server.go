package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	//"encoding/json"
	"flag"
	"log"

	//"math/rand"
	"net"
	//time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tk "Bit_Torrent_Project/tracker/tracker"
	pb "Bit_Torrent_Project/tracker/trackerpb"
)

func loadTLSCredentials() (credentials.TransportCredentials, error) {

	pemClientCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		return nil, err
	}
	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(config), nil
}

func newTrackerServer() *tk.TrackerServer {
	//s := &routeGuideServer{routeNotes: make(map[string][]*pb.RouteNote)}
	//s.loadFeatures(*jsonDBFile)
	tt := &tk.TrackerServer{}
	return tt
}

func main() {
	port := flag.Int("port", 8168, "set port for TCP conection")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	//fmt.Printf("Server in port:%d", *port)

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}
	sr := grpc.NewServer(grpc.Creds(tlsCredentials))

	pb.RegisterTrackerServer(sr, newTrackerServer())
	err2 := sr.Serve(lis)
	if err2 != nil {
		log.Fatalf("failed to listen: %v", err2)
	}
}
