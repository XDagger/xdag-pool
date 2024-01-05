// Generated from https://github.com/streamrail/concurrent-map
package stratum

import (
	"fmt"
	"hash/fnv"
	"sync"
)

// TODO: Add Keys function which returns an array of keys for the map.

// A "thread" safe map of type string:*sync.Map.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type WorkersMap []*WorkersMapShared
type WorkersMapShared struct {
	items        map[string]*sync.Map
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new concurrent map.
func NewWorkersMap() WorkersMap {
	m := make(WorkersMap, SHARD_COUNT)
	for i := 0; i < SHARD_COUNT; i++ {
		m[i] = &WorkersMapShared{items: make(map[string]*sync.Map)}
	}
	return m
}

// Returns shard under given key
func (m WorkersMap) GetShard(key string) *WorkersMapShared {
	hasher := fnv.New32()
	_, _ = hasher.Write([]byte(key))
	return m[int(hasher.Sum32())%SHARD_COUNT]
}

// Sets the given value under the specified key.
func (m *WorkersMap) Set(key string, value *sync.Map) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	shard.items[key] = value
}

// Retrieves an element from map under given key.
func (m WorkersMap) Get(key string) (*sync.Map, bool) {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	defer shard.RUnlock()

	// Get item from shard.
	val, ok := shard.items[key]
	return val, ok
}

// Returns the number of elements within the map.
func (m WorkersMap) Count() int {
	count := 0
	for i := 0; i < SHARD_COUNT; i++ {
		shard := m[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m *WorkersMap) Has(key string) bool {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	defer shard.RUnlock()

	// See if element is within shard.
	_, ok := shard.items[key]
	return ok
}

// Removes an element from the map.
func (m *WorkersMap) Remove(key string) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	defer shard.Unlock()
	delete(shard.items, key)
}

// Checks if map is empty.
func (m *WorkersMap) IsEmpty() bool {
	return m.Count() == 0
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type NameTuple struct {
	Key string
	Val *sync.Map
}

// Returns an iterator which could be used in a for range loop.
func (m WorkersMap) Iter() <-chan NameTuple {
	ch := make(chan NameTuple)
	go func() {
		// Foreach shard.
		for _, shard := range m {
			// Foreach key, value pair.
			shard.RLock()
			for key, val := range shard.items {
				ch <- NameTuple{key, val}
			}
			shard.RUnlock()
		}
		close(ch)
	}()
	return ch
}

// Returns a buffered iterator which could be used in a for range loop.
func (m WorkersMap) IterBuffered() <-chan NameTuple {
	ch := make(chan NameTuple, m.Count())
	go func() {
		// Foreach shard.
		for _, shard := range m {
			// Foreach key, value pair.
			shard.RLock()
			for key, val := range shard.items {
				ch <- NameTuple{key, val}
			}
			shard.RUnlock()
		}
		close(ch)
	}()
	return ch
}

// Retrieves an element from map under given key.
func (m WorkersMap) GetWorkers(key string) []string {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	defer shard.RUnlock()

	// Get item from shard.
	val, ok := shard.items[key]
	if !ok {
		return nil
	}
	var worker []string
	val.Range(func(k, v interface{}) bool {
		n := fmt.Sprintf("%v", k)
		worker = append(worker, n)
		return true
	})
	return worker
}
