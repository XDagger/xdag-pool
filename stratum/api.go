package stratum

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/XDagger/xdagpool/jrpc"
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
	// stats["luck"] = s.getLuckStats()
	// stats["blocks"] = s.getBlocksStats()

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

// func (s *StratumServer) getLuckStats() map[string]interface{} {
// 	now := util.MakeTimestamp()
// 	var variance float64
// 	var totalVariance float64
// 	var blocksCount int
// 	var totalBlocksCount int

// 	s.blocksMu.Lock()
// 	defer s.blocksMu.Unlock()

// 	for k, v := range s.blockStats {
// 		if k >= now-s.luckWindow {
// 			blocksCount++
// 			variance += v.variance
// 		}
// 		if k >= now-s.luckLargeWindow {
// 			totalBlocksCount++
// 			totalVariance += v.variance
// 		} else {
// 			delete(s.blockStats, k)
// 		}
// 	}
// 	if blocksCount != 0 {
// 		variance = variance / float64(blocksCount)
// 	}
// 	if totalBlocksCount != 0 {
// 		totalVariance = totalVariance / float64(totalBlocksCount)
// 	}
// 	result := make(map[string]interface{})
// 	result["variance"] = variance
// 	result["blocksCount"] = blocksCount
// 	result["window"] = s.config.LuckWindow
// 	result["totalVariance"] = totalVariance
// 	result["totalBlocksCount"] = totalBlocksCount
// 	result["largeWindow"] = s.config.LargeLuckWindow
// 	return result
// }

// func (s *StratumServer) getBlocksStats() []interface{} {
// 	now := util.MakeTimestamp()
// 	var result []interface{}

// 	s.blocksMu.Lock()
// 	defer s.blocksMu.Unlock()

// 	for k, v := range s.blockStats {
// 		if k >= now-int64(s.luckLargeWindow) {
// 			block := map[string]interface{}{
// 				"height":    v.height,
// 				"hash":      v.hash,
// 				"variance":  v.variance,
// 				"timestamp": k,
// 			}
// 			result = append(result, block)
// 		} else {
// 			delete(s.blockStats, k)
// 		}
// 	}
// 	return result
// }

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
	if pageSize < 10 || page > 200 {
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
	if pageSize < 10 || page > 200 {
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

// func (s *StratumServer) HashrateRank(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
// 	w.WriteHeader(http.StatusOK)
// 	data := make(map[string]interface{})
// 	vars := mux.Vars(r)
// 	num := vars["page"]
// 	size := vars["pageSize"]
// 	if num == "" || size == "" {
// 		data["code"] = -1
// 		data["msg"] = "parameter empty"
// 		data["data"] = ""
// 		_ = json.NewEncoder(w).Encode(data)
// 		return
// 	}
// 	page, err1 := strconv.Atoi(num)
// 	pageSize, err2 := strconv.Atoi(size)
// 	if err1 != nil || err2 != nil {
// 		data["code"] = -1
// 		data["msg"] = "parameter not number"
// 		data["data"] = ""
// 		_ = json.NewEncoder(w).Encode(data)
// 		return
// 	}
// 	if page < 1 {
// 		page = 1
// 	}
// 	if pageSize < 10 {
// 		pageSize = 10
// 	}
// 	start := (page-1)*pageSize + 1
// 	end := start + pageSize - 1
// 	ranks, count, err := util.HashrateRank.GetRanks(start, end)
// 	if err != nil {
// 		data["code"] = -1
// 		data["msg"] = err.Error()
// 		data["data"] = ""
// 		_ = json.NewEncoder(w).Encode(data)
// 		return
// 	}
// 	data["code"] = 0
// 	data["msg"] = "success"
// 	data["page"] = page
// 	data["pageSize"] = pageSize
// 	data["total"] = count
// 	data["data"] = ranks
// 	_ = json.NewEncoder(w).Encode(data)
// }

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
	if pageSize < 10 || page > 200 {
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
	if pageSize < 10 || page > 200 {
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
	if pageSize < 10 || page > 200 {
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
	workers := s.workers.GetWorkers(address)
	if len(workers) == 0 {
		data["code"] = -1
		data["msg"] = "no workers"
		data["data"] = ""
		_ = json.NewEncoder(w).Encode(data)
		return
	}
	var hashrate = make(map[string]float64)
	var hashrate24h = make(map[string]float64)
	window24h := 24 * time.Hour
	for _, w := range workers {
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

type XdagPoolConfig struct {
	PoolIP               string `json:"poolIp"`
	PoolPort             int    `json:"poolPort"`
	NodeIP               string `json:"nodeIp"`
	NodePort             int    `json:"nodePort"`
	GlobalMinerLimit     int    `json:"globalMinerLimit"`
	MaxConnectMinerPerIP int    `json:"maxConnectMinerPerIp"`
	MaxMinerPerAccount   int    `json:"maxMinerPerAccount"`
	PoolFeeRation        string `json:"poolFeeRation"`
	PoolRewardRation     string `json:"poolRewardRation"`
	PoolDirectRation     string `json:"poolDirectRation"`
	PoolFundRation       string `json:"poolFundRation"`
	Threshold            string `json:"threshold"`
}

func (s *StratumServer) XdagPoolConfig(id uint64, params json.RawMessage) jrpc.Response {
	var rec XdagPoolConfig
	s.config.RLock()
	defer s.config.RUnlock()
	rec.PoolIP = s.config.Stratum.Ports[0].Host
	rec.PoolPort = s.config.Stratum.Ports[0].Port
	n := strings.LastIndex(s.config.NodeRpc, ":")
	if n > 0 && n < len(s.config.NodeRpc)-1 {
		port, err := strconv.Atoi(s.config.NodeRpc[n+1:])
		if err == nil {
			rec.NodePort = port
			rec.NodeIP = (s.config.NodeRpc[:n])
		} else {
			rec.NodeIP = s.config.NodeRpc
		}
	} else {
		rec.NodeIP = s.config.NodeRpc
	}

	rec.GlobalMinerLimit = s.config.Stratum.Ports[0].MaxConn
	rec.MaxConnectMinerPerIP = 0
	rec.MaxMinerPerAccount = 0
	rec.PoolDirectRation = fmt.Sprintf("%v", s.config.PayOut.DirectRation)
	rec.PoolFeeRation = fmt.Sprintf("%v", s.config.PayOut.PoolRation)
	rec.PoolFundRation = "0"
	rec.PoolRewardRation = fmt.Sprintf("%v", s.config.PayOut.RewardRation)
	rec.Threshold = fmt.Sprintf("%v", s.config.PayOut.Threshold)

	return jrpc.EncodeResponse(id, rec, nil)
}

type XdagPoolUpdate struct {
	PoolFeeRation    string `json:"poolFeeRation"`
	PoolRewardRation string `json:"poolRewardRation"`
	PoolDirectRation string `json:"poolDirectRation"`
	Threshold        string `json:"threshold"`
}

func (s *StratumServer) XdagUpdatePoolConfig(id uint64, params json.RawMessage) jrpc.Response {

	var args []json.RawMessage
	var rec XdagPoolUpdate

	if err := json.Unmarshal(params, &args); err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}

	if len(args) != 2 {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("params length error"))
	}

	var password string
	err := json.Unmarshal(args[1], &password)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}

	if !util.ValidatePasswd(s.config.AddressEncrypted, password) {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("password error"))
	}

	err = json.Unmarshal(args[0], &rec)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}

	s.config.Lock()
	defer s.config.Unlock()

	dr, err := strconv.ParseFloat(rec.PoolDirectRation, 64)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}
	s.config.PayOut.DirectRation = dr

	fr, err := strconv.ParseFloat(rec.PoolFeeRation, 64)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}
	s.config.PayOut.PoolRation = fr

	rr, err := strconv.ParseFloat(rec.PoolRewardRation, 64)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}
	s.config.PayOut.RewardRation = rr

	th, err := strconv.Atoi(rec.Threshold)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}
	s.config.PayOut.Threshold = int64(th)

	return jrpc.EncodeResponse(id, "Success", nil)
}

type XdagPoolMiners struct {
	Address      string       `json:"address"`
	Status       string       `json:"status"`
	UnpaidShares float64      `json:"unpaidShares"`
	Hashrate     float64      `json:"hashrate"`
	Workers      []XdagWorker `json:"workers"`
}
type XdagWorker struct {
	Address      string  `json:"address"`
	InBound      int     `json:"inBound"`
	OutBound     int     `json:"outBound"`
	UnpaidShares float64 `json:"unpaidShares"`
	Name         string  `json:"name"`
	Hashrate     float64 `json:"hashrate"`
}

func (s *StratumServer) XdagGetPoolWorkers(id uint64, params json.RawMessage) jrpc.Response {
	var rec []XdagPoolMiners
	now := util.MakeTimestamp()
	var minersWorks = make(map[string]map[string]struct{})
	var miners = make(map[string]*XdagPoolMiners)

	for m := range s.miners.Iter() {
		lastBeat := m.Val.getLastBeat()
		n := strings.Index(m.Key, ".")
		if n < 0 {
			continue
		}
		address := m.Key[:n]
		name := m.Key[n+1:]
		workers, ok1 := minersWorks[address]
		if !ok1 {
			minersWorks[address] = make(map[string]struct{})
			minersWorks[address][name] = struct{}{}
			worker := XdagWorker{
				Address:  m.Val.ip,
				Name:     name,
				Hashrate: m.Val.hashrate(s.estimationWindow),
			}
			miner := XdagPoolMiners{
				Address:      address,
				UnpaidShares: s.backend.GetMinerUnpaid(address),
			}
			miner.Hashrate = worker.Hashrate
			miner.Workers = append(miner.Workers, worker)
			miner.Status = "MINER_ARCHIVE"
			if now-lastBeat < (int64(s.timeout) / 1000000) {
				miner.Status = "MINER_ACTIVE"
			}
			miners[address] = &miner
		} else {
			_, ok2 := workers[name]
			if !ok2 {
				minersWorks[address][name] = struct{}{}
				worker := XdagWorker{
					Address:  m.Val.ip,
					Name:     name,
					Hashrate: m.Val.hashrate(s.estimationWindow),
				}
				miners[address].Hashrate += worker.Hashrate
				miners[address].Workers = append(miners[address].Workers, worker)
				if now-lastBeat < (int64(s.timeout) / 1000000) {
					miners[address].Status = "MINER_ACTIVE"
				}
			}
		}

	}

	rec = append(rec, XdagPoolMiners{
		Address: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Status:  "fee",
		Workers: make([]XdagWorker, 0),
	})
	for _, v := range miners {
		rec = append(rec, *v)
	}

	return jrpc.EncodeResponse(id, rec, nil)
}

type MinerAccount struct {
	Address      string  `json:"address"`
	Timestamp    int64   `json:"timestamp"`
	TotalReward  float64 `json:"total_reward"`
	TotalPayment float64 `json:"total_payment"`
	TotalUnpaid  float64 `json:"total_unpaid"`
}

func (s *StratumServer) XdagMinerAccount(id uint64, params json.RawMessage) jrpc.Response {
	now := util.MakeTimestamp()

	var args []json.RawMessage
	if err := json.Unmarshal(params, &args); err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}
	if len(args) != 1 {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("params length error"))
	}

	var address string
	err := json.Unmarshal(args[0], &address)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}

	if address == "" || !util.ValidateAddress(address) {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("addres is empty or invalid"))
	}
	reward, payment, unpaid, err := s.backend.GetMinerAccount(address)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}

	data := MinerAccount{
		Address:      address,
		Timestamp:    now,
		TotalReward:  reward,
		TotalPayment: payment,
		TotalUnpaid:  unpaid,
	}
	return jrpc.EncodeResponse(id, data, nil)
}

type MinerHashrate struct {
	Address          string           `json:"address"`
	Timestamp        int64            `json:"timestamp"`
	TotalHashrate    float64          `json:"total_hashrate"`
	TotalHashrate24h float64          `json:"total_hashrate24h"`
	TotalOnline      int              `json:"total_online"`
	Hashrate         []WorkerHashrate `json:"hashrate"`
}

type WorkerHashrate struct {
	Name          string  `json:"name"`
	Hashrate      float64 `json:"hashrate"`
	Hashrate24h   float64 `json:"hashrate24h"`
	LastBeat      int64   `json:"lastBeat"`
	ValidShares   int64   `json:"validShares"`
	StaleShares   int64   `json:"staleShares"`
	InvalidShares int64   `json:"invalidShares"`
	Accepts       int64   `json:"accepts"`
	Rejects       int64   `json:"rejects"`
	Ip            string  `json:"ip"`
	Warning       bool    `json:"warning"`
	Timeout       bool    `json:"timeout"`
}

func (s *StratumServer) XdagMinerHashrate(id uint64, params json.RawMessage) jrpc.Response {
	now := util.MakeTimestamp()

	var args []json.RawMessage
	if err := json.Unmarshal(params, &args); err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}
	if len(args) != 1 {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("params length error"))
	}

	var address string
	err := json.Unmarshal(args[0], &address)
	if err != nil {
		return jrpc.EncodeResponse(id, struct{}{}, err)
	}

	if address == "" || !util.ValidateAddress(address) {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("addres is empty or invalid"))
	}
	workers := s.workers.GetWorkers(address)
	if len(workers) == 0 {
		return jrpc.EncodeResponse(id, struct{}{}, errors.New("no workers"))
	}

	var result []WorkerHashrate
	window24h := 24 * time.Hour

	minerHashrate := MinerHashrate{
		Address:   address,
		Timestamp: now,
	}

	dataChan := make(chan WorkerHashrate, len(workers))

	for _, w := range workers {
		go func(adress, w string) {
			m, ok := s.miners.Get(address + "." + w)
			if !ok {
				dataChan <- WorkerHashrate{}
				return
			}

			lastBeat := m.getLastBeat()
			hashrate := m.hashrate(s.estimationWindow)
			hashrate24h := m.hashrate(window24h)
			stats := WorkerHashrate{
				Name:          w,
				Hashrate:      hashrate,
				Hashrate24h:   hashrate24h,
				LastBeat:      lastBeat,
				ValidShares:   atomic.LoadInt64(&m.validShares),
				StaleShares:   atomic.LoadInt64(&m.staleShares),
				InvalidShares: atomic.LoadInt64(&m.invalidShares),
				Accepts:       atomic.LoadInt64(&m.accepts),
				Rejects:       atomic.LoadInt64(&m.rejects),
			}
			if !s.config.Frontend.HideIP {
				stats.Ip = m.ip
			} else {
				stats.Ip = "-"

			}

			if now-lastBeat > (int64(s.timeout/2) / 1000000) {
				stats.Warning = true
			}
			if now-lastBeat > (int64(s.timeout) / 1000000) {
				stats.Timeout = true
			}
			dataChan <- stats
		}(address, w)
	}

	i := 0
	var hashrate float64
	var hashrate24h float64
	totalOnline := 0

	for {
		select {
		case <-time.After(10 * time.Second):
			return jrpc.EncodeResponse(id, struct{}{}, errors.New("timeout"))
		case v := <-dataChan:
			if v.Name != "" {
				result = append(result, v)
			}
			i++
			hashrate += v.Hashrate
			hashrate24h += v.Hashrate24h
			if !v.Timeout {
				totalOnline++
			}

			if i == len(workers) {
				minerHashrate.TotalHashrate = hashrate
				minerHashrate.TotalHashrate24h = hashrate24h
				minerHashrate.TotalOnline = totalOnline
				minerHashrate.Hashrate = result
				return jrpc.EncodeResponse(id, minerHashrate, nil)
			}
		}
	}

}

type PoolHashrate struct {
	Hashrate    float64 `json:"hashrate"`
	Hashrate24h float64 `json:"hashrate24h"`
	Total       int     `json:"total"`
	TotalOnline int     `json:"total_online"`
}

func (s *StratumServer) XdagPoolHashrate(id uint64, params json.RawMessage) jrpc.Response {
	now := util.MakeTimestamp()
	totalhashrate := float64(0)
	totalhashrate24h := float64(0)
	total := 0
	totalOnline := 0
	window24h := 24 * time.Hour

	for m := range s.miners.Iter() {
		lastBeat := m.Val.getLastBeat()
		hashrate := m.Val.hashrate(s.estimationWindow)
		hashrate24h := m.Val.hashrate(window24h)
		totalhashrate += hashrate
		totalhashrate24h += hashrate24h
		total++
		if now-lastBeat <= (int64(s.timeout) / 1000000) {
			totalOnline++
		}
	}
	return jrpc.EncodeResponse(id, PoolHashrate{
		Hashrate:    totalhashrate,
		Hashrate24h: totalhashrate24h,
		Total:       total,
		TotalOnline: totalOnline,
	}, nil)
}
