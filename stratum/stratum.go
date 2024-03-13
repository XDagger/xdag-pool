package stratum

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XDagger/xdagpool/kvstore"

	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
)

type StratumServer struct {
	luckWindow int64
	// luckLargeWindow int64
	roundShares   int64
	blockStats    map[int64]blockEntry
	config        *pool.Config
	miners        MinersMap
	workers       WorkersMap
	blockTemplate atomic.Value
	// upWsClient          *ws.Socket
	timeout          time.Duration
	estimationWindow time.Duration
	purgeWindow      time.Duration
	// purgeLargeWindow time.Duration

	blocksMu   sync.RWMutex
	sessionsMu sync.RWMutex
	sessions   map[*Session]struct{}

	backend *kvstore.KvClient

	upstreamsStates []bool

	maxConcurrency int
}

type blockEntry struct {
	height   int64
	variance float64
	hash     string
}

type Endpoint struct {
	jobSequence uint64
	config      *pool.Port
	difficulty  *big.Int
	instanceId  []byte
	extraNonce  uint32
	targetHex   string
}

type Session struct {
	lastJobHash atomic.Value
	sync.Mutex

	conn    *net.TCPConn
	tlsConn *tls.Conn

	enc *json.Encoder
	ip  string

	login   string
	address string
	id      string
	uid     string

	endpoint  *Endpoint
	validJobs []*Job
}

const (
	MaxReqSize = 10 * 1024
)

func NewStratum(cfg *pool.Config, backend *kvstore.KvClient, msgChan chan pool.Message) *StratumServer {
	stratum := &StratumServer{config: cfg, backend: backend, blockStats: make(map[int64]blockEntry),
		maxConcurrency: cfg.Threads}

	// stratum.upWsClient = ws.NewRpcClient(cfg.NodeWs, cfg.WsSsl)
	// util.Info.Printf("Upstream ws: %s => %s", cfg.NodeName, cfg.NodeWs)

	stratum.miners = NewMinersMap()
	stratum.workers = NewWorkersMap()
	stratum.sessions = make(map[*Session]struct{})

	timeout, _ := time.ParseDuration(cfg.Stratum.Timeout)
	stratum.timeout = timeout

	estimationWindow, _ := time.ParseDuration(cfg.EstimationWindow)
	stratum.estimationWindow = estimationWindow

	purgeWindow, _ := time.ParseDuration(cfg.PurgeWindow)
	stratum.purgeWindow = purgeWindow

	// purgeLargeWindow, _ := time.ParseDuration(cfg.PurgeLargeWindow)
	// stratum.purgeLargeWindow = purgeLargeWindow

	luckWindow, _ := time.ParseDuration(cfg.LuckWindow)
	stratum.luckWindow = int64(luckWindow / time.Millisecond)
	// luckLargeWindow, _ := time.ParseDuration(cfg.LargeLuckWindow)
	// stratum.luckLargeWindow = int64(luckLargeWindow / time.Millisecond)

	purgeIntv := util.MustParseDuration(cfg.PurgeInterval)
	purgeTimer := time.NewTimer(purgeIntv)
	util.Info.Printf("Set purge interval to %v", purgeIntv)

	// purge stale
	go func() {
		for {
			select {
			case <-purgeTimer.C:
				stratum.purgeStale()
				purgeTimer.Reset(purgeIntv)
			}
		}
	}()

	go func(c chan pool.Message) {
		for m := range c {
			if m.MsgType == 1 { // task
				stratum.refreshBlockTemplate(m.MsgContent)
			} else if m.MsgType == 3 { // rewards
				stratum.processRewards(m.MsgContent)
			}
		}
	}(msgChan)

	return stratum
}

func NewEndpoint(cfg *pool.Port) *Endpoint {
	e := &Endpoint{config: cfg}
	e.instanceId = make([]byte, 4)
	_, err := rand.Read(e.instanceId) // random instance id
	if err != nil {
		util.Error.Fatalf("Can't seed with random bytes: %v", err)
	}
	e.targetHex = util.GetTargetHex(e.config.Difficulty) // default 000037EC8EC25E6D
	e.difficulty = big.NewInt(e.config.Difficulty)       //default 300000
	return e
}

func (s *StratumServer) Listen() {
	for _, port := range s.config.Stratum.Ports {
		go func(cfg pool.Port) {
			e := NewEndpoint(&cfg)
			e.Listen(s)
		}(port)
	}
}

func (s *StratumServer) ListenTLS() {
	for _, portTls := range s.config.StratumTls.Ports {
		go func(cfg pool.Port, t pool.StratumTls) {
			e := NewEndpoint(&cfg)
			e.ListenTLS(s, t)
		}(portTls, s.config.StratumTls)
	}
}

func (e *Endpoint) Listen(s *StratumServer) {
	bindAddr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	addr, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		util.Error.Fatalf("Error: %v", err)
	}
	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		util.Error.Fatalf("Error: %v", err)
	}
	defer server.Close()

	util.Info.Printf("Stratum listening on %s", bindAddr)
	accept := make(chan int, e.config.MaxConn)
	n := 0

	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			continue
		}
		_ = conn.SetKeepAlive(true)
		// ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		cs := &Session{conn: conn, ip: conn.RemoteAddr().String(), enc: json.NewEncoder(conn), endpoint: e, address: s.config.Address}
		n += 1

		accept <- n
		go func() {
			s.handleClient(cs, e)
			<-accept
		}()
	}
}

func (e *Endpoint) ListenTLS(s *StratumServer, t pool.StratumTls) {
	bindAddr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)

	cert, err := tls.LoadX509KeyPair(t.TlsCert, t.TlsKey)
	if err != nil {
		util.Error.Fatalf("Error: %v", err)
	}

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{cert}
	tlsConfig.Time = time.Now
	tlsConfig.Rand = rand.Reader

	server, err := tls.Listen("tcp", bindAddr, tlsConfig)
	if err != nil {
		util.Error.Fatalf("Error: %v", err)
	}
	defer server.Close()

	util.Info.Printf("Stratum TLS listening on %s", bindAddr)
	accept := make(chan int, e.config.MaxConn)
	n := 0

	for {
		conn, err := server.Accept()
		if err != nil {
			continue
		}
		util.Info.Printf("Accept Stratum TLS Connection from: %s, to: %s", conn.RemoteAddr().String(), conn.LocalAddr().String())

		ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

		tlsConn, ok := conn.(*tls.Conn)
		if ok {
			state := tlsConn.ConnectionState()
			for _, v := range state.PeerCertificates {
				pKIXPublicKey, _ := x509.MarshalPKIXPublicKey(v.PublicKey)
				util.Info.Printf("x509.MarshalPKIXPublicKey: %v", pKIXPublicKey)
			}
		} else {
			_ = conn.Close()
			continue
		}

		cs := &Session{tlsConn: tlsConn, ip: ip, enc: json.NewEncoder(conn), endpoint: e, address: s.config.Address}
		n += 1

		accept <- n
		go func() {
			s.handleTLSClient(cs, e)
			<-accept
		}()
	}
}

func (s *StratumServer) handleClient(cs *Session, e *Endpoint) {
	connbuff := bufio.NewReaderSize(cs.conn, MaxReqSize)
	s.setDeadline(cs.conn)

	for {
		data, isPrefix, err := connbuff.ReadLine()
		if isPrefix {
			util.Info.Println("Socket flood detected from", cs.ip)
			break
		} else if err == io.EOF {
			if cs.login == "" && cs.id == "" {
				util.Info.Println("Client disconnected from", cs.ip)
			} else {
				util.Info.Printf("Client disconnected: Address: [%s] | Name: [%s] | IP: [%s]", cs.login, cs.id, cs.ip)
			}
			break
		} else if err != nil {
			util.Error.Printf("Error reading from socket: %v | Address: [%s] | Name: [%s] | IP: [%s]", err, cs.login, cs.id, cs.ip)
			break
		}

		// NOTICE: cpuminer-multi sends junk newlines, so we demand at least 1 byte for decode
		// NOTICE: Ns*CNMiner.exe will send malformed JSON on very low diff, not sure we should handle this
		if len(data) > 1 {
			var req JSONRpcReq
			err = json.Unmarshal(data, &req)
			if err != nil {
				util.Error.Printf("Malformed request from %s: %v", cs.ip, err)
				break
			}
			s.setDeadline(cs.conn)
			err = cs.handleMessage(s, e, &req)
			if err != nil {
				util.Error.Printf("handleTCPMessage: %v", err)
				break
			}
		}
	}
	s.removeMiner(cs.uid)
	s.removeSession(cs)
	_ = cs.conn.Close()
}

func (s *StratumServer) handleTLSClient(cs *Session, e *Endpoint) {
	connbuff := bufio.NewReaderSize(cs.tlsConn, MaxReqSize)
	s.setTLSDeadline(cs.tlsConn)

	for {
		data, isPrefix, err := connbuff.ReadLine()
		if isPrefix {
			util.Info.Println("Socket flood detected from", cs.ip)
			break
		} else if err == io.EOF {
			if cs.login == "" && cs.id == "" {
				util.Info.Println("Client disconnected from", cs.ip)
			} else {
				util.Info.Printf("Client disconnected: Address: [%s] | Name: [%s] | IP: [%s]", cs.login, cs.id, cs.ip)
			}
			break
		} else if err != nil {
			util.Error.Printf("Error reading from socket: %v | Address: [%s] | Name: [%s] | IP: [%s]", err, cs.login, cs.id, cs.ip)
			break
		}

		// NOTICE: cpuminer-multi sends junk newlines, so we demand at least 1 byte for decode
		// NOTICE: Ns*CNMiner.exe will send malformed JSON on very low diff, not sure we should handle this
		if len(data) > 1 {
			var req JSONRpcReq
			err = json.Unmarshal(data, &req)
			if err != nil {
				util.Error.Printf("Malformed request from %s: %v", cs.ip, err)
				break
			}
			s.setTLSDeadline(cs.tlsConn)
			err = cs.handleMessage(s, e, &req)
			if err != nil {
				util.Error.Printf("handleTCPMessage: %v", err)
				break
			}
		}
	}
	s.removeMiner(cs.uid)
	s.removeSession(cs)
	_ = cs.tlsConn.Close()
}

func (cs *Session) handleMessage(s *StratumServer, e *Endpoint, req *JSONRpcReq) error {
	if req.Id == nil {
		err := fmt.Errorf("server disconnect request")
		util.Error.Println(err)
		return err
	} else if req.Params == nil {
		err := fmt.Errorf("server RPC request params")
		util.Error.Println(err)
		return err
	}

	// Handle RPC methods
	switch req.Method {

	case "login":
		var params LoginParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			util.Error.Println("Unable to parse params: login")
			return err
		}
		reply, errReply := s.handleLoginRPC(cs, &params)
		if errReply != nil {
			return cs.sendError(req.Id, errReply, true)
		}
		return cs.sendResult(req.Id, &reply)
	case "getjob":
		var params GetJobParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			util.Error.Println("Unable to parse params: getjob")
			return err
		}
		reply, errReply := s.handleGetJobRPC(cs, &params)
		if errReply != nil {
			return cs.sendError(req.Id, errReply, true)
		}
		return cs.sendResult(req.Id, &reply)
	case "submit":
		var params SubmitParams
		err := json.Unmarshal(*req.Params, &params)
		if err != nil {
			util.Error.Println("Unable to parse params: submit")
			return err
		}
		reply, errReply := s.handleSubmitRPC(cs, &params)
		if errReply != nil {
			return cs.sendError(req.Id, errReply, false)
		}
		return cs.sendResult(req.Id, &reply)
	case "keepalived":
		return cs.sendResult(req.Id, &StatusReply{Status: "KEEPALIVED"})
	default:
		errReply := s.handleUnknownRPC(req)
		return cs.sendError(req.Id, errReply, true)
	}
}

func (cs *Session) sendResult(id *json.RawMessage, result interface{}) error {
	cs.Lock()
	defer cs.Unlock()
	message := JSONRpcResp{Id: id, Version: "2.0", Error: nil, Result: result}
	return cs.enc.Encode(&message)
}

func (cs *Session) pushMessage(method string, params interface{}) error {
	cs.Lock()
	defer cs.Unlock()
	message := JSONPushMessage{Version: "2.0", Method: method, Params: params}
	return cs.enc.Encode(&message)
}

func (cs *Session) sendError(id *json.RawMessage, reply *ErrorReply, drop bool) error {
	cs.Lock()
	defer cs.Unlock()
	message := JSONRpcResp{Id: id, Version: "2.0", Error: reply}
	err := cs.enc.Encode(&message)
	if err != nil {
		return err
	}
	if drop {
		return fmt.Errorf("server disconnect request")
	}
	return nil
}

func (s *StratumServer) setDeadline(conn *net.TCPConn) {
	_ = conn.SetDeadline(time.Now().Add(s.timeout))
}

func (s *StratumServer) setTLSDeadline(conn *tls.Conn) {
	_ = conn.SetDeadline(time.Now().Add(s.timeout))
}

func (s *StratumServer) registerSession(cs *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	s.sessions[cs] = struct{}{}
}

func (s *StratumServer) removeSession(cs *Session) {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()
	delete(s.sessions, cs)
}

func (s *StratumServer) isActive(cs *Session) bool {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()
	_, exist := s.sessions[cs]
	return exist
}

func (s *StratumServer) registerMiner(miner *Miner) {
	s.miners.Set(miner.uid, miner)
}

func (s *StratumServer) removeMiner(uid string) {
	s.miners.Remove(uid)
}

func (s *StratumServer) currentBlockTemplate() *BlockTemplate {
	if t := s.blockTemplate.Load(); t != nil {
		return t.(*BlockTemplate)
	}
	return nil
}

func (s *StratumServer) purgeStale() {
	start := time.Now()
	total, err := s.backend.PurgeRecords(s.purgeWindow)
	if err != nil {
		util.Error.Println("Failed to purge  data from backend:", err)
	} else {
		util.Info.Printf("Purged stale stats from backend, %v records affected, elapsed time %v", total, time.Since(start))
	}
}
