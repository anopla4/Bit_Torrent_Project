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
	table [][]*node
	//channel used to black routing table access
	lock chan struct{}
	//array with info of last time when a bucket was changed
	lastChanged [B]time.Time
}

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
		rt.resetRefreshTime(i)
	}
	for i := 0; i <= B; i++ {
		rt.table = append(rt.table, []*node{})
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
func (rt *routingTable) resetRefreshTime(bucket int) {
	<-rt.lock
	rt.lastChanged[bucket] = time.Now()
	rt.lock <- struct{}{}
}
