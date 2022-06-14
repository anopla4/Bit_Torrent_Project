package tracker

import "net"

// PeerTk es una representacion del peer por parte del tracker
type PeerTk struct {
	PeerID string
	IP     net.IP
	Port   int
}
