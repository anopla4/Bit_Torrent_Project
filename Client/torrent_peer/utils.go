package torrent_peer

import (
	"Bit_Torrent_Project/client/client/structures"
	"io"
)

func ReadHandShake(r io.Reader) *structures.Handshake {
	return nil
}

func GetTorrent() {

}

func BuildRequestToTracker() string { return "" }

func BuildMessage(number int) structures.Message {
	return structures.Message{}
}

func BuildTorrentFile() structures.TorrentFile {
	return structures.TorrentFile{}
}

func ReadMessage([]byte) structures.Message {
	return structures.Message{}
}

func Choke() {}
