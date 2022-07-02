package downloader_client

import (
	"Bit_Torrent_Project/client/client/communication"
	"crypto/tls"
	"log"
	"net"
	"time"
)

func SendHandshake(c net.Conn, infoHash [20]byte, peerId string) error {
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	defer c.SetDeadline(time.Time{}) // Disable the deadline

	hsh := communication.Handshake{Pstr: "BitTorrent protocol", PeerId: peerId, InfoHash: infoHash}
	serializedHsh := hsh.Serialize()
	log.Println(serializedHsh)
	_, err := c.Write(serializedHsh)
	if err != nil {
		log.Fatalf("Error while sending request: %v\n", err)
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
	c, err := tls.Dial("tcp", url, tlsConfig)
	if err != nil {
		errChan <- err
		return nil
	}
	log.Println("client: connected to: ", c.RemoteAddr())

	return c
}

func StartClientTCP(url string, infoHash [20]byte, peerID string, errChan chan error) (net.Conn, error) {
	c := StartConnection(url, errChan)
	hshErr := SendHandshake(c, infoHash, peerID)

	if hshErr != nil {
		errChan <- hshErr
		return nil, hshErr
	}

	return c, nil
}
