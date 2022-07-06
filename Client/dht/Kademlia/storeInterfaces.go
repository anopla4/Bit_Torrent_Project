package dht

// StoreManager interface for general data storage
type StoreManager interface {
	Store(key []byte, value []byte) error
	GetValue(key []byte) ([]byte, error)
	GetAllPairs() (map[string][]byte, error)
	Remove(key []byte) bool
}

// KStoreManager interface to manage data storage in kademlia dht
type KStoreManager interface {
	StoreManager
	ExpireKeys() error
	GetKeysToRepublish() ([][]byte, error)
}
