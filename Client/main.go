package main

import (
	"Bit_Torrent_Project/client/client/structures"
	"Bit_Torrent_Project/client/torrent_peer/downloader_client"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var wg = sync.WaitGroup{}

func main() {
	wg.Add(2)

	arguments := os.Args

	requests := make(chan *structures.Message, 50)
	errorChan := make(chan error)
	go StartClientUploader(":"+arguments[1], errorChan)

	fmt.Println("Type start")
	var start string
	_, err := fmt.Scanln(&start)
	if err != nil {
		log.Fatal(err)
	}

	if start == "s" {
		go StartClientDownloader(arguments[2], requests, errorChan)
	}

	if err, ok := <-errorChan; ok {
		fmt.Println(err)
	}

	wg.Wait()
}

func StartClientDownloader(url string, requests chan *structures.Message, errChan chan error) {
	// Dial throw port and ip from peer
	cc, err := downloader_client.StartClientTCP(url)

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
		return
	}
	defer cc.Close()

	// Start message stream through TCP connection
	// requestsErr := make(chan error)
	// responsesErr := make(chan error)

	wgReqResp := sync.WaitGroup{}
	requests <- &structures.Message{
		ID: structures.HAVE,
	}
	wgReqResp.Add(2)
	go func() {
		downloader_client.SendRequests(cc, requests, errChan)
		wgReqResp.Done()
	}()

	go func() {
		downloader_client.ReceiveResponses(cc, errChan)
		wgReqResp.Done()
	}()

	time.Sleep(4 * time.Second)
	requests <- &structures.Message{
		ID: structures.HAVE,
	}
	wgReqResp.Wait()
	go func() {
		for {
			if err, ok := <-errChan; ok {
				fmt.Println(err)
			} else {
				break
			}
		}
	}()
	wg.Done()
}

func StartClientUploader(port string, errChan chan error) {
	fmt.Println("Hello World")

	// Listen to new connections
	err := uploader_client.ServerTCP(port, errChan)

	if err != nil {
		log.Fatalf("Error while starting server: %v\n", err)
		return
	}

	wg.Done()
}
