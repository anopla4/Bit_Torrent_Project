package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"io/ioutil"

	"encoding/json"
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


func exampleTracker() (tk.TorrentsPool, error){
	peers := make(tk.PeersPool, 3)
	peers["peerId1"] = &tk.PeerTk{
		ID: "peerId1",
		State: "seeder",
		Addr: &net.TCPAddr{IP: net.ParseIP("108.56.6.6"),Port: 5000},
	}
	peers["peerIdX"] = &tk.PeerTk{
		ID: "peerIdX",
		State: "lecher",
		Addr: &net.TCPAddr{IP: net.ParseIP("190.122.0.4"),Port: 5000},
	}
	
	torrents := make(tk.TorrentsPool, 5)
	torrents["12345678912345678900"] = &tk.TorrentTk{
		Hash: "12345678912345678900",
		Downloaded: 0,
		Complete: 0,
		Peers: peers,
	}
	torrents["0000000000000000000"] = &tk.TorrentTk{
		Hash: "0000000000000000000",
		Downloaded: 200,
		Complete: 11,
		Peers: peers,
	}
	
	return torrents, nil
}

func jsonPruve(tp tk.TorrentsPool) error{
	jso, err := json.MarshalIndent(tp, "", " ")
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
    //fmt.Println(string(jso))
	e := ioutil.WriteFile("bkdata/torrents.json", jso, fs.ModeExclusive)
	if e != nil {
		fmt.Println(e.Error())
		return e
	}
	fmt.Println()

	tpp := tk.TorrentsPool{}

	err = json.Unmarshal(jso, &tpp)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	//jts, _ := json.MarshalIndent(tpp, "", " ")
	//fmt.Println(string(jts))
	return nil
}

func runPruve(){
	ts, err := exampleTracker()
	if err != nil {
		fmt.Println(err.Error())
		return 
	}
	e := jsonPruve(ts)
	if e != nil {
		fmt.Println(e.Error())
		return 
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {

	pemClientCA, err := ioutil.ReadFile("cert/ca.pem")
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server.pem", "cert/server.key")
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
	return &tk.TrackerServer{Torrents: make(tk.TorrentsPool,10)}
}

func newTrackerServerFromLoad() *tk.TrackerServer {
	tstate, err := ioutil.ReadFile("bkdata/torrents.json")
	if err!= nil{
		log.Println("Failed to load backoup info: " +  err.Error())
		return newTrackerServer()
	}
	var torrentsLoads = tk.TorrentsPool{}
	err = json.Unmarshal(tstate, &torrentsLoads)
	if err != nil {
		log.Println(err.Error())
		return newTrackerServer()
	}
	if len(torrentsLoads) == 0{
		log.Println("Empty backoup")
		return newTrackerServer()
	}
	tt := &tk.TrackerServer{Torrents: torrentsLoads}
	return tt
}

func pritnTracker(ts tk.TrackerServer){
	jts, _ := json.MarshalIndent(ts, "", " ")
	fmt.Println(string(jts))
}

func main() {
	enableTls:= flag.Bool("tls", false, "enable tls security")
	load:= flag.Bool("load", false, "load state from source")
	ip := flag.String("ip", "localhost", "set ip address of the tracker")
	port := flag.Int("port", 8168, "set port for TCP conection")
	flag.Parse()

	//runPruve()
	
	
	var trackerServer *tk.TrackerServer
	if *load{
		trackerServer = newTrackerServerFromLoad()
	}else{
		trackerServer = newTrackerServer()
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *ip, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("Server in: %s port:%d \n", *ip, *port)
	
	var sr *grpc.Server
	if *enableTls == true{
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			log.Fatal("cannot load TLS credentials: ", err)
		}
		sr = grpc.NewServer(grpc.Creds(tlsCredentials))
	}else{
		sr = grpc.NewServer()
	}
	pb.RegisterTrackerServer(sr, trackerServer)
	err2 := sr.Serve(lis)
	if err2 != nil {
		log.Fatalf("failed to listen: %v", err2)
	}
}
