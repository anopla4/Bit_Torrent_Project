package main

import (
	"Bit_Torrent_Project/client/client/communication"
	"Bit_Torrent_Project/client/client/torrent_file"
	"Bit_Torrent_Project/client/client/tracker_communication"
	"Bit_Torrent_Project/client/torrent_peer"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	dht "Bit_Torrent_Project/dht/Kademlia"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/thanhpk/randstr"
	"golang.org/x/exp/slices"
)

var wg = sync.WaitGroup{}

const bootstrapPort = "6888"

func main() {
	peerId := ""
	info := LoadInfo("./info.json")
	if info.PeerId != "" {
		peerId = info.PeerId
	} else {
		peerId = randstr.Hex(10)
	}

	Publish("../b.torrent", peerId, "192.168.169.14")
	// _ = torrent_file.BuildTorrentFile("../VID-20211202-WA0190.mp4", "../b.torrent", "192.168.169.32:8167")

	errChan := make(chan error)

	// Starting server
	csServer := &torrent_peer.ConnectionsState{LastUpload: map[string]time.Time{}, NumberOfBlocksInLast30Seconds: map[string]int{}}
	wg.Add(1)
	go StartClientUploader(info, peerId, csServer, errChan)

	var torrentPath, downloadTo string
	torrentPath = "../c.torrent"
	downloadTo = "../"
	IP := "192.168.169.14"
	bootstrapADDR := IP + ":" + bootstrapPort

	// Starting DHT
	options := &dht.Options{
		ID:                   []byte(peerId),
		IP:                   IP,
		Port:                 6881,
		ExpirationTime:       time.Hour,
		RepublishTime:        time.Minute * 50,
		TimeToDie:            time.Second * 6,
		TimeToRefreshBuckets: time.Minute * 15,
	}
	dhtNode := dht.NewDHT(options)
	exitChan := make(chan string)

	go dhtNode.RunServer(exitChan)

	time.Sleep(time.Second * 2)

	dhtNode.JoinNetwork(bootstrapADDR)

	// Starting downloader
	var download string
	_, err := fmt.Scanln(&download)
	if err != nil {
		log.Fatal(err)
	}
	if download == "d" {
		fmt.Println("Waiting for paths to download...")
		wg.Add(1)
		go func() {
			pieces, infoHash, bitfield, err := Download(torrentPath, downloadTo, IP, peerId, dhtNode)
			if err != nil {
				log.Println(err)
				return
			}
			SaveToInfo(torrentPath, pieces, infoHash, []byte(bitfield))
			wg.Done()
		}()
	}

	go func() {
		for {
			if _, ok := <-exitChan; ok {
				go dhtNode.RunServer(exitChan)
			}
		}
	}()

	wg.Wait()

	defer close(exitChan)
}

func Download(torrentPath string, downloadTo string, IP string, peerId string, dhtNode *dht.DHT) ([]*torrent_peer.PieceResult, [20]byte, string, error) {
	torrentFile, err := torrent_file.OpenTorrentFile(torrentPath)
	csClient := &torrent_peer.ConnectionsState{LastUpload: map[string]time.Time{}, NumberOfBlocksInLast30Seconds: map[string]int{}}

	if err != nil {
		log.Printf("Error while parsing torrent: %v\n", err)
	}
	pieces, infoHash, bitfield, err := torrentFile.DownloadTo(downloadTo, csClient, IP, peerId, dhtNode)

	if err != nil {
		log.Printf("Error while downloading torrent: %v\n", err)
	}
	return pieces, infoHash, bitfield, nil
}

func StartClientUploader(info *uploader_client.ClientInfo, peerID string, cs *torrent_peer.ConnectionsState, errChan chan error) {
	fmt.Println("Hello World")

	// Listen to new connections
	err := uploader_client.ServerTCP(info, peerID, cs, errChan)

	if err != nil {
		log.Fatalf("Error while starting server: %v\n", err)
		return
	}
	wg.Done()
}

func LoadInfo(path string) *uploader_client.ClientInfo {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	var buf uploader_client.ClientInfo
	err = json.Unmarshal(content, &buf)
	if err != nil {
		log.Fatalf("Error during Unmarshal(): %v\n", err)
	}
	return &buf
}

func Publish(path string, peerId string, IP string) {
	tf, tfErr := torrent_file.OpenTorrentFile(path)
	if tfErr != nil {
		log.Printf("Error while opening torrent file %v\n", tfErr)
		return
	}
	trRes := tracker_communication.PublishTorrent(tf.Announce, tf.Info.InfoHash, peerId, IP)
	fmt.Printf("Tracker status response from Publish: %v\n", trRes.GetStatus())
}

func SaveToInfo(path string, pieces []*torrent_peer.PieceResult, infoHash [20]byte, bitfield communication.Bitfield) {
	log.Println("Saving to client's Info...")
	info := LoadInfo("./info.json")
	temp := [][20]byte{}
	newInfo := uploader_client.ClientInfo{}
	newInfo.TorrentFiles = map[string]uploader_client.ClientTorrentFileInfo{}
	newInfo.PeerId = info.PeerId
	hasInfoHash := false
	newIndex := strconv.Itoa(len(newInfo.TorrentFiles) + 1)
	for k, v := range info.TorrentFiles {
		newInfo.TorrentFiles[k] = v
		if infoHash == v.InfoHash {
			hasInfoHash = true
			newIndex = k
		}
	}
	if !hasInfoHash {
		newInfo.TorrentFiles[newIndex] = uploader_client.ClientTorrentFileInfo{}
	}
	temp = append(temp, info.TorrentFiles[newIndex].Pieces...)
	pieceLength := 256
	if hasInfoHash {
		pieceLength = info.TorrentFiles[newIndex].PieceLength
	}
	fmt.Println(bitfield)
	newInfo.TorrentFiles[string(infoHash[:])] = uploader_client.ClientTorrentFileInfo{
		InfoHash:    infoHash,
		Pieces:      temp,
		Path:        path,
		Bitfield:    bitfield,
		PieceLength: pieceLength,
	}
	for _, p := range pieces {
		pInfoHash := sha1.Sum(p.Piece)
		if !slices.Contains(temp, pInfoHash) {
			copy(temp, info.TorrentFiles[string(infoHash[:])].Pieces)
			temp = append(temp, pInfoHash)
		}
	}

	buf, jsonErr := json.MarshalIndent(newInfo, "", " ")

	if jsonErr != nil {
		log.Println("Error while marshaling Info")
	}

	err := ioutil.WriteFile("./info.json", buf, 0777)

	if err != nil {
		log.Println("Error while writing in info file")
	}
}
