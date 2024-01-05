package stratum

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/ws"
	"github.com/gorilla/mux"
)

func (s *StratumServer) StatsIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	hashrate, hashrate24h, totalOnline, miners := s.collectMinersStats()
	stats := map[string]interface{}{
		"miners":      miners,
		"hashrate":    hashrate,
		"hashrate24h": hashrate24h,
		"totalMiners": len(miners),
		"totalOnline": totalOnline,
		"timedOut":    len(miners) - totalOnline,
		"now":         util.MakeTimestamp(),
	}

	stats["upstream"] = ws.Client.Url
	stats["luck"] = s.getLuckStats()
	stats["blocks"] = s.getBlocksStats()

	if t := s.currentBlockTemplate(); t != nil {
		// stats["height"] = t.height
		// stats["diff"] = t.diffInt64
		// roundShares := atomic.LoadInt64(&s.roundShares)
		// stats["variance"] = float64(roundShares) / float64(t.diffInt64)
		stats["prevHash"] = t.jobHash[0:8]
		stats["template"] = true
	}
	_ = json.NewEncoder(w).Encode(stats)
}

// func convertUpstream(u *rpc.RPCClient) map[string]interface{} {
// 	upstream := map[string]interface{}{
// 		"name":             u.Name,
// 		"url":              u.Url.String(),
// 		"sick":             u.Sick(),
// 		"accepts":          atomic.LoadInt64(&u.Accepts),
// 		"rejects":          atomic.LoadInt64(&u.Rejects),
// 		"lastSubmissionAt": atomic.LoadInt64(&u.LastSubmissionAt),
// 		"failsCount":       atomic.LoadInt64(&u.FailsCount),
// 		//"info":             u.Info(),
// 	}
// 	return upstream
// }

func (s *StratumServer) collectMinersStats() (float64, float64, int, []interface{}) {
	now := util.MakeTimestamp()
	var result []interface{}
	totalhashrate := float64(0)
	totalhashrate24h := float64(0)
	totalOnline := 0
	window24h := 24 * time.Hour

	for m := range s.miners.Iter() {
		stats := make(map[string]interface{})
		lastBeat := m.Val.getLastBeat()
		hashrate := m.Val.hashrate(s.estimationWindow)
		hashrate24h := m.Val.hashrate(window24h)
		totalhashrate += hashrate
		totalhashrate24h += hashrate24h
		stats["name"] = m.Key
		stats["hashrate"] = hashrate
		stats["hashrate24h"] = hashrate24h
		stats["lastBeat"] = lastBeat
		stats["validShares"] = atomic.LoadInt64(&m.Val.validShares)
		stats["staleShares"] = atomic.LoadInt64(&m.Val.staleShares)
		stats["invalidShares"] = atomic.LoadInt64(&m.Val.invalidShares)
		stats["accepts"] = atomic.LoadInt64(&m.Val.accepts)
		stats["rejects"] = atomic.LoadInt64(&m.Val.rejects)
		if !s.config.Frontend.HideIP {
			stats["ip"] = m.Val.ip
		}

		if now-lastBeat > (int64(s.timeout/2) / 1000000) {
			stats["warning"] = true
		}
		if now-lastBeat > (int64(s.timeout) / 1000000) {
			stats["timeout"] = true
		} else {
			totalOnline++
		}
		result = append(result, stats)
	}
	return totalhashrate, totalhashrate24h, totalOnline, result
}

func (s *StratumServer) getLuckStats() map[string]interface{} {
	now := util.MakeTimestamp()
	var variance float64
	var totalVariance float64
	var blocksCount int
	var totalBlocksCount int

	s.blocksMu.Lock()
	defer s.blocksMu.Unlock()

	for k, v := range s.blockStats {
		if k >= now-s.luckWindow {
			blocksCount++
			variance += v.variance
		}
		if k >= now-s.luckLargeWindow {
			totalBlocksCount++
			totalVariance += v.variance
		} else {
			delete(s.blockStats, k)
		}
	}
	if blocksCount != 0 {
		variance = variance / float64(blocksCount)
	}
	if totalBlocksCount != 0 {
		totalVariance = totalVariance / float64(totalBlocksCount)
	}
	result := make(map[string]interface{})
	result["variance"] = variance
	result["blocksCount"] = blocksCount
	result["window"] = s.config.LuckWindow
	result["totalVariance"] = totalVariance
	result["totalBlocksCount"] = totalBlocksCount
	result["largeWindow"] = s.config.LargeLuckWindow
	return result
}

func (s *StratumServer) getBlocksStats() []interface{} {
	now := util.MakeTimestamp()
	var result []interface{}

	s.blocksMu.Lock()
	defer s.blocksMu.Unlock()

	for k, v := range s.blockStats {
		if k >= now-int64(s.luckLargeWindow) {
			block := map[string]interface{}{
				"height":    v.height,
				"hash":      v.hash,
				"variance":  v.variance,
				"timestamp": k,
			}
			result = append(result, block)
		} else {
			delete(s.blockStats, k)
		}
	}
	return result
}

func (s *StratumServer) PoolDonateList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	num := vars["page"]
	size := vars["pageSize"]
	if num == "" || size == "" {
		data["code"] = -1
		data["msg"] = "parameter empty"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	page, err1 := strconv.Atoi(num)
	pageSize, err2 := strconv.Atoi(size)
	if err1 != nil || err2 != nil {
		data["code"] = -1
		data["msg"] = "parameter not number"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	start := (page-1)*pageSize + 1
	end := start + pageSize - 1

	amount, count, err := s.backend.GetTotalDonate()
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	list, err := s.backend.GetDonateList(int64(start), int64(end))
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["amount"] = amount
	data["page"] = page
	data["pageSize"] = pageSize
	data["total"] = count
	data["data"] = list
	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) PoolAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})

	rewards, payment, unpaid, donate, err := s.backend.GetPoolAccount()
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["totalRewards"] = rewards
	data["totalPayment"] = payment
	data["totalUnpaid"] = unpaid
	data["totalDonate"] = donate

	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) PoolRewardsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	num := vars["page"]
	size := vars["pageSize"]
	if num == "" || size == "" {
		data["code"] = -1
		data["msg"] = "parameter empty"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	page, err1 := strconv.Atoi(num)
	pageSize, err2 := strconv.Atoi(size)
	if err1 != nil || err2 != nil {
		data["code"] = -1
		data["msg"] = "parameter not number"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	start := (page-1)*pageSize + 1
	end := start + pageSize - 1

	amount, count, err := s.backend.GetTotalPoolRewards()
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	list, err := s.backend.GetPoolRewardsList(int64(start), int64(end))
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["amount"] = amount
	data["page"] = page
	data["pageSize"] = pageSize
	data["total"] = count
	data["data"] = list

	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) HashrateRank(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	num := vars["page"]
	size := vars["pageSize"]
	if num == "" || size == "" {
		data["code"] = -1
		data["msg"] = "parameter empty"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	page, err1 := strconv.Atoi(num)
	pageSize, err2 := strconv.Atoi(size)
	if err1 != nil || err2 != nil {
		data["code"] = -1
		data["msg"] = "parameter not number"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	start := (page-1)*pageSize + 1
	end := start + pageSize - 1
	ranks, count, err := util.HashrateRank.GetRanks(start, end)
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	data["code"] = 0
	data["msg"] = "success"
	data["page"] = page
	data["pageSize"] = pageSize
	data["total"] = count
	data["data"] = ranks
	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) MinerAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	address := vars["address"]
	if address == "" || !util.ValidateAddress(address) {
		data["code"] = -1
		data["msg"] = "addres is empty or invalid"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	reward, payment, unpaid, err := s.backend.GetMinerAccount(address)
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["totalReward"] = reward
	data["totalPayment"] = payment
	data["totalUnpaid"] = unpaid

	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) MinerRewardsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	address := vars["address"]
	if address == "" || !util.ValidateAddress(address) {
		data["code"] = -1
		data["msg"] = "addres is empty address or invalid"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	num := vars["page"]
	size := vars["pageSize"]
	if num == "" || size == "" {
		data["code"] = -1
		data["msg"] = "page parameter empty"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	page, err1 := strconv.Atoi(num)
	pageSize, err2 := strconv.Atoi(size)
	if err1 != nil || err2 != nil {
		data["code"] = -1
		data["msg"] = "page parameter not number"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	start := (page-1)*pageSize + 1
	end := start + pageSize - 1

	amount, count, err := s.backend.MinerTotalRewards(address)
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	list, err := s.backend.MinerRewardsList(address, int64(start), int64(end))
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["amount"] = amount
	data["page"] = page
	data["pageSize"] = pageSize
	data["total"] = count
	data["data"] = list

	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) MinerPaymentList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	address := vars["address"]
	if address == "" || !util.ValidateAddress(address) {
		data["code"] = -1
		data["msg"] = "addres is empty address or invalid"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	num := vars["page"]
	size := vars["pageSize"]
	if num == "" || size == "" {
		data["code"] = -1
		data["msg"] = "page parameter empty"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	page, err1 := strconv.Atoi(num)
	pageSize, err2 := strconv.Atoi(size)
	if err1 != nil || err2 != nil {
		data["code"] = -1
		data["msg"] = "page parameter not number"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	start := (page-1)*pageSize + 1
	end := start + pageSize - 1

	amount, count, err := s.backend.MinerTotalPayment(address)
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	list, err := s.backend.MinerPaymentList(address, int64(start), int64(end))
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["amount"] = amount
	data["page"] = page
	data["pageSize"] = pageSize
	data["total"] = count
	data["data"] = list

	_ = json.NewEncoder(w).Encode(data)
}

func (s *StratumServer) MinerBalanceList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	address := vars["address"]
	if address == "" || !util.ValidateAddress(address) {
		data["code"] = -1
		data["msg"] = "addres is empty address or invalid"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	num := vars["page"]
	size := vars["pageSize"]
	if num == "" || size == "" {
		data["code"] = -1
		data["msg"] = "page parameter empty"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	page, err1 := strconv.Atoi(num)
	pageSize, err2 := strconv.Atoi(size)
	if err1 != nil || err2 != nil {
		data["code"] = -1
		data["msg"] = "page parameter not number"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	start := (page-1)*pageSize + 1
	end := start + pageSize - 1

	count, list, err := s.backend.MinerBalanceList(address, int64(start), int64(end))
	if err != nil {
		data["code"] = -1
		data["msg"] = err.Error()
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}

	data["code"] = 0
	data["msg"] = "success"
	data["page"] = page
	data["pageSize"] = pageSize
	data["total"] = count
	data["data"] = list

	_ = json.NewEncoder(w).Encode(data)
}

// func (s *StratumServer) PoolHashrate(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
// 	w.WriteHeader(http.StatusOK)
// 	data := make(map[string]interface{})

// 	_ = json.NewEncoder(w).Encode(data)
// }

func (s *StratumServer) MinerHashrate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	data := make(map[string]interface{})
	vars := mux.Vars(r)
	address := vars["address"]
	if address == "" || !util.ValidateAddress(address) {
		data["code"] = -1
		data["msg"] = "addres is empty or invalid"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	works := s.workers.GetWorkers(address)
	if len(works) == 0 {
		data["code"] = -1
		data["msg"] = "no works"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	var hashrate = make(map[string]float64)
	var hashrate24h = make(map[string]float64)
	window24h := 24 * time.Hour
	for _, w := range works {
		m, ok := s.miners.Get(address + "." + w)
		if !ok {
			continue
		}
		hashrate[w] = m.hashrate(s.estimationWindow)
		hashrate24h[w] = m.hashrate(window24h)
	}
	var h = make(map[string]interface{})
	h["hashrate"] = hashrate
	h["hashrate24h"] = hashrate24h

	data["code"] = 0
	data["msg"] = "success"
	data["data"] = h
	_ = json.NewEncoder(w).Encode(data)
}
