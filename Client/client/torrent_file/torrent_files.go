package torrent_file

import (
	"Bit_Torrent_Project/client/torrent_peer"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/jackpal/bencode-go"
)

type BencodeTorrent struct {
	Announce string
	Info     BencodeInfo
}

func (bto *BencodeTorrent) ConvertToTorrentFile() (*TorrentFile, error) {
	infoHash, err := bto.Info.hash()

	if err != nil {
		return nil, err
	}

	pieces, err := bto.Info.splitPiecesHash()

	if err != nil {
		return nil, err
	}

	tf := &TorrentFile{
		Announce: bto.Announce,
		Info: TorrentInfo{
			Name:        bto.Info.Name,
			InfoHash:    infoHash,
			PieceLength: bto.Info.PieceLength,
			Pieces:      pieces,
			Length:      bto.Info.Length,
		},
	}
	return tf, nil
}

type BencodeInfo struct {
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece_length"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
}

func (i *BencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buf.Bytes())
	return h, nil
}

func (i *BencodeInfo) splitPiecesHash() ([][20]byte, error) {
	// Create slice of bytes from Pieces string
	buf := []byte(i.Pieces)
	if len(buf)%20 != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}

	piecesHash := make([][20]byte, len(i.Pieces)/20)
	for j := 0; j < len(buf); j += 20 {
		copy(piecesHash[j%20][:], buf[j:j+20])
	}
	return piecesHash, nil
}

type TorrentFile struct {
	Announce string
	Info     TorrentInfo
}

type TorrentInfo struct {
	Name        string     `bencode:"name"`
	InfoHash    [20]byte   `bencode:"info_hash"`
	PieceLength int        `bencode:"piece_length"`
	Pieces      [][20]byte `bencode:"pieces"`
	Length      int        `bencode:"length"`
	// Files       []File   `bencode:"files"`
}

type File struct {
	Length int
	Path   []string
}

func OpenTorrentFile(path string) (*TorrentFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bencodeTorrent := BencodeTorrent{}
	err = bencode.Unmarshal(file, &bencodeTorrent)
	if err != nil {
		return nil, err
	}

	return bencodeTorrent.ConvertToTorrentFile()

}

func (tf *TorrentFile) DownloadTo(path string) error {
	// TODO Request peers to tracker

	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	torrent := torrent_peer.Torrent{
		// TODO Add Peers

		PeerId:      peerID,
		InfoHash:    tf.Info.InfoHash,
		PiecesHash:  tf.Info.Pieces,
		PieceLength: tf.Info.PieceLength,
		Length:      tf.Info.Length,
		Name:        tf.Info.Name,
	}

	res, err := torrent.DownloadFile()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(res)
	if err != nil {
		return err
	}
	return nil
}
