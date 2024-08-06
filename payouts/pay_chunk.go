package payouts

import (
	"errors"

	"github.com/XDagger/xdagpool/kvstore"
	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
)

func batchPayMiners(cfg *pool.Config, backend *kvstore.KvClient) {
	var threshold int64
	var remark string
	cfg.RLock()
	threshold = cfg.PayOut.Threshold
	remark = cfg.PayOut.PaymentRemark
	cfg.RUnlock()

	// find miners balance more than payment threshold
	miners := backend.GetMinersToPay(threshold)
	if len(miners) == 0 {
		return
	}
	var chunkSize int
	if len(remark) > 0 {
		chunkSize = 10
	} else {
		chunkSize = 11
	}
	batchAddress := make([]string, 0, chunkSize)
	batchAmount := make([]int64, 0, chunkSize)
	for address, amount := range miners {

		if amount > 0 {
			batchAddress = append(batchAddress, address)
			batchAmount = append(batchAmount, amount)
		}
		if len(batchAddress) == chunkSize {
			PayChunk(backend, &batchAddress, &batchAmount, remark)
		}
	}

	if len(batchAddress) > 0 {
		PayChunk(backend, &batchAddress, &batchAmount, remark)
	}

}

func PayChunk(backend *kvstore.KvClient, batchAddress *[]string, batchAmount *[]int64, remark string) {
	ms := util.MakeTimestamp()
	ts := ms / 1000
	txHash, err := transfer2chunk(*batchAddress, remark, *batchAmount)
	if err != nil {
		util.Error.Println("transfer chunk reward to miners error", err)
		return
	}
	err = backend.SetChunkPayment(*batchAddress, txHash, remark, *batchAmount, ms, ts)
	if err != nil {
		util.Error.Println("kv store set chunk payment error", txHash, err)
	}
	*batchAddress = (*batchAddress)[:0]
	*batchAmount = (*batchAmount)[:0]
}

func transfer2chunk(miners []string, remark string, amounts []int64) (txHash string, err error) {
	if len(remark) > 0 && !ValidateRemark(remark) {
		return "", errors.New("remark error")
	}

	if len(miners) != len(amounts) || len(miners) > 11 || (len(miners) == 11 && len(remark) > 0) {
		return "", errors.New("transfer chunck size error")
	}

	txHash, err = TransferChunkRpc(amounts, Cfg.Address, miners, remark, BipKey)
	return
	// fmt.Println(amount, Cfg.Address, miner, remark)
	// return getUuid(), nil
}
