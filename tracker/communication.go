package tracker

import (
	"net"
)

// AnnounceQuery es una representacion de una announce get query
type AnnounceQuery struct {
	InfoHash   string //dic bencodificado
	PeerID     string // Id del peer
	IP         net.IP //Ip del peer Opcional
	Port       int16  // Port del per Opcional
	Uploaded   int
	Downloaded int
	Left       int
	Event      int
	NumWant    int //Cantidad de peer que solicita el peer, Default 40
}

// AnnounceResponse es una representacion de una respuesta a un announce get query
type AnnounceResponse struct {
	FailureReason string
	Interval      int
	TrackerID     int
	Complete      int16
	Incomplete    int
	Peers         string
}

/*type ScraperQuery struct {
}*/
