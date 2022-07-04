package dht

import (
	"bytes"
	"errors"
	"log"
	"net"
	"strconv"

	"github.com/jackpal/bencode-go"
)

//Server represent a server for kademlia comunication
type Server struct {
	ip         net.IP
	port       int
	handlerDHT handlerDHT
	Error      error
}

func newServer(ip string, port int, handlerDHT handlerDHT) *Server {
	server := &Server{}
	server.ip = net.ParseIP(ip)
	server.handlerDHT = handlerDHT
	server.port = port

	return server
}

func (s *Server) HandleMessage(l net.PacketConn, addr net.Addr, bufM []byte, nBytes int) {
	var msgQ QueryMessage    //To check if the message is a query
	var msgR ResponseMessage //To check if the message is a query
	var msgErr krpcError
	buf := bytes.NewBuffer(bufM[:nBytes])
	buf2 := bytes.NewBuffer(bufM[:nBytes])
	buf3 := bytes.NewBuffer(bufM[:nBytes])
	if err := bencode.Unmarshal(buf, &msgErr); err != nil {
		panic(err.Error())
	}
	if msgErr.TypeOfMessage == "e" {
		log.Println(msgErr.ErrorKRPC())
		return
	}

	//message correspond to a query
	if err := bencode.Unmarshal(buf2, &msgQ); err == nil && msgQ.TypeOfMessage == "q" {
		if msgQ.QueryName == "ping" {
			msgResponse, errQ := s.handlerDHT.ResponsePing(&msgQ, addr.String())
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := &bytes.Buffer{}
			err = bencode.Marshal(bufResponse, *msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err := l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return
		}
		//manage find_node
		if msgQ.QueryName == "find_node" {
			msgResponse, errQ := s.handlerDHT.ResponseFindNode(&msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, *msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return
		}
		//manage get_peers
		if msgQ.QueryName == "get_peers" {
			msgResponse, errQ := s.handlerDHT.ResponseGetPeers(&msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, *msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return
		}
		if msgQ.QueryName == "announce_peer" {
			msgResponse, errQ := s.handlerDHT.ResponseAnnouncePeers(&msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, *msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return
		}

		//manage announce_peers
		if msgQ.QueryName == "find_node" {
			msgResponse, errQ := s.handlerDHT.ResponseAnnouncePeers(&msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, *msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return
		}
	}
	//TODO:
	// message correspond to a response
	if err := bencode.Unmarshal(buf3, &msgR); err == nil {
		if msgR.TypeOfMessage == "r" {
			go s.handlerDHT.getResponse(&msgR, addr.String())
			return
		}
	}
	// s.handlerDHT.checkAddrInRoutingTable(addr.String())
	panic(errors.New("request type not found"))
}

// RunServer run a server to listen message from peers in network
func (s *Server) RunServer(exit chan string, conn chan *net.UDPConn) {
	l, err := net.ListenPacket("udp", s.ip.String()+":"+strconv.Itoa(s.port))
	log.Printf("serving... %s", l.LocalAddr().String())
	if err != nil {
		go func() { exit <- err.Error() }()
	}
	go func() { conn <- l.(*net.UDPConn) }()

	defer func() {
		err := l.Close()
		if err != nil {
			panic(err)
		}
	}()
	for {
		buf := make([]byte, 1024)
		bytesRead, addr, err := l.ReadFrom(buf)
		if err != nil {
			s.Error = err
			go func() { exit <- err.Error() }()
			break
		}
		go s.HandleMessage(l, addr, buf, bytesRead)

	}
}
