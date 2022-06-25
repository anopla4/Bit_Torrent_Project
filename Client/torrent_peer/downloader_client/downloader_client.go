package downloader_client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func MessageStreamTCP(c net.Conn) error {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(">> ")
		text, _ := reader.ReadString('\n')
		fmt.Fprintf(c, text+"\n")

		message, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			return err
		}

		fmt.Print("->: " + message)
		if strings.TrimSpace(string(text)) == "STOP" {
			fmt.Println("TCP client exiting...")
			return nil
		}
	}
}

func StartClientTCP(url string) (net.Conn, error) {
	cert, tlsErr := tls.LoadX509KeyPair("./SSL/client.pem", "./SSL/client.key")

	if tlsErr != nil {
		log.Fatalf("Error while loading tls keys: %v\n", tlsErr)
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: false}
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
