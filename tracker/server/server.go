package main

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "Bit_Torrent_Project/tracker/trackerpb"
)

// constantes para el status de un Publish Request
const (
	ERROR = 0
	OK = 1
	ALREADYEXISTT = 2;
	ALREADYEXISTP = 3;
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
type AnnounceResponse struct {
	Interval   uint16
	TrackerID  string
	Complete   uint64
	Incomplete uint64
	Peers      string
}

//PeersPool es un diccionario que mapea el peerID con un Peer
type PeersPool map[string]*PeerTk

//PeerTk es la representacion de un Peer para la el Tracker
type PeerTk struct {
	Id         string
	Addr       *net.TCPAddr
	State      string  // Uno entre: seeder, started, completed or stopped
	Uploaded   uint64
	Downloaded uint64
	leftDown   uint64
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

type TrackerServer struct {
	//AnnReq     ParseAnnounce
	//AnnResp    AnnounceResponse
	TkID       string
	Ip         net.IP
	Port       int32
	ListenAddr *net.Listener
	Torrents   TorrentsPool
}

func (tk *TrackerServer) Publish(ctx context.Context, pq *pb.PublishQuery) (*pb.PublishResponse, error){
	ih := pq.GetInfoHash()
	if ih == nil || len(ih) != 20{
		err := fmt.Errorf("InfoHash invalido")
		return nil, err
	}
	infoHash := string(ih[:])
	peerID := pq.GetPeerID()
	if peerID == ""{
		err := fmt.Errorf("No se especifica el PeerID")
		return nil, err
	}
	ip := net.ParseIP(pq.GetIP())
	if ip == nil{
		err := fmt.Errorf("Error con la IP")
		return nil, err
	}
	port := pq.GetPort()
	if port == 0{
		err := fmt.Errorf("No se especifica el puerto")
		return nil, err
	}
	status := tk.publishTorrent(infoHash, peerID, int(port), ip)
	return &pb.PublishResponse{
		Status:status,
	}, nil
}

func (tk *TrackerServer) publishTorrent(infoHash, peerID string, port int, ip net.IP) int32 {
	tp := tk.Torrents
	if ttk, ok := tp[infoHash]; ok{
		tpeers := ttk.Peers
		if _, found := tpeers[peerID]; found{
			return ALREADYEXISTP
		}
		tpeers[peerID] = &PeerTk{
							Id:peerID, 
							Addr:&net.TCPAddr{
											IP:ip, 
											Port:int(port)}, 
							State:"seeder",} 
		return ALREADYEXISTT
	}
	tp[infoHash] = &TorrentTk{
		Hash:infoHash,
		Downloaded:0,
		Complete:1,
		Peers: PeersPool{
				 peerID:&PeerTk{
				 		Id:peerID, 
				 		Addr:&net.TCPAddr{
								IP:ip, 
								Port:port}, 
						State:"seeder"},
					    }}
	return OK
}

//func parseConvert(port uint32) returns (string) {
//	parsedPort := ":"
//}



func (tk *TrackerServer) Announce(ctx context.Context, annq *pb.AnnounceQuery) (*pb.AnnounceResponse, error) {
	pa, err := AnnounceQueryCheck(annq)
	var ar pb.AnnounceResponse
	if err != nil {
		ar.FailureReason = err.Error()
		return &ar, err
	}
	event := pa.Event
	request := pa.Request
	infoHash := pa.InfoHash
	//infoHash = pa.InfoHash
	
	
}

func AnnounceQueryCheck(annPb *pb.AnnounceQuery) (pa ParseAnnounce, err error) {
	infoHash := annPb.GetInfoHash()
	if infoHash == nil|| len(infoHash) != 20{
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
	if ip == nil{
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
	if numwant < 1{
		numwant = 40
	}
	if event != "request" && event != "started" && event != "completed" && event != "stopped"{
		err = fmt.Errorf("No se reconoce el evento especificado")
		return
	}
	pa.Event = event
	pa.NumWant = numwant
	pa.Request = annPb.GetRequest()
	return 
}


func (tk *TrackerServer) handlePublicEvent(pa *ParseAnnounce) {

}

func (tk *TrackerServer) handleStartedEvent(pa *ParseAnnounce) {
	
}

func (tk *TrackerServer) handleCompletedEvent(pa *ParseAnnounce) {
	
}

func (tk *TrackerServer) handleStopedEvent(pa *ParseAnnounce) {
	
}