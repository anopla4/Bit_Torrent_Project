package communication

import (
	"errors"
	"io"
)

type Handshake struct {
	Pstr     string
	PeerId   string
	InfoHash [20]byte
}

func (hsh *Handshake) Serialize() []byte {
	buf := make([]byte, len(hsh.Pstr)+49)
	buf[0] = byte(len(hsh.Pstr))
	curr := 1
	curr += copy(buf[curr:], hsh.Pstr)
	curr += copy(buf[curr:], make([]byte, 8)) // 8 reserved bytes
	curr += copy(buf[curr:], hsh.InfoHash[:])
	_ = copy(buf[curr:], hsh.PeerId[:])
	return buf
}

func DeserializeHandshake(r io.Reader) (*Handshake, error) {
	pstrLength := make([]byte, 1)
	_, err := io.ReadFull(r, pstrLength)

	if err != nil {
		return nil, err
	}

	if 19 != byte(pstrLength[0]) {
		return nil, errors.New("Protocol string does not have the right size.")
	}

	pstr := make([]byte, 19)
	_, pstrErr := io.ReadFull(r, pstr)

	if pstrErr != nil {
		return nil, pstrErr
	}

	if "BitTorrent protocol" != string(pstr) {
		return nil, errors.New("Protocol string is not BitTorrent protocol.")
	}

	reservedBytes := make([]byte, 8)
	_, reservedBytesErr := io.ReadFull(r, reservedBytes)

	if reservedBytesErr != nil {
		return nil, reservedBytesErr
	}

	infoHashSl := make([]byte, 20)
	_, infoHashErr := io.ReadFull(r, infoHashSl)

	if infoHashErr != nil {
		return nil, infoHashErr
	}

	peerIdSl := make([]byte, 20)
	_, peerIdErr := io.ReadFull(r, peerIdSl)

	if peerIdErr != nil {
		return nil, peerIdErr
	}

	infoHash := [20]byte{}
	_ = copy(infoHash[:], infoHashSl)

	peerId := string(peerIdSl)

	return &Handshake{Pstr: string(pstr), InfoHash: infoHash, PeerId: peerId}, nil
}
