package uploader_client

import (
	"Bit_Torrent_Project/client/client/communication"
	"Bit_Torrent_Project/client/torrent_peer"
	"Bit_Torrent_Project/client/torrent_peer/downloader_client"
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"sort"
	"sync"
	"time"
)

const port = "50051"
const MaxUploads = 4

var connectionsGroup = sync.WaitGroup{}

type PeerConnection struct {
	Choked         bool
	PeerChoked     bool
	Peer           string
	Interested     bool
	PeerInterested bool
	c              net.Conn
	InfoHash       [20]byte
}

type Server struct {
	InfoHash     [20]byte // Required
	DownloadRate map[string]int
}

func ServerTCP(info *ClientInfoParsed, peerId string, servers []*Server, cs *torrent_peer.ConnectionsState, errChan chan error) error {
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

	peers := []*PeerConnection{}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certpool,
	}

	// Start listening for new connections
	l, listenErr := tls.Listen("tcp", "localhost"+":"+port, tlsCfg)
	if listenErr != nil {
		fmt.Println(listenErr)
		return listenErr
	}
	defer l.Close()

	go Choke(peers, cs)

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
		go HandleTCPConnection(c, peerId, info, cs, servers, errChan)

		connectionsGroup.Wait()
	}
}

type ClientTorrentFileInfo struct {
	Pieces      []string
	Bitfield    communication.Bitfield
	Path        string
	PieceLength int
}

type ClientTorrentFileInfoParsed struct {
	Pieces      [][20]byte
	Bitfield    communication.Bitfield
	Path        string
	PieceLength int
}

type ClientInfo struct {
	TorrentFiles map[string]ClientTorrentFileInfo
}

type ClientInfoParsed struct {
	TorrentFiles map[[20]byte]ClientTorrentFileInfoParsed
}

func (ci *ClientInfo) ParseInfo() ClientInfoParsed {
	res := ClientInfoParsed{}
	for k, v := range ci.TorrentFiles {
		citfp := ClientTorrentFileInfoParsed{
			Pieces:      [][20]byte{},
			Bitfield:    v.Bitfield,
			Path:        v.Path,
			PieceLength: v.PieceLength,
		}
		for _, ps := range v.Pieces {
			ph := [20]byte{}
			copy(ph[:], []byte(ps))
			citfp.Pieces = append(citfp.Pieces, ph)
		}
		infoHash := [20]byte{}
		copy(infoHash[:], []byte(k))
		res.TorrentFiles[infoHash] = citfp
	}
	return res
}

func HandleTCPConnection(c net.Conn, peerId string, info *ClientInfoParsed, cs *torrent_peer.ConnectionsState, servers []*Server, errChan chan error) {
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

		infoHash, id, hshErr := Handshake(c, peerId, errChan)

		s, found := FindServer(servers, infoHash)

		if !found {
			s = &Server{}
			s.InfoHash = infoHash
			servers = append(servers, s)
		}

		if hshErr != nil {
			return
		}

		pc := &PeerConnection{
			Choked:     true,
			Interested: false,
			Peer:       id,
			c:          c,
			InfoHash:   infoHash,
		}

		bfErr := SendBitfield(c, info, infoHash, errChan)

		if bfErr != nil {
			return
		}

		HandleMessage(c, pc, info, deserializedMessage, cs, s, errChan)
	}
	connectionsGroup.Done()
}

func FindServer(servers []*Server, infoHash [20]byte) (*Server, bool) {
	var s *Server
	for _, server := range servers {
		if server.InfoHash == infoHash {
			s = server
			return s, true
		}
	}
	return s, false
}

func Handshake(c net.Conn, peerId string, errChan chan error) ([20]byte, string, error) {
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	defer c.SetDeadline(time.Time{}) // Disable the deadline

	hsh, err := communication.DeserializeHandshake(c)
	if err != nil {
		errChan <- err
		return [20]byte{}, "", err
	}
	id := hsh.PeerId
	hsh.PeerId = peerId

	hshErr := downloader_client.SendHandshake(c, hsh.InfoHash, peerId)

	if hshErr != nil {
		errChan <- hshErr
		return [20]byte{}, "", hshErr
	}

	return hsh.InfoHash, id, nil
}
func SendBitfield(c net.Conn, info *ClientInfoParsed, infoHash [20]byte, errChan chan error) error {
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	defer c.SetDeadline(time.Time{}) // Disable the deadline

	msg := communication.Message{ID: communication.BITFIELD, Payload: info.TorrentFiles[infoHash].Bitfield}
	serializedMsg := msg.Serialize()
	log.Println(serializedMsg)
	_, err := c.Write(serializedMsg)
	if err != nil {
		log.Printf("Error while sending request: %v\n", err)
		errChan <- err
		return err
	}
	return nil
}

func HandleMessage(c net.Conn, pc *PeerConnection, info *ClientInfoParsed, msg *communication.Message, cs *torrent_peer.ConnectionsState, s *Server, errChan chan error) {
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.SetDeadline(time.Time{}) // Disable the deadline

	if msg.ID == communication.KEEPALIVE {
		return
	}

	switch msg.ID {
	case communication.CHOKE:
		pc.Choked = true
	case communication.UNCHOKE:
		pc.Choked = false
	case communication.INTERESTED:
		pc.PeerInterested = true
	case communication.NOTINTERESTED:
		pc.PeerInterested = false
	case communication.HAVE:
		HandleHave(c, s, msg, pc, info, errChan)
	case communication.BITFIELD:
		break
	case communication.REQUEST:
		if cs.NumberOfUploadPeers < MaxUploads && pc.PeerChoked {
			err := pc.SendUnchoke()
			if err != nil {
				return
			}
		}
		if !pc.Choked {
			HandleRequest(c, msg, pc, info, errChan)
		}
	case communication.PIECE:
		break
	case communication.CANCEL:
		c.Close()
	}
}

func (p *PeerConnection) SendUnchoke() error {
	_, err := p.c.Write((&communication.Message{ID: communication.UNCHOKE}).Serialize())
	if err != nil {
		p.PeerChoked = false
	}
	return err
}
func (p *PeerConnection) SendChoke() error {
	_, err := p.c.Write((&communication.Message{ID: communication.UNCHOKE}).Serialize())
	if err != nil {
		p.PeerChoked = true
	}
	return err
}

func HandleRequest(c net.Conn, msg *communication.Message, pc *PeerConnection, info *ClientInfoParsed, errChan chan error) {
	index, begin, length, err := communication.ParseRequest(*msg)

	if err != nil {
		errChan <- err
		return
	}
	file, err := os.Open(info.TorrentFiles[pc.InfoHash].Path)

	buf := make([]byte, length)
	_, bufErr := file.Read(buf)

	if bufErr != nil {
		log.Println("Error while reading file bytes")
		errChan <- err
		return
	}
	piecePayload := make([]byte, 8+length)
	binary.BigEndian.PutUint32(piecePayload, uint32(index))
	binary.BigEndian.PutUint32(piecePayload[4:8], uint32(begin))
	copy(piecePayload[8:], buf[index*info.TorrentFiles[pc.InfoHash].PieceLength:])

	pieceMsg := communication.Message{ID: communication.PIECE, Payload: piecePayload}

	serializedMsg := pieceMsg.Serialize()
	log.Println(serializedMsg)
	_, err = c.Write(serializedMsg)
	if err != nil {
		log.Fatalf("Error while sending request: %v\n", err)
		errChan <- err
		return
	}
}
func HandleHave(c net.Conn, s *Server, msg *communication.Message, pc *PeerConnection, info *ClientInfoParsed, errChan chan error) {
	_, err := communication.ParseHave(*msg)
	if err != nil {
		log.Fatalf("Error while parsing HAVE message: %v\n", err)
		errChan <- err
		return
	}
	s.DownloadRate[pc.Peer]++
}

func Choke(peers []*PeerConnection, cs *torrent_peer.ConnectionsState) {
	round := 0
	optimisticChoked := PeerConnection{}
	for {
		uploadRateOrder := []*PeerConnection{}
		forOptimisticUnchoke := []*PeerConnection{}
		for _, p := range peers {
			if p.PeerInterested {
				uploadRateOrder = append(uploadRateOrder, p)
				if p.PeerChoked {
					forOptimisticUnchoke = append(forOptimisticUnchoke, p)
				}
			}
		}
		if round%3 == 0 {
			n := rand.Intn(len(forOptimisticUnchoke))
			optimisticChoked = *forOptimisticUnchoke[n]
		}
		sort.Slice(uploadRateOrder, func(i, j int) bool {
			lastTimeI := cs.LastUpload[uploadRateOrder[i].Peer]
			numberOfBlocksI := cs.NumberOfBlocksInLast30Seconds[uploadRateOrder[i].Peer]
			lastTimeJ := cs.LastUpload[uploadRateOrder[j].Peer]
			numberOfBlocksJ := cs.NumberOfBlocksInLast30Seconds[uploadRateOrder[j].Peer]

			return numberOfBlocksI/(int(time.Now().Sub(lastTimeI))) > numberOfBlocksJ/(int(time.Now().Sub(lastTimeJ)))
		})

		isContained := false
		for _, peer := range uploadRateOrder[0:3] {
			peer.SendUnchoke()
			if peer.Peer == optimisticChoked.Peer {
				isContained = true
			}
		}
		if round%3 == 0 {
			if !isContained {
				optimisticChoked.SendUnchoke()
			} else {
				for len(forOptimisticUnchoke) > 0 {
					forOptimisticUnchoke = []*PeerConnection{}
					for _, p := range peers {
						if p.PeerChoked && p.Peer != optimisticChoked.Peer {
							forOptimisticUnchoke = append(forOptimisticUnchoke, p)
						}
					}
					n := rand.Intn(len(forOptimisticUnchoke))
					forOptimisticUnchoke[n].SendUnchoke()
					optimisticChoked = *forOptimisticUnchoke[n]
					if forOptimisticUnchoke[n].PeerInterested {
						break
					}
				}
			}
		}

		time.Sleep(10 * time.Second)
	}
}
