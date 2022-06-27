package dht

import (
	"net"
)

//Server represent a server for kademlia comunication
type Server struct {
	ip     net.IP
	port   int
	caller interface{}
	Error  error
}

func newServer(ip string, port int, caller interface{}) {
	server := &Server{}
	server.ip = net.ParseIP(ip)
	server.caller = caller
}

func (s *Server) HandleMessage(l net.PacketConn, addr net.Addr, buf []byte, nBytes int) error {

	return nil
}
func (s *Server) RunServer(exit chan string) {
	l, err := net.ListenPacket("udp", s.ip.String()+":"+string(s.port))
	if err != nil {
		exit <- err.Error()
	}
	defer l.Close()

	for {
		buf := make([]byte, 1024)
		bytesRead, addr, err := l.ReadFrom(buf)
		if err != nil {
			s.Error = err
			exit <- err.Error()
			break
		}
		go s.HandleMessage(l, addr, buf, bytesRead)
	}
}
