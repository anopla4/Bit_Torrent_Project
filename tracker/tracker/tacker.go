package tracker

import (
	"context"
	"fmt"
	_ "encoding/json"  // testing
	_ "flag"
	_ "log"
	"math/rand"
	"net"
	"time"
	_ "google.golang.org/grpc" // testing

	pb "Bit_Torrent_Project/tracker/trackerpb"
)

// constantes para el status de un Publish Request
const (
	OK            = 1
	ALREADYEXISTT = 2
	ALREADYEXISTP = 3
)

// ParseAnnounce es una representacion de una announce get query
type ParseAnnounce struct {
	InfoHash   string //dic bencodificado
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

// AnnounceResponse es una representacion de una respuesta a un announce get query
//type AnnounceResponse struct {
//	Interval   uint16
//	TrackerID  string
//	Complete   uint64
//	Incomplete uint64
//	Peers      map[string]string
//}

//PeersPool es un diccionario que mapea el peerID con un Peer
type PeersPool map[string]*PeerTk

//PeerTk es la representacion de un Peer para la el Tracker
type PeerTk struct {
	ID         string
	Addr       *net.TCPAddr
	State      string // Uno entre: seeder, started, completed or stopped
	//Uploaded   uint64
	//Downloaded uint64
	//LeftDown   uint64
}

//TorrentsPool es un diccionario que mapea metainfoHash con la informacion del torrent file
type TorrentsPool map[string]*TorrentTk

// TorrentTk es la representacion de un archivo torrent para el tracker
type TorrentTk struct {
	Hash       string
	Downloaded uint64
	Complete   uint64
	Peers      PeersPool
}

//TrackerServer es la representacion de la estructura del tracker
type TrackerServer struct {
	//AnnReq     ParseAnnounce
	//AnnResp    AnnounceResponse
	pb.UnimplementedTrackerServer
	TkID       string
	Interval   uint32
	IP         net.IP
	Port       int32
	ListenAddr *net.Listener
	Torrents   TorrentsPool
}

//Publish maneja el servicio Publish request del tracker recibiendo un PublishQuery y devolviendo un PublishResponse
func (tk *TrackerServer) Publish(ctx context.Context, pq *pb.PublishQuery) (*pb.PublishResponse, error) {
	ih := pq.GetInfoHash()
	if ih == nil || len(ih) != 20 {
		err := fmt.Errorf("InfoHash invalido")
		return nil, err
	}
	infoHash := string(ih[:])
	peerID := pq.GetPeerID()
	if peerID == "" {
		err := fmt.Errorf("No se especifica el PeerID")
		return nil, err
	}
	ip := net.ParseIP(pq.GetIP())
	if ip == nil {
		err := fmt.Errorf("Error con la IP")
		return nil, err
	}
	port := pq.GetPort()
	if port == 0 {
		err := fmt.Errorf("No se especifica el puerto")
		return nil, err
	}
	status := tk.publishTorrent(infoHash, peerID, int(port), ip)
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
	pa, err := announceQueryCheck(annq)
	var ar pb.AnnounceResponse
	if err != nil {
		ar.FailureReason = err.Error()
		return &ar, err
	}
	event := pa.Event
	request := pa.Request
	infoHash := pa.InfoHash
	tt := tk.Torrents
	if ttk, ok := tt[infoHash]; ok {
		ptk, err := ttk.Peers.getPeer(pa.PeerID)
		if err != nil {
			ar.FailureReason = err.Error()
			return &ar, err
		}
		if request {
			peerSet, _ := ttk.Peers.getRandomPeers(pa.PeerID, int(pa.NumWant))
			ar.Peers = peerSet
		}
		switch event {
		case "started":
			ptk.State = "started"
		case "request":
		case "complete":
			ptk.State = "complete"
			ttk.Complete++
			ttk.Downloaded++
		case "stopped":
			ptk.State = "stopped"
		}
		ar.Interval = tk.Interval
		ar.Complete, ar.Incomplete = ttk.Peers.torrentStats()
		return &ar, nil
	}
	err = fmt.Errorf("No coincide el infohash")
	ar.FailureReason = err.Error()
	return &ar, err
}

//para representar un conjunto de enteros
type intset map[int]interface{}

//devuelve las estadisticas de complete e incomplete para un torrent 
func (pp PeersPool) torrentStats()(complete, incomplete int64){
	complete, incomplete = 0,0
	for _ , v := range pp {
		switch v.State{
		case "stopped":
		case "started":
			incomplete++
		default:
			complete++
		}
	}
	return
}

//Devuelve una cantidad pseudorandom de peers en base al numwant
func (pp PeersPool) getRandomPeers(excludeID string, numwant int) (map[string]string, error) {
	maxPeers := len(pp)
	if _, ok := pp[excludeID]; ok {
		maxPeers--
	}
	if numwant > maxPeers {
		numwant = maxPeers
	}
	dif := numwant - maxPeers
	excludes := make(intset, dif)
	if dif > 1 {
		last := rand.Intn(numwant - 1)
		excludes[last] = 0
		for i := 1; i < dif; i++ {
			rand.Seed(time.Now().UnixNano())
			r := rand.Intn(numwant - 1)
			if _, found := excludes[r]; !found {
				excludes[r] = 0
			}
		}
	}
	i := 0
	count := 0
	peers := make(map[string]string, numwant)
	for k, v := range pp {
		if k != excludeID {
			if _, found := excludes[i]; found {
				delete(excludes, i)
			}else{
				switch v.State{
				case "stopped":
				case "started":
				default:
					count++
					peers[v.ID] = v.Addr.String()
				}	
			}
		}
		if count == numwant-1 {
			break
		}
		i++
	}
	return peers, nil
}

//Dado un id, retorna el correspondiente peerTk
func (pp PeersPool) getPeer(id string) (*PeerTk, error) {
	if ptk, ok := pp[id]; ok {
		return ptk, nil
	}
	err := fmt.Errorf("No se reconoce el peer identificado con peerID: %s", id)
	return nil, err
}

//funcion para parsear y chequear un AnnounceQuery, generando un ParseAnnounce
func announceQueryCheck(annPb *pb.AnnounceQuery) (pa ParseAnnounce, err error) {
	infoHash := annPb.GetInfoHash()
	if infoHash == nil || len(infoHash) != 20 {
		err = fmt.Errorf("Infohash vacio")
		return
	}
	pa.InfoHash = string(infoHash[:])
	peerID := annPb.GetPeerID()
	if peerID == "" {
		err = fmt.Errorf("No se especifica el PeerID")
		return
	}
	pa.PeerID = peerID
	ip := net.ParseIP(annPb.GetIP())
	if ip == nil {
		err = fmt.Errorf("Error con la IP")
		return
	}
	pa.IP = ip
	port := annPb.GetPort()
	if port == 0 {
		err = fmt.Errorf("No se especifica el puerto")
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
		err = fmt.Errorf("No se reconoce el evento especificado")
		return
	}
	pa.Event = event
	pa.NumWant = numwant
	pa.Request = annPb.GetRequest()
	return
}

//Scrape procesa el servio Scrape del Tracker tomando un ScraperQuery y devolviendo un ScraperResponse
func (tk *TrackerServer) Scrape(ctx context.Context, sc *pb.ScraperQuery) (*pb.ScraperResponse, error){
	infoHashes, err := ParseScraperRequest(sc)
	var sr pb.ScraperResponse
	if err != nil{
		sr.FailureReason = err.Error()
		return &sr, err
	}
	tt := tk.Torrents
	files := make(map[string]*pb.File, len(infoHashes))
	for _, ih := range infoHashes{
		if ttk, found := tt[ih]; found{
			_, incomplete := ttk.Peers.torrentStats()
			files[ih] = &pb.File{
								Incomplete:incomplete,
								Complete: int64(ttk.Complete),
								Downloaded: int64(ttk.Downloaded),
								}
		}
	}
	sr.Files = files
	return &sr, nil
}

//ParseScraperRequest valida la peticion de un ScrapeQuery y devuelve la lista de infoHashes
// que fueron requeridos
func ParseScraperRequest(sc *pb.ScraperQuery) ([]string, error){
	infoHashes := sc.GetInfoHash()
	n := len(infoHashes)
	if n == 0{
		err := fmt.Errorf("No se especifica ningun infoHash")
		return nil, err
	}
	parseInfoHashes := make([]string, n)
	for idx, ih := range infoHashes{
		parseInfoHashes[idx] = string(ih[:])
	}
	return parseInfoHashes, nil
}

