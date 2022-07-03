package dht

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKbucket(t *testing.T) {
	kbucket := newKbucket()
	assert.IsType(t, []*node{}, kbucket.bucket)
	assert.Equal(t, K, kbucket.k)
	assert.Equal(t, true, time.Now().After(kbucket.lastChanged))
}
