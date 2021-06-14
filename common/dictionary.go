package common

import (
	"sync"
)

type ValueDictionary struct {
	value map[string]interface{}
	lock  sync.RWMutex
}

// Set adds a new item to the dictionary
func (d *ValueDictionary) Set(k string, v interface{}) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.value == nil {
		d.value = make(map[string]interface{})
	}
	d.value[k] = v
}

// Get returns the value associated with the key
func (d *ValueDictionary) Get(k string) interface{} {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.value[k]
}

// Has returns true if the key exists in the dictionary
func (d *ValueDictionary) Has(k string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()
	_, ok := d.value[k]
	return ok
}
