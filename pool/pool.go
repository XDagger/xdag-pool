package pool

import (
	"encoding/json"
	"sync"
)

const Version = "0.1.2"
const PoolKey = "" // it can make pool boot/reboot without interfering.

type StorageConfig struct {
	Endpoint          string `json:"endpoint"`
	PasswordEncrypted string `json:"passwordEncrypted"`
	Password          string `json:"-"`
	Database          int64  `json:"database"`
	PoolSize          int    `json:"poolSize"`
}

type PayOutConfig struct {
	PoolRation      float64 `json:"poolRation"`
	RewardRation    float64 `json:"rewardRation"`
	DirectRation    float64 `json:"directRation"`
	PoolFeeAddress  string  `json:"poolFeeAddress"`
	Threshold       int64   `json:"threshold"`
	PaymentInterval string  `json:"paymentInterval"`
	Mode            string  `json:"mode"`
	PaymentRemark   string  `json:"paymentRemark"`
}

type Config struct {
	sync.RWMutex
	AddressEncrypted string     `json:"addressEncrypted"`
	Address          string     `json:"-"`
	Log              Log        `json:"log"`
	Stratum          Stratum    `json:"stratum"`
	StratumTls       StratumTls `json:"stratumTls"`
	EstimationWindow string     `json:"estimationWindow"`
	LuckWindow       string     `json:"luckWindow"`
	// LargeLuckWindow  string     `json:"largeLuckWindow"`

	PurgeInterval string `json:"purgeInterval"`
	PurgeWindow   string `json:"purgeWindow"`
	// PurgeLargeWindow string `json:"purgeLargeWindow"`

	Threads  int      `json:"threads"`
	Frontend Frontend `json:"frontend"`

	Coin    string        `json:"coin"`
	KvRocks StorageConfig `json:"kvrocks"`

	NodeName string `json:"node_name"`
	NodeRpc  string `json:"node_rpc"`
	NodeWs   string `json:"node_ws"`
	WsSsl    bool   `json:"ws_ssl"`

	RxMode          string `json:"rx_mode"`
	WalletEncrypted string `json:"walletEncrypted"`
	WalletPswd      string `json:"-"`

	PayOut PayOutConfig `json:"payout"`
}

type Stratum struct {
	Enabled bool   `json:"enabled"`
	Timeout string `json:"timeout"`
	Ports   []Port `json:"listen"`
}

type Port struct {
	Difficulty int64  `json:"diff"` //default 300000
	Host       string `json:"host"`
	Port       int    `json:"port"`
	MaxConn    int    `json:"maxConn"`
}

type StratumTls struct {
	Enabled bool   `json:"enabled"`
	Timeout string `json:"timeout"`
	Ports   []Port `json:"listen"`
	TlsCert string `json:"tlsCert"`
	TlsKey  string `json:"tlsKey"`
}

type Upstream struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout string `json:"timeout"`
}

type Frontend struct {
	Enabled  bool   `json:"enabled"`
	Listen   string `json:"listen"`
	Login    string `json:"login"`
	Password string `json:"password"`
	HideIP   bool   `json:"hideIP"`
}

type Log struct {
	LogSetLevel int `json:"logSetLevel"`
}

// ws reward message from xdaj
type XdagjReward struct {
	TxBlock     string  `json:"txBlock"`
	PreHash     string  `json:"preHash"`
	Share       string  `json:"share"`
	Amount      float64 `json:"amount"`
	Fee         float64 `json:"fee"`
	DonateBlock string  `json:"donateBlock"`
	Donate      float64 `json:"donate"`
}

type Message struct {
	MsgType    int             `json:"msgType"` // 1:task, 2:share, 3:rewards
	MsgContent json.RawMessage `json:"msgContent"`
}
type Task struct {
	PreHash  string `json:"preHash"`
	TashSeed string `json:"taskSeed"`
}
type XdagjTask struct {
	Data      Task   `json:"task"`
	Timestamp uint64 `json:"taskTime"`
	Index     int    `json:"taskIndex"`
}
