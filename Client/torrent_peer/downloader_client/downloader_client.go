package downloader_client

import (
	"Bit_Torrent_Project/client/client/communication"
	"crypto/tls"
	"log"
	"net"
)

func SendHandshake(infoHash [20]byte, peerId [20]byte, c net.Conn, errChan chan error) {
	hsh := communication.Handshake{Pstr: "BitTorrent protocol", PeerId: peerId, InfoHash: infoHash}
	serializedHsh := hsh.Serialize()
	log.Println(serializedHsh)
	_, err := c.Write(serializedHsh)
	if err != nil {
		log.Fatalf("Error while sending request: %v\n", err)
		errChan <- err
	}
}

func StartClientTCP(url string, errChan chan error) net.Conn {
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
