package dht

import (
	"crypto/rand"
	"math/big"
	"net"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNetworkNode(t *testing.T) {
	node := NewNetworkNode("10.6.100.25", "3128")
	assert.IsType(t, []byte{}, node.ID, "node.ID is not []byte")
	ipTest := net.IP{}
	assert.IsType(t, ipTest, node.IP, "node.ip is not net.IP type")
	assert.Equal(t, "10.6.100.25", node.IP.String(), "node ip is not the same passed as argument")
	assert.Equal(t, 3128, node.Port, "node.port is not the same passed as argument")
}

func TestNewNode(t *testing.T) {
	ntNode := NewNetworkNode("10.6.100.25", "3128")
	ntNode.ID = make([]byte, 20)
	_, err := rand.Read(ntNode.ID)
	if err != nil {
		t.Fatal(err)
	}
	node := newNode(ntNode)
	assert.Equal(t, ntNode.ID, node.ID, "incorrect id")
	assert.Equal(t, ntNode.IP.String(), ntNode.IP.String(), "incorrect ip addres")
	assert.Equal(t, ntNode.Port, ntNode.Port, "incorrect port")
}

func TestCompactIN(t *testing.T) {
	ntNode := NewNetworkNode("10.6.100.25", "3128")
	ntNode.ID = make([]byte, 20)
	_, err := rand.Read(ntNode.ID)
	if err != nil {
		t.Fatal(err)
	}
	node := newNode(ntNode)
	assert.Equal(t, node.Port, ntNode.Port)
	compactInfo := node.CompactInfo()
	assert.IsType(t, []byte{}, compactInfo, "contact info type has to be []byte")
	newnode := NewNodeFromCompactInfo(compactInfo)
	assert.Equal(t, ntNode.ID, newnode.ID, "incorrect id from compact info")
	assert.Equal(t, ntNode.IP.String(), newnode.IP.String(), "incorrect ip addres from compact info")
	assert.Equal(t, ntNode.Port, newnode.Port, "incorrect port from compact info")
}

func TestEqualsNodes(t *testing.T) {
	ntNode := NewNetworkNode("10.6.100.25", "3128")
	ntNode.ID = make([]byte, 20)
	_, err := rand.Read(ntNode.ID)
	if err != nil {
		t.Fatal(err)
	}
	node := newNode(ntNode)
	compactInfo := node.CompactInfo()
	newnode := NewNodeFromCompactInfo(compactInfo)
	equals := equalsNodes(ntNode, newnode.NetworkNode, false)
	assert.Equal(t, true, equals, "problem with equalsNodes method")
}

func TestGetDistance(t *testing.T) {
	id := getIDWithB(uint8(0))
	id1 := getIDWithB(uint8(3))
	dist := new(big.Int)
	dist.SetBytes(id1)
	assert.Equal(t, dist, getDistance(id, id1))
}
func TestNodeList(t *testing.T) {
	id := getIDWithB(uint8(0))
	nl := &nodeList{Comparator: id}
	id1 := getIDWithB(uint8(3))
	ntNode := NewNetworkNode("10.6.100.25", "3128")
	ntNode.ID = id1
	node0 := newNode(ntNode)
	id2 := getIDWithB(uint8(5))
	ntNode1 := NewNetworkNode("10.6.100.27", "3128")
	ntNode1.ID = id2
	node1 := newNode(ntNode1)
	nl.AppendUnique([]*node{node0, node1})
	assert.Equal(t, 2, nl.Len())
	sort.Sort(nl)
	assert.Equal(t, true, equalsNodes(node0.NetworkNode, nl.Nodes[0], false))
	assert.Equal(t, true, equalsNodes(node1.NetworkNode, nl.Nodes[1], false))
	nl.Swap(0, 1)
	assert.Equal(t, true, equalsNodes(node1.NetworkNode, nl.Nodes[0], false))
	assert.Equal(t, true, equalsNodes(node0.NetworkNode, nl.Nodes[1], false))
	nl.RemoveNode(node0.NetworkNode)
	assert.Equal(t, 1, nl.Len())
	assert.Equal(t, true, equalsNodes(node1.NetworkNode, nl.Nodes[0], false))
}

func getIDWithB(b byte) []byte {
	return []byte{b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b}
}
