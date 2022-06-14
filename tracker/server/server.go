package main

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/anoppa/Bit_Torrent_Project/tracker/trackerpb"
)

// AnnounceQuery es una representacion de una announce get query
type AnnounceQuery struct {
	InfoHash   string //dic bencodificado
	PeerID     string // Id del peer
	IP         net.IP //Ip del peer Opcional
	Port       int16  // Port del peer Opcional
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      string
	NumWant    uint32 //Cantidad de peer que solicita el peer, Default 50
}

// AnnounceResponse es una representacion de una respuesta a un announce get query
type AnnounceResponse struct {
	Interval   uint16
	TrackerID  string
	Complete   uint64
	Incomplete uint
	Peers      string
}

//PeersPool es un diccionario que mapea el peerAddr con un Peer
type PeersPool map[string]*PeerTk

//PeerTk es la representacion de un Peer para la el Tracker
type PeerTk struct {
	Id         string
	Addr       *net.TCPAddr
	lastTime   time.Time
	Uploaded   uint64
	Downloaded uint64
	leftDown   uint64
}

//TorrentsPool es un diccionario que mapea metainfoHash con el correspondiente torrent file
type TorrentsPool map[string]*TorrentTk

// TorrentTk es la representacion de un archivo torrent para el tracker
type TorrentTk struct {
	Hash       string
	Name       string
	Downloaded uint64
	Peers      PeersPool
}

type TrackerServer struct {
	AnnReq     AnnounceQuery
	AnnResp    AnnounceResponse
	TkID       string
	Ip         net.IP
	Port       string
	ListenAddr *net.Listener
	Torrents   TorrentsPool
}

func (tk *TrackerServer) Announce(ctx context.Context, annq *pb.AnnounceQuery) (*pb.AnnounceResponse, error) {

}

func AnnounceQueryCheck(annPb *pb.AnnounceQuery) (checedAnnounceQuery AnnounceQuery, err error) {
	checkedAnnounceQ := make(AnnounceQuery)
	infoHash := annPb.GetInfoHash()
	if infoHash == "" {
		err = fmt.Errorf("Infohash vacio")
		return
	}
	checkedAnnounceQ.InfoHash = infoHash
	peerID := annPb.GetPeerID
	if peerID == "" {
		err = fmt.Errorf("No se especifica el PeerID")
		return
	}
	checedAnnounceQuery.PeerID = peerID
	ip := annPb.GetIP()
	if ip == ""{
		err = fmt.Errorf("No se especifica el Ip")
		return
	}
	if 

}
