package uploader_client

import (
	"Bit_Torrent_Project/client/client/communication"
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

var connectionsGroup = sync.WaitGroup{}

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

	// Start listening for new connections
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
		log.Println("Connection accepted")

		tlsCon, ok := c.(*tls.Conn)
		if ok {
			log.Println("ok = true")
			state := tlsCon.ConnectionState()
			for _, v := range state.PeerCertificates {
				log.Print(x509.MarshalPKIXPublicKey(v.PublicKey))
			}
		}

		connectionsGroup.Add(1)
		go HandleTCPConnection(c, errChan)

		connectionsGroup.Wait()
	}
}

func HandleTCPConnection(c net.Conn, errChan chan error) {
	for {
		deadlineErr := c.SetReadDeadline(time.Now().Add(5 * time.Second))
		if deadlineErr != nil {
			errChan <- deadlineErr
			break
		}
		deserializedMessage, err := communication.Deserialize(bufio.NewReader(c))
		if err != nil {
			fmt.Println(err)
			errChan <- err
			break
		}
		log.Println(deserializedMessage.ID)

	}
	connectionsGroup.Done()
}
