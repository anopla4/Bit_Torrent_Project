package dht

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDHT(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	assert.IsType(t, &DHT{}, dht)
	assert.IsType(t, &routingTable{}, dht.routingTable)
	assert.Equal(t, options, dht.options)
	assert.IsType(t, &KStore{}, dht.store)
	assert.IsType(t, map[string]*ExpectedResponse{}, dht.expectedResponses)
}

func TestStoreLocal(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	id1 := getIDWithB(uint8(3))
	ip := "10.6.100.21"
	port := "3128"
	err := dht.Store(id1, ip, port)
	if err != nil {
		fmt.Println(err.Error())
	}
	addr := net.UDPAddr{IP: net.ParseIP(ip), Port: 3128}
	addrS := addr.String()
	assert.Equal(t, true, dht.store.checkKeyValuePair(string(id1), addrS))
}

func TestGetPeerWhenInfohashLocal(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)
	id1 := getIDWithB(uint8(3))
	ip := "10.6.100.21"
	port := "3128"
	err := dht.Store(id1, ip, port)
	if err != nil {
		fmt.Println(err.Error())
	}
	addr := net.UDPAddr{IP: net.ParseIP(ip), Port: 3128}
	addrS := addr.String()

	value, ok := dht.GetPeers(id1)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(value))
	assert.Equal(t, addrS, value[0])
}

func TestCheckExpirationTime(t *testing.T) {
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
	id1 := getIDWithB(uint8(3))
	ip := "10.6.100.21"
	port := "3128"
	err := dht.Store(id1, ip, port)
	if err != nil {
		fmt.Println(err.Error())
	}
	addr := net.UDPAddr{IP: net.ParseIP(ip), Port: 3128}
	addrS := addr.String()

	value, ok := dht.GetPeers(id1)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(value))
	assert.Equal(t, addrS, value[0])
	go dht.checkForExpirationTime()

	time.Sleep(time.Second * 3)
	_, ok = dht.GetPeers(id1)
	assert.Equal(t, false, ok)
}

func TestUpdateNode(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	dht := newDHT(options)

	id2 := make([]byte, 20)
	copy(id2[:5], id[:5])
	id2 = append(id2[:5], getIDWithB(uint8(3))[:15]...)
	n := newNode(&NetworkNode{ID: id2, IP: net.ParseIP("10.6.100.76"), port: 3128})
	dht.Update(n)
	assert.Equal(t, 1, dht.routingTable.getTotalKnownNodes())
	infos := dht.FindNode(id2)

	assert.Equal(t, 1, len(infos))
	assert.Equal(t, string(n.CompactInfo()), infos[0])

}
