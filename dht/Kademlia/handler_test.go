package dht

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	assert.IsType(t, &handlerDHT{}, hd)
}

func TestUpdateNodeOfQueryMessage(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args1 := map[string]string{"id": string(getIDWithB(uint8(3))), "info_hash": "node1//node2", "port": "3128"}
	msg1, _ := newQueryMessage("announce_peer", args1)
	addr := "10.6.100.56:3333"
	err := hd.updateNodeOfMessage(msg1, addr)
	assert.Nil(t, err)
	assert.Equal(t, 1, dht.routingTable.getTotalKnownNodes())
}

func TestUpdateNodeOfResponseMessage(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args1 := map[string]string{"id": string(getIDWithB(uint8(3))), "nodes": "node1//node2"}
	msg1, err := newResponseMessage("find_node", "12223", args1)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}

	addr := "10.6.100.56:3333"
	err = hd.updateNodeOfResponse(msg1, addr)
	assert.Nil(t, err)
	assert.Equal(t, 1, dht.routingTable.getTotalKnownNodes())
}

func TestResponsePing(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args1 := map[string]string{"id": string(getIDWithB(uint8(3)))}
	msg1, err := newQueryMessage("ping", args1)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}

	addr := "10.6.100.56:3333"
	msgR, err := hd.ResponsePing(msg1, addr)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}
	assert.IsType(t, &ResponseMessage{}, msgR)
	assert.Equal(t, msg1.TransactionID, msgR.TransactionID)
	assert.Equal(t, string(id), msgR.Response["id"])
	assert.Equal(t, 1, dht.routingTable.getTotalKnownNodes())
}

func TestResponseFindNode(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args0 := map[string]string{"id": string(getIDWithB(uint8(3)))}
	msg0, err0 := newQueryMessage("ping", args0)
	if err0 != nil {
		fmt.Println(err0.ErrorKRPC())
	}

	addr0 := "10.6.100.56:3333"
	_, err0 = hd.ResponsePing(msg0, addr0)
	if err0 != nil {
		fmt.Println(err0.ErrorKRPC())
	}
	args2 := map[string]string{"id": string(getIDWithB(uint8(4)))}
	msg2, err2 := newQueryMessage("ping", args2)
	if err2 != nil {
		fmt.Println(err2.ErrorKRPC())
	}

	addr2 := "10.6.100.56:3333"
	_, err2 = hd.ResponsePing(msg2, addr2)
	if err2 != nil {
		fmt.Println(err2.ErrorKRPC())
	}
	args1 := map[string]string{"id": string(getIDWithB(uint8(3))), "target": string(id)}
	msg1, err := newQueryMessage("find_node", args1)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}

	addr := "10.6.100.56:3333"
	msgR, err := hd.ResponseFindNode(msg1, addr)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}
	assert.IsType(t, &ResponseMessage{}, msgR)
	assert.Equal(t, msg1.TransactionID, msgR.TransactionID)
	assert.Equal(t, string(id), msgR.Response["id"])
	assert.Equal(t, 2, dht.routingTable.getTotalKnownNodes())
	values, ok := msgR.Response["nodes"]
	valuesR := strings.Split(values, "//")
	assert.True(t, ok)
	assert.Equal(t, 2, len(valuesR))
}

func TestResponseGetPeers(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args0 := map[string]string{"id": string(getIDWithB(uint8(3)))}
	msg0, err0 := newQueryMessage("ping", args0)
	if err0 != nil {
		fmt.Println(err0.ErrorKRPC())
	}

	addr0 := "10.6.100.56:3333"
	_, err0 = hd.ResponsePing(msg0, addr0)
	if err0 != nil {
		fmt.Println(err0.ErrorKRPC())
	}
	args2 := map[string]string{"id": string(getIDWithB(uint8(4)))}
	msg2, err2 := newQueryMessage("ping", args2)
	if err2 != nil {
		fmt.Println(err2.ErrorKRPC())
	}

	addr2 := "10.6.100.56:3333"
	_, err2 = hd.ResponsePing(msg2, addr2)
	if err2 != nil {
		fmt.Println(err2.ErrorKRPC())
	}
	args1 := map[string]string{"id": string(getIDWithB(uint8(3))), "info_hash": string(getIDWithB(3))}
	msg1, err := newQueryMessage("get_peers", args1)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}

	addr := "10.6.100.56:3333"
	msgR, err := hd.ResponseGetPeers(msg1, addr)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
	}
	assert.IsType(t, &ResponseMessage{}, msgR)
	assert.Equal(t, msg1.TransactionID, msgR.TransactionID)
	assert.Equal(t, string(id), msgR.Response["id"])
	assert.Equal(t, 2, dht.routingTable.getTotalKnownNodes())
	values, ok := msgR.Response["nodes"]
	assert.True(t, ok)
	valuesR := strings.Split(values, "//")
	assert.Equal(t, 2, len(valuesR))
	_, ok = msgR.Response["values"]
	assert.False(t, ok)

	id1 := getIDWithB(uint8(3))
	ip := "10.6.100.21"
	port := "3128"
	errS := dht.Store(id1, ip, port)
	assert.Nil(t, errS)
	args4 := map[string]string{"id": string(getIDWithB(uint8(3))), "info_hash": string(getIDWithB(3))}
	msg4, err4 := newQueryMessage("get_peers", args4)
	if err4 != nil {
		fmt.Println(err4.ErrorKRPC())
	}

	addr4 := "10.6.100.56:3333"
	msgR4, err4 := hd.ResponseGetPeers(msg4, addr4)
	if err4 != nil {
		fmt.Println(err4.ErrorKRPC())
	}
	assert.IsType(t, &ResponseMessage{}, msgR4)
	assert.Equal(t, msg4.TransactionID, msgR4.TransactionID)
	assert.Equal(t, string(id), msgR4.Response["id"])
	assert.Equal(t, 2, dht.routingTable.getTotalKnownNodes())
	values1, ok1 := msgR4.Response["values"]
	assert.True(t, ok1)
	valuesR1 := strings.Split(values1, "//")
	assert.Equal(t, 1, len(valuesR1))
	_, ok = msgR4.Response["nodes"]
	assert.False(t, ok)
}

func TestResponseAnnouncePeers(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args0 := map[string]string{"id": string(getIDWithB(uint8(3))), "info_hash": string(getIDWithB(uint8(3))), "port": "3332"}
	msg0, err0 := newQueryMessage("announce_peer", args0)
	if err0 != nil {
		fmt.Println(err0.ErrorKRPC())
		panic(err0)
	}
	addr0 := "10.6.100.56:3333"
	msgR, err0 := hd.ResponseAnnouncePeers(msg0, addr0)
	if err0 != nil {
		fmt.Println(err0.ErrorKRPC())
		panic(err0)
	}

	assert.IsType(t, &ResponseMessage{}, msgR)
	assert.Equal(t, msg0.TransactionID, msgR.TransactionID)
	assert.Equal(t, 1, dht.routingTable.getTotalKnownNodes())
	args4 := map[string]string{"id": string(getIDWithB(uint8(6))), "info_hash": string(getIDWithB(uint8(3)))}
	msg4, err4 := newQueryMessage("get_peers", args4)
	if err4 != nil {
		fmt.Println(err4.ErrorKRPC())
		panic(err4)
	}

	addr4 := "10.6.100.56:3333"
	msgR4, err4 := hd.ResponseGetPeers(msg4, addr4)
	if err4 != nil {
		fmt.Println(err4.ErrorKRPC())
	}
	assert.IsType(t, &ResponseMessage{}, msgR4)
	assert.Equal(t, msg4.TransactionID, msgR4.TransactionID)
	assert.Equal(t, string(id), msgR4.Response["id"])
	assert.Equal(t, 2, dht.routingTable.getTotalKnownNodes())
	values1, ok1 := msgR4.Response["values"]
	assert.True(t, ok1)
	valuesR1 := strings.Split(values1, "//")
	assert.Equal(t, 1, len(valuesR1))
	_, ok := msgR4.Response["nodes"]
	assert.False(t, ok)
}

func TestGetResponse(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Second * 2,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	hd := &handlerDHT{dht: dht}
	args := map[string]string{"id": string(getIDWithB(uint8(6))), "info_hash": string(getIDWithB(uint8(3)))}
	msg, err := newQueryMessage("get_peers", args)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
		panic(err)
	}
	addr := "10.6.100.56:3333"

	expR := &ExpectedResponse{
		idQuery:      getIDWithB(3),
		mess:         msg,
		recieverADDR: addr,
		resChan:      make(chan *ResponseMessage),
		timeToDie:    time.Now().Add(time.Hour),
	}
	dht.expectedResponses[msg.TransactionID] = expR

	args1 := map[string]string{"id": string(getIDWithB(uint8(7))), "values": "10.6.100.71:3332//10.6.7.8:3131"}
	msgR, err := newResponseMessage("get_peers", msg.TransactionID, args1)
	if err != nil {
		fmt.Println(err.ErrorKRPC())
		panic(err)
	}
	addr1 := "10.6.100.56:3333"
	assert.Equal(t, 1, len(dht.expectedResponses))
	hd.getResponse(msgR, addr1)
	respChan := expR.resChan
	resp := <-respChan
	assert.Equal(t, msgR, resp)
}
