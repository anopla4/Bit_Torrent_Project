package dht

type dht struct {
	routingTable *routingTable
	store        KStore
	options      *Options
	// sendedToken  map[string]string
}
type Options struct {
	ID   []byte
	IP   string
	Port string
}

func (dht *dht) Update(node *node) {
	//TODO
}

//TODO
func (dht *dht) FindNode(target []byte) []string {
	return []string{}
}

//TODO
func (dht *dht) GetPeers(infohash []byte) ([]string, bool) {
	return []string{}, true
}

//TODO
func (dht *dht) Store(infohash []byte, ip string, port string) error {
	return nil
}
