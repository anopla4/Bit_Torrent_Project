package torrent_peer

import (
	"errors"
	"math/rand"
)

func SelectPiece(mode string, pieces []*PieceTask, peers []*Client) (int, error) {
	switch mode {
	case "random first":
		return RandomFirst(pieces), nil
	case "rarest first":
		return RarestFirst(pieces, peers), nil
	default:
		return -1, errors.New("Unkonown pieces selection mode")
	}
}

func RandomFirst(pieces []*PieceTask) int {
	return rand.Intn(len(pieces))
}

func RarestFirst(pieces []*PieceTask, peers []*Client) int {
	p := 0
	min := len(peers) + 1
	for i, piece := range pieces {
		pc := NumberOfPeers(peers, piece.Index)
		if pc < min {
			p = i
			min = pc
		}
	}
	return p
}

func NumberOfPeers(peers []*Client, index int) int {
	count := 0
	for _, p := range peers {
		if p.Bitfield.HasPiece(index) {
			count += 1
		}
	}
	return count
}

func EndGame() {}
