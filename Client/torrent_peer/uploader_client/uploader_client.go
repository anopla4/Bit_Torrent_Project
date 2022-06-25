package uploader_client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func ServerTCP(port string) error {
	l, listenErr := net.Listen("tcp", port)
	if listenErr != nil {
		fmt.Println(listenErr)
		return listenErr
	}
	defer l.Close()

	c, err := l.Accept()

	if err != nil {
		log.Fatalf("Error while accepting next connection: %v\n", err)
	}

	var buf = bufio.NewReader(c)
	for {
		deadlineErr := c.SetReadDeadline(time.Now().Add(3 * time.Second))
		if deadlineErr != nil {
			fmt.Println(deadlineErr)
			return deadlineErr
		}
		netData, err := buf.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return err
		}
		if strings.TrimSpace(string(netData)) == "STOP" {
			fmt.Println("Exiting TCP server!")
			return nil
		}

		fmt.Print("-> ", string(netData))
		t := time.Now()
		myTime := t.Format(time.RFC3339) + "\n"
		_, writeErr := c.Write([]byte(myTime))

		if writeErr != nil {
			log.Fatalf("Error while writing to the connection: %v\n", writeErr)
			return writeErr
		}

	}
}
