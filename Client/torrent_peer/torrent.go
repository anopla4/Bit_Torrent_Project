package torrent_peer

import (
	"Bit_Torrent_Project/client/client/peer"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
)

type Torrent struct {
	Peers       []peer.Peer
	PeerId      [20]byte
	InfoHash    [20]byte
	PiecesHash  [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type pieceTask struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	piece []byte
}

type pieceProgress struct {
	index         int
	client        *Client
	buf           []byte
	downloaded    int
	pendingBlocks int
	requested     int
}

func (pr *pieceResult) check(pt *pieceTask) error {
	hash := sha1.Sum(pr.piece)
	if !bytes.Equal(hash[:], pt.hash[:]) {
		return fmt.Errorf("Index %d failed integrity check", pt.index)
	}
	return nil
}

func (t *Torrent) pieceLength(index int) int {
	if index*t.PieceLength > t.Length {
		return t.Length - index*t.PieceLength
	}
	return t.PieceLength
}

func (t *Torrent) DownloadFile() ([]byte, error) {
	log.Printf("Starting downloading %v file", t.Name)

	// Channel of pieces downloads tasks to be completed
	tasks := make(chan *pieceTask, len(t.PiecesHash))

	// Channel of pieces results
	responses := make(chan *pieceResult)

	for i, p := range t.PiecesHash {
		l := t.pieceLength(i)
		tasks <- &pieceTask{index: i, hash: p, length: l}
	}

	errChan := make(chan error, 1)
	for _, peer := range t.Peers {
		go t.StartPeerDownload(peer, tasks, responses, errChan)
	}

	// Catch errors in downloads
	go func(errChan chan error) {
		for {
			if connErr, ok := <-errChan; ok {
				log.Println(connErr)
				break
			}
		}
	}(errChan)

	// Collect results from peers
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PiecesHash) {
		if res, ok := <-responses; ok {
			copy(buf[(res.index*t.PieceLength):], res.piece)
			donePieces++

			percent := float64(donePieces) / float64(len(t.PiecesHash)) * 100
			log.Printf("(%0.2f%%) Downloaded piece #%d\n", percent, res.index)
		} else {
			break
		}
	}
	close(tasks)
	close(responses)
	close(errChan)
	return buf, nil
}

func Choke() {}
