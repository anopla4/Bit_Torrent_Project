package piece_selection

import (
	"Bit_Torrent_Project/client/torrent_peer"
	"errors"
	"math/rand"
)

func SelectPiece(mode string, pieces []torrent_peer.PieceTask, peers []*torrent_peer.Client) (int, error) {
	switch mode {
	case "random first":
		return RandomFirst(pieces), nil
	case "rarest first":
		return RarestFirst(pieces, peers), nil
	default:
		return -1, errors.New("Unkonown pieces selection mode")
	}
}

func RandomFirst(pieces []torrent_peer.PieceTask) int {
	return rand.Intn(len(pieces))
}

func RarestFirst(pieces []torrent_peer.PieceTask, peers []*torrent_peer.Client) int {
	p := 0
	min := len(peers) + 1
	for _, piece := range pieces {
		pc := NumberOfPeers(peers, piece.Index)
		if pc < min {
			p = piece.Index
			min = pc
		}
	}
	return p
}

func NumberOfPeers(peers []*torrent_peer.Client, index int) int {
	count := 0
	for _, p := range peers {
		if p.Bitfield.HasPiece(index) {
			count += 1
		}
	}
	return count
}

func EndGame() {}
