package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/goji/httpauth"
	"github.com/gorilla/mux"
	"golang.org/x/term"

	"github.com/XDagger/xdagpool/ws"
	"github.com/XDagger/xdagpool/xdago/base58"
	"github.com/XDagger/xdagpool/xdago/cryptography"
	bip "github.com/XDagger/xdagpool/xdago/wallet"

	"github.com/XDagger/xdagpool/kvstore"
	"github.com/XDagger/xdagpool/payouts"
	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/stratum"
	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/xdago/common"
)

var (
	LatestTag           = ""
	LatestTagCommitSHA1 = ""
	LatestCommitSHA1    = ""
	BuildTime           = ""
	ReleaseType         = ""
)

var cfg pool.Config
var backend *kvstore.KvClient = nil
var msgChan chan pool.Message

func startStratum() {
	if cfg.Threads > 0 {
		runtime.GOMAXPROCS(cfg.Threads)
		util.Info.Printf("Running with %v threads", cfg.Threads)
	} else {
		n := runtime.NumCPU()
		runtime.GOMAXPROCS(n)
		util.Info.Printf("Running with default %v threads", n)
	}

	s := stratum.NewStratum(&cfg, backend, msgChan)
	if cfg.Frontend.Enabled {
		go startFrontend(&cfg, s)
	}

	if cfg.StratumTls.Enabled {
		s.ListenTLS()
	}

	if cfg.Stratum.Enabled {
		s.Listen()
	}

	quit := make(chan bool)
	<-quit
}

func startFrontend(cfg *pool.Config, s *stratum.StratumServer) {
	r := mux.NewRouter()
	r.HandleFunc("/stats", s.StatsIndex)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./www/")))
	var err error
	if len(cfg.Frontend.Password) > 0 {
		auth := httpauth.SimpleBasicAuth(cfg.Frontend.Login, cfg.Frontend.Password)
		err = http.ListenAndServe(cfg.Frontend.Listen, auth(r))
	} else {
		err = http.ListenAndServe(cfg.Frontend.Listen, r)
	}
	if err != nil {
		util.Error.Fatal(err)
	}
}

//func startNewrelic() {
//	// Run NewRelic
//	if cfg.NewrelicEnabled {
//		nr := gorelic.NewAgent()
//		nr.Verbose = cfg.NewrelicVerbose
//		nr.NewrelicLicense = cfg.NewrelicKey
//		nr.NewrelicName = cfg.NewrelicName
//		nr.Run()
//	}
//}

func readConfig(cfg *pool.Config) {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	configFileName, _ = filepath.Abs(configFileName)
	log.Printf("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatal("File error: ", err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&cfg); err != nil {
		log.Fatal("Config error: ", err.Error())
	}
}

func readSecurityPass() ([]byte, error) {
	fmt.Printf("Enter Security Password: \n")
	var fd int
	if term.IsTerminal(int(syscall.Stdin)) {
		fd = int(syscall.Stdin)
	} else {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return nil, errors.New("error allocating terminal")
		}
		defer tty.Close()
		fd = int(tty.Fd())
	}

	SecurityPass, err := term.ReadPassword(fd)
	if err != nil {
		return nil, err
	}
	return SecurityPass, nil
}

func readWalletPass() ([]byte, error) {
	fmt.Printf("Enter Wallet Password: \n")
	var fd int
	if term.IsTerminal(int(syscall.Stdin)) {
		fd = int(syscall.Stdin)
	} else {
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return nil, errors.New("error allocating terminal")
		}
		defer tty.Close()
		fd = int(tty.Fd())
	}

	WalletPass, err := term.ReadPassword(fd)
	if err != nil {
		return nil, err
	}
	return WalletPass, nil
}

func decryptPoolConfigure(cfg *pool.Config, passBytes []byte) error {
	b, err := util.Ae64Decode(cfg.AddressEncrypted, passBytes)
	if err != nil {
		return err
	}
	cfg.Address = string(b)

	// check address
	if !util.ValidateAddress(cfg.Address) {
		return errors.New("decryptPoolConfigure: ValidateAddress")
	}

	// if cfg.Redis.Enabled {
	b, err = util.Ae64Decode(cfg.KvRocks.PasswordEncrypted, passBytes)
	if err != nil {
		return err
	}
	cfg.KvRocks.Password = string(b)
	// }

	// if cfg.RedisFailover.Enabled {
	// 	b, err = util.Ae64Decode(cfg.RedisFailover.PasswordEncrypted, passBytes)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cfg.RedisFailover.Password = string(b)
	// }

	return nil
}

func OptionParse() {
	var showVer bool
	flag.BoolVar(&showVer, "v", false, "show build version")

	flag.Parse()

	if showVer {
		fmt.Printf("Latest Tag: %s\n", LatestTag)
		fmt.Printf("Latest Tag Commit SHA1: %s\n", LatestTagCommitSHA1)
		fmt.Printf("Latest Commit SHA1: %s\n", LatestCommitSHA1)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Release Type: %s\n", ReleaseType)
		os.Exit(0)
	}
}

func main() {
	OptionParse()
	readConfig(&cfg)
	rand.Seed(time.Now().UTC().UnixNano())

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var wg sync.WaitGroup

	// init log file
	_ = os.Mkdir("logs", os.ModePerm)
	iLogFile := "logs/info.log"
	eLogFile := "logs/error.log"
	sLogFile := "logs/share.log"
	bLogFile := "logs/block.log"
	util.InitLog(iLogFile, eLogFile, sLogFile, bLogFile, cfg.Log.LogSetLevel)

	// set rlimit nofile value
	util.SetRLimit(800000)

	secPassBytes, err := readSecurityPass()
	if err != nil {
		util.Error.Fatal("Read Security Password error: ", err.Error())
	}

	err = decryptPoolConfigure(&cfg, secPassBytes)
	if err != nil {
		util.Error.Fatal("Decrypt Pool Configure error: ", err.Error())
	}

	walletPass, err := readWalletPass()
	if err != nil {
		util.Error.Fatal("Read Wallet Password error: ", err.Error())
	}

	hasWallet, bigAddress := connectBipWallet(string(walletPass[:]))
	if !hasWallet {
		util.Error.Fatal("Read Wallet files error")
	}

	if bigAddress != cfg.Address {
		util.Error.Fatal("Wallet Account Address and Pool Address in Config File are not equal.")
	}

	backend = kvstore.NewKvClient(&cfg.KvRocks, cfg.Coin)

	if backend == nil {
		util.Error.Fatal("Backend is Nil: maybe redis/redisFailover config is invalid")
	}

	pong, err := backend.Check()
	if err != nil {
		util.Error.Printf("Can't establish connection to backend: %v", err)
	} else {
		util.Info.Printf("Backend check reply: %v", pong)
	}

	defer func() {
		if r := recover(); r != nil {
			util.Error.Println(string(debug.Stack()))
		}
	}()

	msgChan = make(chan pool.Message, 512)
	ws.NewClient(cfg.NodeWs, cfg.WsSsl, msgChan)
	//startNewrelic()
	startStratum()

	wg.Add(1)
	go func() {
		defer wg.Done()
		payouts.PaymentTask(ctx, &cfg, backend)
	}()

	wg.Wait()
	fmt.Println("Stratum server shutdown.")
}

func connectBipWallet(password string) (bool, string) {
	util.Info.Println("Initializing cryptography...")
	util.Info.Println("Reading wallet...")
	pwd, _ := os.Executable()
	pwd = filepath.Dir(pwd)
	wallet := bip.NewWallet(path.Join(pwd, common.BIP32_WALLET_FOLDER, common.BIP32_WALLET_FILE_NAME))
	res := wallet.UnlockWallet(password)
	if res && wallet.IsHdWalletInitialized() {
		payouts.BipWallet = &wallet
		payouts.BipKey = payouts.BipWallet.GetDefKey()
		b := cryptography.ToBytesAddress(payouts.BipWallet.GetDefKey())
		bipAddress := base58.ChkEnc(b[:])
		payouts.BipAddress = bipAddress
		return true, bipAddress
	}
	return false, ""
}
