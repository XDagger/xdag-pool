package util

import (
	"errors"
	"sync"
	"time"

	"github.com/wangjia184/sortedset"
)

type Rank struct {
	Login    string
	Hashrate float64
}
type SortedHashrate struct {
	lastLock      sync.RWMutex
	curLock       sync.RWMutex
	set           [2]*sortedset.SortedSet
	Last          *sortedset.SortedSet
	Current       *sortedset.SortedSet
	step          uint64
	interval      int
	totalHashrate float64
}

var HashrateRank *SortedHashrate

// interval: rank refresh interval (minutes)
func NewHashrateRank(interval int) {
	HashrateRank := &SortedHashrate{}
	HashrateRank.set[0] = sortedset.New()
	HashrateRank.Current = HashrateRank.set[0]
	HashrateRank.interval = interval

	go func() {
		<-time.After(15 * time.Minute) //init first rank
		HashrateRank.Next()

		for range time.Tick(time.Duration(interval) * time.Minute) {
			HashrateRank.Next()
		}
	}()
}

// switch current sortedset
func (s *SortedHashrate) Next() {
	s.lastLock.Lock()
	defer s.lastLock.Unlock()
	s.curLock.Lock()
	defer s.curLock.Unlock()

	n := s.step % 2
	s.Last = s.set[n]

	s.step += 1
	n = s.step % 2
	s.set[n] = sortedset.New()
	s.Current = s.set[n]
	s.totalHashrate = float64(s.Last.PopMax().Score()) / float64(s.interval)
}

// accumulate share diff for miners
func (s *SortedHashrate) IncShareByKey(key string, share int64) {
	s.curLock.Lock()
	defer s.curLock.Unlock()

	var score sortedset.SCORE
	node := s.Current.GetByKey(key)
	if node != nil {
		score = node.Score()
	}
	s.Current.AddOrUpdate(key, score+sortedset.SCORE(share), nil)
}

// start begin with 1, get [start,end] items, get all items when [1, -1]
func (s *SortedHashrate) GetRanks(start, end int) ([]Rank, error) {
	s.lastLock.RLock()
	defer s.lastLock.RUnlock()
	if s.Last == nil {
		return nil, errors.New("hashrate rank not ready")
	}
	if start < 1 || start > end {
		return nil, errors.New("input rank number error")
	}
	nodes := s.Last.GetByRankRange(-1*start, -1*end, false)
	var ranks = make([]Rank, len(nodes))
	for i := 0; i < len(nodes); i++ {
		r := Rank{
			Login:    nodes[i].Key(),
			Hashrate: float64(nodes[i].Score()) / float64(s.interval),
		}
		ranks = append(ranks, r)
	}
	return ranks, nil
}
