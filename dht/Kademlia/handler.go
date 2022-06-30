package dht

import (
	"net"
)

type handlerDHT struct {
	dht dht
}

func newHandlerDHT(dht dht) *handlerDHT {
	return &handlerDHT{dht}
}

func (hd *handlerDHT) updateNodeOfMessage(msg *QueryMessage, addr string) krpcErroInt {
	nodeQuerying := &node{}

	if _, in := msg.Arguments["id"]; !in {
		return newProtocolError("id key is required for ping request")
	}
	value := msg.Arguments["id"]
	nodeQuerying.ID = value.([]byte)
	addrUDP, errADDR := net.ResolveUDPAddr("udp", addr)
	if errADDR != nil {
		return newGenericError("error resolving udp addr: " + errADDR.Error())
	}
	nodeQuerying.IP = addrUDP.IP
	nodeQuerying.port = addrUDP.Port

	hd.dht.Update(nodeQuerying)
	return nil
}

// ResponsePing return message in response to a ping query
func (hd *handlerDHT) ResponsePing(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	args := map[string]interface{}{}
	args["id"] = hd.dht.options.ID
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
	errUpd := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUpd
}

// ResponseFindNode return message in response to a find_node query
func (hd *handlerDHT) ResponseFindNode(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	//arguments for response
	args := map[string]interface{}{}
	args["id"] = hd.dht.options.ID

	//find nearest node to target id
	target := msg.Arguments["target"].([]byte)
	args["nodes"] = hd.dht.FindNode(target)

	//create responce message instance
	msgResponse, errK := newResponseMessage(msg.QueryName, msg.TransactionID, args)
	if errK != nil {
		return nil, errK
	}

	//update node status in routing table
	errUpd := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUpd
}

// ResponseGetPeers return message in response to a get_peers query
func (hd *handlerDHT) ResponseGetPeers(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	//arguments for response
	args := map[string]interface{}{}
	args["id"] = hd.dht.options.ID

	// //generate random token
	// token := make([]byte, 1)
	// rand.Read(token)
	// args["token"] = token

	// //store token and addr of querying node
	// hd.dht.sendedToken[string(token)] = addr

	infohash := msg.Arguments["info_hash"].([]byte)
	peers, ok := hd.dht.GetPeers(infohash)
	if ok {
		args["values"] = peers
	} else {
		args["nodes"] = peers
	}

	//create responce message instance
	msgResponse, errK := newResponseMessage(msg.QueryName, msg.TransactionID, args)
	if errK != nil {
		return nil, errK
	}

	//update node status in routing table
	errUpd := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUpd
}

func (hd *handlerDHT) ResponseAnnouncePeers(msg *QueryMessage, addr string) (*ResponseMessage, krpcErroInt) {
	//arguments for response
	args := map[string]interface{}{}
	args["id"] = hd.dht.options.ID

	infohash := msg.Arguments["info_hash"].([]byte)
	port := msg.Arguments["port"].(string)
	ip, err := net.ResolveIPAddr("udp", addr)
	if err != nil {
		return nil, newGenericError(err.Error())
	}
	err = hd.dht.Store(infohash, ip.String(), port)
	if err != nil {
		return nil, newGenericError(err.Error())
	}

	//create responce message instance
	msgResponse, errK := newResponseMessage(msg.QueryName, msg.TransactionID, args)
	if errK != nil {
		return nil, errK
	}

	//update node status in routing table
	errUpd := hd.updateNodeOfMessage(msg, addr)
	return msgResponse, errUpd
}
