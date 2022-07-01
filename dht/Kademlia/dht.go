package dht

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/jackpal/bencode-go"
)

type dht struct {
	routingTable      *routingTable
	store             KStore
	options           *Options
	conn              *net.PacketConn
	expectedResponses map[string]*ExpectedResponse
	// sendedToken  map[string]string
}
type Options struct {
	ID             []byte
	IP             string
	Port           int
	ExpirationTime time.Duration
	RepublishTime  time.Duration
	TimeToDie      time.Duration
}

type ExpectedResponse struct {
	idQuery      []byte
	mess         *QueryMessage
	recieverADDR string
	resChan      chan map[string]interface{}
	timeToDie    time.Time
}

func (dht *dht) Update(node *node) {
	dht.routingTable.updateBucketOfNode(node)
}

func (dht *dht) GetID() []byte {
	return dht.options.ID
}
func (dht *dht) Ping() {

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

func (dht *dht) GetExpextedResponse(transactionID []byte) (*ExpectedResponse, bool) {
	if expResponse, ok := dht.expectedResponses[string(transactionID)]; ok {
		return expResponse, ok
	}
	return nil, false
}

func (dht *dht) RemoveExpectedResponse(transactionID []byte) {
	delete(dht.expectedResponses, string(transactionID))
}

func (dht *dht) RunServer(exit chan string) {
	hd := handlerDHT{
		dht: dht,
	}

	server := newServer(dht.options.IP, dht.options.Port, hd)
	conn := make(chan *net.PacketConn)
	go server.RunServer(exit, conn)
	dht.conn = <-conn
}

// SendMessage send a query message to an addres throug dht.conn
func (dht *dht) SendMessage(query *QueryMessage, expectingResponse bool, addr string) (*ExpectedResponse, error) {
	buffer := bytes.NewBuffer(make([]byte, 1024))
	err := bencode.Marshal(buffer, query)
	if err != nil {
		return nil, err
	}
	conn := *(dht.conn)
	addrN, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	_, err = conn.WriteTo(buffer.Bytes(), addrN)
	if err != nil {
		return nil, err
	}
	if expectingResponse {
		expectedResponse := &ExpectedResponse{}
		expectedResponse.idQuery = []byte(query.TransactionID)
		expectedResponse.mess = query
		expectedResponse.recieverADDR = addr
		expectedResponse.timeToDie = time.Now().Add(dht.options.TimeToDie)
		dht.expectedResponses[query.TransactionID] = expectedResponse
		return expectedResponse, nil
	}
	return nil, nil

}
