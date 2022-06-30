package dht

import (
	"bytes"
	"errors"
	"log"
	"net"

	"github.com/jackpal/bencode-go"
)

//Server represent a server for kademlia comunication
type Server struct {
	ip         net.IP
	port       int
	handlerDHT handlerDHT
	Error      error
}

func newServer(ip string, port int, handlerDHT handlerDHT) {
	server := &Server{}
	server.ip = net.ParseIP(ip)
	server.handlerDHT = handlerDHT
}

func (s *Server) HandleMessage(l net.PacketConn, addr net.Addr, bufM []byte, nBytes int) error {
	msgQ := &QueryMessage{}    //To check if the message is a query
	msgR := &ResponseMessage{} //To check if the message is a query
	msgErr := &krpcError{}
	buf := bytes.NewBuffer(bufM[:nBytes])

	if err := bencode.Unmarshal(buf, msgErr); err == nil {
		log.Println("error: " + msgErr.ErrorKRPC())
		panic(msgErr.ErrorKRPC())
	}
	//message correspond to a query
	if err := bencode.Unmarshal(buf, msgQ); err == nil {
		if msgQ.QueryName == "ping" {
			msgResponse, errQ := s.handlerDHT.ResponsePing(msgQ, addr.String())
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return nil
		}
		//manage find_node
		if msgQ.QueryName == "find_node" {
			msgResponse, errQ := s.handlerDHT.ResponseFindNode(msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return nil
		}
		//manage get_peers
		if msgQ.QueryName == "find_peers" {
			msgResponse, errQ := s.handlerDHT.ResponseGetPeers(msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return nil
		}

		//manage announce_peers
		if msgQ.QueryName == "find_node" {
			msgResponse, errQ := s.handlerDHT.ResponseAnnouncePeers(msgQ, addr.String())
			//Pensar en hacer esta parte mas extensible
			if errQ != nil {
				log.Println("error: " + msgErr.ErrorKRPC())
			}
			bufResponse := bytes.Buffer{}
			err = bencode.Marshal(&bufResponse, msgResponse)
			if err != nil {
				log.Println("error: " + err.Error())
				panic(err)
			}
			_, err = l.WriteTo(bufResponse.Bytes(), addr)
			if err != nil {
				log.Println("error while writting to addr: " + addr.String())
				panic(err)
			}
			return nil
		}
	}
	// message correspond to a response
	if err := bencode.Unmarshal(buf, msgR); err == nil {
		//TODO
		return nil
	}

	return errors.New("request type not found")
}
func (s *Server) RunServer(exit chan string) {
	l, err := net.ListenPacket("udp", s.ip.String()+":"+string(s.port))
	if err != nil {
		exit <- err.Error()
	}
	defer func() {
		err := l.Close()
		if err != nil {
			log.Println("error while closing server: " + err.Error())
			panic(err)
		}
	}()

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
