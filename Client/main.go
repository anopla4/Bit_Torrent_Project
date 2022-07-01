package main

import (
	"Bit_Torrent_Project/client/client/torrent_file"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	"fmt"
	"log"
	"os"
)

func main() {
	arguments := os.Args

	torrentPath := arguments[1]
	downloadTo := arguments[2]

	torrentFile, err := torrent_file.OpenTorrentFile(torrentPath)

	if err != nil {
		log.Printf("Error while parsing torrent: %v\n", err)
	}

	err = torrentFile.DownloadTo(downloadTo)

	if err != nil {
		log.Printf("Error while downloading torrent: %v\n", err)
	}
}

func StartClientUploader(port string, errChan chan error) {
	fmt.Println("Hello World")

	// Listen to new connections
	err := uploader_client.ServerTCP(port, errChan)

	if err != nil {
		log.Fatalf("Error while starting server: %v\n", err)
		return
	}
}
