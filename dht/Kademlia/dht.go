package dht

import (
	"net"
	"strconv"
	"time"
)

type dht struct {
	routingTable *routingTable
	store        KStore
	options      *Options
	// sendedToken  map[string]string
}
type Options struct {
	ID             []byte
	IP             string
	Port           string
	ExpirationTime time.Duration
	RepublishTime  time.Duration
}

func (dht *dht) Update(node *node) {
	dht.routingTable.updateBucketOfNode(node)
}

// FindNode return a list with compactInfo of k nearest node to target
func (dht *dht) FindNode(target []byte) []string {
	nl := dht.routingTable.getNearestNodes(K, target)
	nodesInfo := []string{}
	for _, n := range nl.Nodes {
		nodesInfo = append(nodesInfo, string(newNode(n).CompactInfo()))
	}
	return nodesInfo
}

// GetPeers returns addr of peer containing the infohash in case it is in the storage or k nearest nodes in other case(also true in first casem false otherwise)
func (dht *dht) GetPeers(infohash []byte) ([]string, bool) {
	if values, ok := dht.store.data[string(infohash)]; ok {
		return values, true
	}
	nodes := dht.FindNode(infohash)
	return nodes, false
}

// Store save info in dht storage
func (dht *dht) Store(infohash []byte, ip string, port string) error {
	expTime := time.Now().Add(dht.options.ExpirationTime)
	repTime := time.Now().Add(dht.options.RepublishTime)
	portI, _ := strconv.Atoi(port)
	addr := net.UDPAddr{IP: net.ParseIP(ip), Port: portI}
	err := dht.store.Store(infohash, []byte(addr.String()), expTime, repTime)
	return err
}
