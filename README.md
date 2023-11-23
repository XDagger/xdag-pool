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


<!-- ### Linux

Use Ubuntu 16.04 LTS.

    sudo apt-get install libssl-dev
    sudo apt-get install git cmake build-essential pkg-config libboost-all-dev libreadline-dev doxygen libsodium-dev libzmq5-dev
    sudo apt-get install liblmdb-dev libevent-dev libjson-c-dev uuid-dev

Use Ubuntu 18.04 LTS.

    sudo apt-get install libssl1.0-dev
    sudo apt-get install git cmake build-essential pkg-config libboost-all-dev libreadline-dev doxygen libsodium-dev libzmq5-dev 
    sudo apt-get install liblmdb-dev libevent-dev libjson-c-dev uuid-dev


Compile Monero source (with shared libraries option):

    git clone --recursive https://github.com/monero-project/monero.git
    cd monero
    git checkout tags/v0.17.0.0 -b v0.17.0.0
    cmake -DBUILD_SHARED_LIBS=1 -DMANUAL_SUBMODULES=1 .
    make

Install Golang and required packages:

    sudo apt install software-properties-common
    sudo add-apt-repository ppa:longsleep/golang-backports
    sudo apt-get update
    sudo apt-get install golang-go

Clone:

    git clone https://github.com/XDagger/xdagpool.git
    cd xmrpool

Build stratum:

    export MONERO_DIR=[path_of_monero] 
    cmake .
    make
    make -f Makefile_build_info

`MONERO_DIR=/path/to/monero` is optional, not needed if both `monero` and `xmrpool` is in the same directory like `/opt/src/`. By default make will search for monero libraries in `../monero`. You can just run `cmake .`.

### Mac OS X

Compile Monero source:

    git clone --recursive https://github.com/monero-project/monero.git
    cd monero
    git checkout tags/v0.17.0.0 -b v0.17.0.0
    cmake .
    make

Install Golang and required packages:

    brew update && brew install go

Clone stratum:

    git clone https://github.com/XDagger/xdagpool.git
    cd xmrpool

Build stratum:

    MONERO_DIR=[path_of_monero]  
    cmake .
    make
    make -f Makefile_build_info

### Running Stratum

    ./build/bin/xmrpool config.json

If you need to bind to privileged ports and don't want to run from `root`:

    sudo apt-get install libcap2-bin
    sudo setcap 'cap_net_bind_service=+ep' /path/to/xmrpool -->

## Configuration

Configuration is self-describing, just copy *config.example.json* to *config.json* and run stratum with path to config file as 1st argument.

```javascript
{
  // Pool Address for rewards
  // AES: CBC, Key Size: 128bits, IV and Secret Key: 16 characters long( add '*' if length not enough)
  "addressEncrypted": "YOUR-ADDRESS-ENCRYPTED",
  "threads": 4,

  "estimationWindow": "15m",
  "luckWindow": "24h",
  "largeLuckWindow": "72h",

  "stratum": {
    // Socket timeout
    "timeout": "2m",

    "listen": [
      {
        "host": "0.0.0.0",
        "port": 1111,
        "diff": 5000,
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
		"passwordEncrypted": "oXyI5OTy+nRTshESi80X8KKSjDiLksuw1mhwRg2z0Ic="
	},

	"payout": {
		"poolRation": 5.0,
		"fundRation": 5.0,
		"rewardRation": 5.0,
		"directRation": 5.0,
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

Copy your wallet data folder xdagj_wallet to pool path.