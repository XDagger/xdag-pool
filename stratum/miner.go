package stratum

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/ws"
	"github.com/XDagger/xdagpool/xdago/base58"
)

type Job struct {
	jobHash string
	sync.RWMutex
	id          string
	extraNonce  uint32
	submissions map[string]struct{}
}

type Miner struct {
	lastBeat      int64
	startedAt     int64
	validShares   int64
	invalidShares int64
	staleShares   int64
	accepts       int64
	rejects       int64
	shares        map[int64]int64
	sync.RWMutex
	id  string
	ip  string
	uid string

	maxConcurrency int
}

func (job *Job) submit(nonce string) bool {
	job.Lock()
	defer job.Unlock()
	if _, exist := job.submissions[nonce]; exist {
		return true
	}
	job.submissions[nonce] = struct{}{}
	return false
}

func NewMiner(uid, id string, ip string, maxConcurrency int) *Miner {
	shares := make(map[int64]int64)
	return &Miner{uid: uid, id: id, ip: ip, shares: shares, maxConcurrency: maxConcurrency}
}

func (cs *Session) getJob(t *BlockTemplate) *JobReplyData {
	hash := cs.lastJobHash.Swap(t.jobHash)

	if hash == t.jobHash {
		return &JobReplyData{}
	}

	extraNonce := atomic.AddUint32(&cs.endpoint.extraNonce, 1) // increase extraNonce
	blob := t.nextBlob(cs.address, extraNonce, cs.endpoint.instanceId)
	id := atomic.AddUint64(&cs.endpoint.jobSequence, 1)
	job := &Job{
		id:         strconv.FormatUint(id, 10),
		extraNonce: extraNonce,
		jobHash:    t.jobHash,
	}
	job.submissions = make(map[string]struct{})
	cs.pushJob(job)
	reply := &JobReplyData{Algo: "rx/0", JobId: job.id, Blob: blob, Target: cs.endpoint.targetHex,
		// Height:   t.height,
		SeedHash: hex.EncodeToString(t.seedHash)}
	return reply
}

func (cs *Session) pushJob(job *Job) {
	cs.Lock()
	defer cs.Unlock()
	cs.validJobs = append(cs.validJobs, job)

	if len(cs.validJobs) > 4 {
		cs.validJobs = cs.validJobs[1:]
	}
}

func (cs *Session) findJob(id string) *Job {
	cs.Lock()
	defer cs.Unlock()
	for _, job := range cs.validJobs {
		if job.id == id {
			return job
		}
	}
	return nil
}

func (m *Miner) heartbeat() {
	now := util.MakeTimestamp()
	atomic.StoreInt64(&m.lastBeat, now)
}

func (m *Miner) getLastBeat() int64 {
	return atomic.LoadInt64(&m.lastBeat)
}

func (m *Miner) storeShare(diff int64) {
	now := util.MakeTimestamp() / 1000
	m.Lock()
	m.shares[now] += diff
	m.Unlock()
}

func (m *Miner) hashrate(estimationWindow time.Duration) float64 {
	now := util.MakeTimestamp() / 1000
	totalShares := int64(0)
	window := int64(estimationWindow / time.Second)
	boundary := now - m.startedAt

	if boundary > window {
		boundary = window
	}

	m.Lock()
	for k, v := range m.shares {
		if k < now-86400 {
			delete(m.shares, k)
		} else if k >= now-window {
			totalShares += v
		}
	}
	m.Unlock()
	return float64(totalShares) / float64(boundary)
}

func (m *Miner) processShare(s *StratumServer, cs *Session, job *Job, t *BlockTemplate,
	nonce string, result string) bool {
	// r := s.rpc()

	// 32 bytes (reserved) = 20 bytes (pool owner wallet address) + 4 bytes (extraNonce) + 4 bytes (instanceId) + 4 bytes (share nonce)
	shareBuff := make([]byte, len(t.buffer)*2)

	copy(shareBuff, t.buffer)

	addr, _, _ := base58.ChkDec(s.config.Address)
	copy(shareBuff[len(t.buffer)+12:], addr[:len(addr)-4]) // without checksum bytes

	enonce := make([]byte, 4)
	binary.BigEndian.PutUint32(enonce, job.extraNonce)
	copy(shareBuff[len(t.buffer):], enonce[:]) // 4 bytes

	copy(shareBuff[len(t.buffer)+4:], cs.endpoint.instanceId[:3]) // 3 bytes of instanceId

	nonceBuff, _ := hex.DecodeString(nonce) // 32bits (4 bytes) share nonce sent by miner
	copy(shareBuff[len(t.buffer)+7:], nonceBuff[:4])

	shareBuff[len(t.buffer)+11] = cs.endpoint.instanceId[3] // 1 bytes of instanceId

	hashBytes := util.RxHash(shareBuff)

	if hex.EncodeToString(hashBytes) != result {
		util.Error.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		util.ShareLog.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		atomic.AddInt64(&m.invalidShares, 1)
		return false
	}

	hashDiff, ok := util.GetHashDifficulty(hashBytes)
	if !ok {
		util.Error.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		util.ShareLog.Printf("Bad hash from miner %v.%v@%v", cs.login, cs.id, cs.ip)
		atomic.AddInt64(&m.invalidShares, 1)
		return false
	}

	block := hashDiff.Cmp(cs.endpoint.difficulty) >= 0

	// nonceHex := hex.EncodeToString(nonceBuff)
	// extraHex := hex.EncodeToString(enonce)
	// instanceIdHex := hex.EncodeToString(cs.endpoint.instanceId)
	// paramIn := []string{nonceHex, extraHex, instanceIdHex}
	share := hex.EncodeToString(shareBuff[len(t.buffer):])

	if block {
		fmt.Println("rxhash:", util.GetRxHashStr(hashBytes))
		shareU64 := binary.LittleEndian.Uint64(hashBytes[24:32]) // share hash high 8 bytes to uint64 used to compare hash
		minShare := s.backend.IsMinShare(t.jobHash, cs.login, share, shareU64)

		// block
		if minShare {
			err := ws.Submit(hex.EncodeToString(shareBuff[:32]),
				hex.EncodeToString(shareBuff[32:]), t.taskIndex)
			if err != nil {
				// atomic.AddInt64(&m.rejects, 1)
				// atomic.AddInt64(&r.Rejects, 1)
				util.Error.Printf("Block rejected at hash %s: %v", t.jobHash, err)
				util.BlockLog.Printf("Block rejected at hash %s: %v", t.jobHash, err)
			}
		}
		// _, err := r.SubmitBlock(hex.EncodeToString(shareBuff)) //TODO: send pool address + share
		// if err != nil {
		// 	atomic.AddInt64(&m.rejects, 1)
		// 	atomic.AddInt64(&r.Rejects, 1)
		// 	util.Error.Printf("Block rejected at height %d: %v", t.height, err)
		// 	util.BlockLog.Printf("Block rejected at height %d: %v", t.height, err)
		// } else {
		// blockFastHash := hex.EncodeToString(util.FastHash(shareBuff))
		// now := util.MakeTimestamp()
		// roundShares := atomic.SwapInt64(&s.roundShares, 0)
		// ratio := float64(roundShares) / float64(t.diffInt64)
		// s.blocksMu.Lock()
		// s.blockStats[now] = blockEntry{height: t.height, hash: blockFastHash, variance: ratio}
		// s.blocksMu.Unlock()
		// atomic.AddInt64(&m.accepts, 1)
		// atomic.AddInt64(&r.Accepts, 1)
		// atomic.StoreInt64(&r.LastSubmissionAt, now)

		exist, err := s.backend.WriteBlock(cs.login, cs.id, share, cs.endpoint.difficulty.Int64(),
			shareU64, t.timestamp, t.jobHash)
		if exist {
			ms := util.MakeTimestamp()
			ts := ms / 1000

			err := s.backend.WriteInvalidShare(ms, ts, cs.login, cs.id, cs.endpoint.difficulty.Int64())
			if err != nil {
				util.Error.Println("Failed to insert invalid share data into backend:", err)
			}
			return false
		}
		if err != nil {
			util.Error.Println("Failed to insert block candidate into backend:", err)
			util.BlockLog.Println("Failed to insert block candidate into backend:", err)
		} else {
			util.Info.Printf("Inserted block %s to backend", t.jobHash)
			util.BlockLog.Printf("Inserted block %s to backend", t.jobHash)
		}

		util.Info.Printf("Block %s found by miner %v.%v@%v", t.jobHash, cs.login, cs.id, cs.ip)
		util.BlockLog.Printf("Block %s found by miner %v.%v@%v", t.jobHash, cs.login, cs.id, cs.ip)

		// Immediately refresh current BT and send new jobs
		// s.refreshBlockTemplate(true)
		// }
	} else {
		// invalid share
		ms := util.MakeTimestamp()
		ts := ms / 1000
		err := s.backend.WriteRejectShare(ms, ts, cs.login, cs.id, cs.endpoint.difficulty.Int64())
		if err != nil {
			util.Error.Println("Failed to insert reject share data into backend:", err)
			return false
		}
		util.Error.Printf("Rejected low difficulty share of %v from %v.%v@%v", hashDiff, cs.login, cs.id, cs.ip)
		util.ShareLog.Printf("Rejected low difficulty share of %v from %v.%v@%v", hashDiff, cs.login, cs.id, cs.ip)
		atomic.AddInt64(&m.invalidShares, 1)
		return false
	}

	atomic.AddInt64(&s.roundShares, cs.endpoint.config.Difficulty)
	atomic.AddInt64(&m.validShares, 1)
	m.storeShare(cs.endpoint.config.Difficulty)

	util.Info.Printf("Valid share of %v at difficulty %v from %v.%v@%v", hashDiff, cs.endpoint.config.Difficulty, cs.login, cs.id, cs.ip)
	util.ShareLog.Printf("Valid share of %v at difficulty %v from %v.%v@%v", hashDiff, cs.endpoint.config.Difficulty, cs.login, cs.id, cs.ip)
	return true
}
