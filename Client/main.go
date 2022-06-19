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

func main() {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func(port string) {
		fmt.Println("Hello World")

		uploader_client.StartServer(port)

		wg.Done()
	}("50051")

	go func(port string) {
		time.Sleep(20 * time.Second)
		cc, err := downloader_client.StartClient(port)

		if err != nil {
			log.Fatalf("Could not connect: %v", err)
			return
		}
		defer cc.Close()
		c := torrent_peerpb.NewDownloadServiceClient(cc)
		requests := []*torrent_peerpb.Message{
			{
				Length: 1, Id: 1,
			},
			{
				Length: 1, Id: 2,
			},
			{
				Length: 1, Id: 3,
			},
		}
		downloader_client.MessageStream(c, requests)

		wg.Done()
	}("6881")

	wg.Wait()
}
