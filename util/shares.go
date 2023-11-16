package util

import (
	"sync"
	"time"
)

type shares struct {
	lastLock sync.RWMutex
	curLock  sync.RWMutex
	set      [2]*map[string]struct{}
	Last     *map[string]struct{}
	Current  *map[string]struct{}
	step     uint64
}

var MinedShares *shares

func NewMinedShares() {
	MinedShares = &shares{}
	MinedShares.set[0] = new(map[string]struct{})
	MinedShares.Current = MinedShares.set[0]

	go func() {
		for range time.Tick(15 * time.Minute) {
			MinedShares.Next()
		}
	}()
}

func (m *shares) Next() {
	m.lastLock.Lock()
	defer m.lastLock.Unlock()
	m.curLock.Lock()
	defer m.curLock.Unlock()

	n := m.step % 2
	m.Last = m.set[n]

	m.step += 1
	n = m.step % 2
	m.set[n] = new(map[string]struct{})
	m.Current = m.set[n]
}

// func (m *shares) Set(key string) {
// 	m.curLock.Lock()
// 	defer m.curLock.Unlock()

// 	(*m.Current)[key] = struct{}{}
// }

// check duplicated shares
func (m *shares) ShareExist(key string) bool {
	m.curLock.Lock()
	defer m.curLock.Unlock()

	m.lastLock.RLock()
	defer m.lastLock.RUnlock()

	_, ok1 := (*m.Current)[key]

	var ok2 bool
	if m.Last != nil {
		_, ok2 = (*m.Last)[key]
	}

	if !ok1 && !ok2 {
		(*m.Current)[key] = struct{}{}
	}

	return ok1 || ok2
}
