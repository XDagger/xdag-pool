package main

import (
	"fmt"
	"path"
	"testing"

	"github.com/XDagger/xdagpool/kvstore"
	"github.com/XDagger/xdagpool/payouts"
	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/xdago/common"
	bip "github.com/XDagger/xdagpool/xdago/wallet"
)

func TestFieldTypes(t *testing.T) {
	s := payouts.FieldChunkTypes(false, true, true, 10)
	fmt.Println(s)

	s = payouts.FieldChunkTypes(false, false, true, 11)
	fmt.Println(s)

	s = payouts.FieldChunkTypes(false, true, false, 6)
	fmt.Println(s)

	s = payouts.FieldChunkTypes(true, false, false, 5)
	fmt.Println(s)
}

func TestChunkSign(t *testing.T) {
	// init log file
	// _ = os.Mkdir("../logs", os.ModePerm)
	iLogFile := "logs/info.log"
	eLogFile := "logs/error.log"
	sLogFile := "logs/share.log"
	bLogFile := "logs/block.log"
	util.InitLog(iLogFile, eLogFile, sLogFile, bLogFile, 10)

	wallet := bip.NewWallet(path.Join(".", common.BIP32_WALLET_FOLDER, common.BIP32_WALLET_FILE_NAME))
	res := wallet.UnlockWallet("111")
	if res && wallet.IsHdWalletInitialized() {
		key := wallet.GetDefKey()

		r, s, n := payouts.ChunkTranxSign("0000000000000000000000000000000000000000000000000000000000000000", key)
		fmt.Println(r, s, n)
	}
}

func TestChunkBlock(t *testing.T) {
	to := []string{"4duPWMbYUgAifVYkKDCWxLvRRkSByf5gb", "Fve2AF8NrEPjNcAj5BABTBeqn7LW7WfeT", "Fii9BuhR1KogfNzWbtSH1YJgQQDwFMomK"}
	value := []int64{2 * 1e9, 7 * 1e9, 15 * 1e9}

	iLogFile := "logs/info.log"
	eLogFile := "logs/error.log"
	sLogFile := "logs/share.log"
	bLogFile := "logs/block.log"
	util.InitLog(iLogFile, eLogFile, sLogFile, bLogFile, 10)

	wallet := bip.NewWallet(path.Join(".", common.BIP32_WALLET_FOLDER, common.BIP32_WALLET_FILE_NAME))
	res := wallet.UnlockWallet("111")
	if res && wallet.IsHdWalletInitialized() {
		key := wallet.GetDefKey()

		block, total := payouts.TransactionChunkBlock("Dd2KRkRceHtx7ep3qWHVAHEdjYoyPpAYx", to, "hello", value, key)
		fmt.Println(total)
		fmt.Println(block)
	}

}

func TestChunkRpc(t *testing.T) {
	to := []string{"4duPWMbYUgAifVYkKDCWxLvRRkSByf5gb", "Fve2AF8NrEPjNcAj5BABTBeqn7LW7WfeT", "2UqAZBFqA3CbWa6oTR3PYi6BW2wXGK9tY"}
	value := []int64{2 * 1e9, 3 * 1e9, 4 * 1e9}
	cfg.NodeRpc = "http://testnet-rpc.xdagj.org:10001"
	payouts.Cfg = &cfg
	iLogFile := "logs/info.log"
	eLogFile := "logs/error.log"
	sLogFile := "logs/share.log"
	bLogFile := "logs/block.log"
	util.InitLog(iLogFile, eLogFile, sLogFile, bLogFile, 10)

	wallet := bip.NewWallet(path.Join(".", common.BIP32_WALLET_FOLDER, common.BIP32_WALLET_FILE_NAME))
	res := wallet.UnlockWallet("111")
	if res && wallet.IsHdWalletInitialized() {
		// fmt.Println(wallet.GetMnemonic())
		key := wallet.GetDefKey()

		hash, err := payouts.TransferChunkRpc(value, "Fii9BuhR1KogfNzWbtSH1YJgQQDwFMomK", to, "hello", key)
		fmt.Println(hash)
		fmt.Println(err)
	}

}

func TestPayChunk(t *testing.T) {
	to := []string{"4duPWMbYUgAifVYkKDCWxLvRRkSByf5gb", "Fve2AF8NrEPjNcAj5BABTBeqn7LW7WfeT", "Fii9BuhR1KogfNzWbtSH1YJgQQDwFMomK"}
	value := []int64{2 * 1e9, 3 * 1e9, 4 * 1e9}
	cfg.NodeRpc = "http://testnet-rpc.xdagj.org:10001"
	cfg.Coin = "xdag"
	cfg.KvRocks = pool.StorageConfig{
		Endpoint:          "127.0.0.1:6379",
		PasswordEncrypted: "MbRmWtAs7GA2E1B6ioBSoQ==",
		Password:          "123456",
		Database:          0,
		PoolSize:          10,
	}
	cfg.Address = "Dd2KRkRceHtx7ep3qWHVAHEdjYoyPpAYx"
	payouts.Cfg = &cfg
	iLogFile := "logs/info.log"
	eLogFile := "logs/error.log"
	sLogFile := "logs/share.log"
	bLogFile := "logs/block.log"
	util.InitLog(iLogFile, eLogFile, sLogFile, bLogFile, 10)

	wallet := bip.NewWallet(path.Join(".", common.BIP32_WALLET_FOLDER, common.BIP32_WALLET_FILE_NAME))
	res := wallet.UnlockWallet("111")
	if res && wallet.IsHdWalletInitialized() {
		payouts.BipKey = wallet.GetDefKey()
		backend = kvstore.NewKvClient(&cfg.KvRocks, cfg.Coin)

		payouts.PayChunk(backend, &to, &value, "test pay")

	}

}
