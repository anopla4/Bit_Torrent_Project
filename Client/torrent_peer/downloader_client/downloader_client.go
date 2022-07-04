package downloader_client

import (
	"Bit_Torrent_Project/client/client/communication"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
)

func SendHandshake(c net.Conn, infoHash [20]byte, peerId string) error {
	// _ = c.SetDeadline(time.Now().Add(10 * time.Second))
	// defer c.SetDeadline(time.Time{}) // Disable the deadline

	hsh := communication.Handshake{Pstr: "BitTorrent protocol", PeerId: peerId, InfoHash: infoHash}
	serializedHsh := hsh.Serialize()
	_, err := c.Write(serializedHsh)
	if err != nil {
		log.Fatalf("Error while sending handshake: %v\n", err)
		return err
	}
	return nil
}

func StartConnection(url string, errChan chan error) net.Conn {
	cert, tlsErr := tls.LoadX509KeyPair("./SSL/client.pem", "./SSL/client.key")

	if tlsErr != nil {
		log.Printf("Error while loading tls keys: %v\n", tlsErr)
		errChan <- tlsErr
		return nil
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
	fmt.Printf("Dialing to %s...\n", url)
	c, err := tls.Dial("tcp", url, tlsConfig)
	if err != nil {
		errChan <- err
		return nil
	}
	log.Println("client: connected")

	return c
}

func StartClientTCP(url string, infoHash [20]byte, id string, peerId string, errChan chan error) (net.Conn, error) {
	c := StartConnection(url, errChan)
	hshErr := SendHandshake(c, infoHash, id)

	if hshErr != nil {
		errChan <- hshErr
		return nil, hshErr
	}

	hsh, err := communication.DeserializeHandshake(c)

	if err != nil {
		errChan <- hshErr
		return nil, hshErr
	}

	fmt.Printf("Response from handshake: %v\n", hsh)

	if hsh.PeerId != peerId {
		return nil, errors.New("Peer ids does not match")
	}

	return c, nil
}
