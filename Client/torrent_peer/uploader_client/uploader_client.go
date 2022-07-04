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

const port = "50052"
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

func ServerTCP(info *ClientInfo, peerId string, cs *torrent_peer.ConnectionsState, errChan chan error) error {
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

	// peers := []*PeerConnection{}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certpool,
	}

	// Start listening for new connections
	l, listenErr := tls.Listen("tcp", "192.168.169.14"+":"+port, tlsCfg)
	if listenErr != nil {
		fmt.Println(listenErr)
		return listenErr
	}
	defer l.Close()

	// go Choke(peers, cs)

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
		go HandleTCPConnection(c, peerId, info, cs, errChan)

		connectionsGroup.Wait()
	}
}

type ClientTorrentFileInfo struct {
	Pieces      [][20]byte
	Bitfield    communication.Bitfield
	Path        string
	PieceLength int
	InfoHash    [20]byte
}

type ClientInfo struct {
	TorrentFiles map[string]ClientTorrentFileInfo
}

func HandleTCPConnection(c net.Conn, peerId string, info *ClientInfo, cs *torrent_peer.ConnectionsState, errChan chan error) {
	// deadlineErr := c.SetReadDeadline(time.Now().Add(30 * time.Second))
	// if deadlineErr != nil {
	// 	errChan <- deadlineErr
	// 	break
	// }

	infoHash, id, hshErr := Handshake(c, peerId, errChan)

	if hshErr != nil {
		errChan <- hshErr
		return
	}
	bfErr := SendBitfield(c, info, infoHash, errChan)
	fmt.Println("InfoHash", infoHash)

	if bfErr != nil {
		errChan <- bfErr
		return
	}

	pc := &PeerConnection{
		Choked:     true,
		Interested: false,
		Peer:       id,
		c:          c,
		InfoHash:   infoHash,
	}
	_ = pc.SendUnchoke()
	for {
		deserializedMessage, err := communication.Deserialize(bufio.NewReader(c))
		if err != nil {
			fmt.Println(err)
			errChan <- err
			break
		}
		log.Println("Message:", deserializedMessage.ID)
		HandleMessage(c, pc, info, deserializedMessage, cs, errChan)
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
	// _ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	// defer c.SetReadDeadline(time.Time{}) // Disable the deadline

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
func SendBitfield(c net.Conn, info *ClientInfo, infoHash [20]byte, errChan chan error) error {
	// _ = c.SetDeadline(time.Now().Add(10 * time.Second))
	// defer c.SetDeadline(time.Time{}) // Disable the deadline

	msg := communication.Message{ID: communication.BITFIELD, Payload: getClientTorrentFileInfo(info, infoHash).Bitfield}
	serializedMsg := msg.Serialize()
	log.Println("Bitfield:", serializedMsg)
	_, err := c.Write(serializedMsg)
	if err != nil {
		log.Printf("Error while sending request: %v\n", err)
		errChan <- err
		return err
	}
	return nil
}

func HandleMessage(c net.Conn, pc *PeerConnection, info *ClientInfo, msg *communication.Message, cs *torrent_peer.ConnectionsState, errChan chan error) {
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.SetDeadline(time.Time{}) // Disable the deadline

	if msg.ID == communication.KEEPALIVE {
		return
	}

	switch msg.ID {
	case communication.CHOKE:
		log.Println("Choke received")
		pc.Choked = true
	case communication.UNCHOKE:
		log.Println("Unchoke received")
		log.Println("Remote Addr ", c.RemoteAddr().String())
		log.Println("Local  Addr ", c.LocalAddr().String())
		pc.Choked = false
	case communication.INTERESTED:
		log.Println("Interested received")
		pc.PeerInterested = true
	case communication.NOTINTERESTED:
		log.Println("NotInterested received")
		pc.PeerInterested = false
	case communication.HAVE:
		log.Println("Have received")
		HandleHave(c, cs, msg, pc, info, errChan)
	case communication.BITFIELD:
		break
	case communication.REQUEST:
		log.Println("Request received")
		if cs.NumberOfUploadPeers < MaxUploads && pc.PeerChoked {
			err := pc.SendUnchoke()
			if err != nil {
				return
			}
		}
		log.Println(pc.Choked)
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
	//fmt.Println("3")
	//time.Sleep(3 * time.Second)
	_, err := p.c.Write((&communication.Message{ID: communication.UNCHOKE}).Serialize())
	if err != nil {
		p.PeerChoked = false
	}
	return err
}
func (p *PeerConnection) SendChoke() error {
	//fmt.Println("4")
	//time.Sleep(3 * time.Second)

	_, err := p.c.Write((&communication.Message{ID: communication.CHOKE}).Serialize())
	if err != nil {
		p.PeerChoked = true
	}
	return err
}

func HandleRequest(c net.Conn, msg *communication.Message, pc *PeerConnection, info *ClientInfo, errChan chan error) {
	index, begin, length, err := communication.ParseRequest(*msg)
	log.Println("Handling request...")
	if err != nil {
		log.Println("Error:", err)
		errChan <- err
		return
	}
	ci := getClientTorrentFileInfo(info, pc.InfoHash)
	log.Println("Client info", ci)
	file, err := os.Open(ci.Path)
	log.Println("File:", file)
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
	copy(piecePayload[8:], buf[index*getClientTorrentFileInfo(info, pc.InfoHash).PieceLength:])

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

func getClientTorrentFileInfo(info *ClientInfo, infoHash [20]byte) ClientTorrentFileInfo {
	for _, tf := range info.TorrentFiles {
		if tf.InfoHash == infoHash {
			return tf
		}
	}
	return ClientTorrentFileInfo{}
}

func HandleHave(c net.Conn, cs *torrent_peer.ConnectionsState, msg *communication.Message, pc *PeerConnection, info *ClientInfo, errChan chan error) {
	_, err := communication.ParseHave(*msg)
	if err != nil {
		log.Fatalf("Error while parsing HAVE message: %v\n", err)
		errChan <- err
		return
	}
	if time.Since(cs.LastUpload[pc.Peer]) > 30*time.Second {
		cs.LastUpload[pc.Peer] = time.Now()
		cs.NumberOfBlocksInLast30Seconds[pc.Peer] = 0
	} else {
		cs.NumberOfBlocksInLast30Seconds[pc.Peer]++
	}
}

func Choke(peers []*PeerConnection, cs *torrent_peer.ConnectionsState) {
	round := 0
	optimisticChoked := PeerConnection{}
	unchoked := []*PeerConnection{}

	for len(peers) > 0 {
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

			return numberOfBlocksI/(int(time.Since(lastTimeI))) > numberOfBlocksJ/(int(time.Since(lastTimeJ)))
		})

		isContained := false
		for _, peer := range uploadRateOrder[0:3] {
			peer.SendUnchoke()
			unchoked = append(unchoked, peer)

			if peer.Peer == optimisticChoked.Peer {
				isContained = true
			}
		}
		if round%3 == 0 {
			if !isContained {
				optimisticChoked.SendUnchoke()
				unchoked = append(unchoked, &optimisticChoked)
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
					unchoked = append(unchoked, forOptimisticUnchoke[n])

					optimisticChoked = *forOptimisticUnchoke[n]
					if forOptimisticUnchoke[n].PeerInterested {
						break
					}
				}
			}
		}
		for _, p := range peers {
			if !contains(unchoked, p) {
				p.SendChoke()
			}
		}
		time.Sleep(10 * time.Second)
	}
}
func contains(peers []*PeerConnection, peer *PeerConnection) bool {
	for _, p := range peers {
		if p == peer {
			return true
		}
	}
	return false
}
