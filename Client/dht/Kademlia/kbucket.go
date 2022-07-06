package dht

import "time"

type kbucket struct {
	bucket      []*node
	k           int
	lastChanged time.Time
}

func newKbucket() *kbucket {
	b := kbucket{}
	b.k = K
	b.lastChanged = time.Now()
	return &b
}
