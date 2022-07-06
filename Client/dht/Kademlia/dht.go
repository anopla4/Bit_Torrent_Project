package dht

import (
	"bytes"
	"errors"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackpal/bencode-go"
)

type DHT struct {
	RoutingTable      *RoutingTable
	store             *KStore
	options           *Options
	conn              *net.UDPConn
	expectedResponses map[string]*ExpectedResponse
	// sendedToken  map[string]string
}
type Options struct {
	ID                   []byte
	IP                   string
	Port                 int
	ExpirationTime       time.Duration
	RepublishTime        time.Duration
	TimeToDie            time.Duration
	TimeToRefreshBuckets time.Duration
}

type ExpectedResponse struct {
	idQuery      []byte
	mess         *QueryMessage
	recieverADDR string
	resChan      chan *ResponseMessage
	timeToDie    time.Time
}

func (dht *DHT) Update(node *node) {
	dht.RoutingTable.updateBucketOfNode(node)
}

func (dht *DHT) GetID() []byte {
	return dht.options.ID
}

func NewDHT(options *Options) *DHT {
	dht := &DHT{}
	dht.options = options

	rt, _ := newRoutingTable(options)
	dht.RoutingTable = rt

	st := NewKStore()
	dht.store = st

	dht.expectedResponses = map[string]*ExpectedResponse{}
	return dht
}

func (dht *DHT) Ping(addr string) {
	log.Println("sending ping message to " + addr)
	args := map[string]string{"id": string(dht.options.ID)}
	pingMsg, err := newQueryMessage("ping", args)
	if err != nil {
		log.Println(err.ErrorKRPC())
	}

	// fmt.Println(msgD.QueryName)
	log.Println(pingMsg.Arguments["id"] == string(dht.options.ID))
	expResponse, errQ := dht.SendMessage(pingMsg, true, addr)
	if errQ != nil {
		log.Println(errQ.Error())
	}
	respChan := expResponse.resChan
	select {
	case <-respChan:
		log.Println("receiving ping response from " + addr)
		return
	case <-time.After(dht.options.TimeToDie):
		log.Println("failed receiving ping response")
		dht.RemoveExpectedResponse([]byte(pingMsg.TransactionID))
		//do something with penalize
	}
}

// FindNode return a list with compactInfo of k nearest node to target
func (dht *DHT) FindNode(target []byte) []string {
	nl := dht.RoutingTable.getNearestNodes(K, target)
	nodesInfo := []string{}
	for _, n := range nl.Nodes {
		nodesInfo = append(nodesInfo, string(newNode(n).CompactInfo()))
	}
	return nodesInfo
}

// GetPeers returns addr of peer containing the infohash in case it is in the storage or k nearest nodes in other case(also true in first casem false otherwise)
func (dht *DHT) GetPeers(infohash []byte) ([]string, bool) {
	if values, ok := dht.store.data[string(infohash)]; ok {
		return values, true
	}
	nodes := dht.FindNode(infohash)
	return nodes, false
}

// Store save info in dht storage
func (dht *DHT) Store(infohash []byte, ip string, port string) error {
	log.Println("received announce from ", ip, ":", port)
	expTime := time.Now().Add(dht.options.ExpirationTime) //TODO: inverse proportionality assignation of expirationTime
	repTime := time.Now().Add(dht.options.RepublishTime)
	portI, _ := strconv.Atoi(port)
	addr := net.UDPAddr{IP: net.ParseIP(ip), Port: portI}
	err := dht.store.Store(infohash, []byte(addr.String()), expTime, repTime)
	return err
}

func (dht *DHT) GetExpextedResponse(transactionID []byte) (*ExpectedResponse, bool) {
	if expResponse, ok := dht.expectedResponses[string(transactionID)]; ok {
		return expResponse, ok
	}
	return nil, false
}

func (dht *DHT) LookUP(id string, queryType string) (values []string, closest []string, err error) {
	nl := dht.RoutingTable.getNearestNodes(ALPHA, []byte(id))
	if nl.Len() == 0 {
		return nil, nil, errors.New("no nodes known")
	}
	closestNode := nl.Nodes[0]

	for {
		expectedResponsesLU := []*ExpectedResponse{}
		for i := 0; i < ALPHA && i < nl.Len(); i++ {
			n := nl.Nodes[i]
			addr := net.UDPAddr{IP: n.IP, Port: n.Port}
			q := &QueryMessage{}
			if queryType == "find_node" {
				var err krpcErroInt
				q, err = newQueryMessage(queryType, map[string]string{"id": string(n.ID), "target": id})
				if err != nil {
					return nil, nil, errors.New(err.ErrorKRPC())
				}
			} else {
				if queryType == "get_peers" {
					var err krpcErroInt
					q, err = newQueryMessage(queryType, map[string]string{"id": string(n.ID), "info_hash": id})
					if err != nil {
						return nil, nil, errors.New(err.ErrorKRPC())
					}
				} else {
					return nil, nil, errors.New("not valid query type for node lookup")
				}
			}
			er, err := dht.SendMessage(q, true, addr.String())
			if err != nil {
				return nil, nil, err
			}
			expectedResponsesLU = append(expectedResponsesLU, er)
		}
		respChan := make(chan *ResponseMessage)
		for i := 0; i < len(expectedResponsesLU); i++ {
			go func(r *ExpectedResponse, index int) {
				select {
				case response := <-r.resChan:
					if response == nil {
						dht.PenalizeNode(nl.Nodes[index])
						dht.RemoveExpectedResponse(r.idQuery)
						return
					}
					go func() { respChan <- response }()
				case <-time.After(dht.options.TimeToDie):
					dht.PenalizeNode(nl.Nodes[index])
					dht.RemoveExpectedResponse(r.idQuery)
					return
				}
			}(expectedResponsesLU[i], i)
		}
		results := []*ResponseMessage{}
		expR := len(expectedResponsesLU)
		if expR > 0 {
		WAIT:
			for {
				select {
				case result := <-respChan:
					if result == nil {
						expR--
						continue
					}
					results = append(results, result)
					if len(results) == expR {
						close(respChan)
						break WAIT
					}
				case <-time.After(dht.options.TimeToDie):
					close(respChan)
					break WAIT
				}
			}

			for _, result := range results {
				switch queryType {
				case "find_node":
					nodesInfo := strings.Split(result.Response["nodes"], "//")

					nodes := []*node{}
					for _, nodeInfo := range nodesInfo {
						newN := NewNodeFromCompactInfo([]byte(nodeInfo))
						// if bytes.Equal(newN.ID, dht.options.ID) {
						// 	continue
						// }
						nodes = append(nodes, newN)
					}
					nl.AppendUnique(nodes)

				case "get_peers":
					if values, ok := result.Response["values"]; ok {
						return strings.Split(values, "//"), nil, nil
					}
					nodesInfo := strings.Split(result.Response["nodes"], "//")
					nodes := []*node{}
					for _, nodeInfo := range nodesInfo {
						nodes = append(nodes, NewNodeFromCompactInfo([]byte(nodeInfo)))
					}
					nl.AppendUnique(nodes)

				}
			}

		}
		if nl.Len() == 0 {
			return nil, nil, nil
		}
		sort.Sort(nl)
		//no received node nearest than the last nearest one
		if equalsNodes(nl.Nodes[0], closestNode, false) {
			for _, n := range nl.Nodes {
				nN := newNode(n)
				info := nN.CompactInfo()
				closest = append(closest, string(info))
			}
			return nil, closest, nil
		}
		closestNode = nl.Nodes[0]

	}
}

func (dht *DHT) PenalizeNode(node *NetworkNode) {

}
func (dht *DHT) AnnouncePeer(key string, port int) {
	_, nodesToRequest, err := dht.LookUP(key, "find_node")
	log.Println("announcing peer in dht in port:", port)
	if err != nil {
		log.Println(err)
		return
	}
	args := map[string]string{"id": string(dht.options.ID), "info_hash": key, "port": strconv.Itoa(port)}
	msg, errQ := newQueryMessage("announce_peer", args)

	if err != nil {
		log.Println(errQ)
		return
	}
	for _, nodeInfo := range nodesToRequest {
		n := NewNodeFromCompactInfo([]byte(nodeInfo))

		addr := net.UDPAddr{IP: n.IP, Port: n.Port}
		go func() {
			expRes, err := dht.SendMessage(msg, true, addr.String())
			if err != nil {
				log.Println(err)
				return
			}
			respChan := expRes.resChan
			select {
			case <-respChan:
				return
			case <-time.After(dht.options.TimeToDie):
				dht.RemoveExpectedResponse([]byte(msg.TransactionID))
				//buscar el nodo en la tabla de ruta y penalizarlo
			}
		}()
	}
}

func (dht *DHT) RemoveExpectedResponse(transactionID []byte) {
	delete(dht.expectedResponses, string(transactionID))
}

func (dht *DHT) RunServer(exit chan string) {
	hd := handlerDHT{
		dht: dht,
	}
	server := newServer(dht.options.IP, dht.options.Port, hd)
	conn := make(chan *net.UDPConn)
	go server.RunServer(exit, conn)
	dht.conn = <-conn
	go dht.RefreshBuckets()
	go dht.checkForExpirationTime()
	go dht.CheckExpectedResponses()
}

// SendMessage send a query message to an addres throug dht.conn
func (dht *DHT) SendMessage(query *QueryMessage, expectingResponse bool, addr string) (*ExpectedResponse, error) {
	buffer := &bytes.Buffer{}
	err := bencode.Marshal(buffer, *query)
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
		expectedResponse.resChan = make(chan *ResponseMessage)
		expectedResponse.timeToDie = time.Now().Add(dht.options.TimeToDie)
		dht.expectedResponses[query.TransactionID] = expectedResponse
		return expectedResponse, nil
	}
	return nil, nil

}

func (dht *DHT) checkForExpirationTime() {
	ticker := time.NewTicker(dht.options.ExpirationTime)
	for {
		<-ticker.C
		err := dht.store.ExpireKeys()
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

// func (dht *DHT) checkForRepublish() {
// 	ticker := time.NewTicker(dht.options.RepublishTime)
// 	for {
// 		<-ticker.C
// 		keysToRepublish, valuesToRepublish, err := dht.store.GetKeyValuesToRepublish(dht.options.RepublishTime)
// 		if err != nil {
// 			panic(err)
// 		}
// 		for i := 0; i < len(keysToRepublish); i++ {
// 			go dht.AnnouncePeer(keysToRepublish[i], valuesToRepublish[i])
// 		}
// 	}
// }

func (dht *DHT) CheckExpectedResponses() {
	ticker := time.NewTicker(2 * dht.options.TimeToDie)

	for {
		<-ticker.C
		for key, value := range dht.expectedResponses {
			if value.timeToDie.Before(time.Now()) {
				delete(dht.expectedResponses, key)
			}
		}
	}
}

func (dht *DHT) GetPeersToDownload(infohash []byte) ([]string, error) {
	v, ok := dht.GetPeers(infohash)
	if ok {
		return v, nil
	}
	values, _, err := dht.LookUP(string(infohash), "get_peers")
	if err != nil {
		return nil, err
	}
	if values == nil {
		return []string{}, nil
	}
	return values, nil
}

func (dht *DHT) RefreshBuckets() {
	for i := 0; i < B; i++ {
		go dht.RefreshBucket(i)
	}
}

func (dht *DHT) RefreshBucket(bucket int) {
	ticker := time.NewTicker(dht.options.TimeToRefreshBuckets)
	for {
		<-ticker.C
		if dht.RoutingTable.table[bucket].lastChanged.Add(dht.options.TimeToRefreshBuckets).Before(time.Now()) {
			id := dht.RoutingTable.getRandomIDFromBucket(bucket)
			go dht.LookUP(string(id), "find_node")
		}
	}
}

func (dht *DHT) JoinNetwork(addr string) {
	dht.Ping(addr)
	go func() {
		_, _, err := dht.LookUP(string(dht.options.ID), "find_node")
		if err != nil {
			log.Println("failed LookUp: ", err)
			return
		}
	}()
}

//TODO:
//2- JoinNetwork function Creo q hecho
//3- check bucket time Creo q hecho
//4- implement penalize and unpenalize each time received a response from a node(last can be done in update function)
//5- check the problem with republish
