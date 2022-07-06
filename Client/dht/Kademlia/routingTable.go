package dht

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"net"
	"sort"
	"sync"
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

type RoutingTable struct {
	Self *NetworkNode
	//routing table of nodes(160*k)
	table []*kbucket
	//channel used to black routing table access
	lock sync.Mutex
	//array with info of last time when a bucket was changed
	// lastChanged []time.Time
}

//create new instance of RoutingTable
func newRoutingTable(options *Options) (*RoutingTable, error) {
	rt := &RoutingTable{}
	rt.Self = &NetworkNode{}
	rt.lock = sync.Mutex{}
	if options.ID != nil {
		rt.Self.ID = options.ID
	} else {
		id := make([]byte, 20)
		_, err := rand.Read(id)
		if err != nil {
			return nil, err
		}
		rt.Self.ID = id
	}
	table := []*kbucket{}

	for i := 0; i < B; i++ {
		table = append(table, newKbucket())
	}
	rt.table = table
	for i := 0; i < B; i++ {
		rt.resetLastTimeChanged(i, false)
	}

	if options.IP == "" || options.Port == 0 {
		return nil, errors.New("Port and IP required")
	}
	rt.setIP(options.IP)
	err := rt.setPort(options.Port)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func (rt *RoutingTable) setIP(ip string) {
	rt.Self.IP = net.ParseIP(ip)
}

func (rt *RoutingTable) setPort(port int) error {
	rt.Self.Port = port
	return nil
}
func (rt *RoutingTable) resetLastTimeChanged(bucket int, blocked bool) {
	if !blocked {
		rt.lock.Lock()
		defer rt.lock.Unlock()
	}
	rt.table[bucket].lastChanged = time.Now()
}

func (rt *RoutingTable) getLastTimeChanged(bucket int) time.Time {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	return rt.table[bucket].lastChanged
}

func (rt *RoutingTable) GetTotalKnownNodes() int {
	rt.lock.Lock()
	defer rt.lock.Unlock()

	total := 0
	for _, bucket := range rt.table {
		total += len(bucket.bucket)
	}
	return total
}

//index of correspondent bucket for node(first bite different)
func (rt *RoutingTable) getFirstDifBitBucketIndex(node []byte) int {
	for i := 0; i < len(node); i++ {
		xor := node[i] ^ rt.Self.ID[i]
		for j := 0; j < 8; j++ {
			val := xor & (1 << (7 - j))
			if val > 0 {
				return B - 8*i - j - 1
			}
		}
	}
	return 0
}

//check if node with id nodeID
func (rt *RoutingTable) nodeInBucket(nodeID []byte, bucket int, blocked bool) int {
	if !blocked {
		rt.lock.Lock()
		defer rt.lock.Unlock()
	}
	for index, n := range rt.table[bucket].bucket {
		if bytes.Equal(n.ID, nodeID) {
			return index
		}
	}
	return -1
}

//return the num nearest nodes to a node with ID nodeID
func (rt *RoutingTable) getNearestNodes(num int, nodeID []byte) *nodeList {
	nl := &nodeList{Comparator: nodeID}
	rt.lock.Lock()
	defer rt.lock.Unlock()

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
		j++
	}
	toAdd := num

	for num > 0 && len(indexes) > 0 && toAdd > 0 {
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
func (rt *RoutingTable) RemoveNode(nodeID []byte) {
	rt.lock.Lock()
	defer rt.lock.Unlock()

	index := rt.getFirstDifBitBucketIndex(nodeID)

	for i, v := range rt.table[index].bucket {
		if bytes.Equal(v.ID, nodeID) {
			rt.table[index].bucket = append(rt.table[index].bucket[:i], rt.table[index].bucket[i+1:]...)
		}
	}
}

//len of bucket i
func (rt *RoutingTable) getTotalNodesInBucket(b int) int {
	rt.lock.Lock()
	defer rt.lock.Unlock()

	return len(rt.table[b].bucket)
}

//get random id from bucket b space
func (rt *RoutingTable) getRandomIDFromBucket(b int) []byte {
	// rt.lock.Lock()
	// defer func() { rt.lock <- struct{}{} }()

	id := rt.Self.ID

	equalsBytes := int(math.Ceil((160.0-float64(b))/8.0)) - 1

	newID := []byte{}
	for i := 0; i < equalsBytes; i++ {
		newID = append(newID, id[i])
	}

	firstDif := (160-b)%8 - 1
	var firstDifByte uint8 = 0

	for i := 0; i < firstDif; i++ {
		firstDifByte += uint8(id[equalsBytes] & (1 << (7 - i)))
	}

	if uint8(id[equalsBytes])&1<<(7-firstDif) == 0 {
		firstDifByte += 1 << (7 - firstDif)
	}
	for i := firstDif + 1; i < 8; i++ {
		aux, _ := rand.Int(rand.Reader, big.NewInt(2))
		firstDifByte += uint8(int(aux.Int64()) << (7 - i))
	}
	newID = append(newID, firstDifByte)
	for i := equalsBytes + 1; i < 20; i++ {
		newByte, _ := rand.Int(rand.Reader, big.NewInt(256))
		newID = append(newID, uint8(newByte.Int64()))
	}
	return newID
}

//put node at the top(normally last seen node)
func (rt *RoutingTable) updateBucketOfNode(node *node) {
	rt.lock.Lock()
	defer rt.lock.Unlock()

	indexBucket := rt.getFirstDifBitBucketIndex(node.ID)

	indexNode := rt.nodeInBucket(node.ID, indexBucket, true)

	if indexNode == -1 {
		rt.table[indexBucket].bucket = append(rt.table[indexBucket].bucket, node)
		rt.table[indexBucket].lastChanged = time.Now()
	} else {
		bucket := rt.table[indexBucket].bucket
		n := bucket[indexNode]
		bucket = append(bucket[:indexNode], bucket[indexNode+1:]...)
		bucket = append(bucket, n)
		rt.table[indexBucket].bucket = bucket
		rt.table[indexBucket].lastChanged = time.Now()
	}
}

// return the nodes in bucket correspondent to id closest than routing table id
func (rt *RoutingTable) getNodesInBucketClosestThanRT(id []byte) []*node {
	rt.lock.Lock()
	defer rt.lock.Unlock()

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
