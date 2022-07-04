package main

import (
	"Bit_Torrent_Project/client/client/torrent_file"
	"Bit_Torrent_Project/client/client/tracker_communication"
	"Bit_Torrent_Project/client/torrent_peer"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

var wg = sync.WaitGroup{}

func main() {
	// peerID := randstr.Hex(10)

	arguments := os.Args

	peerId := arguments[1]

	info := LoadInfo("./info.json")
	errChan := make(chan error)
	csServer := &torrent_peer.ConnectionsState{LastUpload: map[string]time.Time{}, NumberOfBlocksInLast30Seconds: map[string]int{}}
	wg.Add(1)
	go StartClientUploader(info, peerId, csServer, errChan)
	// Start downloader
	var download string
	_, err := fmt.Scanln(&download)
	if err != nil {
		log.Fatal(err)
	}
	if download == "d" {
		fmt.Println("Waiting for paths to download...")
		var torrentPath, downloadTo string
		_, err := fmt.Scanln(&torrentPath, &downloadTo)
		if err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		go Download(torrentPath, downloadTo, peerId)
	}
	wg.Wait()
	// Publish("../a.torrent", "192.168.169.32:8167", peerId, "192.168.169.14")
	// _ = torrent_file.BuildTorrentFile("../test.txt", "../a.torrent", "192.168.169.32:8167")
}

func Download(torrentPath string, downloadTo string, peerId string) {
	torrentFile, err := torrent_file.OpenTorrentFile(torrentPath)
	fmt.Println("Pieces hashes:", torrentFile.Info.Pieces)
	csClient := &torrent_peer.ConnectionsState{LastUpload: map[string]time.Time{}, NumberOfBlocksInLast30Seconds: map[string]int{}}

	if err != nil {
		log.Printf("Error while parsing torrent: %v\n", err)
	}
	err = torrentFile.DownloadTo(downloadTo, csClient, peerId)

	if err != nil {
		log.Printf("Error while downloading torrent: %v\n", err)
	}
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

func Publish(path string, trackerUrl string, peerId string, IP string) {
	tf, tfErr := torrent_file.OpenTorrentFile(path)
	if tfErr != nil {
		log.Printf("Error while opening torrent file %v\n", tfErr)
		return
	}
	trRes := tracker_communication.PublishTorrent(trackerUrl, tf.Info.InfoHash, peerId, IP)
	fmt.Printf("Tracker status response from Publish: %v\n", trRes.GetStatus())
}
