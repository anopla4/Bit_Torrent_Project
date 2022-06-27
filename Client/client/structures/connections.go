package structures

import (
	"encoding/binary"
	"io"
)

type IDMessage uint8

const (
	CHOKE         IDMessage = 0
	UNCHOKE       IDMessage = 1
	INTERESTED    IDMessage = 2
	NOTINTERESTED IDMessage = 3
	HAVE          IDMessage = 4
	BITFIELD      IDMessage = 5
	PIECE         IDMessage = 6
	CANCEL        IDMessage = 7
)

type Message struct {
	ID      IDMessage
	Payload []byte
}

type Bitfield []byte

func (bf *Bitfield) HasPiece(index int) bool { return false }
func (bf *Bitfield) SetPiece(index int)      {}

func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4)
	}
	length := uint32(len(m.Payload) + 1) // +1 for id
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)
	return buf
}

func Deserialize(r io.Reader) (*Message, error) {
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	_, err = io.ReadFull(r, messageBuf)
	if err != nil {
		return nil, err
	}

	m := &Message{
		ID:      IDMessage(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	return m, nil
}

type ConnectionState struct {
	ChokedState     [2]bool
	InterestedState [2]bool
	Peer1           *Peer
	Peer2           *Peer
}

type Handshake struct {
	Pstr     string
	PeerId   [20]byte
	InfoHash [20]byte
}

func (hsh *Handshake) Serialize() {

}
