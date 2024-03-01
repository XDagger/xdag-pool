package stratum

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XDagger/xdagpool/payouts"
	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
)

var noncePattern *regexp.Regexp

const defaultWorkerId = "0"

func init() {
	noncePattern, _ = regexp.Compile("^[0-9a-f]{8}$")
}

func (s *StratumServer) handleLoginRPC(cs *Session, params *LoginParams) (*JobReply, *ErrorReply) {
	address, id := extractWorkerId(params.Login, params.Pass)
	if !util.ValidateAddress(address) {
		util.Error.Printf("Invalid address %s used for login by %s", address, cs.ip)
		return nil, &ErrorReply{Code: -1, Message: "Invalid address used for login"}
	}

	t := s.currentBlockTemplate()
	if t == nil {
		return nil, &ErrorReply{Code: -1, Message: "Job not ready"}
	}

	cs.login = address
	cs.id = id
	cs.uid = address + "." + id

	miner, ok := s.miners.Get(cs.uid)
	if !ok {
		miner = NewMiner(cs.uid, cs.id, cs.ip, s.maxConcurrency)
		s.registerMiner(miner)
	}

	ids, ok := s.workers.Get(address)
	if !ok {
		m := new(sync.Map)
		m.Store(id, struct{}{})
		s.workers.Set(address, m)
	} else {
		ids.Store(id, struct{}{})
	}

	util.Info.Printf("Miner connected %s.%s@%s", cs.login, cs.id, cs.ip)

	s.registerSession(cs)
	miner.heartbeat()

	return &JobReply{Id: cs.id, Job: cs.getJob(t), Status: "OK"}, nil
}

func (s *StratumServer) handleGetJobRPC(cs *Session, params *GetJobParams) (*JobReplyData, *ErrorReply) {
	miner, ok := s.miners.Get(cs.login + "." + params.Id)
	if !ok {
		return nil, &ErrorReply{Code: -1, Message: "Unauthenticated"}
	}
	t := s.currentBlockTemplate()
	if t == nil {
		return nil, &ErrorReply{Code: -1, Message: "Job not ready"}
	}
	miner.heartbeat()
	return cs.getJob(t), nil
}

func (s *StratumServer) handleSubmitRPC(cs *Session, params *SubmitParams) (*StatusReply, *ErrorReply) {
	miner, ok := s.miners.Get(cs.login + "." + params.Id)
	if !ok {
		return nil, &ErrorReply{Code: -1, Message: "Unauthenticated"}
	}
	miner.heartbeat()

	job := cs.findJob(params.JobId)
	if job == nil {
		return nil, &ErrorReply{Code: -1, Message: "Invalid job id"}
	}

	if !noncePattern.MatchString(params.Nonce) {
		return nil, &ErrorReply{Code: -1, Message: "Malformed nonce"}
	}
	nonce := strings.ToLower(params.Nonce)
	exist := job.submit(nonce)
	if exist {
		atomic.AddInt64(&miner.invalidShares, 1)
		return nil, &ErrorReply{Code: -1, Message: "Duplicate share"}
	}

	t := s.currentBlockTemplate()
	if job.jobHash != t.jobHash {
		util.Error.Printf("Stale share for job %s from %s.%s@%s", job.jobHash, cs.login, cs.id, cs.ip)
		util.ShareLog.Printf("Stale share for job %s from %s.%s@%s", job.jobHash, cs.login, cs.id, cs.ip)
		atomic.AddInt64(&miner.staleShares, 1)
		return nil, &ErrorReply{Code: -1, Message: "Block expired"}
	}

	validShare := miner.processShare(s, cs, job, t, nonce, params.Result)
	if !validShare {
		return nil, &ErrorReply{Code: -1, Message: "Low difficulty share"}
	}
	return &StatusReply{Status: "OK"}, nil
}

func (s *StratumServer) handleUnknownRPC(req *JSONRpcReq) *ErrorReply {
	util.Error.Printf("Unknown RPC method: %v", req)
	return &ErrorReply{Code: -1, Message: "Invalid method"}
}

func (s *StratumServer) broadcastNewJobs() {
	t := s.currentBlockTemplate()
	if t == nil {
		return
	}
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	count := len(s.sessions)
	util.Info.Printf("Broadcasting new jobs to %d miners", count)

	start := time.Now()
	bcast := make(chan int, 1024)
	n := 0

	for m := range s.sessions {
		n++
		bcast <- n
		go func(cs *Session) {
			reply := cs.getJob(t)
			err := cs.pushMessage("job", &reply)
			<-bcast
			if err != nil {
				util.Error.Printf("Job transmit error to %s: %v", cs.ip, err)
				s.removeSession(cs)
			} else {
				if cs.tlsConn != nil {
					s.setTLSDeadline(cs.tlsConn)
				}
				if cs.conn != nil {
					s.setDeadline(cs.conn)
				}
			}
		}(m)
	}
	util.Info.Printf("Jobs broadcast finished %s", time.Since(start))
}

func (s *StratumServer) refreshBlockTemplate(msg json.RawMessage) {
	newBlock := s.fetchBlockTemplate(msg)
	if newBlock {
		s.broadcastNewJobs()
	}
}

func (s *StratumServer) processRewards(msg json.RawMessage) {
	var rewards []pool.XdagjReward
	err := json.Unmarshal(msg, &rewards)
	if err == nil {
		for _, v := range rewards {
			payouts.ProcessReward(s.config, s.backend, v)
		}
	} else {
		util.Error.Println("unmarshal rewards error", err)
	}

}

func extractWorkerId(loginWorkerPair, pass string) (string, string) {
	parts := strings.SplitN(loginWorkerPair, ".", 2)
	if len(parts) > 1 {
		return parts[0], parts[1]
	} else if len(pass) > 0 && pass != "x" {
		return loginWorkerPair, pass
	}
	return loginWorkerPair, defaultWorkerId
}
