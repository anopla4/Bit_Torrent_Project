package torrent_peer

import (
	"Bit_Torrent_Project/client/client/communication"
	"Bit_Torrent_Project/client/client/peer"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"time"
	"trackerpb"
)

type Torrent struct {
	Peers       []peer.Peer
	PeerId      string
	InfoHash    [20]byte
	PiecesHash  [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type PieceTask struct {
	Index  int
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

type downloadProgress struct {
	bitfield      communication.Bitfield
	downloaded    int
	pendingPieces int
	requested     int
	left          uint64
}

func (pr *pieceResult) check(pt *PieceTask) error {
	hash := sha1.Sum(pr.piece)
	if !bytes.Equal(hash[:], pt.hash[:]) {
		return fmt.Errorf("Index %d failed integrity check", pt.Index)
	}
	return nil
}

func (t *Torrent) pieceLength(index int) int {
	if index*t.PieceLength > t.Length {
		return t.Length - index*t.PieceLength
	}
	return t.PieceLength
}

func (t *Torrent) DownloadFile(tc trackerpb.TrackerClient, IP net.IP, cs *ConnectionsState, port string) ([]byte, error) {
	log.Printf("Starting downloading %v file", t.Name)

	// Channel of pieces downloads tasks to be completed
	// tasks := make(chan *PieceTask, len(t.PiecesHash))
	tasks := []*PieceTask{}

	// Channel of pieces results
	responses := make(chan *pieceResult)

	for i, p := range t.PiecesHash {
		l := t.pieceLength(i)
		// tasks <- &PieceTask{Index: i, hash: p, length: l}
		tasks = append(tasks, &PieceTask{Index: i, hash: p, length: l})
	}

	errChan := make(chan error, 1)
	dp := &downloadProgress{
		bitfield:      make([]byte, 8),
		downloaded:    0,
		pendingPieces: len(t.PiecesHash),
		requested:     0,
		left:          uint64(t.Length),
	}
	peers := []*Client{}
	for _, peer := range t.Peers {
		go t.StartPeerDownload(dp, peer, tasks, responses, tc, IP, port, peers, cs, errChan)
	}

	go Choke(peers, cs)

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
	close(responses)
	close(errChan)
	return buf, nil
}

func Choke(peers []*Client, cs *ConnectionsState) {
	round := 0
	optimisticChoked := Client{}
	unchoked := []*Client{}
	for {
		uploadRateOrder := []*Client{}
		forOptimisticUnchoke := []*Client{}
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
			lastTimeI := cs.LastUpload[uploadRateOrder[i].PeerId]
			numberOfBlocksI := cs.NumberOfBlocksInLast30Seconds[uploadRateOrder[i].PeerId]
			lastTimeJ := cs.LastUpload[uploadRateOrder[j].PeerId]
			numberOfBlocksJ := cs.NumberOfBlocksInLast30Seconds[uploadRateOrder[j].PeerId]

			return numberOfBlocksI/(int(time.Now().Sub(lastTimeI))) > numberOfBlocksJ/(int(time.Now().Sub(lastTimeJ)))
		})

		isContained := false
		for _, peer := range uploadRateOrder[0:3] {
			peer.SendUnchoke()
			unchoked = append(unchoked, peer)
			if peer.PeerId == optimisticChoked.PeerId {
				isContained = true
			}
		}
		if round%3 == 0 {
			if !isContained {
				optimisticChoked.SendUnchoke()
				unchoked = append(unchoked, &optimisticChoked)
			} else {
				for len(forOptimisticUnchoke) > 0 {
					forOptimisticUnchoke = []*Client{}
					for _, p := range peers {
						if p.PeerChoked && p.PeerId != optimisticChoked.PeerId {
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

func contains(peers []*Client, peer *Client) bool {
	for _, p := range peers {
		if p == peer {
			return true
		}
	}
	return false
}
