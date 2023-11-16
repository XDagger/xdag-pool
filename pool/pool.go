package pool

type StorageConfig struct {
	Enabled           bool   `json:"enabled"`
	Endpoint          string `json:"endpoint"`
	PasswordEncrypted string `json:"passwordEncrypted"`
	Password          string `json:"-"`
	Database          int64  `json:"database"`
	PoolSize          int    `json:"poolSize"`
}

type StorageConfigFailover struct {
	Enabled           bool     `json:"enabled"`
	MasterName        string   `json:"masterName"`
	SentinelEndpoints []string `json:"sentinelEndpoints"`
	PasswordEncrypted string   `json:"passwordEncrypted"`
	Password          string   `json:"-"`
	Database          int64    `json:"database"`
	PoolSize          int      `json:"poolSize"`
}

type UnlockerConfig struct {
	Enabled         bool    `json:"enabled"`
	PoolRation      float64 `json:"poolRation"`
	FundRation      float64 `json:"fundRation"`
	RewardRation    float64 `json:"rewardRation"`
	DirectRation    float64 `json:"directRation"`
	PoolFeeAddress  string  `json:"poolFeeAddress"`
	Donate          bool    `json:"donate"`
	Threshold       int64   `json:"threshold"`
	PaymentInterval string  `json:"paymentInterval"`
	KeepTxFees      bool    `json:"keepTxFees"`
	Mode            string  `json:"mode"`
	PaymentRemark   string  `json:"paymentRemark"`
	DaemonHost      string  `json:"daemonHost"`
	DaemonPort      int     `json:"daemonPort"`
	Timeout         string  `json:"timeout"`
}

type Config struct {
	AddressEncrypted      string     `json:"addressEncrypted"`
	Address               string     `json:"-"`
	Log                   Log        `json:"log"`
	Stratum               Stratum    `json:"stratum"`
	StratumTls            StratumTls `json:"stratumTls"`
	BlockRefreshInterval  string     `json:"blockRefreshInterval"`
	UpstreamCheckInterval string     `json:"upstreamCheckInterval"`
	Upstream              []Upstream `json:"upstream"`
	EstimationWindow      string     `json:"estimationWindow"`
	LuckWindow            string     `json:"luckWindow"`
	LargeLuckWindow       string     `json:"largeLuckWindow"`
	HashRateExpiration    string     `json:"hashRateExpiration"`

	PurgeInterval       string `json:"purgeInterval"`
	HashrateWindow      string `json:"hashrateWindow"`
	HashrateLargeWindow string `json:"hashrateLargeWindow"`

	Threads  int      `json:"threads"`
	Frontend Frontend `json:"frontend"`

	Coin          string                `json:"coin"`
	Redis         StorageConfig         `json:"redis"`
	RedisFailover StorageConfigFailover `json:"redisFailover"`

	NodeRpc       string         `json:"node_rpc"`
	BlockUnlocker UnlockerConfig `json:"unlocker"`
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
	TxBlock string  `json:"txBlock"`
	PreHash string  `json:"preHash"`
	Share   string  `json:"share"`
	Amount  float64 `json:"amount"`
	Fee     float64 `json:"fee"`
}
