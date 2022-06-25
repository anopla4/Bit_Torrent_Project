package main

import (
	"Bit_Torrent_Project/client/torrent_peer/downloader_client"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	"fmt"
	"log"
	"os"
	"sync"
)

var wg = sync.WaitGroup{}

func main() {

	wg.Add(2)

	arguments := os.Args

	go StartClientUploader(":" + arguments[1])

	fmt.Println("Type start")
	var start string
	_, err := fmt.Scanln(&start)
	if err != nil {
		log.Fatal(err)
	}

	if start == "s" {
		go StartClientDownloader(arguments[2])
	}

	wg.Wait()
}

func StartClientDownloader(url string) {
	// Dial throw port and ip from peer
	cc, err := downloader_client.StartClientTCP(url)

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
		return
	}
	defer cc.Close()

	// Start message stream through TCP connection
	streamErr := downloader_client.MessageStreamTCP(cc)

	if streamErr != nil {
		log.Fatalf("Could not connect: %v", err)
		return
	}

	wg.Done()
}

func StartClientUploader(port string) {
	fmt.Println("Hello World")

	// Listen to new connections
	err := uploader_client.ServerTCP(port)

	if err != nil {
		log.Fatalf("Error while starting server: %v\n", err)
		return
	}

	wg.Done()
}
