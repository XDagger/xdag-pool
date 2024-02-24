package payouts

import (
	"context"
	"time"

	"github.com/XDagger/xdagpool/kvstore"
	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
)

// var backend *kvstore.KvClient = nil

func PaymentTask(ctx context.Context, cfg *pool.Config, backend *kvstore.KvClient) {
	interval, err := time.ParseDuration(cfg.PayOut.PaymentInterval)
	if err != nil {
		interval = 10 * time.Minute
	}
	ticker := time.NewTicker(interval) // every 10 minutes
	for {
		select {
		case <-ctx.Done():
			util.Info.Println("exit payment task")
			return
		case <-ticker.C:
			payMiners(cfg, backend)
		}
	}
}

func payMiners(cfg *pool.Config, backend *kvstore.KvClient) {
	cfg.RLock()
	defer cfg.RUnlock()
	// find miners balance more than payment threshold
	miners := backend.GetMinersToPay(cfg.PayOut.Threshold)
	if len(miners) == 0 {
		return
	}
	for address, amount := range miners {
		if amount > 0 {
			payMiner(backend, address, cfg.PayOut.PaymentRemark, float64(amount)/float64(1e9))
		}
	}
}

func payMiner(backend *kvstore.KvClient, miner, remark string, amount float64) {
	ms := util.MakeTimestamp()
	ts := ms / 1000
	txHash, err := transfer2miner(miner, remark, amount)
	if err != nil {
		util.Error.Println("transfer reward to miner error", miner, amount, err)
		return
	}
	err = backend.SetPayment(miner, txHash, remark, amount, ms, ts)
	if err != nil {
		util.Error.Println("kv store set payment error", miner, txHash, amount, err)
	}
}

// func payFund(backend *kvstore.KvClient, fund, jobHash, remark string, amount float64) {
// 	ms := util.MakeTimestamp()
// 	ts := ms / 1000
// 	txHash, err := transfer2miner(fund, remark, amount)
// 	if err != nil {
// 		util.Error.Println("transfer donate to fund error", fund, amount, err)
// 		return
// 	}
// 	err = backend.SetFund(fund, txHash, jobHash, remark, amount, ms, ts)
// 	if err != nil {
// 		util.Error.Println("kv store set donate error", fund, txHash, amount, err)
// 	}
// }

func transfer2miner(miner, remark string, amount float64) (txHash string, err error) {
	txHash, err = TransferRpc(amount, Cfg.Address, miner, remark, BipKey)
	return
	// fmt.Println(amount, Cfg.Address, miner, remark)
	// return getUuid(), nil

}

// uuid as unique remark for a transfer transaction
// func getUuid() string {
// 	uuid := uuid.New().String()
// 	return strings.Replace(uuid, "-", "", -1)
// }
