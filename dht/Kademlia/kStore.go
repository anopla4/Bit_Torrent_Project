package dht

import (
	"bytes"
	"errors"
	"time"
)

// KStore struct for data storage
type KStore struct {
	data              map[string][]byte //map info_hash to data
	lock              chan struct{}
	expirationTimeMap map[string]time.Time //map key to expiration time
	republishTimeMap  map[string]time.Time //map key to republish time
}

//NewKStore return a new instance of KStore
func NewKStore() *KStore {
	st := &KStore{}

	st.data = map[string][]byte{}
	st.lock = make(chan struct{})
	st.lock <- struct{}{}
	st.expirationTimeMap = map[string]time.Time{}
	st.republishTimeMap = map[string]time.Time{}

	return st
}

// Store add a new key, value pair to the storage
func (st *KStore) Store(key []byte, data []byte, expirationTime time.Time, republishTime time.Time) error {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	if time.Now().After(expirationTime) || time.Now().After(republishTime) {
		return errors.New("expiration time and republish time have to be after current time")
	}
	keyString := string(key)
	if dataCurrent, in := st.data[keyString]; in {
		st.expirationTimeMap[keyString] = expirationTime
		st.republishTimeMap[keyString] = republishTime
		if !bytes.Equal(dataCurrent, data) {
			st.data[keyString] = data
		}
		return nil
	}

	st.data[keyString] = data
	st.expirationTimeMap[keyString] = expirationTime
	st.republishTimeMap[keyString] = republishTime
	return nil
}

// GetValue return data for given key, error if the key is not in storage
func (st *KStore) GetValue(key []byte) ([]byte, error) {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	if data, in := st.data[string(key)]; in {
		return data, nil
	}
	return []byte{}, errors.New("Not value found for given key")
}

// GetAllPairs return all pairs in store
func (st *KStore) GetAllPairs() (map[string][]byte, error) {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	return st.data, nil
}

// Remove delete key if exist in store
func (st *KStore) Remove(key []byte) bool {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	keyString := string(key)
	if _, in := st.data[keyString]; in {
		delete(st.data, keyString)
		delete(st.republishTimeMap, keyString)
		delete(st.expirationTimeMap, keyString)
		return true
	}
	return false
}

// ExpireKeys delete key, value pairs expired
func (st *KStore) ExpireKeys() error {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()

	for key, expTime := range st.expirationTimeMap {
		if time.Now().After(expTime) {
			delete(st.data, key)
			delete(st.expirationTimeMap, key)
			delete(st.republishTimeMap, key)
		}
	}
	return nil
}

// GetKeysToRepublish return key wich repTime is before time.Now
func (st *KStore) GetKeysToRepublish() ([][]byte, error) {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()

	keysToRepublish := [][]byte{}
	for key, repTime := range st.republishTimeMap {
		if time.Now().After(repTime) {
			keysToRepublish = append(keysToRepublish, []byte(key))
		}
	}
	return keysToRepublish, nil
}
