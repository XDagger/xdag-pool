package randomx

import (
	"encoding/hex"
	"sync"
)

type RxHash struct {
	sync.RWMutex
	CurrentSeed string
	LastSeed    string
	Cache       Cache
	Dataset     Dataset
	Vm          VM
}

var Rx *RxHash

func init() {
	Rx = &RxHash{}
	flags := GetFlags()
	cache, _ := AllocCache(flags)
	ds, _ := AllocDataset(flags)
	Rx.Cache = cache
	Rx.Dataset = ds
}

func (h *RxHash) NewSeed(seed []byte) {
	h.Lock()
	defer h.Unlock()

	if h.CurrentSeed == hex.EncodeToString(seed) {
		return
	}

	h.LastSeed = h.CurrentSeed
	h.CurrentSeed = hex.EncodeToString(seed)
	flags := GetFlags()
	InitCache(h.Cache, seed)

	count := DatasetItemCount()

	var wg sync.WaitGroup
	var workerNum = uint32(4)
	for i := uint32(0); i < workerNum; i++ {
		wg.Add(1)
		a := (count * i) / workerNum
		b := (count * (i + 1)) / workerNum
		go func() {
			defer wg.Done()
			InitDataset(h.Dataset, h.Cache, a, b-a)
		}()
	}
	wg.Wait()

	vm, _ := CreateVM(h.Cache, h.Dataset, flags, FlagFullMEM)
	DestroyVM(h.Vm)
	h.Vm = vm
}

func (h *RxHash) NewSeedSlow(seed []byte) {
	h.Lock()
	defer h.Unlock()
	if h.CurrentSeed == hex.EncodeToString(seed) {
		return
	}

	h.LastSeed = h.CurrentSeed
	h.CurrentSeed = hex.EncodeToString(seed)
	flags := GetFlags()
	InitCache(h.Cache, seed)

	vm, _ := CreateVM(h.Cache, nil, flags)
	DestroyVM(h.Vm)
	h.Vm = vm
}

func (h *RxHash) IsCurrentSeed(seed string) bool {
	h.RLock()
	defer h.RUnlock()
	return h.CurrentSeed == seed
}

func (h *RxHash) CalculateHash(input []byte) []byte {
	h.Lock()
	defer h.Unlock()
	return CalculateHash(h.Vm, input)
}
