package torrent_peer

import (
	"Bit_Torrent_Project/client/client/communication"
	"Bit_Torrent_Project/client/client/peer"
	"Bit_Torrent_Project/client/torrent_peer/downloader_client"
	"net"
	"strconv"
	"time"
)

var MaxBlocks = 5
var MaxBlockSize = 16384

type Client struct {
	Connection net.Conn
	Peer       peer.Peer
	Choked     bool
	Bitfield   communication.Bitfield
	InfoHash   [20]byte
	PeerId     [20]byte
}

func (c *Client) SendUnchoke() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.UNCHOKE}).Serialize())
	return err
}
func (c *Client) SendChoke() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.CHOKE}).Serialize())
	return err
}
func (c *Client) SendInterested() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.INTERESTED}).Serialize())
	return err
}
func (c *Client) SendNotInterested() error {
	_, err := c.Connection.Write((&communication.Message{ID: communication.NOTINTERESTED}).Serialize())
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

func StartConnectionWithPeer(peer peer.Peer, infoHash [20]byte, peerId [20]byte, errChan chan error) (*Client, error) {
	cc := downloader_client.StartClientTCP(string(peer.IP)+":"+strconv.Itoa(int(peer.Port)), errChan)

	defer cc.Close()
	return nil, nil
}

func DownloadPiece(c *Client, task *pieceTask) (*pieceResult, error) {
	pp := pieceProgress{
		index:  task.index,
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

				err := c.SendRequest(task.index, pp.requested, blockSize)
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
		}
	}

	return &pieceResult{index: task.index, piece: pp.buf}, nil
}

func (t *Torrent) StartPeerDownload(peer peer.Peer, tasks chan *pieceTask, responses chan *pieceResult, errChan chan error) {
	// Dial throw port and ip from peer
	client, err := StartConnectionWithPeer(peer, t.InfoHash, t.PeerId, errChan)

	if err != nil {
		errChan <- err
		return
	}
	defer client.Connection.Close()

	// Wait for being unchoked and send interested message
	client.SendUnchoke()
	client.SendInterested()

	for {
		if task, ok := <-tasks; ok {
			// Ask for piece
			if !client.Bitfield.HasPiece(task.index) {
				tasks <- task
				continue
			}

			// Download piece
			piece, err := DownloadPiece(client, task)
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

			client.SendHave(task.index)
			responses <- piece
		} else {
			break
		}
	}

}
