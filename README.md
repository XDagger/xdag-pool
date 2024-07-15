# xdagPool

High performance xdag mining stratum with Web-interface written in Golang.

**Stratum feature list:**

* Be your own pool
* Rigs availability monitoring
* Keep track of accepts, rejects, blocks stats
* Easy detection of sick rigs
* Concurrent shares processing
* config encrypted by pool password
* Beautiful Web-interface

![](screenshot.png)

## Installation

Dependencies:

  * go-1.20
  * RandomX

  enter clib/randomx, build randomx library with CMakeLists.txt

## Encrypt tool

### build
```
$> cd tools
$> go build ./encrypt.go
```

### usage
```
./encrypt [-h] [-p pool password] [-a address] [-w wallet password] [-k kv store password]
```
## Configuration

Configuration is self-describing, just copy *config.example.json* to *config.json* and run stratum with path to config file as 1st argument.

```javascript
{
  // Pool Address for rewards, pool key: 12345678
  // AES: CBC, Key Size: 128bits, IV and Secret Key: 16 characters long( add '*' if length not enough)
  "addressEncrypted": "G6LLHMvWi6HiysT+PuCWXhuaTWOxbHlEocNf5ilWAy+e7KsjAGPVOu1PBgIxxeFD",
  "threads": 4,

  //hashrate estimation
  "estimationWindow": "15m",
  "luckWindow": "24h",

  // purge stale kv store data, remain recent 3 days data
  "purgeInterval": "3h",
  "purgeWindow": "72h",

  // randomx mode: fast(3G ram), light(300M ram)
  "rx_mode":"fast",

  //AES encrypted wallet password by pool key
  "walletEncrypted": "9FilIh6x3WdWaC74YGg3qw==",
  "stratum": {
    // Socket timeout
    "timeout": "2m",

    "listen": [
      {
        "host": "0.0.0.0",
        "port": 1111,
        "diff": 20000,
        "maxConn": 32768
      }
    ]
  },

  "frontend": {
    "enabled": true,
    "listen": "0.0.0.0:8082",
    "login": "admin",
    "password": "",
    "hideIP": false
  },

  "kvrocks": {
		"endpoint": "127.0.0.1:6379",
		"poolSize": 10,
		"database": 0,
    //123456
		"passwordEncrypted": "MbRmWtAs7GA2E1B6ioBSoQ=="
	},

	"payout": {
		"poolRation": 5.0,
		"rewardRation": 0.0,
		"directRation": 0.0,
    // threshhold to pay miner
		"threshold": 3,
		"paymentInterval": "10m",
    // solo or equal
		"mode": "equal",
		"paymentRemark": "http://mypool.com"
	}

 }
```

You must use ``<address>.WorkerID`` as username in your miner. If there is no workerID specified your rig stats will be merged under `0` worker.

Copy your wallet data folder ``xdagj_wallet`` to pool path.

To skip password input, modify code pool/pool.go and put your pool key in the code.

```
const PoolKey = "12345678" // it can make pool boot/reboot without interfering.
```

## RPC

### xdag_poolConfig
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"xdag_poolConfig","params":[],"id":1}'
```

#### response
```
{"jsonrpc":"2.0","id":1,"result":
{"poolIp":"127.0.0.1","poolPort":7001,"nodeIp":"127.0.0.1","nodePort":8001,"globalMiner
Limit":8192,"maxConnectMinerPerIp":256,"maxMinerPerAccount":256,"poolFeeRation":"5.0","
poolRewardRation":"5.0","poolDirectRation":"5.0","poolFundRation":"0.0","threshold":"3"}}
```

### xdag_updatePoolConfig
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"xdag_updatePoolConfig","params":[{"poolFeeRation":"4","poolRewardRation":"4","poolDirectRation":"4","threshold":"4"},"pool_password"],"id":1}'
```
#### response
```
{"jsonrpc":"2.0","id":1,"result":"Success"}
```

### xdag_getPoolWorkers
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data
'{"jsonrpc":"2.0","method":"xdag_getPoolWorkers","params":[],"id":1}'
```

#### response
```
json {"jsonrpc":"2.0","id":1,"result":
[{"address":"pCuGwAx/THicdSMFiy7vPgixsSP9AVRQ","status":"fee","unpaidShares":0.0,"hashr
ate":2.2551405187698493E-18,"workers":[]},
{"address":"oJA3+RpvYRb0eJKdUo38XTxLRhqircNa","status":"MINER_ACTIVE","unpaidShares":0.
02088520508135681,"hashrate":3.9752369225781053E-4,"workers":
[{"address":"172.31.100.234:48466","inBound":145,"outBound":438,"unpaidShares":0.020885
20508135681,"name":"wb1","hashrate":3.9752369225781053E-4}]},
{"address":"3oQtj/YtlIl8PGNWtqFR0QNo0dXJ+FDq","status":"MINER_ACTIVE","unpaidShares":1.
5918582804382894E-8,"hashrate":2.8651250602006866E-8,"workers":
[{"address":"172.31.100.234:53154","inBound":201,"outBound":260,"unpaidShares":1.591858
2804382894E-8,"name":"", "hashrate":2.8651250602006866E-8}]},
{"address":"jWx51h049qtJH68Zd0buxwW5xPML4GZR","status":"MINER_ACTIVE","unpaidShares":4.
186463512272553E-4,"hashrate":0.0012218697372245008,"workers":
[{"address":"172.31.100.234:48488","inBound":108,"outBound":327,"unpaidShares":1.893309
5261087333E-4,"name":"wb2","hashrate":5.014658633706342E-4}
{"address":"172.31.100.234:48490","inBound":89,"outBound":295,"unpaidShares":4.18646351
2272553E-4,"name":"wb3","hashrate":3.704957506989368E-4},
{"address":"172.31.100.234:48492","inBound":102,"outBound":295,"unpaidShares":1.6494257
81936861E-4,"name":"wb4","hashrate":2.4399312515244604E-4}]}]}
```

### xdag_poolHashrate
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data
'{"jsonrpc":"2.0","method":"xdag_poolHashrate","params":[],"id":1}'
```

#### response
```
json {
  "jsonrpc": "2.0",
  "result": {
    "hashrate": 600,
    "hashrate24h": 6.25,
    "total": 2, // total miners
    "total_online": 2 // online miners
  },
  "id": 1
}
```

### xdag_minerHashrate
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data
'{"jsonrpc":"2.0","method":"xdag_minerHashrate","params":["miner's address"],"id":1}'
```

#### response
```
json {
  "jsonrpc": "2.0",
  "result": {
    "address": "miner's address",
    "timestamp": 1715694978524,
    "total_hashrate": 222.22222222222223,
    "total_hashrate24h": 0,
    "total_online": 2,
    "hashrate": [
      {
        "name": "test",
        "hashrate": 44.44444444444444,
        "hashrate24h": 0.46296296296296297,
        "lastBeat": 1715694938551,
        "validShares": 2,
        "staleShares": 0,
        "invalidShares": 0,
        "accepts": 0,
        "rejects": 0,
        "ip": "127.0.0.1:53148",
        "warning": false,
        "timeout": false
      },
      {
        "name": "rig02",
        "hashrate": 177.77777777777777,
        "hashrate24h": 1.8518518518518519,
        "lastBeat": 1715694950748,
        "validShares": 8,
        "staleShares": 0,
        "invalidShares": 0,
        "accepts": 0,
        "rejects": 0,
        "ip": "192.168.10.5:7548",
        "warning": false,
        "timeout": false
      }
    ]
  },
  "id": 1
}
```

### xdag_minerAccount
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data
'{"jsonrpc":"2.0","method":"xdag_minerAccount","params":["miner's address"],"id":1}'
```

#### response
```
json {
  "jsonrpc": "2.0",
  "result": {
    "address": "miner's address",
    "timestamp": 1715695039585,
    "total_reward": 225.795999996,
    "total_payment": 56.448999999,
    "total_unpaid": 169.346999997
  },
  "id": 1
}
```

### xdag_poolVersion
#### request
```
curl http://127.0.0.1:8082/api -s -X POST -H "Content-Type: application/json" --data
'{"jsonrpc":"2.0","method":"xdag_poolVersion","params":[""],"id":1}'
```

#### response
```
json {
  "jsonrpc": "2.0",
  "result": "0.1.0",
  "id": 1
}
```