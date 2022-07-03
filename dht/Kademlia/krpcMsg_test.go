package dht

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/jackpal/bencode-go"
	"github.com/stretchr/testify/assert"
)

func TestNewGenericError(t *testing.T) {
	err := newGenericError("a generic error has ocurred")
	assert.IsType(t, &GenericError{}, err)
	assert.Equal(t, "e", err.message.TypeOfMessage)
	assert.IsType(t, "3333", err.message.TransactionID)
	assert.Equal(t, 1, len([]byte(err.message.TransactionID)))
	assert.Equal(t, "201", err.ErrorField[0])
	assert.Equal(t, "a generic error has ocurred", err.ErrorKRPC())
}

func TestNewServerError(t *testing.T) {
	err := newServerError("a server error has ocurred")
	assert.IsType(t, &ServerError{}, err)
	assert.Equal(t, "e", err.message.TypeOfMessage)
	assert.IsType(t, "3333", err.message.TransactionID)
	assert.Equal(t, 1, len([]byte(err.message.TransactionID)))
	assert.Equal(t, "202", err.ErrorField[0])
	assert.Equal(t, "a server error has ocurred", err.ErrorKRPC())
}

func TestNewProtocolError(t *testing.T) {
	err := newProtocolError("a protocol error has ocurred")
	assert.IsType(t, &ProtocolError{}, err)
	assert.Equal(t, "e", err.message.TypeOfMessage)
	assert.IsType(t, "3333", err.message.TransactionID)
	assert.Equal(t, 1, len([]byte(err.message.TransactionID)))
	assert.Equal(t, "203", err.ErrorField[0])
	assert.Equal(t, "a protocol error has ocurred", err.ErrorKRPC())
}

func TestNewMethodUnknownError(t *testing.T) {
	err := newMethodUnknownError("method unknown")
	assert.IsType(t, &MethodUnknownError{}, err)
	assert.Equal(t, "e", err.message.TypeOfMessage)
	assert.IsType(t, "3333", err.message.TransactionID)
	assert.Equal(t, 1, len([]byte(err.message.TransactionID)))
	assert.Equal(t, "204", err.ErrorField[0])
	assert.Equal(t, "method unknown", err.ErrorKRPC())
}

func TestNewPingQueryMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk"}
	msg, err := newQueryMessage("ping", args)
	assert.Nil(t, err)
	assert.Equal(t, "q", msg.TypeOfMessage)
	assert.Equal(t, "ping", msg.QueryName)
	assert.IsType(t, "3333", msg.message.TransactionID)
	assert.Equal(t, 1, len([]byte(msg.message.TransactionID)))
	assert.IsType(t, map[string]string{}, msg.Arguments)
	assert.Equal(t, 1, len(msg.Arguments))
	_, ok := msg.Arguments["id"]
	assert.Equal(t, true, ok)
	value := msg.Arguments["id"]
	assert.Equal(t, "kkkk", value)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newQueryMessage("ping", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)
}

func TestNewFindNodeQueryMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk", "target": "3rl67ee"}
	msg, err := newQueryMessage("find_node", args)
	assert.Nil(t, err)
	assert.Equal(t, "q", msg.TypeOfMessage)
	assert.Equal(t, "find_node", msg.QueryName)
	assert.IsType(t, "3333", msg.message.TransactionID)
	assert.Equal(t, 1, len([]byte(msg.message.TransactionID)))
	assert.IsType(t, map[string]string{}, msg.Arguments)
	assert.Equal(t, 2, len(msg.Arguments))
	_, ok := msg.Arguments["id"]
	assert.Equal(t, true, ok)
	value := msg.Arguments["id"]
	assert.Equal(t, "kkkk", value)
	_, ok = msg.Arguments["target"]
	assert.Equal(t, true, ok)
	value = msg.Arguments["target"]
	assert.Equal(t, "3rl67ee", value)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newQueryMessage("find_node", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)

	args3 := map[string]string{"id": "kkkk", "tar": "3456"}
	_, err2 := newQueryMessage("find_node", args3)
	assert.NotNil(t, err2)
	assert.IsType(t, &ProtocolError{}, err2)
}

func TestNewGetPeersQueryMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk", "info_hash": "3rl67ee"}
	msg, err := newQueryMessage("get_peers", args)
	assert.Nil(t, err)
	assert.Equal(t, "q", msg.TypeOfMessage)
	assert.Equal(t, "get_peers", msg.QueryName)
	assert.IsType(t, "3333", msg.message.TransactionID)
	assert.Equal(t, 1, len([]byte(msg.message.TransactionID)))
	assert.IsType(t, map[string]string{}, msg.Arguments)
	assert.Equal(t, 2, len(msg.Arguments))
	_, ok := msg.Arguments["id"]
	assert.Equal(t, true, ok)
	value := msg.Arguments["id"]
	assert.Equal(t, "kkkk", value)
	_, ok = msg.Arguments["info_hash"]
	assert.Equal(t, true, ok)
	value = msg.Arguments["info_hash"]
	assert.Equal(t, "3rl67ee", value)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newQueryMessage("get_peers", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)

	args3 := map[string]string{"id": "kkkk", "infohash": "3456"}
	_, err2 := newQueryMessage("get_peers", args3)
	assert.NotNil(t, err2)
	assert.IsType(t, &ProtocolError{}, err2)
}

func TestNewAnnouncePeerQueryMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk", "info_hash": "3rl67ee", "port": "3128"}
	msg, err := newQueryMessage("announce_peer", args)
	assert.Nil(t, err)
	assert.Equal(t, "q", msg.TypeOfMessage)
	assert.Equal(t, "announce_peer", msg.QueryName)
	assert.IsType(t, "3333", msg.message.TransactionID)
	assert.Equal(t, 1, len([]byte(msg.message.TransactionID)))
	assert.IsType(t, map[string]string{}, msg.Arguments)
	assert.Equal(t, 3, len(msg.Arguments))
	_, ok := msg.Arguments["id"]
	assert.Equal(t, true, ok)
	value := msg.Arguments["id"]
	assert.Equal(t, "kkkk", value)
	_, ok = msg.Arguments["info_hash"]
	assert.Equal(t, true, ok)
	value = msg.Arguments["info_hash"]
	assert.Equal(t, "3rl67ee", value)
	_, ok = msg.Arguments["port"]
	assert.Equal(t, true, ok)
	value = msg.Arguments["port"]
	assert.Equal(t, "3128", value)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newQueryMessage("announce_peer", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)

	args3 := map[string]string{"id": "kkkk", "info_hash": "3456", "por": "3128"}
	_, err2 := newQueryMessage("announce_peer", args3)
	assert.NotNil(t, err2)
	assert.IsType(t, &ProtocolError{}, err2)

	args4 := map[string]string{"id": "kkkk", "info_hash": "3456", "port": "3128"}
	_, err3 := newQueryMessage("announce_per", args4)
	assert.NotNil(t, err3)
	assert.IsType(t, &MethodUnknownError{}, err3)
}

func TestNewPingResponseMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk"}
	msg, err := newResponseMessage("ping", "4444", args)
	assert.Nil(t, err)
	assert.Equal(t, "r", msg.TypeOfMessage)
	assert.Equal(t, "4444", msg.TransactionID)
	assert.IsType(t, map[string]string{}, msg.Response)
	assert.Equal(t, 1, len(msg.Response))
	_, ok := msg.Response["id"]
	assert.Equal(t, true, ok)
	value := msg.Response["id"]
	assert.Equal(t, "kkkk", value)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newResponseMessage("ping", "4444", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)
}

func TestNewFindNodeResponseMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk", "nodes": "3rl67ee//456"}
	msg, err := newResponseMessage("find_node", "3333", args)
	assert.Nil(t, err)
	assert.Equal(t, "r", msg.TypeOfMessage)
	assert.Equal(t, "3333", msg.TransactionID)
	assert.IsType(t, map[string]string{}, msg.Response)
	assert.Equal(t, 2, len(msg.Response))
	_, ok := msg.Response["id"]
	assert.Equal(t, true, ok)
	value := msg.Response["id"]
	assert.Equal(t, "kkkk", value)
	_, ok = msg.Response["nodes"]
	assert.Equal(t, true, ok)
	value1 := msg.Response["nodes"]
	assert.Equal(t, "3rl67ee//456", value1)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newResponseMessage("find_node", "tid", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)

	args3 := map[string]string{"id": "kkkk", "tar": "3456"}
	_, err2 := newResponseMessage("find_node", "tid", args3)
	assert.NotNil(t, err2)
	assert.IsType(t, &ProtocolError{}, err2)
}

func TestNewGetPeersResponseMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk", "nodes": "3rl67ee//456"}
	msg, err := newResponseMessage("get_peers", "3333", args)
	assert.Nil(t, err)
	assert.Equal(t, "r", msg.TypeOfMessage)
	assert.Equal(t, "3333", msg.TransactionID)
	assert.IsType(t, map[string]string{}, msg.Response)
	assert.Equal(t, 2, len(msg.Response))
	_, ok := msg.Response["id"]
	assert.Equal(t, true, ok)
	value := msg.Response["id"]
	assert.Equal(t, "kkkk", value)
	_, ok = msg.Response["nodes"]
	assert.Equal(t, true, ok)
	value1 := msg.Response["nodes"]
	assert.Equal(t, "3rl67ee//456", value1)

	args = map[string]string{"id": "kkkk", "values": "3rl67ee//456"}
	msg, err = newResponseMessage("get_peers", "3333", args)
	assert.Nil(t, err)
	assert.Equal(t, "r", msg.TypeOfMessage)
	assert.Equal(t, "3333", msg.TransactionID)
	assert.IsType(t, map[string]string{}, msg.Response)
	assert.Equal(t, 2, len(msg.Response))
	_, ok = msg.Response["id"]
	assert.Equal(t, true, ok)
	value = msg.Response["id"]
	assert.Equal(t, "kkkk", value)
	_, ok = msg.Response["nodes"]
	assert.Equal(t, false, ok)
	_, ok = msg.Response["values"]
	assert.Equal(t, true, ok)
	value1 = msg.Response["values"]
	assert.Equal(t, "3rl67ee//456", value1)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newResponseMessage("get_peers", "tid", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)

	args3 := map[string]string{"id": "kkkk", "node": "3456"}
	_, err2 := newResponseMessage("get_peers", "tid", args3)
	assert.NotNil(t, err2)
	assert.IsType(t, &ProtocolError{}, err2)
}

func TestNewAnnouncePeersResponseMessage(t *testing.T) {
	args := map[string]string{"id": "kkkk"}
	msg, err := newResponseMessage("announce_peer", "4444", args)
	assert.Nil(t, err)
	assert.Equal(t, "r", msg.TypeOfMessage)
	assert.Equal(t, "4444", msg.TransactionID)
	assert.IsType(t, map[string]string{}, msg.Response)
	assert.Equal(t, 1, len(msg.Response))
	_, ok := msg.Response["id"]
	assert.Equal(t, true, ok)
	value := msg.Response["id"]
	assert.Equal(t, "kkkk", value)

	args2 := map[string]string{"i": "kkkk"}
	_, err1 := newResponseMessage("announce_peer", "4444", args2)
	assert.NotNil(t, err1)
	assert.IsType(t, &ProtocolError{}, err1)
}

func TestBencodeResponseMessages(t *testing.T) {
	args := map[string]string{"id": "kkkk"}
	msg, _ := newResponseMessage("ping", "4444", args)
	bufer := &bytes.Buffer{}
	err := bencode.Marshal(bufer, *msg)
	assert.Nil(t, err)
	msgEncoded := bufer.Bytes()
	var msgDecoded ResponseMessage
	bufer1 := bytes.NewBuffer(msgEncoded)
	err = bencode.Unmarshal(bufer1, &msgDecoded)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Nil(t, err)
	assert.Equal(t, *msg, msgDecoded)

	args1 := map[string]string{"id": "kkkk", "nodes": "node1//node2"}
	msg1, _ := newResponseMessage("get_peers", "4444", args1)
	bufer2 := &bytes.Buffer{}
	err1 := bencode.Marshal(bufer2, *msg1)
	assert.Nil(t, err1)
	msgEncoded1 := bufer2.Bytes()
	var msgDecoded1 ResponseMessage
	bufer3 := bytes.NewBuffer(msgEncoded1)
	err = bencode.Unmarshal(bufer3, &msgDecoded1)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Nil(t, err)
	assert.Equal(t, *msg1, msgDecoded1)
}

func TestBencodeQueryMessages(t *testing.T) {
	args := map[string]string{"id": "kkkk"}
	msg, _ := newQueryMessage("ping", args)
	bufer := &bytes.Buffer{}
	err := bencode.Marshal(bufer, *msg)
	assert.Nil(t, err)
	msgEncoded := bufer.Bytes()
	var msgDecoded QueryMessage
	bufer1 := bytes.NewBuffer(msgEncoded)
	err = bencode.Unmarshal(bufer1, &msgDecoded)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Nil(t, err)
	assert.Equal(t, *msg, msgDecoded)

	args1 := map[string]string{"id": "kkkk", "info_hash": "node1//node2", "port": "3128"}
	msg1, _ := newQueryMessage("announce_peer", args1)
	bufer2 := &bytes.Buffer{}
	err1 := bencode.Marshal(bufer2, *msg1)
	assert.Nil(t, err1)
	msgEncoded1 := bufer2.Bytes()
	var msgDecoded1 QueryMessage
	bufer3 := bytes.NewBuffer(msgEncoded1)
	err = bencode.Unmarshal(bufer3, &msgDecoded1)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Nil(t, err)
	assert.Equal(t, *msg1, msgDecoded1)
}
