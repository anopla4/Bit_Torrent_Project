package structures

import "net"

type Peer struct {
	Id   string
	IP   net.IP
	Port uint16
}

type TorrentFile struct {
	Announce string
	Info     *InfoDictionary
}

type InfoDictionary struct {
	Name        string
	InfoHash    [20]byte
	PieceLength int
	Pieces      string
	Length      int
	Files       []File
}

type File struct {
	Length int
	Path   []string
}
