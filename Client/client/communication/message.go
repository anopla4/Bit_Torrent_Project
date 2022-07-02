package communication

import (
	"encoding/binary"
	"fmt"
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
	REQUEST       IDMessage = 8
)

type Message struct {
	ID      IDMessage
	Payload []byte
}

type Bitfield []byte

func (bf Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	return bf[byteIndex]>>(7-offset)&1 != 0
}
func (bf Bitfield) SetPiece(index int) {
	byteIndex := index / 8
	offset := index % 8
	// silently discard invalid bounded index
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}

	bf[byteIndex] |= 1 << (7 - offset)
}

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

func BuildHaveMessage(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &Message{ID: HAVE, Payload: payload}
}
func BuildRequestMessage(index int, begin int, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))
	return &Message{ID: REQUEST, Payload: payload}
}

func ParseHave(haveMsg Message) (int, error) {
	if haveMsg.ID != HAVE {
		return 0, fmt.Errorf("Expected HAVE (ID %d), got ID %d", HAVE, haveMsg.ID)
	}
	if len(haveMsg.Payload) != 4 {
		return 0, fmt.Errorf("Expected payload length 4, got length %d", len(haveMsg.Payload))
	}
	index := int(binary.BigEndian.Uint32(haveMsg.Payload))
	return index, nil
}
func ParsePiece(index int, buf []byte, pieceMsg Message) (int, error) {
	if pieceMsg.ID != PIECE {
		return 0, fmt.Errorf("Expected HAVE (ID %d), got ID %d", HAVE, pieceMsg.ID)
	}
	payload := pieceMsg.Payload
	if len(payload) < 8 {
		return 0, fmt.Errorf("Payload too short. %d < 8", len(payload))
	}
	parsedIndex := int(binary.BigEndian.Uint32(payload[0:4]))
	if parsedIndex != index {
		return 0, fmt.Errorf("Expected index %d, got %d", index, parsedIndex)
	}
	parsedBegin := int(binary.BigEndian.Uint32(payload[4:8]))
	if parsedBegin >= len(buf) {
		return 0, fmt.Errorf("Begin offset too high. %d >= %d", parsedBegin, len(buf))
	}
	data := payload[8:]
	if parsedBegin+len(data) > len(buf) {
		return 0, fmt.Errorf("Data too long [%d] for offset %d with length %d", len(data), parsedBegin, len(buf))
	}
	copy(buf[parsedBegin:], data)
	return len(data), nil
}
