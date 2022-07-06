package torrent_peer

import (
	"Bit_Torrent_Project/Client/client/communication"
	"Bit_Torrent_Project/Client/client/peer"
	"Bit_Torrent_Project/Client/torrent_peer/downloader_client"
	dht "Bit_Torrent_Project/Client/dht/Kademlia"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
	"Bit_Torrent_Project/Client/trackerpb"
)

var MaxBlocks = 5
var MaxBlockSize = 16

type Client struct {
	Connection     net.Conn
	Peer           peer.Peer
	Choked         bool
	PeerChoked     bool
	Interested     bool
	PeerInterested bool
	Bitfield       communication.Bitfield
	InfoHash       [20]byte
	PeerId         string
}

type ConnectionsState struct {
	LastUpload                    map[string]time.Time
	NumberOfBlocksInLast30Seconds map[string]int
	NumberOfUploadPeers           int
}

// Sends UNCHOKE message
func (c *Client) SendUnchoke() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.UNCHOKE}).Serialize())
	if err != nil {
		c.PeerChoked = false
	}
	return err
}

// Sends CHOKE message
func (c *Client) SendChoke() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.CHOKE}).Serialize())
	if err != nil {
		c.PeerChoked = true
	}
	return err
}

// Sends INTERESTED message
func (c *Client) SendInterested() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.INTERESTED}).Serialize())
	if err != nil {
		c.Interested = true
	}
	return err
}

// Sends NOTINTERESTED message
func (c *Client) SendNotInterested() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.NOTINTERESTED}).Serialize())
	if err != nil {
		c.Interested = false
	}
	return err
}

// Sends HAVE message
func (c *Client) SendHave(index int) error {
	msg := communication.BuildHaveMessage(index)
	_, err := c.Connection.Write(msg.Serialize())
	return err
}

// Sends REQUEST message
func (c *Client) SendRequest(index int, begin int, length int) error {
	fmt.Println("Index", index)
	fmt.Println("Begin", begin)
	fmt.Println("Length", length)
	msg := communication.BuildRequestMessage(index, begin, length)
	log.Println("Request message", msg)
	_, err := c.Connection.Write(msg.Serialize())
	return err
}

// Receives BITFIELD message
func RecvBitfield(conn net.Conn) (communication.Bitfield, error) {
	// _ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	// defer conn.SetDeadline(time.Time{}) // Disable the deadline

	msg, err := communication.Deserialize(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		err := fmt.Errorf("Expected bitfield but got %v", msg)
		return nil, err
	}
	if msg.ID != communication.BITFIELD {
		err := fmt.Errorf("Expected bitfield but got ID %d", msg.ID)
		return nil, err
	}

	return msg.Payload, nil
}

// Starts new TCP connection with peer, sends handshake and returns a Client object
func StartConnectionWithPeer(peer peer.Peer, infoHash [20]byte, peerId string, peers []*Client, errChan chan error) (*Client, error) {
	fmt.Println("Starting connection with peer...")
	fmt.Printf("Address: %s\n", peer.IP.String()+":"+strconv.Itoa(int(peer.Port)))
	c, err := downloader_client.StartClientTCP(peer.IP.String()+":"+strconv.Itoa(int(peer.Port)), infoHash, peerId, peer.Id, errChan)
	if err != nil {
		errChan <- err
		return nil, err
	}

	fmt.Println("Receiving bitfield...")
	bf, bfErr := RecvBitfield(c)
	fmt.Println("Bitfield:", bf)

	if bfErr != nil {
		errChan <- bfErr
		return nil, bfErr
	}

	client := &Client{
		Connection:     c,
		Choked:         true,
		Interested:     false,
		PeerChoked:     true,
		PeerInterested: false,
		Bitfield:       bf,
		Peer:           peer,
		InfoHash:       infoHash,
		PeerId:         peerId,
	}
	peers = append(peers, client)

	return client, nil

}

// Downloads piece from c peer
func DownloadPiece(c *Client, task *PieceTask, cs *ConnectionsState) (*PieceResult, error) {
	log.Printf("Starting %v piece download...", task.Index)

	pp := pieceProgress{
		index:  task.Index,
		client: c,
		buf:    make([]byte, task.length),
	}

	// Set deadline for download
	// errSD := c.Connection.SetDeadline(time.Now().Add(30 * time.Second))
	// if errSD != nil {
	// 	return nil, errSD
	// }
	// defer c.Connection.SetDeadline(time.Time{}) // Disable the deadline
	fmt.Println("Downloaded:", pp.downloaded)
	fmt.Println("Task length:", task.length)
	for pp.downloaded < task.length {
		// If unchoked, ask for pieces blocks until reach maxBlocks unfulfilled blocks
		if !c.Choked {
			for pp.pendingBlocks < MaxBlocks && pp.requested < task.length {
				blockSize := MaxBlockSize

				// Last block might be shorter than the typical block
				if task.length-pp.requested < blockSize {
					blockSize = task.length - pp.requested
				}

				log.Printf("Starting %v, %v request sent...", task.Index, pp.requested)
				err := c.SendRequest(task.Index, pp.requested, blockSize)
				log.Println("Request sent")
				if err != nil {
					return nil, err
				}

				pp.pendingBlocks++
				pp.requested += blockSize
			}
		}

		// Read message
		msg, err := communication.Deserialize(c.Connection)
		if err != nil {
			return nil, err
		}

		switch msg.ID {
		case communication.UNCHOKE:
			log.Println("Unchoke received")
			c.Choked = false
		case communication.CHOKE:
			log.Println("Choke received")
			c.Choked = true
		case communication.INTERESTED:
			log.Println("Interested received")
			c.PeerInterested = true
		case communication.NOTINTERESTED:
			log.Println("NotInterested received")
			c.PeerInterested = false
		case communication.HAVE:
			log.Println("Have received")
			index, err := communication.ParseHave(*msg)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			pp.client.Bitfield.SetPiece(index)
		case communication.PIECE:
			log.Println("Piece received")
			n, err := communication.ParsePiece(pp.index, pp.buf, *msg)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			pp.downloaded += n
			pp.pendingBlocks--
			if time.Since(cs.LastUpload[c.Peer.Id]) > time.Duration(30*time.Second) {
				cs.LastUpload[c.Peer.Id] = time.Now()
				cs.NumberOfBlocksInLast30Seconds[c.Peer.Id] = 0
			} else {
				cs.LastUpload[c.Peer.Id] = time.Now()
				cs.NumberOfBlocksInLast30Seconds[c.Peer.Id]++
			}
		}
	}

	return &PieceResult{Index: task.Index, Piece: pp.buf}, nil
}

// Sends handshake, receives bitfield from peer and manages pieces downloads
func (t *Torrent) StartPeerDownload(dp *downloadProgress, peer peer.Peer, tasks []*PieceTask, responses chan *PieceResult,
	tc trackerpb.TrackerClient, IP string, port string, peers []*Client, cs *ConnectionsState,
	errChan chan error, dhtNode *dht.DHT) {
	fmt.Println("Starting peer download...")

	// Dial throw port and ip from peer
	fmt.Println("Info hash:", t.InfoHash)
	client, err := StartConnectionWithPeer(peer, t.InfoHash, t.PeerId, peers, errChan)
	fmt.Println("Connected to peer...")

	if err != nil {
		errChan <- err
		return
	}
	defer func() {
		_ = client.SendNotInterested()
		client.Connection.Close()
	}()

	// Send Unchoke and Interested message
	err = client.SendUnchoke()
	fmt.Println("Unchoking peer:", client.Peer.IP, client.Peer.Port)
	if err != nil {
		errChan <- err
		return
	}

	err = client.SendInterested()
	if err != nil {
		errChan <- err
		return
	}

	p, _ := strconv.Atoi(port)
	left := dp.left
	fmt.Println("Sending started announce to tracker...")
	announce := trackerpb.AnnounceQuery{
		InfoHash: t.InfoHash[:],
		PeerID:   t.PeerId,
		IP:       IP,
		Port:     int32(p),
		Event:    "started",
		Left:     uint64(dp.left),
	}
	ctx := context.Background()
	var done chan struct{}
	var doneKeepAliveToDHT chan struct{}

	if tc != nil {
		tRes, tResErr := tc.Announce(ctx, &announce)
		done = make(chan struct{})

		if tResErr != nil {
			errChan <- err
			log.Printf("Error in tracker's Announce response: %v\n", tResErr)
			return
		}
		interval := tRes.GetInterval()
		fmt.Println("Interval from announce response:", interval)
		go t.KeepAliveToTracker(interval, IP, p, &left, dp, ctx, tc, done)
	}

	mode := "random first"
	for len(tasks) > 0 {
		i, psErr := SelectPiece(mode, tasks, peers)
		task := tasks[i]
		fmt.Println("Task>>>>", task)
		tasks = remove(tasks, i)
		if psErr != nil {
			errChan <- err
			log.Printf("Error while selecting piece: %v\n", err)
			break
		}

		// Ask for piece
		if !client.Bitfield.HasPiece(task.Index) {
			log.Println("Peer does not have piece")
			tasks = append(tasks, task)
			continue
		}

		dp.requested += task.length

		// Download piece
		piece, err := DownloadPiece(client, task, cs)
		log.Printf("Piece downloaded %v\n", piece)

		if err != nil {
			errChan <- err
			log.Println("Error downloading piece")

			tasks = append(tasks, task)
			continue
		}

		log.Println("Checking piece...")
		err = piece.check(task)

		if err != nil {
			errChan <- err
			tasks = append(tasks, task)
			continue
		}
		log.Println("Piece checked")
		if mode == "random first" {
			//AnnouncePeer to DHT

			if dhtNode != nil {
				dhtNode.AnnouncePeer(string(t.InfoHash[:]), p)
				go func(doneKeepAliveToDHT chan struct{}) {
					for {
						select {
						case <-time.After(time.Second * 100):
							if _, ok := <-doneKeepAliveToDHT; ok {
								dhtNode.AnnouncePeer(string(t.InfoHash[:]), p)
							}
						case <-doneKeepAliveToDHT:
							return
						}
					}
				}(doneKeepAliveToDHT)
			}
			mode = "rarest first"
		}
		dp.bitfield.SetPiece(task.Index)
		dp.pendingPieces--
		dp.downloaded += task.length
		dp.requested -= task.length
		left -= uint64(task.length)
		_ = client.SendHave(task.Index)
		if tc != nil {
			log.Println("Sending announce to tracker...")
			announce := trackerpb.AnnounceQuery{
				InfoHash:   t.InfoHash[:],
				PeerID:     t.PeerId,
				IP:         IP,
				Port:       int32(p),
				Event:      "request",
				Left:       left,
				Request:    false,
				Downloaded: uint64(dp.downloaded),
			}
			log.Println("Announcing to tracker...")
			_, tResErr := tc.Announce(ctx, &announce)
			if tResErr != nil {
				errChan <- tResErr
			}
		}

		responses <- piece
	}
	log.Println("Closing Keep Alive channels")
	close(done)
	close(doneKeepAliveToDHT)
}

// Sends Announce to tracker to keep it up to date
func (t *Torrent) KeepAliveToTracker(interval uint32, IP string, port int, left *uint64, dp *downloadProgress, ctx context.Context, tc trackerpb.TrackerClient, done chan struct{}) {
	for {
		//PREGUNTAR A ANA
		select {
		case <-done:
			return
		case <-time.After(time.Second * 10):
			announce := trackerpb.AnnounceQuery{
				InfoHash:   t.InfoHash[:],
				PeerID:     t.PeerId,
				IP:         IP,
				Port:       int32(port),
				Event:      "request",
				Left:       *left,
				Request:    false,
				Downloaded: uint64(dp.downloaded),
			}

			_, tResErr := tc.Announce(ctx, &announce)

			if tResErr != nil {
				log.Printf("Error in tracker keepalive: %v\n", tResErr)
				return
			}
		}
	}
}

func remove(slice []*PieceTask, s int) []*PieceTask {
	return append(slice[:s], slice[s+1:]...)
}
