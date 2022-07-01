package peer

import "net"

type Peer struct {
	Id   string
	IP   net.IP
	Port uint16
}

func (p *Peer) String() string {
	return p.Id
}
