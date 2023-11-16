package payouts

import (
	"encoding/hex"
	"errors"

	"github.com/XDagger/xdagpool/kvstore"
	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/xdago/base58"
	"github.com/XDagger/xdagpool/xdago/secp256k1"
	bip "github.com/XDagger/xdagpool/xdago/wallet"
)

var BipWallet *bip.Wallet
var BipAddress string
var BipKey *secp256k1.PrivateKey

const CommunityAddress = "4duPWMbYUgAifVYkKDCWxLvRRkSByf5gb"

func ProcessReward(cfg *pool.Config, backend *kvstore.KvClient, reward pool.XdagjReward) {
	ms := util.MakeTimestamp()
	ts := ms / 1000
	login, err := addressFromShare(reward.Share)
	if err != nil {
		util.Error.Println("invalid share in xdagj reward")
		return
	}

	// is the reward's share submitted by this pool?
	if !backend.IsPoolShare(reward.PreHash, reward.Share) {
		backend.SetLostReward(login, reward, ms, ts)
		return
	}
	err = backend.SetWinReward(login, reward, ms, ts)
	if err != nil {
		util.Error.Println("store win set error")
		return
	}

	dividend(cfg, backend, login, reward, ms, ts)

}

func dividend(cfg *pool.Config, backend *kvstore.KvClient, login string, reward pool.XdagjReward, ms, ts int64) {
	poolFee := reward.Amount * cfg.BlockUnlocker.PoolRation / 100.0     // for pool owner
	fundFee := reward.Amount * cfg.BlockUnlocker.FundRation / 100.0     //for community fund
	rewardFee := reward.Amount * cfg.BlockUnlocker.RewardRation / 100.0 // reward to lowest hash finder
	directFee := reward.Amount * cfg.BlockUnlocker.DirectRation / 100.0 // divided equally to every miner

	payFund(backend, CommunityAddress, reward.PreHash,
		cfg.BlockUnlocker.PaymentRemark, fundFee)

	divideAmount := reward.Amount - poolFee - fundFee - directFee
	if cfg.BlockUnlocker.Mode == "solo" {
		backend.SetFinderReward(login, reward, divideAmount, ms, ts)
	} else {
		backend.SetFinderReward(login, reward, rewardFee, ms, ts)
	}

	if cfg.BlockUnlocker.Mode == "solo" && cfg.BlockUnlocker.DirectRation > 0 {
		backend.DivideSolo(login, reward, directFee, ms, ts)
	} else {
		if cfg.BlockUnlocker.DirectRation > 0 {
			divideAmount = divideAmount - rewardFee
			backend.DivideEqual(login, reward, directFee, divideAmount, ms, ts)
		} else {
			backend.DivideEqual(login, reward, 0, divideAmount, ms, ts)
		}

	}
}

func addressFromShare(share string) (string, error) {
	if len(share) != 64 {
		return "", errors.New("share length error")
	}
	b, err := hex.DecodeString(share[:40])
	if err != nil {
		return "", err
	}
	return base58.ChkEnc(b), nil
}
