package uploader_client

import (
	"Bit_Torrent_Project/client/client/structures"
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"
)

func ServerTCP(port string, errChan chan error) error {
	// Setting SSL certificate
	cert, err := tls.LoadX509KeyPair("./SSL/server.pem", "./SSL/server.key")
	if err != nil {
		log.Fatalf("Error loading certificate: %v\n", err)
	}

	certpool := x509.NewCertPool()
	pem, err := ioutil.ReadFile("./SSL/ca.pem")
	if err != nil {
		log.Fatalf("Failed to read client certificate authority: %v", err)
	}
	if !certpool.AppendCertsFromPEM(pem) {
		log.Fatalf("Can't parse client certificate authority")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certpool,
	}

	l, listenErr := tls.Listen("tcp", port, tlsCfg)
	if listenErr != nil {
		fmt.Println(listenErr)
		return listenErr
	}
	defer l.Close()

	// Handle connections
	for {
		c, err := l.Accept()

		if err != nil {
			log.Fatalf("Error while accepting next connection: %v\n", err)
			return err
		}
		log.Printf("Connection accepted: %v\n", c)

		tlsCon, ok := c.(*tls.Conn)
		if ok {
			log.Println("ok = true")
			state := tlsCon.ConnectionState()
			for _, v := range state.PeerCertificates {
				log.Print(x509.MarshalPKIXPublicKey(v.PublicKey))
			}
		}
		doneChan := make(chan struct{})
		go HandleTCPConnection(c, errChan, doneChan)

		if connErr := <-errChan; connErr != nil {
			fmt.Println(connErr)
		}
		<-doneChan
		// go func() {
		// 	for {
		// 		if connErr, ok := <-errChan; ok {
		// 			log.Fatalln(connErr)
		// 			break
		// 		}
		// 	}
		// }()
	}
}

func HandleTCPConnection(c net.Conn, errChan chan error, doneChan chan struct{}) {
	for {
		deadlineErr := c.SetReadDeadline(time.Now().Add(5 * time.Second))
		if deadlineErr != nil {
			fmt.Println(deadlineErr)
			errChan <- deadlineErr
			break
		}
		deserializedMessage, err := structures.Deserialize(bufio.NewReader(c))
		log.Println(deserializedMessage)
		if err != nil {
			fmt.Println(err)
			errChan <- err
			break
		}

	}
	close(doneChan)
}
