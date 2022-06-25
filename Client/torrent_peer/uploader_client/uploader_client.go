package uploader_client

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func ServerTCP(port string) error {
	// Setting SSL certificate
	cert, err := tls.LoadX509KeyPair("./SSL/cert.pem", "./SSL/key.pem")
	if err != nil {
		log.Fatalf("Error loading certificate: %v\n", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	l, listenErr := tls.Listen("tcp", port, tlsCfg)
	if listenErr != nil {
		fmt.Println(listenErr)
		return listenErr
	}
	defer l.Close()

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

		errChan := make(chan error)
		go HandleTCPConnection(c, errChan)

		go func(errChan chan error) {
			for {
				if connErr, ok := <-errChan; ok {
					fmt.Println(connErr)
					break
				} else {
					break
				}
			}
		}(errChan)
	}
}

func HandleTCPConnection(c net.Conn, errorChan chan error) {
	var buf = bufio.NewReader(c)
	for {
		deadlineErr := c.SetReadDeadline(time.Now().Add(10 * time.Second))
		if deadlineErr != nil {
			fmt.Println(deadlineErr)
			errorChan <- deadlineErr
			break
		}
		netData, err := buf.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			errorChan <- err
			break
		}
		if strings.TrimSpace(string(netData)) == "STOP" {
			fmt.Println("Exiting TCP server!")
		}

		fmt.Print("-> ", string(netData))
		t := time.Now()
		myTime := t.Format(time.RFC3339) + "\n"
		_, writeErr := c.Write([]byte(myTime))

		if writeErr != nil {
			log.Fatalf("Error while writing to the connection: %v\n", writeErr)
			errorChan <- writeErr
			break
		}

	}
}
