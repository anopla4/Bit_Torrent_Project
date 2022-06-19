package main

import (
	"Bit_Torrent_Project/client/torrent_peer/downloader_client"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	"fmt"
	"log"
	"sync"
	"time"
	"torrent_peerpb"
)

var wg = sync.WaitGroup{}

func main() {

	wg.Add(2)

	fmt.Println("Input server ip and port")
	var ip, port string
	_, err := fmt.Scanln(&ip, &port)
	if err != nil {
		log.Fatal(err)
	}

	go StartClientUploader(ip, port)

	fmt.Println("Input peer ip and port")
	var peer_ip, peer_port string
	_, err = fmt.Scanln(&peer_ip, &peer_port)
	if err != nil {
		log.Fatal(err)
	}

	requests := make(chan *torrent_peerpb.Message, 4)
	requests <- &torrent_peerpb.Message{
		Length: 1, Id: 1,
	}

	go StartClientDownloader(peer_ip, peer_port, requests)

	time.Sleep(4 * time.Second)
	requests <- &torrent_peerpb.Message{
		Length: 1, Id: 2,
	}
	requests <- &torrent_peerpb.Message{
		Length: 1, Id: 3,
	}

	wg.Wait()
}

func StartClientDownloader(ip string, port string, requests chan *torrent_peerpb.Message) {
	// Dial throw port and ip from peer
	cc, err := downloader_client.StartClient(port)

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
		return
	}
	defer cc.Close()

	// Initialize Download service in connection
	c := torrent_peerpb.NewDownloadServiceClient(cc)

	downloader_client.MessageStream(c, requests)

	wg.Done()
}

func StartClientUploader(ip string, port string) {
	fmt.Println("Hello World")

	// Listen to new connections
	uploader_client.StartServer(port)

	wg.Done()
}
