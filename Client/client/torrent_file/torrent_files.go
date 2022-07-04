package torrent_file

import (
	"Bit_Torrent_Project/client/client/peer"
	"Bit_Torrent_Project/client/client/tracker_communication"
	"Bit_Torrent_Project/client/torrent_peer"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"trackerpb"

	"github.com/jackpal/bencode-go"
	"github.com/thanhpk/randstr"
)

const port = "50051"

type BencodeTorrent struct {
	Announce string
	Info     BencodeInfo
}

func (bto *BencodeTorrent) ConvertToTorrentFile() (*TorrentFile, error) {
	infoHash := bto.Info.InfoHash
	fmt.Println("Pieces strings hashes:", bto.Info.Pieces)
	pieces, err := bto.Info.splitPiecesHash()

	if err != nil {
		return nil, err
	}
	fmt.Println("Bencode pieces:", pieces)

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
	Name        string   `bencode:"name"`
	InfoHash    [20]byte `bencode:"info_hash"`
	PieceLength int      `bencode:"piece_length"`
	Pieces      string   `bencode:"pieces"`
	Length      int      `bencode:"length"`
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

// func (i *BencodeInfo) splitPiecesHash() ([][20]byte, error) {
// 	// Create slice of bytes from Pieces string
// 	buf := []byte(i.Pieces)
// 	if len(buf)%20 != 0 {
// 		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
// 		return nil, err
// 	}

// 	piecesHash := make([][20]byte, len(i.Pieces)/20)
// 	for j := 0; j < len(buf); j += 20 {
// 		copy(piecesHash[j%20][:], buf[j:j+20])
// 	}
// 	return piecesHash, nil
// }

func (i *BencodeInfo) splitPiecesHash() ([][20]byte, error) {
	hashLen := 20 // Length of SHA-1 hash
	buf := []byte(i.Pieces)
	if len(buf)%hashLen != 0 {
		err := fmt.Errorf("Received malformed pieces of length %d", len(buf))
		return nil, err
	}
	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
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
	// Files       []File   `bencode:"files"` // TODO
}

type File struct {
	Length int
	Path   []string
}

func BuildTorrentFile(filePath string, dstPath string, trackerUrl string) error {
	file, err := os.Open(filePath)

	if err != nil {
		log.Printf("Error while opening file: %v\n", err)
		return err
	}
	defer file.Close()
	// Name
	name := file.Name()

	// InfoHash
	infoHash := []byte(randstr.Hex(10))
	infoHashArr := [20]byte{}
	copy(infoHashArr[:], infoHash)
	fmt.Println(infoHashArr)
	// Piece length
	pieceLength := 256

	// File length
	fi, StErr := file.Stat()
	if StErr != nil {
		return StErr
	}

	length := fi.Size()
	// Pieces
	buf := make([]byte, length)
	_, bufErr := file.Read(buf)

	if bufErr != nil {
		return bufErr
	}

	piecesHash := ""
	if int(length)/pieceLength == 0 {
		hash := sha1.Sum(buf[0:])
		piecesHash += string(hash[:])
	}

	for i := 0; i < int(length)/pieceLength; i++ {
		begin := i * pieceLength
		end := (i + 1) * pieceLength
		if end > len(buf) {
			end = len(buf)
		}
		hash := sha1.Sum(buf[begin:end])
		piecesHash += string(hash[:])
	}

	torrentFile := BencodeTorrent{
		Announce: trackerUrl,
		Info: BencodeInfo{
			Name:        name,
			InfoHash:    infoHashArr,
			PieceLength: pieceLength,
			Pieces:      piecesHash,
			Length:      int(length),
		},
	}
	var buffer bytes.Buffer
	_ = bencode.Marshal(&buffer, torrentFile)
	err = os.WriteFile(dstPath, buffer.Bytes(), 0777)
	if err != nil {
		log.Printf("Error while writing torrent file: %v\n", err)
		return err
	}
	return nil
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
	fmt.Println("Bencoded torrent:", bencodeTorrent)

	return bencodeTorrent.ConvertToTorrentFile()

}

func (tf *TorrentFile) DownloadTo(path string, cs *torrent_peer.ConnectionsState, peerID string) error {
	fmt.Println("DownloadTo...")
	c, ctx, err := tracker_communication.TrackerClient(tf.Announce)
	IP := "192.168.169.14"

	fmt.Println("Requesting peers...")
	peersDict := tracker_communication.RequestPeers(c, tf.Announce, tf.Info.InfoHash, string(peerID[:]), IP, ctx)
	if err != nil {
		fmt.Printf("Error while requesting peers: %v\n", err)
		return err
	}

	peers := make([]peer.Peer, 0, len(peersDict))

	for k, v := range peersDict {
		ip := strings.Split(v, ":")
		p, _ := strconv.Atoi(ip[1])
		peers = append(peers, peer.Peer{Id: k, IP: net.ParseIP(ip[0]), Port: uint16(p)})
	}

	torrent := torrent_peer.Torrent{
		Peers:       peers,
		PeerId:      peerID,
		InfoHash:    tf.Info.InfoHash,
		PiecesHash:  tf.Info.Pieces,
		PieceLength: tf.Info.PieceLength,
		Length:      tf.Info.Length,
		Name:        tf.Info.Name,
	}
	tc := trackerpb.NewTrackerClient(c)

	p, _ := strconv.Atoi(port)

	res, err := torrent.DownloadFile(tc, IP, cs, port)
	if err != nil {
		return err
	}
	defer func() {
		announce := trackerpb.AnnounceQuery{
			InfoHash: tf.Info.InfoHash[:],
			PeerID:   peerID,
			IP:       IP,
			Port:     int32(p),
			Event:    "stopped",
		}
		_, _ = tc.Announce(ctx, &announce)
	}()

	announce := trackerpb.AnnounceQuery{
		InfoHash: tf.Info.InfoHash[:],
		PeerID:   peerID,
		IP:       IP,
		Port:     int32(p),
		Event:    "completed",
	}
	_, tResErr := tc.Announce(ctx, &announce)
	if tResErr != nil {
		return tResErr
	}

	outFile, err := os.Create(path + torrent.Name)
	if err != nil {
		log.Printf("Error while creating destiny file: %v\n", err)
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(res)
	if err != nil {
		return err
	}
	return nil
}
