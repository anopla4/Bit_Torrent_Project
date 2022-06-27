package dht

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math/big"
	"net"
	"sort"
	"strconv"
	"time"
)

const (
	//K max number of nodes in a bucket
	K = 20
	// ALPHA max number of parallel on network calls
	ALPHA = 3
	// B number of bits in infohash space
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

//check if node with id nodeID
func (rt *routingTable) nodeInBucket(nodeID []byte, bucket int) int {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()
	for index, n := range rt.table[bucket].bucket {
		if bytes.Equal(n.ID, nodeID) {
			return index
		}
	}
	return -1
}

//return the num nearest nodes to a node with ID nodeID
func (rt *routingTable) getNearestNodes(num int, nodeID []byte) *nodeList {
	nl := &nodeList{}

	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()

	index := rt.getFirstDifBitBucketIndex(nodeID)

	i := index - 1
	j := index + 1
	indexes := []int{index}
	for i >= 0 || j < B {
		if i >= 0 {
			indexes = append(indexes, i)
		}
		if j < B {
			indexes = append(indexes, j)
		}
		i--
		j--
	}

	toAdd := num

	for num > 0 && len(indexes) > 0 {
		index, indexes = indexes[0], indexes[1:]
		for _, v := range rt.table[index].bucket {
			nl.AppendUnique([]*node{v})
			toAdd--
			if toAdd == 0 {
				break
			}
		}
	}
	sort.Sort(nl)
	return nl
}

//remove node with id nodeID from routing table
func (rt *routingTable) RemoveNode(nodeID []byte) {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()

	index := rt.getFirstDifBitBucketIndex(nodeID)

	for i, v := range rt.table[index].bucket {
		if bytes.Equal(v.ID, nodeID) {
			rt.table[index].bucket = append(rt.table[index].bucket[:i], rt.table[index].bucket[i+1:]...)
		}
	}
}

//len of bucket i
func (rt *routingTable) getTotalNodesInBucket(b int) int {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()

	return len(rt.table[b].bucket)
}

//get random id from bucket b space
func (rt *routingTable) getRandomIDFromBucket(b int) []byte {
	// <-rt.lock
	// defer func() { rt.lock <- struct{}{} }()

	id := rt.Self.ID

	equalsBytes := b / 8

	newID := []byte{}
	for i := 0; i < equalsBytes; i++ {
		newID = append(newID, id[i])
	}

	firstDif := b % 8

	var firstDifByte uint8

	for i := 0; i < firstDif; i++ {
		firstDifByte += uint8(id[equalsBytes] & (1 << (7 - i)))
	}

	for i := firstDif; i < 8; i++ {
		aux, _ := rand.Int(rand.Reader, big.NewInt(2))
		firstDifByte += uint8(int(aux.Int64()) << (7 - i))
	}

	for i := equalsBytes; i < 20; i++ {
		newByte, _ := rand.Int(rand.Reader, big.NewInt(256))
		newID = append(newID, uint8(newByte.Int64()))
	}
	return newID
}

//put node at the top(normally last seen node)
func (rt *routingTable) updateBucketOfNode(node []byte) {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()

	indexBucket := rt.getFirstDifBitBucketIndex(node)

	indexNode := rt.nodeInBucket(node, indexBucket)

	if indexNode == -1 {
		panic(errors.New("The node does not exist."))
	} else {
		bucket := rt.table[indexBucket].bucket
		n := bucket[indexNode]
		bucket = append(bucket[:indexNode], bucket[indexNode+1:]...)
		bucket = append(bucket, n)
		rt.table[indexBucket].bucket = bucket
		rt.table[indexBucket].lastChanged = time.Now()
	}
}

func (rt *routingTable) getNodesInBucketClosestThanRT(id []byte) []*node {
	<-rt.lock
	defer func() { rt.lock <- struct{}{} }()

	indexBucket := rt.getFirstDifBitBucketIndex(id)

	nodes := []*node{}
	for _, n := range rt.table[indexBucket].bucket {
		d1 := getDistance(rt.Self.ID, id)
		d2 := getDistance(n.ID, id)
		dif := d1.Sub(d2, d1)
		if dif.Sign() == -1 {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

//TODO:
//1- mark node as seen(put node at top of bucket) //DONE
//2- get_total_nodes_in_bucket //DONE
//3- get_random_id_from_bucket //Done
//Done:
//setSelfAddr
//reset and get last_time for bucket i
//does node exist in bucket
//get nearest nodes
//getBucketIndexFromDifferingBit
// remove node
//DUDA(todavia no se si lo necesito):
//getAllNodesInBucketCloserThan OJOOOO: TENGO Q HACERLO PA ASIGNAR LOS EXPIRATION TIME DONE
