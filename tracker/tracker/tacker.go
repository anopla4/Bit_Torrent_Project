package tracker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"grpc-go/peer"
	"io/ioutil"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	//_ "encoding/json"
	_ "flag"
	"fmt"

	//_ "log"
	"net"
	"sync"
	"time"

	//"google.golang.org/grpc"
	//"google.golang.org/grpc/peer"
	//_ "google.golang.org/grpc" // testing

	//ttp "Bit_Torrent_Project/tracker/tracker_tracker_protocol"
	pb "Bit_Torrent_Project/tracker/trackerpb"
)

// constantes para el status de un Publish Request
const (
	OK            = 1
	ALREADYEXISTT = 2
	ALREADYEXISTP = 3

	ALREADYKNOWOK = 4
	UNKNOWNREDKEY = 5
	FAILEDIP      = 6
	FAILEDPORT    = 7
	FAILED        = 8
)

//constantes de tiempo
const (
	INTERVALTKTKTIME = 600
)

// ParseAnnounce es una representacion de una announce get query
type ParseAnnounce struct {
	InfoHash   string //del torrent
	PeerID     string // Id del peer
	IP         net.IP //Ip del peer Opcional
	Port       int    // Port del peer Opcional
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      string
	NumWant    uint32 //Cantidad de peer que solicita el peer, Default 50
	Request    bool   // Indica si quiere recibir peers
}

//PeersPool es un diccionario que mapea el peerID con un Peer
type PeersPool map[string]*PeerTk

//PeerTk es la representacion de un Peer para la el Tracker
type PeerTk struct {
	ID    string       `json:"peer_id"`
	Addr  *net.TCPAddr `json:"peer_addr"`
	State string       `json:"peer_state"` // Uno entre: seeder, started, completed or stopped
}

//TorrentsPool es un diccionario que mapea metainfoHash con la informacion del torrent file
type TorrentsPool map[string]*TorrentTk

// TorrentTk es la representacion de un archivo torrent para el tracker
type TorrentTk struct {
	Hash       string    `json:"infoHash"`
	Downloaded uint64    `json:"downloaded"`
	Complete   uint64    `json:"complete"`
	Peers      PeersPool `json:"peers"`
}

// BkTracker es la representacion de un backoup tracker para el tracker
type BkTracker struct {
	IP   net.IP `json:"ip"`
	Port int    `json:"port"`
}

//TrackersPool es un diccionario que mapea un addr con su tracker correspondiente
type TrackersPool map[string]*BkTracker

//TrackerServer es la representacion de la estructura del tracker
type TrackerServer struct {
	pb.UnimplementedTrackerServer
	TkID       string        `json:"tk_Id"`
	Interval   uint32        `json:"interval"`
	IP         net.IP        `json:"ip"`
	Port       int32         `json:"port"`
	ListenAddr *net.Listener `json:"listen_addr"`
	sync.RWMutex
	Torrents TorrentsPool `json:"torrents_pool"`

	//Tracker to Tracker Comunication Block
	//ttp.UnimplementedTrackerComunicationServer
	RedKey         []byte
	BackupTrackers TrackersPool `json:"bk_trackers"`
}

//AddBkTracker agrega un BkTracker a la lista del tracker
func (tk *TrackerServer) AddBkTracker(ip net.IP, port int) error {
	addr := net.JoinHostPort(ip.String(), fmt.Sprint(port))
	if _, found := tk.BackupTrackers[addr]; found {
		err := fmt.Errorf("Already have this bktracker %s", addr)
		log.Println(err.Error())
		return err
	}
	tk.BackupTrackers[addr] = &BkTracker{IP: ip, Port: port}
	log.Printf("Bk Tracker register: %s \n", addr)
	return nil
}

func RePublishRequestManager(listAddr []string, key []byte, infoHash, peerId, ip string, port int32) {
	for _, addr := range listAddr {
		if addr != "" {
			fmt.Printf("Make a RePublis request to %s \n", addr)
			go makeRePublishRequest(addr, key, infoHash, peerId, ip, port)
		}
	}
	log.Println("Finish RePublish")
}

func makeRePublishRequest(addr string, key []byte, infoHash, peerID, IP string, port int32) {
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
	client := pb.NewTrackerClient(conn)

	m := pb.RePublishQuery{
		InfoHash: []byte(infoHash),
		PeerID:   peerID,
		IP:       IP,
		Port:     port,
		RedKey:   key,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.RePublish(ctx, &m)
	if err != nil {
		log.Printf("The call for RePublish request to %s was an error : %s\n", addr, err.Error())
	} else {
		log.Printf("RePublish Response from %s with status %v\n", addr, res.Status)
	}
}

//Publish maneja el servicio Publish request del tracker recibiendo un PublishQuery y devolviendo un PublishResponse
func (tk *TrackerServer) Publish(ctx context.Context, pq *pb.PublishQuery) (*pb.PublishResponse, error) {
	log.Println("Incoming publish... ")

	ih := pq.GetInfoHash()
	if ih == nil || len(ih) != 20 {
		err := fmt.Errorf("Publish error--> Invalid InfoHash")
		log.Println(err.Error())
		return nil, err
	}
	infoHash := string(ih[:])
	peerID := pq.GetPeerID()
	if peerID == "" {
		err := fmt.Errorf("Publish error--> Blank field: PeerID")
		log.Println(err.Error())
		return nil, err
	}
	ip := net.ParseIP(pq.GetIP())
	if ip == nil {
		err := fmt.Errorf("ublish Error---> Unreacheable field: IP")
		log.Println(err.Error())
		return nil, err
	}
	port := pq.GetPort()
	if port == 0 {
		err := fmt.Errorf("Publish Error---> Unreacheable field: port")
		log.Println(err.Error())
		return nil, err
	}
	tk.Lock()
	defer tk.Unlock()
	status := tk.publishTorrent(infoHash, peerID, int(port), ip)
	if status == OK {
		log.Printf("Publish torrent-----> %s by peer: %s\n", infoHash, peerID)
		listAddr := make([]string, len(tk.BackupTrackers))
		key := tk.RedKey
		for addr := range tk.BackupTrackers {
			if addr != "" {
				listAddr = append(listAddr, addr)
			}
		}
		log.Println(len(listAddr))
		fmt.Println(fmt.Sprint(listAddr))
		go RePublishRequestManager(listAddr, key, infoHash, peerID, ip.String(), port)
	}
	return &pb.PublishResponse{
		Status: status,
	}, nil
}

func (tk *TrackerServer) publishTorrent(infoHash, peerID string, port int, ip net.IP) int32 {
	tp := tk.Torrents
	if ttk, ok := tp[infoHash]; ok {
		tpeers := ttk.Peers
		if _, found := tpeers[peerID]; found {
			return ALREADYEXISTP
		}
		tpeers[peerID] = &PeerTk{
			ID: peerID,
			Addr: &net.TCPAddr{
				IP:   ip,
				Port: int(port)},
			State: "seeder"}
		return ALREADYEXISTT
	}
	tp[infoHash] = &TorrentTk{
		Hash:       infoHash,
		Downloaded: 0,
		Complete:   1,
		Peers: PeersPool{
			peerID: &PeerTk{
				ID: peerID,
				Addr: &net.TCPAddr{
					IP:   ip,
					Port: port},
				State: "seeder"},
		}}
	return OK
}

//Announce maneja el servicio Announce request del tracker recibiendo un AnnounceQuery y devolviendo un AnnounceResponse
func (tk *TrackerServer) Announce(ctx context.Context, annq *pb.AnnounceQuery) (*pb.AnnounceResponse, error) {
	if p, ok := peer.FromContext(ctx); ok {
		fmt.Println(p)
	}
	log.Println("Incoming Announce... ")

	pa, err := announceQueryCheck(annq)
	var ar pb.AnnounceResponse
	if err != nil {
		ar.FailureReason = err.Error()
		log.Println(err.Error())
		return &ar, err
	}
	event := pa.Event
	request := pa.Request
	infoHash := pa.InfoHash
	tk.RLock()
	defer tk.RUnlock()
	tt := tk.Torrents
	if ttk, ok := tt[infoHash]; ok {
		if request {
			peerSet, _ := ttk.Peers.getRandomPeers(pa.PeerID, int(pa.NumWant))
			ar.Peers = peerSet
		}
		switch event {
		case "started":
			_, err := ttk.Peers.getPeer(pa.PeerID)
			if err != nil {
				ttk.Peers[pa.PeerID] = &PeerTk{
					ID:    pa.PeerID,
					Addr:  &net.TCPAddr{IP: pa.IP, Port: pa.Port},
					State: "started"}
			} else {
				ttk.Peers[pa.PeerID].State = "started"
			}
		case "request":
		case "complete":
			ttk.Peers[pa.PeerID].State = "complete"
			ttk.Complete++
			ttk.Downloaded++
			///RePublish block
			listAddr := make([]string, len(tk.BackupTrackers))
			key := tk.RedKey
			for addr := range tk.BackupTrackers {
				listAddr = append(listAddr, addr)
			}
			go RePublishRequestManager(listAddr, key, infoHash, pa.PeerID, pa.IP.String(), int32(pa.Port))
		case "stopped":
			ttk.Peers[pa.PeerID].State = "stopped"
		}
		ar.Interval = tk.Interval
		ar.Complete, ar.Incomplete = ttk.Peers.torrentStats()
		log.Printf("Annunce OK---> %s for peer: %s", infoHash, pa.PeerID)
		return &ar, nil
	}
	err = fmt.Errorf("Announce Error---> Dont match the infohash")
	log.Println(err.Error())
	ar.FailureReason = err.Error()
	return &ar, err
}

//para representar un conjunto de enteros
type intset map[int]interface{}

//devuelve las estadisticas de complete e incomplete para un torrent
func (pp PeersPool) torrentStats() (complete, incomplete int64) {
	complete, incomplete = 0, 0
	for _, v := range pp {
		switch v.State {
		case "stopped":
		case "started":
			incomplete++
		default:
			complete++
		}
	}
	return
}

//Devuelve una cantidad de peers en base al numwant
func (pp PeersPool) getRandomPeers(excludeID string, numwant int) (map[string]string, error) {
	maxPeers := len(pp)
	if _, ok := pp[excludeID]; ok {
		maxPeers--
	}
	if numwant > maxPeers {
		numwant = maxPeers
	}
	count := 0
	peers := make(map[string]string, numwant)
	for k, v := range pp {
		if k != excludeID {
			switch v.State {
			case "stopped":
			case "started":
			default:
				count++
				peers[v.ID] = v.Addr.String()
			}
		}
		if count == numwant-1 {
			break
		}
	}
	for k, v := range pp {
		if count == numwant-1 {
			break
		}
		if k != excludeID {
			switch v.State {
			case "started":
				count++
				peers[v.ID] = v.Addr.String()
			}
		}
	}

	return peers, nil
}

//Dado un id, retorna el correspondiente peerTk
func (pp PeersPool) getPeer(id string) (*PeerTk, error) {
	if ptk, ok := pp[id]; ok {
		return ptk, nil
	}
	err := fmt.Errorf("Unrecognized peerID: %s", id)
	return nil, err
}

//funcion para parsear y chequear un AnnounceQuery, generando un ParseAnnounce
func announceQueryCheck(annPb *pb.AnnounceQuery) (pa ParseAnnounce, err error) {
	infoHash := annPb.GetInfoHash()
	if infoHash == nil || len(infoHash) != 20 {
		err = fmt.Errorf("Announce Error---> Empty Infohash")
		return
	}
	pa.InfoHash = string(infoHash[:])
	peerID := annPb.GetPeerID()
	if peerID == "" {
		err = fmt.Errorf("Announce Error---> Blank field:  PeerID")
		return
	}
	pa.PeerID = peerID
	ip := net.ParseIP(annPb.GetIP())
	if ip == nil {
		err = fmt.Errorf("Announce Error---> Unreacheable field: IP")
		return
	}
	pa.IP = ip
	port := annPb.GetPort()
	if port == 0 {
		err = fmt.Errorf("Announce Error---> Unreacheable field: port")
		return
	}
	pa.Port = int(port)
	pa.Uploaded = annPb.GetUploaded()
	pa.Downloaded = annPb.GetDownloaded()
	pa.Left = annPb.GetLeft()
	event := annPb.GetEvent()
	numwant := annPb.GetNumWant()
	if numwant < 1 {
		numwant = 40
	}
	if event != "request" && event != "started" && event != "completed" && event != "stopped" {
		err = fmt.Errorf("Dont recognize the event especified. By protocol, event must be one among: request, started, stopped, completed")
		return
	}
	pa.Event = event
	pa.NumWant = numwant
	pa.Request = annPb.GetRequest()
	return
}

//Scrape procesa el servio Scrape del Tracker tomando un ScraperQuery y devolviendo un ScraperResponse
func (tk *TrackerServer) Scrape(ctx context.Context, sc *pb.ScraperQuery) (*pb.ScraperResponse, error) {
	infoHashes, err := ParseScraperRequest(sc)
	var sr pb.ScraperResponse
	if err != nil {
		sr.FailureReason = err.Error()
		log.Println(err.Error())
		return &sr, err
	}
	tk.RLock()
	defer tk.RUnlock()
	tt := tk.Torrents
	files := make(map[string]*pb.File, len(infoHashes))
	for _, ih := range infoHashes {
		if ttk, found := tt[ih]; found {
			_, incomplete := ttk.Peers.torrentStats()
			files[ih] = &pb.File{
				Incomplete: incomplete,
				Complete:   int64(ttk.Complete),
				Downloaded: int64(ttk.Downloaded),
			}
		}
	}
	sr.Files = files
	return &sr, nil
}

//ParseScraperRequest valida la peticion de un ScrapeQuery y devuelve la lista de infoHashes
// que fueron requeridos
func ParseScraperRequest(sc *pb.ScraperQuery) ([]string, error) {
	infoHashes := sc.GetInfoHash()
	n := len(infoHashes)
	if n == 0 {
		err := fmt.Errorf("Scrape Error---> Empty infoHash")
		return nil, err
	}
	parseInfoHashes := make([]string, n)
	for idx, ih := range infoHashes {
		parseInfoHashes[idx] = string(ih[:])
	}
	return parseInfoHashes, nil
}

//Tracker to Tracker protocol

//RePublish es el metodo para menejar la solicitud de RePublish entre los trackers
func (tk *TrackerServer) RePublish(ctx context.Context, rpq *pb.RePublishQuery) (*pb.RePublishResponse, error) {
	ih := rpq.GetInfoHash()
	if ih == nil || len(ih) != 20 {
		err := fmt.Errorf("RePublish Error---> Invalid InfoHash")
		log.Println(err.Error())
		return nil, err
	}
	infoHash := string(ih[:])
	peerID := rpq.GetPeerID()
	if peerID == "" {
		err := fmt.Errorf("RePublish Error---> Blank field: PeerID")
		log.Println(err.Error())
		return nil, err
	}
	ip := net.ParseIP(rpq.GetIP())
	if ip == nil {
		err := fmt.Errorf("RePublish Error---> Unreacheable field: IP")
		log.Println(err.Error())
		return nil, err
	}
	port := rpq.GetPort()
	if port == 0 {
		err := fmt.Errorf("RePublish Error---> Unreacheable field: port")
		log.Println(err.Error())
		return nil, err
	}
	inKey := rpq.GetRedKey()
	tk.Lock()
	defer tk.Unlock()
	if string(inKey[:]) != string(tk.RedKey[:]) {
		err := fmt.Errorf("RePublish Error---> The <redKey> is incorrect")
		log.Println(err.Error())
		return &pb.RePublishResponse{
			Status: UNKNOWNREDKEY,
		}, err
	}
	status := tk.publishTorrent(infoHash, peerID, int(port), ip)
	if status == OK {
		log.Printf("RePublish torrent:%s from source peerId:%s\n", infoHash, peerID)
	}
	return &pb.RePublishResponse{
		Status: status,
	}, nil
}

//KnowMe es el metodo para manejar la solicitud KnowMeRequest entre los trackers
func (tk *TrackerServer) KnowMe(ctx context.Context, kr *pb.KnowMeRequest) (*pb.KnowMeResponse, error) {
	inKey := kr.GetRedKey()

	port := kr.GetPort()
	log.Println(port)
	ip := net.ParseIP(kr.GetIP())
	if ip == nil {
		err := fmt.Errorf("KnowMe Error---> Unreachable field: IP")
		log.Println(err.Error())
		return &pb.KnowMeResponse{
			Status: FAILEDIP,

			RepeatInterval: INTERVALTKTKTIME,
		}, err
	}
	if port == 0 {
		err := fmt.Errorf("KnowMe Error---> Unreachable field: port")
		log.Println(err.Error())
		return &pb.KnowMeResponse{
			Status:         FAILEDPORT,
			RepeatInterval: INTERVALTKTKTIME,
		}, err
	}
	tk.Lock()
	if string(inKey[:]) != string(tk.RedKey[:]) {
		err := fmt.Errorf("KnowMe Error---> The <redKey> is incorrect")
		log.Println(err.Error())
		return &pb.KnowMeResponse{
			Status:         UNKNOWNREDKEY,
			RepeatInterval: INTERVALTKTKTIME,
		}, err
	}
	hotsport := net.JoinHostPort(ip.String(), fmt.Sprint(port))
	log.Println(hotsport)
	bktks := tk.BackupTrackers
	if _, found := bktks[hotsport]; found {
		return &pb.KnowMeResponse{
			Status:         FAILED,
			RepeatInterval: INTERVALTKTKTIME,
		}, nil
	}
	bktks[hotsport] = &BkTracker{IP: ip, Port: int(port)}
	tk.Unlock()
	log.Println("KnowMe complete, backup included")
	return &pb.KnowMeResponse{
		Status:         OK,
		RepeatInterval: INTERVALTKTKTIME,
	}, nil
}
