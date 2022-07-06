package torrent_file

import (
	"Bit_Torrent_Project/Client/client/peer"
	"Bit_Torrent_Project/Client/client/tracker_communication"
	dht "Bit_Torrent_Project/Client/dht/Kademlia"
	"Bit_Torrent_Project/Client/torrent_peer"
	"Bit_Torrent_Project/Client/trackerpb"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	//"Bit_Torrent_Project/Client/client/peer"
	//"Bit_Torrent_Project/Client/client/tracker_communication"
	//"Bit_Torrent_Project/Client/torrent_peer"
	//dht "Bit_Torrent_Project/Client/dht/Kademlia"
	//"bytes"
	//"crypto/sha1"
	//"fmt"
	//"log"
	//"net"
	//"os"
	//"strconv"
	//"strings"
	//"trackerpb"
	//
	"github.com/jackpal/bencode-go"
	"github.com/thanhpk/randstr"
)

const port = "50051"

type BencodeTorrent struct {
	Announce string
	Info     BencodeInfo
}

// Converts a BencodeTorrent int a TorrentFile
func (bto *BencodeTorrent) ConvertToTorrentFile() (*TorrentFile, error) {
	infoHash := bto.Info.InfoHash
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
	Name        string   `bencode:"name"`
	InfoHash    [20]byte `bencode:"info_hash"`
	PieceLength int      `bencode:"piece_length"`
	Pieces      string   `bencode:"pieces"`
	Length      int      `bencode:"length"`
}

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

// Builds and saves .torrent of filePath in dstPath
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
		fmt.Println(hash)
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

// Opens .torrent from path
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

// Downloads file with .torrent in path
func (tf *TorrentFile) DownloadTo(path string, cs *torrent_peer.ConnectionsState, IP string, peerID string, dhtNode *dht.DHT) ([]*torrent_peer.PieceResult, [20]byte, string, error) {
	fmt.Println("DownloadTo...")
	c, ctx, err := tracker_communication.TrackerClient(tf.Announce)
	if err != nil {
		fmt.Printf("Error while connecting to tracker: %v\n", err)
		return nil, [20]byte{}, "", err
	}

	fmt.Println("Requesting peers...")
	peersDict, err := tracker_communication.RequestPeers(c, tf.Announce, tf.Info.InfoHash, string(peerID[:]), IP, ctx)

	if err != nil {
		log.Printf("Error while requesting peers from tracker: %v\n", err)
		addrs, errDHT := dhtNode.GetPeersToDownload(tf.Info.InfoHash[:])
		if errDHT != nil {
			log.Printf("Error while requesting peers from dht: %v\n", err)
			return nil, [20]byte{}, "", errDHT
		}
		peersDict = map[string]string{}
		for i, addr := range addrs {
			peersDict[strconv.Itoa(i)] = addr
		}

	}

	peers := make([]peer.Peer, 0, len(peersDict))

	for k, v := range peersDict {
		ip := strings.Split(v, ":")
		p, _ := strconv.Atoi(ip[1])
		peers = append(peers, peer.Peer{Id: k, IP: net.ParseIP(ip[0]), Port: uint16(p)})
	}
	fmt.Println("Peers:", peers)
	torrent := torrent_peer.Torrent{
		Peers:       peers,
		PeerId:      peerID,
		InfoHash:    tf.Info.InfoHash,
		PiecesHash:  tf.Info.Pieces,
		PieceLength: tf.Info.PieceLength,
		Length:      tf.Info.Length,
		Name:        tf.Info.Name,
	}
	var tc trackerpb.TrackerClient
	if err != nil {
		tc = nil
	} else {

		tc = trackerpb.NewTrackerClient(c)
	}

	p, _ := strconv.Atoi(port)

	pieces, infoHash, bitfield, res, err := torrent.DownloadFile(tc, IP, cs, port, nil)//dhtNode)
	if err != nil {
		return nil, [20]byte{}, "", err
	}
	defer func() {
		log.Println("Stopping client...")
		announce := trackerpb.AnnounceQuery{
			InfoHash: tf.Info.InfoHash[:],
			PeerID:   peerID,
			IP:       IP,
			Port:     int32(p),
			Event:    "stopped",
		}
		_, _ = tc.Announce(ctx, &announce)
	}()

	log.Println("Sending completed to tracker...")
	announce := trackerpb.AnnounceQuery{
		InfoHash: tf.Info.InfoHash[:],
		PeerID:   peerID,
		IP:       IP,
		Port:     int32(p),
		Event:    "completed",
	}
	_, tResErr := tc.Announce(ctx, &announce)
	if tResErr != nil {
		return nil, [20]byte{}, "", tResErr
	}

	outFile, err := os.Create(path + torrent.Name)
	if err != nil {
		log.Printf("Error while creating destiny file: %v\n", err)
		return nil, [20]byte{}, "", err
	}
	defer outFile.Close()
	_, err = outFile.Write(res)
	if err != nil {
		return nil, [20]byte{}, "", err
	}
	return pieces, infoHash, bitfield, nil
}
