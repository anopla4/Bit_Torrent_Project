package main

import (
	dht "dht/Kademlia"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	if os.Args[1] == "1" {
		client1()
	} else {
		port, _ := strconv.Atoi(os.Args[1])
		client2(port)
	}
}

func client1() {
	id := getIDWithB(uint8(5))
	options := &dht.Options{
		ID:                   id,
		IP:                   "127.0.0.1",
		Port:                 5456,
		ExpirationTime:       time.Minute * 10,
		RepublishTime:        time.Minute * 5,
		TimeToDie:            time.Second,
		TimeToRefreshBuckets: time.Minute * 15,
	}
	dht1 := dht.NewDHT(options)
	exitChan := make(chan string)
	go dht1.RunServer(exitChan)
	time.Sleep(time.Second * 20)
	dht1.AnnouncePeer(string(getIDWithB(8)), 3030)
	select {
	case msg := <-exitChan:
		log.Println(msg)
	case <-time.After(time.Minute * 5):
		log.Println("closed client")
	}
}

func client2(port int) {
	id := getIDWithB(uint8(port % 10))
	options := &dht.Options{
		ID:                   id,
		IP:                   "127.0.0.1",
		Port:                 port,
		ExpirationTime:       time.Minute,
		RepublishTime:        time.Minute,
		TimeToDie:            time.Second,
		TimeToRefreshBuckets: time.Minute * 15,
	}
	dht1 := dht.NewDHT(options)
	exitChan := make(chan string)
	go dht1.RunServer(exitChan)
	time.Sleep(time.Second * 10)
	dht1.JoinNetwork("127.0.0.1:5456")
	time.Sleep(time.Second * 2)

	log.Printf("nodes known... %d", dht1.RoutingTable.GetTotalKnownNodes())

	time.Sleep(time.Second * 10)
	values, err := dht1.GetPeersToDownload(getIDWithB(8))
	if err != nil {
		log.Println(err.Error())
	}

	if port%4 == 0 || port%4 == 1 {
		dht1.AnnouncePeer(string(getIDWithB(8)), 3039)
	}
	for _, val := range values {

		fmt.Printf("el infohash esta en : %s", val)
	}
	time.Sleep(time.Second * 3)
	values, err = dht1.GetPeersToDownload(getIDWithB(8))
	if err != nil {
		log.Println(err.Error())
	}
	for _, val := range values {

		fmt.Printf("el infohash esta en : %s", val)
	}

	select {
	case msg := <-exitChan:
		log.Println(msg)
	case <-time.After(time.Minute * 5):
		log.Println("closed client")
	}
}
func getIDWithB(b byte) []byte {
	return []byte{b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b}
}
