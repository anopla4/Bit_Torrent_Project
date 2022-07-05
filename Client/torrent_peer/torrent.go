package torrent_peer

import (
	"Bit_Torrent_Project/client/client/communication"
	"Bit_Torrent_Project/client/client/peer"
	dht "Bit_Torrent_Project/dht/Kademlia"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"
	"trackerpb"

	"github.com/cheggaaa/pb"
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

type PieceResult struct {
	Index int
	Piece []byte
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

func (pr *PieceResult) check(pt *PieceTask) error {
	hash := sha1.Sum(pr.Piece)
	log.Println("Piece in ckecking>>>>:", pr.Piece)
	log.Println("Hash:", hash)
	log.Println("Piece task hash:", pt.hash)
	if !bytes.Equal(hash[:], pt.hash[:]) {
		return fmt.Errorf("Index %d failed integrity check", pt.Index)
	}
	return nil
}

func (t *Torrent) pieceLength(index int) int {
	if (index+1)*t.PieceLength > t.Length {
		return t.Length - index*t.PieceLength
	}
	return t.PieceLength
}

// Downloads file from peers in torrent t
func (t *Torrent) DownloadFile(tc trackerpb.TrackerClient, IP string, cs *ConnectionsState, port string, dhtNode *dht.DHT) ([]*PieceResult, [20]byte, string, []byte, error) {
	log.Printf("Starting downloading %v file\n", t.Name)
	wg := sync.WaitGroup{}
	// Channel of pieces downloads tasks to be completed
	tasks := []*PieceTask{}

	// Channel of pieces results
	responses := make(chan *PieceResult, 8)

	for i, p := range t.PiecesHash {
		l := t.pieceLength(i)
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
	fmt.Println("Peers: ", t.Peers)
	for _, peer_ := range t.Peers {
		fmt.Printf("Peer: %v\n", peer_)
		wg.Add(1)
		go func(p peer.Peer) {
			t.StartPeerDownload(dp, p, tasks, responses, tc, IP, port, peers, cs, errChan)
			log.Println("Exiting peer goroutine>>>>>")
			wg.Done()
		}(peer_)
	}

	wg.Add(1)
	doneChoke := make(chan struct{})
	go func() {
		Choke(peers, cs, doneChoke)
		wg.Done()
	}()

	// Catch errors in downloads
	wg.Add(1)
	go func(errChan chan error) {
		for {
			if connErr, ok := <-errChan; ok {
				log.Println(connErr)
			} else {
				log.Println("Exiting error goroutine>>>>>")
				break
			}
		}
		log.Println("Errors done")
		wg.Done()
	}(errChan)

	// Collect results from peers
	buf := make([]byte, t.Length)

	// start new bar
	bar := pb.New(t.Length).SetUnits(pb.U_BYTES)
	bar.Start()
	pieces := []*PieceResult{}
	wg.Add(1)
	go func() {
		donePieces := 0
		for donePieces < len(t.PiecesHash) {
			if res, ok := <-responses; ok {
				log.Println("Response:", res)
				log.Println("Copying to buffer")
				copy(buf[(res.Index*t.PieceLength):], res.Piece)

				for i := 0; i < len(res.Piece); i++ {
					bar.Increment()
				}

				donePieces++
				fmt.Println("Done pieces>>>>", donePieces)
				fmt.Println("Total pieces>>>>", len(t.PiecesHash))
				pieces = append(pieces, res)
				percent := float64(donePieces) / float64(len(t.PiecesHash)) * 100
				log.Printf("(%0.2f%%) Downloaded piece #%d\n", percent, res.Index)
			} else {
				log.Println("Exiting responses goroutine>>>>>")
				close(errChan)
				wg.Done()
				return
			}
		}
		log.Println("Responses done")
		close(errChan)
		close(responses)
		wg.Done()
	}()
	close(doneChoke)
	wg.Wait()
	log.Println("Exiting from DownloadFile")
	bar.Finish()
	return pieces, t.InfoHash, string(dp.bitfield), buf, nil
}

// Chokes and unchokes peers for load balance
func Choke(peers []*Client, cs *ConnectionsState, done chan struct{}) {
	round := 0
	optimisticChoked := Client{}
	unchoked := []*Client{}
	for {
		if _, ok := <-done; ok {
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
			if round%3 == 0 && len(forOptimisticUnchoke) > 0 {
				n := rand.Intn(len(forOptimisticUnchoke))
				optimisticChoked = *forOptimisticUnchoke[n]
			}
			sort.Slice(uploadRateOrder, func(i, j int) bool {
				lastTimeI := cs.LastUpload[uploadRateOrder[i].PeerId]
				numberOfBlocksI := cs.NumberOfBlocksInLast30Seconds[uploadRateOrder[i].PeerId]
				lastTimeJ := cs.LastUpload[uploadRateOrder[j].PeerId]
				numberOfBlocksJ := cs.NumberOfBlocksInLast30Seconds[uploadRateOrder[j].PeerId]

				return numberOfBlocksI/(int(time.Since(lastTimeI))) > numberOfBlocksJ/(int(time.Since(lastTimeJ)))
			})

			isContained := false
			toUnchoke := uploadRateOrder
			if len(uploadRateOrder) >= 3 {
				toUnchoke = uploadRateOrder[0:3]
			}
			for _, peer := range toUnchoke {
				_ = peer.SendUnchoke()
				unchoked = append(unchoked, peer)
				if peer.PeerId == optimisticChoked.PeerId {
					isContained = true
				}
			}
			if round%3 == 0 && len(forOptimisticUnchoke) > 0 {
				if !isContained {
					_ = optimisticChoked.SendUnchoke()
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
						_ = forOptimisticUnchoke[n].SendUnchoke()
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
					_ = p.SendChoke()
				}
			}
		} else {
			log.Println("Exiting choke algorithm...")
			break
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
