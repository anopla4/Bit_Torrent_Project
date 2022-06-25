package downloader_client

import (
	"bufio"
	"fmt"
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
	c, err := net.Dial("tcp", url)
	if err != nil {
		return nil, err
	}
	return c, nil
}
