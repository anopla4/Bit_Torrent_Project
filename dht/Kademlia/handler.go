package dht

import (
	"net"
	"strings"
)

type handlerDHT struct {
	dht kademlia
}

// func newHandlerDHT(dht dht) *handlerDHT {
// 	return &handlerDHT{dht}
// }

func (hd *handlerDHT) updateNodeOfMessage(msg *QueryMessage, addr string) krpcErroInt {
	nodeQuerying := &node{}

	if _, in := msg.Arguments["id"]; !in {
		return newProtocolError("id key is required for ping request")
	}
	value := msg.Arguments["id"]
	nodeQuerying.ID = []byte(value)
	addrUDP, errADDR := net.ResolveUDPAddr("udp", addr)
	if errADDR != nil {
		return newGenericError("error resolving udp addr: " + errADDR.Error())
	}
	nodeQuerying.IP = addrUDP.IP
	nodeQuerying.port = addrUDP.Port

	hd.dht.Update(nodeQuerying)
	return nil
}

func (hd *handlerDHT) updateNodeOfResponse(msg *ResponseMessage, addr string) krpcErroInt {
	nodeR := &node{}

	if _, in := msg.Response["id"]; !in {
		hd.checkAddrInRoutingTable(addr)
		return newProtocolError("id key is required")
	}
	value := msg.Response["id"]
	nodeR.ID = []byte(value)
	addrUDP, errADDR := net.ResolveUDPAddr("udp", addr)
	if errADDR != nil {
		return newGenericError("error resolving udp addr: " + errADDR.Error())
	}
	nodeR.IP = addrUDP.IP
	nodeR.port = addrUDP.Port

	//TODO:
	//actualizar en 0 la cantidad de respuestas
	hd.dht.Update(nodeR)
	return nil
}

// ResponsePing return message in response to a ping query
func (hd *handlerDHT) ResponsePing(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	args := map[string]string{}
	args["id"] = string(hd.dht.GetID())
	msgResponse, err := newResponseMessage("ping", msg.TransactionID, args)
	if err != nil {
		return nil, err
	}
	// nodeQuerying := &node{}

	// if _, in := msg.Arguments["id"]; !in {
	// 	return nil, newProtocolError("id key is required for ping request")
	// }
	// value := msg.Arguments["id"]
	// nodeQuerying.ID = value.([]byte)
	// addrUDP, errADDR := net.ResolveUDPAddr("udp", addr)
	// if errADDR != nil {
	// 	return nil, newGenericError("error resolving udp addr: " + errADDR.Error())
	// }
	// nodeQuerying.IP = addrUDP.IP
	// nodeQuerying.port = addrUDP.Port

	// hd.dht.Update(nodeQuerying)
	errUDP := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUDP
}

// ResponseFindNode return message in response to a find_node query
func (hd *handlerDHT) ResponseFindNode(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	//arguments for response
	args := map[string]string{}
	args["id"] = string(hd.dht.GetID())

	//find nearest node to target id
	target := []byte(msg.Arguments["target"])
	nodes := strings.Join(hd.dht.FindNode(target), "//")
	args["nodes"] = nodes

	//create responce message instance
	msgResponse, errK := newResponseMessage(msg.QueryName, msg.TransactionID, args)
	if errK != nil {
		return nil, errK
	}

	//update node status in routing table
	errUDP := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUDP
}

// ResponseGetPeers return message in response to a get_peers query
func (hd *handlerDHT) ResponseGetPeers(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	//arguments for response
	args := map[string]string{}
	args["id"] = string(hd.dht.GetID())

	// //generate random token
	// token := make([]byte, 1)
	// rand.Read(token)
	// args["token"] = token

	// //store token and addr of querying node
	// hd.dht.sendedToken[string(token)] = addr

	infohash := []byte(msg.Arguments["info_hash"])
	peers, ok := hd.dht.GetPeers(infohash)
	if ok {
		args["values"] = strings.Join(peers, "//")
	} else {
		args["nodes"] = strings.Join(peers, "//")
	}

	//create responce message instance
	msgResponse, errK := newResponseMessage(msg.QueryName, msg.TransactionID, args)
	if errK != nil {
		return nil, errK
	}

	//update node status in routing table
	errUDP := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUDP
}

func (hd *handlerDHT) ResponseAnnouncePeers(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	//arguments for response
	args := map[string]string{}
	args["id"] = string(hd.dht.GetID())

	infohash := msg.Arguments["info_hash"]
	port := msg.Arguments["port"]
	ip, err := net.ResolveIPAddr("udp", addr)
	if err != nil {
		return nil, newGenericError(err.Error())
	}
	err = hd.dht.Store([]byte(infohash), ip.String(), port)
	if err != nil {
		return nil, newGenericError(err.Error())
	}

	//create responce message instance
	msgResponse, errK := newResponseMessage(msg.QueryName, msg.TransactionID, args)
	if errK != nil {
		return nil, errK
	}

	//update node status in routing table
	errUDP := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUDP
}

//update node info in case message is corrupted
func (hd *handlerDHT) checkAddrInRoutingTable(addr string) {
	//TODO:
}

func (hd *handlerDHT) getResponse(msg *ResponseMessage, addr string) {
	if expResponse, ok := hd.dht.GetExpextedResponse([]byte(msg.TransactionID)); ok {
		if addr != expResponse.recieverADDR {
			hd.dht.RemoveExpectedResponse(expResponse.idQuery)
			resChan := expResponse.resChan
			resChan <- nil
			goto UPD
		}
		if expResponse.mess.TypeOfMessage == "find_node" {
			if _, in := msg.Response["id"]; !in {
				hd.dht.RemoveExpectedResponse([]byte(msg.TransactionID))
				resChan := expResponse.resChan
				resChan <- nil
				goto UPD
			}
			if _, in := msg.Response["nodes"]; !in {
				hd.dht.RemoveExpectedResponse([]byte(msg.TransactionID))
				resChan := expResponse.resChan
				resChan <- nil
				goto UPD
			}
			resChan := expResponse.resChan
			resChan <- msg
		} else {
			if expResponse.mess.TypeOfMessage == "get_peers" {
				if _, in := msg.Response["id"]; !in {
					hd.dht.RemoveExpectedResponse([]byte(msg.TransactionID))
					resChan := expResponse.resChan
					resChan <- nil
					goto UPD
				}
				_, in := msg.Response["nodes"]
				_, inn := msg.Response["values"]
				if !in && !inn {
					hd.dht.RemoveExpectedResponse([]byte(msg.TransactionID))
					resChan := expResponse.resChan
					resChan <- nil
					goto UPD
				}
				resChan := expResponse.resChan
				resChan <- msg
			} else {
				if expResponse.mess.TypeOfMessage == "ping" {
					if _, in := msg.Response["id"]; !in {
						hd.dht.RemoveExpectedResponse([]byte(msg.TransactionID))
						resChan := expResponse.resChan
						resChan <- nil
						goto UPD
					}
					resChan := expResponse.resChan
					resChan <- msg
				}
				// else {
				// 	if expResponse.mess.TypeOfMessage == "announce_peer" {
				// 		if _, in := msg.Response["id"]; !in {
				// 			hd.dht.RemoveExpectedResponse([]byte(msg.TransactionID))
				// 			goto UPD
				// 		}
				// 		resChan := *(&expResponse.resChan)
				// 		resChan <- msg.Response

				// }
			}
		}

	}
UPD:
	//update node status in routing table
	errUDP := hd.updateNodeOfResponse(msg, addr)
	if errUDP != nil {
		panic(errUDP)
	}

}
