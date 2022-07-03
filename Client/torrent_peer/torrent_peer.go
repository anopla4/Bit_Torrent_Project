package torrent_peer

import (
	"Bit_Torrent_Project/client/client/communication"
	"Bit_Torrent_Project/client/client/peer"
	"Bit_Torrent_Project/client/torrent_peer/downloader_client"
	"Bit_Torrent_Project/client/torrent_peer/uploader_client"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
	"trackerpb"
)

var MaxBlocks = 5
var MaxBlockSize = 16384

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

func (c *Client) SendUnchoke() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.UNCHOKE}).Serialize())
	if err != nil {
		c.PeerChoked = false
	}
	return err
}
func (c *Client) SendChoke() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.CHOKE}).Serialize())
	if err != nil {
		c.PeerChoked = true
	}
	return err
}
func (c *Client) SendInterested() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.INTERESTED}).Serialize())
	if err != nil {
		c.Interested = true
	}
	return err
}
func (c *Client) SendNotInterested() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.NOTINTERESTED}).Serialize())
	if err != nil {
		c.Interested = false
	}
	return err
}
func (c *Client) SendHave(index int) error {
	msg := communication.BuildHaveMessage(index)
	_, err := c.Connection.Write(msg.Serialize())
	return err
}
func (c *Client) SendRequest(index int, begin int, length int) error {
	msg := communication.BuildRequestMessage(index, begin, length)
	_, err := c.Connection.Write(msg.Serialize())
	return err
}

func RecvBitfield(conn net.Conn) (communication.Bitfield, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

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

func StartConnectionWithPeer(peer peer.Peer, infoHash [20]byte, peerId string, peers []*Client, errChan chan error) (*Client, error) {
	c, err := downloader_client.StartClientTCP(string(peer.IP)+":"+strconv.Itoa(int(peer.Port)), infoHash, peerId, errChan)
	if err != nil {
		errChan <- err
		return nil, err
	}

	defer c.Close()

	bf, bfErr := RecvBitfield(c)

	if bfErr != nil {
		errChan <- bfErr
		return nil, bfErr
	}

	client := &Client{
		Connection: c,
		Choked:     true,
		Interested: false,
		Bitfield:   bf,
		Peer:       peer,
		InfoHash:   infoHash,
		PeerId:     peerId,
	}

	peers = append(peers, client)

	return client, nil

}

func DownloadPiece(c *Client, task *PieceTask, cs *ConnectionsState) (*pieceResult, error) {
	pp := pieceProgress{
		index:  task.Index,
		client: c,
		buf:    make([]byte, task.length),
	}

	// Set deadline for download
	errSD := c.Connection.SetDeadline(time.Now().Add(30 * time.Second))
	if errSD != nil {
		return nil, errSD
	}
	defer c.Connection.SetDeadline(time.Time{}) // Disable the deadline

	for pp.downloaded < task.length {
		// If unchoked, ask for pieces blocks until reach maxBlocks unfulfilled blocks
		if !c.Choked {
			for pp.pendingBlocks < MaxBlocks && pp.requested < task.length {
				blockSize := MaxBlockSize
				// Last block might be shorter than the typical block
				if task.length-pp.requested < blockSize {
					blockSize = task.length - pp.requested
				}

				err := c.SendRequest(task.Index, pp.requested, blockSize)
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
			c.Choked = false
		case communication.CHOKE:
			c.Choked = true
		case communication.INTERESTED:
			c.PeerInterested = true
		case communication.NOTINTERESTED:
			c.PeerInterested = false
		case communication.HAVE:
			index, err := communication.ParseHave(*msg)
			if err != nil {
				return nil, err
			}
			pp.client.Bitfield.SetPiece(index)
		case communication.PIECE:
			n, err := communication.ParsePiece(pp.index, pp.buf, *msg)
			if err != nil {
				return nil, err
			}
			pp.downloaded += n
			pp.pendingBlocks--
			if time.Now().Sub(cs.LastUpload[c.Peer.Id]) > time.Duration(30*time.Second) {
				cs.LastUpload[c.Peer.Id] = time.Now()
				cs.NumberOfBlocksInLast30Seconds[c.Peer.Id] = 0
			} else {
				cs.LastUpload[c.Peer.Id] = time.Now()
				cs.NumberOfBlocksInLast30Seconds[c.Peer.Id]++
			}
		}
	}

	return &pieceResult{index: task.Index, piece: pp.buf}, nil
}

func (t *Torrent) StartPeerDownload(dp *downloadProgress, peer peer.Peer, tasks chan *PieceTask, responses chan *pieceResult,
	tc trackerpb.TrackerClient, IP net.IP, port string, peers []*Client, servers []*uploader_client.Server, cs *ConnectionsState,
	errChan chan error) {
	// Dial throw port and ip from peer
	client, err := StartConnectionWithPeer(peer, t.InfoHash, t.PeerId, peers, errChan)

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
	announce := trackerpb.AnnounceQuery{
		InfoHash: t.InfoHash[:],
		PeerID:   t.PeerId,
		IP:       IP.String(),
		Port:     int32(p),
		Event:    "started",
		Left:     uint64(dp.left),
	}
	ctx := context.Background()

	tRes, tResErr := tc.Announce(ctx, &announce)

	if tResErr != nil {
		errChan <- err
		log.Printf("Error in tracker's Announce response: %v\n", tResErr)
		return
	}
	interval := tRes.GetInterval()

	go t.KeepAliveToTracker(interval, IP, p, &left, dp, ctx, tc)

	for {
		if task, ok := <-tasks; ok {
			// Ask for piece
			if !client.Bitfield.HasPiece(task.Index) {
				tasks <- task
				continue
			}

			dp.requested += task.length

			// Download piece
			piece, err := DownloadPiece(client, task, cs)
			if err != nil {
				errChan <- err
				tasks <- task
				return
			}

			err = piece.check(task)
			if err != nil {
				errChan <- err
				tasks <- task
				continue
			}

			dp.bitfield.SetPiece(task.Index)
			dp.pendingPieces--
			dp.downloaded += task.length
			dp.requested -= task.length
			left -= uint64(task.length)
			_ = client.SendHave(task.Index)

			announce := trackerpb.AnnounceQuery{
				InfoHash:   t.InfoHash[:],
				PeerID:     t.PeerId,
				IP:         IP.String(),
				Port:       int32(p),
				Event:      "request",
				Left:       left,
				Request:    false,
				Downloaded: uint64(dp.downloaded),
			}

			tRes, tResErr = tc.Announce(ctx, &announce)
			responses <- piece
		} else {
			break
		}
	}

}
func (t *Torrent) KeepAliveToTracker(interval uint32, IP net.IP, port int, left *uint64, dp *downloadProgress, ctx context.Context, tc trackerpb.TrackerClient) {

	for {
		announce := trackerpb.AnnounceQuery{
			InfoHash:   t.InfoHash[:],
			PeerID:     t.PeerId,
			IP:         IP.String(),
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

		time.Sleep(time.Duration(interval) * 1000000000)
	}

}
