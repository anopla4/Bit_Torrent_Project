package dht

import (
	"crypto/rand"
	"errors"
	"net"
	"strconv"
	"time"
)

const (
	//max number of nodes in a bucket
	K = 20
	//max number of parallel on network calls
	ALPHA = 3
	//number of bits in infohash space
	B = 160
)

type routingTable struct {
	Self *NetworkNode
	//routing table of nodes(160*k)
	table []*kbucket
	//channel used to black routing table access
	lock chan struct{}
	//array with info of last time when a bucket was changed
	lastChanged [B]time.Time
}

//create new instance of routingTable
func newRoutingTable(options *Options) (*routingTable, error) {
	rt := &routingTable{}

	rt.Self = &NetworkNode{}
	rt.lock = make(chan struct{})

	if options.ID == nil {
		rt.Self.ID = options.ID
	} else {
		id := make([]byte, 20)
		_, err := rand.Read(id)
		if err != nil {
			return nil, err
		}
		rt.Self.ID = id
	}
	for i := 0; i <= B; i++ {
		rt.resetLastTimeChanged(i)
	}
	for i := 0; i <= B; i++ {
		rt.table = append(rt.table, newKbucket())
	}
	if options.IP == "" || options.Port == "" {
		return nil, errors.New("Port and IP required")
	}
	rt.setIP(options.IP)
	err := rt.setPort(options.Port)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (rt *routingTable) setIP(ip string) {
	rt.Self.IP = net.ParseIP(ip)
}

func (rt *routingTable) setPort(port string) error {
	p, err := strconv.Atoi(port)
	if err == nil {
		return err
	}
	rt.Self.port = p
	return nil
}
func (rt *routingTable) resetLastTimeChanged(bucket int) {
	<-rt.lock
	rt.table[bucket].lastChanged = time.Now()
	rt.lock <- struct{}{}
}

func (rt *routingTable) getLastTimeChanged(bucket int) time.Time {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()
	return rt.table[bucket].lastChanged
}

func (rt *routingTable) getTotalKnownNodes() int {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()

	total := 0
	for _, bucket := range rt.table {
		total += len(bucket.bucket)
	}
	return total
}

func (rt *routingTable) getNearestNodes(num int, node []byte) *nodeList {
	return &nodeList{}
}

//index of correspondent bucket for node(first bite different)
func (rt *routingTable) getFirstDifBitBucketIndex(node []byte) int {
	for i := 0; i < len(node); i++ {
		xor := node[i] ^ rt.Self.ID[i]
		for j := 0; j < 8; j++ {
			val := xor & (1 << (7 - j))
			if val > 0 {
				return B - 8*i + j - 1
			}
		}
	}
	return 0
}
