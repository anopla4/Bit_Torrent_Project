package downloader_client

import (
	"Bit_Torrent_Project/client/client/structures"
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
)

func SendRequests(c net.Conn, requests chan *structures.Message, errors chan error) {
	for {
		if req, ok := <-requests; ok {
			serializedReq := req.Serialize()
			log.Println(serializedReq)
			_, err := c.Write(serializedReq)
			if err != nil {
				log.Fatalf("Error while sending request: %v\n", err)
				errors <- err
				break
			}
		} else {
			break
		}
	}
}

func ReceiveResponses(c net.Conn, errors chan error) {
	for {
		message, err := structures.Deserialize(bufio.NewReader(c))
		if err != nil {
			errors <- err
			break
		}
		log.Fatalln(message)
	}
}

func StartClientTCP(url string) (net.Conn, error) {
	cert, tlsErr := tls.LoadX509KeyPair("./SSL/client.pem", "./SSL/client.key")

	if tlsErr != nil {
		log.Fatalf("Error while loading tls keys: %v\n", tlsErr)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
	c, err := tls.Dial("tcp", url, tlsConfig)
	if err != nil {
		return nil, err
	}
	log.Println("client: connected to: ", c.RemoteAddr())

	state := c.ConnectionState()
	for _, v := range state.PeerCertificates {
		// fmt.Println(x509.MarshalPKIXPublicKey(v.PublicKey))
		fmt.Println(v.Subject)
	}
	log.Println("client: handshake: ", state.HandshakeComplete)

	return c, nil
}
