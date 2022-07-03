package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"container/list"
	"encoding/json"
	"flag"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tk "Bit_Torrent_Project/tracker/tracker"
	ttp "Bit_Torrent_Project/tracker/tracker_tracker_protocol"
	pb "Bit_Torrent_Project/tracker/trackerpb"
)

//func exampleTracker() (tk.TorrentsPool, error) {
//	peers := make(tk.PeersPool, 3)
//	peers["peerId1"] = &tk.PeerTk{
//		ID:    "peerId1",
//		State: "seeder",
//		Addr:  &net.TCPAddr{IP: net.ParseIP("108.56.6.6"), Port: 5000},
//	}
//	peers["peerIdX"] = &tk.PeerTk{
//		ID:    "peerIdX",
//		State: "lecher",
//		Addr:  &net.TCPAddr{IP: net.ParseIP("190.122.0.4"), Port: 5000},
//	}
//
//	torrents := make(tk.TorrentsPool, 5)
//	torrents["12345678912345678900"] = &tk.TorrentTk{
//		Hash:       "12345678912345678900",
//		Downloaded: 0,
//		Complete:   0,
//		Peers:      peers,
//	}
//	torrents["0000000000000000000"] = &tk.TorrentTk{
//		Hash:       "0000000000000000000",
//		Downloaded: 200,
//		Complete:   11,
//		Peers:      peers,
//	}
//
//	return torrents, nil
//}
//
//func jsonPruve(tp tk.TorrentsPool) error {
//	jso, err := json.MarshalIndent(tp, "", " ")
//	if err != nil {
//		fmt.Println(err.Error())
//		return err
//	}
//	//fmt.Println(string(jso))
//	e := ioutil.WriteFile("bkdata/torrents.json", jso, fs.ModeExclusive)
//	if e != nil {
//		fmt.Println(e.Error())
//		return e
//	}
//	fmt.Println()
//
//	tpp := tk.TorrentsPool{}
//
//	err = json.Unmarshal(jso, &tpp)
//	if err != nil {
//		fmt.Println(err.Error())
//		return err
//	}
//	return nil
//}
//
//func runPruve() {
//	ts, err := exampleTracker()
//	if err != nil {
//		fmt.Println(err.Error())
//		return
//	}
//	e := jsonPruve(ts)
//	if e != nil {
//		fmt.Println(e.Error())
//		return
//	}
//}

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

func loadBkTrackers() (tk.TrackersPool, error) {
	btks, err := ioutil.ReadFile("bkdata/bktrackers.json")
	if err != nil {
		log.Println("Failed to load backoup info: " + err.Error())
		return make(tk.TrackersPool, 10), err
	}
	var bktkLoads = tk.TrackersPool{}
	err = json.Unmarshal(btks, &bktkLoads)
	if err != nil {
		log.Println(err.Error())
		return make(tk.TrackersPool, 10), err
	}
	if len(bktkLoads) == 0 {
		log.Println("Empty backoup")
		return make(tk.TrackersPool, 10), err
	}
	return bktkLoads, nil
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

func getNewBkTrackersAddrs(source string) (list.List, error) {
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
	alreadyKnows, err := loadBkTrackers()
	for _, t := range aTk {
		if t.IP == nil || t.Port == 0 {
			log.Println("Bad addres for new backoup tracker")
			continue
		}
		addr := fmt.Sprintf("%s:%s", t.IP.String(), fmt.Sprint(t.Port))
		if err == nil {
			if _, found := alreadyKnows[addr]; found {
				continue
			}
		}
		queue.PushBack(addr)
	}
	return *queue, nil
}

func trackerTrackerCommunication(parentCtx context.Context, tkQueue list.List, key, myIP string, myPort int32, ch chan<- tk.BkTracker) {
	for tkQueue.Len() > 0 {
		e := tkQueue.Front() // First element
		addr := fmt.Sprint(e.Value)
		go makeKnowMeConnection(parentCtx, addr, key, myIP, myPort, ch)
		tkQueue.Remove(e) // Dequeue
	}
}

func putBkTrackersIntoServer(parentCtx context.Context, ts *tk.TrackerServer, ch <-chan tk.BkTracker) {
loop:
	for {
		select {
		case <-parentCtx.Done():
			break loop
		case bk := <-ch:
			addr := fmt.Sprintf("%s:%s", bk.IP.String(), fmt.Sprint(bk.Port))
			ts.Lock()
			ts.BackoupTrackers[addr] = &bk
			ts.Unlock()
		default:
			continue
		}
	}
}

func makeKnowMeConnection(parentCtx context.Context, addr, key, myIP string, myPort int32, ch chan<- tk.BkTracker) {
	cert, _ := tls.LoadX509KeyPair("cert/cert.pem", "cert/key.pem")
	caCert, _ := ioutil.ReadFile("cert/cert.pem")

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{RootCAs: caCertPool, Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.Printf("KnowMe connection error--> impossible connect: %v\n", err)
		return
	}
	defer conn.Close()
	client := ttp.NewTrackerComunicationClient(conn)

	m := ttp.KnowMeRequest{
		RedKey: key,
		IP:     myIP,
		Port:   myPort,
	}
	ctx, cancel := context.WithTimeout(parentCtx, 20*time.Second)
	defer cancel()
	res, err := client.KnowMe(ctx, &m)
	if err != nil {
		log.Printf("The call for KnowMe request to %s was an error : %s\n", addr, err.Error())
	} else {
		log.Printf("Response from %s with status %v\n", addr, res.Status)
		if res.Status == tk.OK || res.Status == tk.ALREADYKNOWOK {
			s := strings.Split(addr, ":")
			p, _ := strconv.Atoi(s[1])
			ch <- tk.BkTracker{IP: net.ParseIP(s[0]), Port: p, LastSeen: time.Now()}
		}
	}
}

func saveTorrents(torrents *tk.TorrentsPool, wg *sync.WaitGroup) {
	defer wg.Done()
	torJso, err := json.MarshalIndent(torrents, "", " ")
	if err != nil {
		log.Println(fmt.Sprint(time.Now()) + ": Error while jsonMarshall was working :" + err.Error())
		return
	}
	e := ioutil.WriteFile("bkdata/torrents.json", torJso, fs.FileMode(os.O_WRONLY))
	if e != nil {
		log.Println(fmt.Sprint(time.Now()) + ": Error " + e.Error())
	}
}

func saveBkTk(bktrackers *tk.TrackersPool, wg *sync.WaitGroup) {
	defer wg.Done()
	bkTkJso, err := json.MarshalIndent(bktrackers, "", " ")
	if err != nil {
		log.Println(fmt.Sprint(time.Now()) + ": Error while jsonMarshall was woring :" + err.Error())
		return
	}
	e := ioutil.WriteFile("bkdata/bktrackers.json", bkTkJso, fs.FileMode(os.O_WRONLY))
	if e != nil {
		log.Fatal("Shet")
		log.Println(fmt.Sprint("Error " + e.Error()))
	}
}

func saveData(ctx context.Context, ts *tk.TrackerServer, saveTime int) {
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			time.Sleep(time.Duration(saveTime))
			ts.RLock()
			var wg sync.WaitGroup
			wg.Add(2)
			go saveTorrents(&ts.Torrents, &wg)
			go saveBkTk(&ts.BackoupTrackers, &wg)
			wg.Wait()
			ts.RUnlock()
		}
	}
}

func main() {
	saveTime := flag.Int("saveTime", 600, "seconds time between two save copies")
	newBTkFromSource := flag.String("nBkTk", "", "set the path of json with backoup tracker directions")
	redKeySeed := flag.String("redKey", "", "set the password for the tk to tk communication")
	enableTlS := flag.Bool("tls", false, "enablecurity")
	load := flag.Bool("load", false, "load state from source")
	ip := flag.String("ip", "localhost", "set ip address of the tracker")
	port := flag.Int("port", 8168, "set port for TCP conection on GRPC server")
	flag.Parse()
	if *redKeySeed == "" {
		log.Fatalln("The redKey flag is required for security of tacker communication. Set '-redKey' flag")
	}
	redKey := sha256.Sum256([]byte(*redKeySeed))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var trackerServer *tk.TrackerServer
	if *load {
		trackerServer = newTrackerServerFromLoad(string(redKey[:]))
		trackerServer.BackoupTrackers, _ = loadBkTrackers()
	} else {
		trackerServer = newTrackerServer(string(redKey[:]))
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *ip, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

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
	log.Printf("Server in: %s port:%d \n", *ip, *port)

	if *newBTkFromSource == "" {
		log.Println("No source for backoup tracker. Nothing happen, but the red is less strong")
	} else {
		tkQueue, err := getNewBkTrackersAddrs(*newBTkFromSource)
		if err != nil {
			log.Println("Problems loading backoup tracker. Nothing happen, but the red is less strong> " + err.Error())
		} else {
			bkchanel := make(chan tk.BkTracker, tkQueue.Len())
			go putBkTrackersIntoServer(ctx, trackerServer, bkchanel)
			go trackerTrackerCommunication(ctx, tkQueue, *redKeySeed, *ip, int32(*port), bkchanel)
		}
	}
	go saveData(ctx, trackerServer, *saveTime)

	err2 := sr.Serve(lis)
	if err2 != nil {
		log.Fatalf("failed to listen: %v", err2)
	}
}
