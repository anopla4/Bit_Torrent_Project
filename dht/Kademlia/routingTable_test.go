package dht

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRoutingTable(t *testing.T) {
	id := getIDWithB(uint8(3))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, &NetworkNode{}, rt.Self)
	assert.Equal(t, string(id), string(rt.Self.ID))
	assert.Equal(t, "10.6.100.75", rt.Self.IP.String())
	assert.Equal(t, 3128, rt.Self.Port)
	assert.IsType(t, []*kbucket{}, rt.table)
	assert.Equal(t, B, len(rt.table))
	assert.IsType(t, make(chan struct{}), rt.lock)
	<-rt.lock
	close(rt.lock)
}

func TestFirstDifByte(t *testing.T) {
	id := getIDWithB(uint8(3))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	id2 := make([]byte, 20)
	copy(id2[:5], id[:5])
	id2 = append(id2[:5], getIDWithB(uint8(5))[:18]...)
	index := rt.getFirstDifBitBucketIndex(id2)
	assert.Equal(t, 160-5*8-6, index)
	<-rt.lock
	close(rt.lock)
}

func TestUpdateNodeInBucket(t *testing.T) {
	id := getIDWithB(uint8(3))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	id2 := make([]byte, 20)
	copy(id2[:5], id[:5])
	id2 = append(id2[:5], getIDWithB(uint8(5))[:15]...)
	n := newNode(&NetworkNode{ID: id2, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n)
	assert.Equal(t, 1, rt.GetTotalKnownNodes())
	assert.Equal(t, 0, rt.nodeInBucket(n.ID, 160-5*8-6, true))
	id3 := make([]byte, 20)
	copy(id3[:5], id[:5])
	id3 = append(id3[:5], getIDWithB(uint8(7))[:15]...)
	n1 := newNode(&NetworkNode{ID: id3, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n1)
	assert.Equal(t, 2, rt.GetTotalKnownNodes())
	assert.Equal(t, 1, rt.nodeInBucket(n1.ID, 160-5*8-6, true))
	<-rt.lock
	close(rt.lock)
}

func TestNodesInBucketNearestThanRTID(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	id2 := make([]byte, 20)
	copy(id2[:5], id[:5])
	id2 = append(id2[:5], getIDWithB(uint8(3))[:15]...)
	n := newNode(&NetworkNode{ID: id2, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n)
	id3 := make([]byte, 20)
	copy(id3[:5], id[:5])
	id3 = append(id3[:5], getIDWithB(uint8(7))[:15]...)
	n1 := newNode(&NetworkNode{ID: id3, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n1)
	id4 := make([]byte, 20)
	copy(id4[:5], id[:5])
	id4 = append(id4[:5], getIDWithB(uint8(2))[:15]...)

	nl := rt.getNodesInBucketClosestThanRT(id4)
	assert.Equal(t, 1, len(nl))
	<-rt.lock
	close(rt.lock)
}

func TestRemoveNode(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	id2 := make([]byte, 20)
	copy(id2[:5], id[:5])
	id2 = append(id2[:5], getIDWithB(uint8(3))[:15]...)
	n := newNode(&NetworkNode{ID: id2, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n)
	id3 := make([]byte, 20)
	copy(id3[:5], id[:5])
	id3 = append(id3[:5], getIDWithB(uint8(7))[:15]...)
	n1 := newNode(&NetworkNode{ID: id3, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n1)
	rt.RemoveNode(id2)
	assert.Equal(t, 1, rt.GetTotalKnownNodes())
	assert.Equal(t, -1, rt.nodeInBucket(id2, 160-5*8-6, true))
	<-rt.lock
	close(rt.lock)
}

func TestGetNearestNodes(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	id2 := make([]byte, 20)
	copy(id2[:5], id[:5])
	id2 = append(id2[:5], getIDWithB(uint8(3))[:15]...)
	n := newNode(&NetworkNode{ID: id2, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n)
	id3 := make([]byte, 20)
	copy(id3[:5], id[:5])
	id3 = append(id3[:5], getIDWithB(uint8(7))[:15]...)
	n1 := newNode(&NetworkNode{ID: id3, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n1)
	id4 := make([]byte, 20)
	copy(id4[:5], id[:5])
	id4 = append(id4[:5], getIDWithB(uint8(2))[:15]...)
	n2 := newNode(&NetworkNode{ID: id4, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n2)
	id5 := make([]byte, 20)
	copy(id5[:5], id[:5])
	id5 = append(id5[:5], getIDWithB(uint8(1))[:15]...)
	n3 := newNode(&NetworkNode{ID: id5, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n3)
	id6 := make([]byte, 20)
	copy(id6[:5], id[:5])
	id6 = append(id6[:5], getIDWithB(uint8(8))[:15]...)
	n4 := newNode(&NetworkNode{ID: id6, IP: net.ParseIP("10.6.100.76"), Port: 3128})
	rt.updateBucketOfNode(n4)
	nl := rt.getNearestNodes(6, id3)
	assert.Equal(t, id3, nl.Nodes[0].ID)
	assert.Equal(t, 5, nl.Len())
	<-rt.lock
	close(rt.lock)
}

func TestGetRandomIdForBucket(t *testing.T) {
	id := getIDWithB(uint8(5))
	options := &Options{
		ID:             id,
		IP:             "10.6.100.75",
		Port:           3128,
		ExpirationTime: time.Minute,
		RepublishTime:  time.Minute,
		TimeToDie:      time.Second,
	}
	rt, err := newRoutingTable(options)
	if err != nil {
		t.Fatal(err)
	}
	bucket := 114
	idRandom := rt.getRandomIDFromBucket(bucket)
	assert.Equal(t, 20, len(idRandom))
	index := rt.getFirstDifBitBucketIndex(idRandom)
	assert.Equal(t, 114, index)
	<-rt.lock
	close(rt.lock)
}
