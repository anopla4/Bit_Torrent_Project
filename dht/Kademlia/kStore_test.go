package dht

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStore(t *testing.T) {
	store := NewKStore()
	assert.IsType(t, map[string][]string{}, store.data)
	assert.IsType(t, map[string]time.Time{}, store.expirationTimeMap)
	assert.IsType(t, map[string]time.Time{}, store.republishTimeMap)
	assert.IsType(t, make(chan struct{}), store.lock)
	<-store.lock
	close(store.lock)
}

func TestStore(t *testing.T) {
	store := NewKStore()
	id := getIDWithB(uint8(2))
	data := getIDWithB(uint8(3))
	data2 := getIDWithB(uint8(2))
	err := store.Store(id, data, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(store.data))
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	values, ok := store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, string(data2), values[0])
	assert.Equal(t, string(data), values[1])

	<-store.lock
	close(store.lock)
}

func TestGetValues(t *testing.T) {
	store := NewKStore()
	id := getIDWithB(uint8(2))
	data := getIDWithB(uint8(3))
	data2 := getIDWithB(uint8(4))
	err := store.Store(id, data, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	values, ok := store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, string(data), values[0])
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	values, ok = store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, string(data2), values[1])
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	values, ok = store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, string(data2), values[1])
	id1 := getIDWithB(uint8(8))
	err = store.Store(id1, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	values, ok = store.GetValues(id1)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, string(data2), values[0])
	_, ok = store.GetValues(getIDWithB(uint8(6)))
	assert.Equal(t, false, ok)
	<-store.lock
	close(store.lock)
}

func TestRemoveKey(t *testing.T) {
	store := NewKStore()
	id := getIDWithB(uint8(2))
	data := getIDWithB(uint8(3))
	data2 := getIDWithB(uint8(4))
	err := store.Store(id, data, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	id1 := getIDWithB(uint8(8))
	err = store.Store(id1, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	rm := store.Remove(id1)
	assert.Equal(t, true, rm)
	assert.Equal(t, 1, len(store.data))
	_, ok := store.GetValues(id1)
	assert.Equal(t, false, ok)
	values, ok := store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, string(data2), values[1])

	<-store.lock
	close(store.lock)
}

func TestRemoveKeyValue(t *testing.T) {
	store := NewKStore()
	id := getIDWithB(uint8(2))
	data := getIDWithB(uint8(3))
	data2 := getIDWithB(uint8(4))
	err := store.Store(id, data, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	store.RemoveKeyValue(string(id), string(data2), false)
	values, ok := store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, string(data), values[0])
	assert.Equal(t, 1, len(store.republishTimeMap))
	<-store.lock
	close(store.lock)
}

func TestExpireKeys(t *testing.T) {
	store := NewKStore()
	id := getIDWithB(uint8(2))
	data := getIDWithB(uint8(3))
	data2 := getIDWithB(uint8(4))
	err := store.Store(id, data, time.Now().Add(time.Second*2), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)
	err = store.ExpireKeys()
	if err != nil {
		t.Fatal(err)
	}
	values, ok := store.GetValues(id)
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, string(data2), values[0])
	<-store.lock
	close(store.lock)
}

func TestGetKeyValuesToRepublish(t *testing.T) {
	store := NewKStore()
	id := getIDWithB(uint8(2))
	data := getIDWithB(uint8(3))
	data2 := getIDWithB(uint8(4))
	err := store.Store(id, data, time.Now().Add(time.Second*2), time.Now().Add(time.Second*2))
	if err != nil {
		t.Fatal(err)
	}
	err = store.Store(id, data2, time.Now().Add(time.Second*5), time.Now().Add(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)
	keys, values, err := store.GetKeyValuesToRepublish(time.Second * 3)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(keys))
	assert.Equal(t, 1, len(values))
	assert.Equal(t, keys[0], string(id))
	assert.Equal(t, values[0], string(data))
	<-store.lock
	close(store.lock)
}
