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
	NumWant    int //Cantidad de peer que solicita el peer, Default 50
}

//IAnnounceResponse representa el tipo de una respuesta a una peticion Announce
type IAnnounceResponse interface{}

// AnnounceResponse es una representacion de una respuesta a un announce get query
type AnnounceResponse struct {
	Interval   int
	TrackerID  int
	Complete   int
	Incomplete int
	Peers      string
}

//ScraperQuery representa el contenido de una peticion Scrape al Tracker
type ScraperQuery struct {
	InfoHash []string
}

//File representa un file dentro de una una respuesta a peticion Scraper
type File struct {
	Incomplete int
	Complete   int
	Downloaded int
}

//ScraperResponse representa el contenido de una respuesta a peticion Scraper
type ScraperResponse struct {
	Files map[string]File
}

//ErrorResponse representa la respuesta cuando ocurre un error en el proceseo de algun
//request en el Tracker
type ErrorResponse struct {
	FailureReason string
}
