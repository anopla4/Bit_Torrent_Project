package dht

type kademlia interface {
	Ping(addr string)
	Update(*node)
	GetID() []byte
	FindNode(target []byte) []string
	GetPeers(infohash []byte) ([]string, bool)
	Store(infohash []byte, ip string, port string) error
	GetExpextedResponse(transactionID []byte) (*ExpectedResponse, bool)
	RemoveExpectedResponse(transactionID []byte)
}
