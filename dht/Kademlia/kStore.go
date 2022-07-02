package dht

import (
	"errors"
	"sort"
	"strings"
	"time"
)

// KStore struct for data storage in kademlia
type KStore struct {
	data              map[string][]string //map info_hash to data
	lock              chan struct{}
	expirationTimeMap map[string]time.Time //map key to expiration time
	republishTimeMap  map[string]time.Time //map key to republish time
}

//NewKStore return a new instance of KStore
func NewKStore() *KStore {
	st := &KStore{}

	st.data = map[string][]string{}
	st.lock = make(chan struct{})
	st.lock <- struct{}{}
	st.expirationTimeMap = map[string]time.Time{}
	st.republishTimeMap = map[string]time.Time{}

	return st
}

func (st *KStore) checkKeyValuePair(key string, value string) bool {
	if values, ok := st.data[key]; ok {
		for _, valueC := range values {
			if value == valueC {
				return true
			}
		}
	}
	return false
}

// Store add a new key, value pair to the storage
func (st *KStore) Store(key []byte, data []byte, expirationTime time.Time, republishTime time.Time) error {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	if time.Now().After(expirationTime) || time.Now().After(republishTime) {
		return errors.New("expiration time and republish time have to be after current time")
	}
	keyString := string(key)
	dataString := string(data)
	//combine key and data for time maps
	keyKD := keyString + "/data:" + dataString
	if _, in := st.expirationTimeMap[keyKD]; in {
		st.expirationTimeMap[keyKD] = expirationTime
		st.republishTimeMap[keyKD] = republishTime
		// if !st.checkKeyValuePair(keyString, dataString) {
		// 	st.data[keyString] = append(st.data[keyString], dataString)
		// }
		return nil
	}
	if _, ok := st.data[keyString]; ok {
		// if !st.checkKeyValuePair(keyString, dataString) {
		// 	st.data[keyString] = append(st.data[keyString], dataString)
		// }
		index := sort.SearchStrings(st.data[keyString], dataString)
		if index == len(st.data[keyString]) {
			st.data[keyString] = append(st.data[keyString], dataString)
		} else {
			if st.data[keyString][index] != dataString {
				st.data[keyString] = append(st.data[keyString][:index], append([]string{dataString}, st.data[keyString][index:]...)...)
			}
		}
		st.expirationTimeMap[keyString] = expirationTime
		st.republishTimeMap[keyString] = republishTime
		return nil
	}
	st.data[keyString] = []string{dataString}
	st.expirationTimeMap[keyString] = expirationTime
	st.republishTimeMap[keyString] = republishTime
	return nil
}

// GetValue return data for given key, error if the key is not in storage
func (st *KStore) GetValues(key []byte) ([]string, error) {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	if data, in := st.data[string(key)]; in {
		return data, nil
	}
	return []string{}, errors.New("Not value found for given key")
}

// GetAllPairs return all pairs in store
func (st *KStore) GetAllPairs() (map[string][]string, error) {
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
		for k, _ := range st.expirationTimeMap {
			if strings.Split(k, "/data:")[0] == keyString {
				delete(st.republishTimeMap, k)
				delete(st.expirationTimeMap, k)
			}
		}
		return true
	}
	return false
}

func (st *KStore) RemoveKeyValue(key string, value string) bool {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()
	if _, in := st.data[key]; in {
		index := sort.SearchStrings(st.data[key], value)
		if index == len(st.data[key]) || st.data[key][index] != value {
			return false
		} else {
			if len(st.data[key]) == 1 {
				delete(st.data, key)
			} else {
				st.data[key] = append(st.data[key][:index], st.data[key][index+1:]...)
			}
		}
		return true
	}
	return false
}

// ExpireKeys delete key, value pairs expired
func (st *KStore) ExpireKeys() error {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()

	for keyKD, expTime := range st.expirationTimeMap {
		if time.Now().After(expTime) {
			// delete(st.data, key)
			keyS := strings.Split(keyKD, "/data:")
			key := keyS[0]
			value := keyS[1]
			st.RemoveKeyValue(key, value)
			delete(st.expirationTimeMap, keyKD)
			delete(st.republishTimeMap, keyKD)
		}
	}
	return nil
}

// GetKeyValuesToRepublish return string of the form key and values wich repTime is before time.Now
func (st *KStore) GetKeyValuesToRepublish(timeRep time.Duration) ([]string, []string, error) {
	<-st.lock
	defer func() { st.lock <- struct{}{} }()

	keysToRepublish := []string{}
	valueToRepublish := []string{}
	for keyKD, repTime := range st.republishTimeMap {
		if time.Now().After(repTime) {
			aux := strings.Split(keyKD, "/data:")
			key := aux[0]
			value := aux[1]
			keysToRepublish = append(keysToRepublish, key)
			valueToRepublish = append(valueToRepublish, value)
			st.republishTimeMap[keyKD] = time.Now().Add(timeRep)
		}
	}
	return keysToRepublish, valueToRepublish, nil
}
