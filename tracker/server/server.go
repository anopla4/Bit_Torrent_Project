package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"io/ioutil"

	"container/list"
	"encoding/json"
	"flag"
	"log"

	//"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	tk "Bit_Torrent_Project/tracker/tracker"
	ttp "Bit_Torrent_Project/tracker/tracker_tracker_protocol"
	pb "Bit_Torrent_Project/tracker/trackerpb"
)

func exampleTracker() (tk.TorrentsPool, error) {
	peers := make(tk.PeersPool, 3)
	peers["peerId1"] = &tk.PeerTk{
		ID:    "peerId1",
		State: "seeder",
		Addr:  &net.TCPAddr{IP: net.ParseIP("108.56.6.6"), Port: 5000},
	}
	peers["peerIdX"] = &tk.PeerTk{
		ID:    "peerIdX",
		State: "lecher",
		Addr:  &net.TCPAddr{IP: net.ParseIP("190.122.0.4"), Port: 5000},
	}

	torrents := make(tk.TorrentsPool, 5)
	torrents["12345678912345678900"] = &tk.TorrentTk{
		Hash:       "12345678912345678900",
		Downloaded: 0,
		Complete:   0,
		Peers:      peers,
	}
	torrents["0000000000000000000"] = &tk.TorrentTk{
		Hash:       "0000000000000000000",
		Downloaded: 200,
		Complete:   11,
		Peers:      peers,
	}

	return torrents, nil
}

func jsonPruve(tp tk.TorrentsPool) error {
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

func runPruve() {
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

	caCert, err := ioutil.ReadFile("cert/cert.pem")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	serverCert, err := tls.LoadX509KeyPair("cert/cert.pem", "cert/key.pem")
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	tlsConfig.BuildNameToCertificate()

	return credentials.NewTLS(tlsConfig), nil
}

func newTrackerServer(key string) *tk.TrackerServer {
	return &tk.TrackerServer{Torrents: make(tk.TorrentsPool, 10),
		RedKey:          key,
		BackoupTrackers: make(tk.TrackersPool, 5)}
}

func newTrackerServerFromLoad(key string) *tk.TrackerServer {
	tstate, err := ioutil.ReadFile("bkdata/torrents.json")
	if err != nil {
		log.Println("Failed to load backoup info: " + err.Error())
		return newTrackerServer(key)
	}
	var torrentsLoads = tk.TorrentsPool{}
	err = json.Unmarshal(tstate, &torrentsLoads)
	if err != nil {
		log.Println(err.Error())
		return newTrackerServer(key)
	}
	if len(torrentsLoads) == 0 {
		log.Println("Empty backoup")
		return newTrackerServer(key)
	}
	tt := &tk.TrackerServer{Torrents: torrentsLoads,
		RedKey:          key,
		BackoupTrackers: make(tk.TrackersPool, 5)}
	return tt
}

func pritnTracker(ts tk.TrackerServer) {
	jts, _ := json.MarshalIndent(ts, "", " ")
	fmt.Println(string(jts))
}

func loadBkTrackers(source string) (list.List, error) {
	jsonTrackers, err := ioutil.ReadFile(source)
	if err != nil {
		return list.List{}, err
	}
	var aTk []tk.BkTracker
	err = json.Unmarshal(jsonTrackers, &aTk)
	if err != nil {
		return list.List{}, err
	}
	queue := list.New()
	for _, t := range aTk {
		if t.IP == nil || t.Port == 0 {
			log.Println("Bad addr for backoup tracker")
		} else {
			queue.PushBack(fmt.Sprintf("%s:%s", t.IP.String(), fmt.Sprint(t.Port)))
		}
	}
	return *queue, nil
}

func trackerTrackerCommunication(tkQueue list.List, key, myIP string, myPort int32, tls bool) {
	for tkQueue.Len() > 0 {
		e := tkQueue.Front() // First element
		addr := fmt.Sprint(e.Value)
		go makeKnowMeConnection(addr, key, myIP, myPort, tls)
		tkQueue.Remove(e) // Dequeue
	}
}

func makeKnowMeConnection(addr, key, myIP string, myPort int32, tls bool) {
	//addr := fmt.Sprintf("%s:%s", host, port)
	//var opts []grpc.DialOption
	//if *tls {
	//	if *caFile == "" {
	//		*caFile = data.Path("x509/ca_cert.pem")
	//	}
	//	creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
	//	if err != nil {
	//		log.Fatalf("Failed to create TLS credentials %v", err)
	//	}
	//	opts = append(opts, grpc.WithTransportCredentials(creds))
	//} else {
	//	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	//}
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("impossible connect: %v", err)
	}
	defer conn.Close()
	client := ttp.NewTrackerComunicationClient(conn)

	m := ttp.KnowMeRequest{
		RedKey: key,
		IP:     myIP,
		Port:   myPort,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := client.KnowMe(ctx, &m)
	if err != nil {
		log.Printf("The call for KnowMe request to %s was an error : %s", addr, err.Error())
	} else {
		log.Printf("Response from %s with status %v", addr, res.Status)
	}
}

//func saveData(ctx context.Context, ts *tk.TrackerServer, saveTime int, startTime time.Time) {
//	var lastSave time.Time = startTime
//	for {
//		select {
//		case <-ctx.Done():
//			break
//		default:
//			time.Sleep(time.Duration(saveTime))
//			ts.Tr
//			jso, err := json.MarshalIndent(tp, "", " ")
//			if err != nil {
//				fmt.Println(err.Error())
//				return err
//			}
//			//fmt.Println(string(jso))
//			e := ioutil.WriteFile("bkdata/torrents.json", jso, fs.ModeExclusive)
//			if e != nil {
//				fmt.Println(e.Error())
//				return e
//			}
//			fmt.Println()
//				}
//			}
//}

func main() {
	//saveTime := flag.Int("saveTime", 120, "seconds time between two save copies")
	backoupTkFromSource := flag.String("backoupTkFromSource", "", "set the path of json with backoup tracker directions")
	redKeySeed := flag.String("redKey", "", "set the password for the tk to tk communication")
	enableTlS := flag.Bool("tls", false, "enable tls security")
	load := flag.Bool("load", false, "load state from source")
	ip := flag.String("ip", "localhost", "set ip address of the tracker")
	port := flag.Int("port", 8168, "set port for TCP conection on GRPC server")
	//tcomClientPort := flag.Int("port", 8007, "set port for tracker-to-tracker request TCP conection")
	//tcomListenPort := flag.Int("port", 8008, "set port for tracker-to-tracker listen TCP conection")
	flag.Parse()
	if *redKeySeed == "" {
		log.Fatalln("The redKey flag is required for security of tacker communication. Set '-redKey' flag")
	}
	redKey := sha256.Sum256([]byte(*redKeySeed))
	if *backoupTkFromSource == "" {
		log.Println("No source for backoup tracker. Nothing happen, but the red is less strong")
	} else {
		tkQueue, err := loadBkTrackers(*backoupTkFromSource)
		if err != nil {
			log.Println("Problems loading backoup tracker. Nothing happen, but the red is less strong")
		} else {
			go trackerTrackerCommunication(tkQueue, *redKeySeed, *ip, int32(*port), *enableTlS)
		}
	}
	//if (*port == *tcomClientPort) || (*port == *tcomListenPort) {
	//	log.Fatalln("Equal port for GRPC server and tracker-to-tracker comunication")
	//}
	//if *tcomClientPort == *tcomListenPort {
	//	log.Fatalln("Equal port for tracker-to-tracker listen and request comunication")
	//}
	//runPruve()
	var trackerServer *tk.TrackerServer
	if *load {
		trackerServer = newTrackerServerFromLoad(string(redKey[:]))
	} else {
		trackerServer = newTrackerServer(string(redKey[:]))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *ip, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("Server in: %s port:%d \n", *ip, *port)

	var sr *grpc.Server
	if *enableTlS {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			log.Fatal("cannot load TLS credentials: ", err)
		}
		sr = grpc.NewServer(grpc.Creds(tlsCredentials))
	} else {
		sr = grpc.NewServer()
	}
	pb.RegisterTrackerServer(sr, trackerServer)
	err2 := sr.Serve(lis)
	if err2 != nil {
		log.Fatalf("failed to listen: %v", err2)
	}
}
