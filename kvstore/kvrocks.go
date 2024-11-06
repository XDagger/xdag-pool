package kvstore

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

const expireDuration = 30 * time.Minute

type KvClient struct {
	client *redis.Client
	prefix string
}

func NewKvClient(cfg *pool.StorageConfig, prefix string) *KvClient {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Endpoint,
		Password: cfg.Password,
		DB:       int(cfg.Database),
		PoolSize: cfg.PoolSize,
	})
	return &KvClient{client: client, prefix: prefix}
}

func (r *KvClient) Check() (string, error) {
	return r.client.Ping(ctx).Result()
}

func (r *KvClient) WriteInvalidShare(ms, ts int64, login, id string, diff int64) error {
	cmd := r.client.ZAdd(ctx, r.formatKey("invalidhashrate"), redis.Z{Score: float64(ts), Member: join(diff, login, id, ms)})
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

func (r *KvClient) WriteRejectShare(ms, ts int64, login, id string, diff int64) error {
	cmd := r.client.ZAdd(ctx, r.formatKey("rejecthashrate"), redis.Z{Score: float64(ts), Member: join(diff, login, id, ms)})
	if cmd.Err() != nil {
		return cmd.Err()
	}
	return nil
}

// func (r *KvClient) writeShare(tx redis.Pipeliner, ms, ts int64, login, id string, diff int64, expire time.Duration) {
// 	tx.HIncrBy(ctx, r.formatKey("shares", "roundCurrent"), login, diff)
// 	tx.ZAdd(ctx, r.formatKey("hashrate"), redis.Z{Score: float64(ts), Member: join(diff, login, id, ms)})
// 	tx.ZAdd(ctx, r.formatKey("hashrate", login), redis.Z{Score: float64(ts), Member: join(diff, id, ms)})
// 	tx.Expire(ctx, r.formatKey("hashrate", login), expire) // Will delete hashrates for miners that gone
// 	// tx.HSet(ctx, r.formatKey("miners", login), "lastShare", strconv.FormatInt(ts, 10))
// }

func (r *KvClient) WriteBlock(login, id, share string, diff int64, shareU64 uint64,
	timestamp uint64, jobHash string) (bool, error) {
	// exist, err := r.checkPoWExist(login, params)
	// if err != nil {
	// 	return false, err
	// }

	exist := util.MinedShares.ShareExist(share)

	if exist {
		return true, nil
	}

	// util.HashrateRank.IncShareByKey(login, diff)   // accumulated for hashrate rank
	// util.HashrateRank.IncShareByKey("total", diff) // accumulated for total hashrate

	tx := r.client.TxPipeline()
	// defer tx.Close()

	ms := util.MakeTimestamp()
	ts := ms / 1000

	// r.writeShare(tx, ms, ts, login, id, diff, window)
	tx.HSet(ctx, r.formatKey("workers", login+"."+id), "lastShare", strconv.FormatInt(ts, 10))
	tx.HSet(ctx, r.formatKey("miners", login), "lastShare", strconv.FormatInt(ts, 10))
	tx.HSet(ctx, r.formatKey("stats"), "lastShare", strconv.FormatInt(ts, 10))
	// tx.HDel(ctx, r.formatKey("stats"), "roundShares")
	// tx.ZIncrBy(ctx, r.formatKey("finders"), 1, login)
	// tx.HIncrBy(ctx, r.formatKey("miners", login), "blocksFound", 1)

	// tx.Rename(ctx, r.formatKey("shares", "roundCurrent"), r.formatRound(int64(height), params[0]))
	// tx.HGetAll(ctx, r.formatRound(int64(height), params[0]))
	tx.HIncrBy(ctx, r.formatKey("pool", jobHash), "diff", diff) // accumulate pool diff of the job
	tx.HIncrBy(ctx, r.formatKey("job", jobHash), login, diff)   // accumulate miners diff of the job (identified by job hash)
	tx.Expire(ctx, r.formatKey("job", jobHash), expireDuration)
	tx.Expire(ctx, r.formatKey("pool", jobHash), expireDuration)
	// cmds, err := tx.Exec(ctx)
	_, err := tx.Exec(ctx)
	if err != nil {
		return false, err
	}
	// else {
	// 	sharesMap, _ := cmds[10].(*redis.MapStringStringCmd).Result()
	// 	totalShares := int64(0)
	// 	for _, v := range sharesMap {
	// 		n, _ := strconv.ParseInt(v, 10, 64)
	// 		totalShares += n
	// 	}
	// 	hashHex := strings.Join(params, ":")
	// 	s := join(hashHex, ts, roundDiff, totalShares)
	// 	cmd := r.client.ZAdd(ctx, r.formatKey("blocks", "candidates"), redis.Z{Score: float64(height), Member: s})
	// 	return false, cmd.Err()
	// }
	return false, nil
}
func (r *KvClient) SetMinerReward(login, txHash, jobHash string, reward float64, ms, ts int64) error {
	tx := r.client.TxPipeline()
	tx.HIncrBy(ctx, r.formatKey("account", login), "reward", int64(reward*1e9))
	tx.HIncrBy(ctx, r.formatKey("account", login), "unpaid", int64(reward*1e9))
	tx.ZAdd(ctx, r.formatKey("rewards", jobHash), redis.Z{Score: float64(ts), Member: join(reward, ms, txHash, login)})
	tx.ZAdd(ctx, r.formatKey("rewards", login), redis.Z{Score: float64(ts), Member: join(reward, ms, txHash, jobHash)})
	tx.ZAdd(ctx, r.formatKey("balance", login), redis.Z{Score: float64(ts), Member: join("reward", reward, ms, txHash, jobHash)})
	_, err := tx.Exec(ctx)
	return err
}

// func (r *KvClient) AddWaiting(jobHash string) {
// 	_, err := r.client.SAdd(ctx, r.formatKey("waiting"), jobHash).Result()
// 	if err != nil {
// 		util.Error.Println("add job waiting  set error", jobHash, err)
// 	}
// }

func (r *KvClient) SetWinReward(login string, reward pool.XdagjReward, ms, ts int64) error {
	// res, err := r.client.SMove(ctx, r.formatKey("waiting"), r.formatKey("win"), reward.PreHash).Result()
	// if err != nil {
	// 	return err
	// }
	// if !res { // preHash not in waiting set
	// 	return errors.New("moved key not exist in source")
	// }

	tx := r.client.TxPipeline()
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "rewards", int64(reward.Amount*1e9))
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "unpaid", int64(reward.Amount*1e9))
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "donate", int64(reward.Donate*1e9))
	tx.ZAdd(ctx, r.formatKey("pool", "rewards"), redis.Z{Score: float64(ts),
		Member: join(reward.Amount, reward.Fee, ms, reward.TxBlock, reward.PreHash, login, reward.Share)})
	tx.ZAdd(ctx, r.formatKey("pool", "donate"), redis.Z{Score: float64(ts),
		Member: join(reward.Donate, ms, reward.PreHash, reward.DonateBlock)}).Result()
	tx.Del(ctx, r.formatKey("submit", reward.PreHash)).Result()
	_, err := tx.Exec(ctx)
	return err
}

// func (r *KvClient) SetLostReward(login string, reward pool.XdagjReward, ms, ts int64) {
// 	_, err := r.client.SMove(ctx, r.formatKey("waiting"), r.formatKey("lost"), reward.PreHash).Result()
// 	if err != nil {
// 		util.Error.Println("store lost set error", reward.PreHash, err)
// 	}
// }

func (r *KvClient) SetPayment(login, txHash, remark string, payment float64, ms, ts int64) error {
	tx := r.client.TxPipeline()
	tx.HIncrBy(ctx, r.formatKey("account", login), "payment", int64(payment*1e9))
	tx.HIncrBy(ctx, r.formatKey("account", login), "unpaid", -1*int64(payment*1e9))
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "payment", int64(payment*1e9))
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "unpaid", -1*int64(payment*1e9))
	tx.ZAdd(ctx, r.formatKey("payment", login), redis.Z{Score: float64(ts),
		Member: join(payment, ms, txHash, remark)})
	tx.ZAdd(ctx, r.formatKey("balance", login), redis.Z{Score: float64(ts),
		Member: join("payment", payment, ms, txHash, remark)})
	_, err := tx.Exec(ctx)
	return err
}

func (r *KvClient) SetChunkPayment(logins []string, txHash, remark string, payments []int64, ms, ts int64) error {
	if len(logins) != len(payments) {
		return errors.New("set payment chunck size not match")
	}
	var total int64
	for _, v := range payments {
		total += v
	}
	tx := r.client.TxPipeline()
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "payment", total)
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "unpaid", -1*total)

	for i := 0; i < len(logins); i++ {
		tx.HIncrBy(ctx, r.formatKey("account", logins[i]), "payment", payments[i])
		tx.HIncrBy(ctx, r.formatKey("account", logins[i]), "unpaid", -1*payments[i])

		tx.ZAdd(ctx, r.formatKey("payment", logins[i]), redis.Z{Score: float64(ts),
			Member: join(float64(payments[i])/float64(1e9), ms, txHash, remark)})
		tx.ZAdd(ctx, r.formatKey("balance", logins[i]), redis.Z{Score: float64(ts),
			Member: join("payment", float64(payments[i])/float64(1e9), ms, txHash, remark)})
	}
	_, err := tx.Exec(ctx)
	return err
}

// func (r *KvClient) SetFund(fund, txHash, jobHash, remark string, payment float64, ms, ts int64) error {
// 	_, err := r.client.ZAdd(ctx, r.formatKey("donate", fund), redis.Z{Score: float64(ts),
// 		Member: join(payment, ms, txHash, jobHash, remark)}).Result()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// WARNING: Must run it periodically to flush out of window hashrate entries
func (r *KvClient) FlushStaleStats(window, largeWindow time.Duration) (int64, error) {
	now := util.MakeTimestamp() / 1000
	max := fmt.Sprint("(", now-int64(window/time.Second))
	total, err := r.client.ZRemRangeByScore(ctx, r.formatKey("hashrate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}

	n, err := r.client.ZRemRangeByScore(ctx, r.formatKey("invalidhashrate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}
	total += n

	n, err = r.client.ZRemRangeByScore(ctx, r.formatKey("rejecthashrate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}
	total += n

	var c uint64
	miners := make(map[string]struct{})
	max = fmt.Sprint("(", now-int64(largeWindow/time.Second))

	for {
		var keys []string
		var err error
		keys, c, err = r.client.Scan(ctx, c, r.formatKey("hashrate", "*"), 100).Result()
		if err != nil {
			return total, err
		}
		for _, row := range keys {
			login := strings.Split(row, ":")[2]
			if _, ok := miners[login]; !ok {
				n, err := r.client.ZRemRangeByScore(ctx, r.formatKey("hashrate", login), "-inf", max).Result()
				if err != nil {
					return total, err
				}
				miners[login] = struct{}{}
				total += n
			}
		}
		if c == 0 {
			break
		}
	}

	return total, nil
}

// get min share rxhash  (high 8 bytes of rxhash as uint64) of a job
func (r *KvClient) IsMinShare(jobHash, login, share string, shareU64 uint64) bool {

	z, err := r.client.ZRangeWithScores(ctx, r.formatKey("mini", jobHash), 0, 0).Result()
	if err != nil {
		util.Error.Printf("Get %s min share failed %v", jobHash, err)
		return false
	}
	if len(z) == 0 || uint64(z[0].Score) > shareU64 {
		tx := r.client.TxPipeline()
		tx.SAdd(ctx, r.formatKey("submit", jobHash), share) //store submitted share
		tx.ZAdd(ctx, r.formatKey("mini", jobHash), redis.Z{Score: float64(shareU64), Member: login})
		if len(z) == 0 {
			tx.Expire(ctx, r.formatKey("submit", jobHash), expireDuration)
			tx.Expire(ctx, r.formatKey("mini", jobHash), expireDuration)
		}
		_, err := tx.Exec(ctx)
		if err != nil {
			util.Error.Println("store submitted min share error", err)
		}
		r.client.ZRemRangeByRank(ctx, r.formatKey("mini", jobHash), 1, -1).Result() // delete bigger hash, remain min hash and its miner address

		return true
	}
	return false
}

func (r *KvClient) IsPoolShare(jobHash, share string) bool {
	ok, err := r.client.SIsMember(ctx, r.formatKey("submit", jobHash), share).Result()
	if err != nil {
		util.Error.Println("check pool submitted share error", err)
		return false
	}
	return ok
}

// get all miners and their unpaid amount  which unpaid amount bigger than threshold
func (r *KvClient) GetMinersToPay(threshold int64) map[string]int64 {
	miners := make(map[string]int64)
	thresholdInt := threshold * 1e9
	iter := r.client.Scan(ctx, 0, r.formatKey("account*"), 0).Iterator()
	for iter.Next(ctx) {
		address := iter.Val()
		unpaid, err := r.client.HGet(ctx, iter.Val(), "unpaid").Int64()
		if err == nil {
			if unpaid > thresholdInt {
				miners[address[13:]] = unpaid
			}
		} else {
			util.Error.Println("iter miner unpaid error", address, err)
		}

	}
	if err := iter.Err(); err != nil {
		util.Error.Println("scan miner unpaid error", err)
		return nil
	}
	return miners
}

// get all miners and their diff proportion  which participated in a job
func (r *KvClient) GetProportion(jobHash string) map[string]float64 {
	miners := make(map[string]float64)
	poolDiff, _ := r.client.HGet(ctx, r.formatKey("pool", jobHash), "diff").Int64()
	fields, err := r.client.HKeys(ctx, r.formatKey("job", jobHash)).Result()
	if err != nil {
		util.Error.Println("get miners diff error", err)
		return nil
	}
	for _, address := range fields {
		diff, _ := r.client.HGet(ctx, r.formatKey("job", jobHash), address).Int64()
		miners[address] = float64(diff) / float64(poolDiff)
	}
	return miners
}

// get all miners addresses which participated in a job and max diff miner
func (r *KvClient) GetMinerName(jobHash string) []string {
	// var maxDiff int64
	var miners []string
	iter := r.client.Scan(ctx, 0, r.formatKey("job", jobHash), 0).Iterator()
	for iter.Next(ctx) {
		address := iter.Val()
		// diff, _ := r.client.HGet(ctx, r.formatKey("job", jobHash), address).Int64()
		// if diff > maxDiff {
		// 	maxDiff = diff
		// 	maxMiner = address
		// }
		miners = append(miners, address)
	}
	return miners
}

// set lowest hash finder reward of a job
func (r *KvClient) SetFinderReward(login string, reward pool.XdagjReward, fee float64, ms, ts int64) {
	if fee*float64(1e9) < 1.0 {
		return
	}
	// minimum hash finder
	raw, err := r.client.ZRange(ctx, r.formatKey("mini", reward.PreHash), 0, 0).Result()
	if err != nil {
		util.Error.Println("get lowest hash finder by job error", reward.PreHash, err)
		return
	}
	if len(raw) == 0 {
		util.Error.Println("lowest hash finder not found", reward.PreHash)
		return
	}
	err = r.SetMinerReward(raw[0], reward.TxBlock, reward.PreHash, fee, ms, ts)
	if err != nil {
		util.Error.Println("store hash finder reward error", reward.PreHash, err)
		return
	}
}

// func (r *KvClient) DivideSolo(login string, reward pool.XdagjReward, fee float64, ms, ts int64) {
// 	miners := r.GetMinerName(reward.PreHash)
// 	if len(miners) == 0 {
// 		util.Error.Println("solo direct reward miners count is 0", reward.PreHash)
// 		return
// 	}
// 	directPerMiner := fee / float64(len(miners))
// 	for _, miner := range miners {
// 		err := r.SetMinerReward(miner, reward.TxBlock, reward.PreHash, directPerMiner, ms, ts)
// 		if err != nil {
// 			util.Error.Println("store solo direct reward error", reward.PreHash, miner, directPerMiner, err)
// 			// continue
// 		}
// 	}
// }

func (r *KvClient) DivideEqual(login string, reward pool.XdagjReward, fee, amount float64, ms, ts int64) {
	miners := r.GetProportion(reward.PreHash)
	if len(miners) == 0 {
		util.Error.Println("equal direct reward miners count is 0", reward.PreHash)
		return
	}
	var directPerMiner float64
	if fee > 0 {
		directPerMiner = fee / float64(len(miners))
	}

	for miner, ratio := range miners {
		part := ratio*amount + directPerMiner
		err := r.SetMinerReward(miner, reward.TxBlock, reward.PreHash, part, ms, ts)
		if err != nil {
			util.Error.Println("store equal direct reward error", reward.PreHash, miner, part, err)
			continue
		}
	}
}

func (r *KvClient) formatKey(args ...interface{}) string {
	return join(r.prefix, join(args...))
}

func join(args ...interface{}) string {
	s := make([]string, len(args))
	for i, v := range args {
		switch x := v.(type) {
		case string:
			s[i] = x
		case int64:
			s[i] = strconv.FormatInt(x, 10)
		case uint64:
			s[i] = strconv.FormatUint(x, 10)
		case float64:
			s[i] = strconv.FormatFloat(x, 'f', 9, 64)
		case bool:
			if x {
				s[i] = "1"
			} else {
				s[i] = "0"
			}
		case *big.Int:
			if x != nil {
				s[i] = x.String()
			} else {
				s[i] = "0"
			}
		default:
			panic("Invalid type specified for conversion")
		}
	}
	return strings.Join(s, ":")
}

func (r *KvClient) PurgeRecords(window time.Duration) (int64, error) {
	now := util.MakeTimestamp() / 1000
	max := fmt.Sprint("(", now-int64(window/time.Second))
	total, err := r.client.ZRemRangeByScore(ctx, r.formatKey("pool", "donate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}

	n, err := r.client.ZRemRangeByScore(ctx, r.formatKey("pool", "rewards"), "-inf", max).Result()
	if err != nil {
		return total, err
	}
	total += n

	n, err = r.client.ZRemRangeByScore(ctx, r.formatKey("rejecthashrate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}
	total += n

	n, err = r.client.ZRemRangeByScore(ctx, r.formatKey("invalidhashrate"), "-inf", max).Result()
	if err != nil {
		return total, err
	}
	total += n

	var c uint64

	max = fmt.Sprint("(", now-int64(window/time.Second))

	for {
		miners := make(map[string]struct{})
		var keys []string
		var err error
		keys, c, err = r.client.Scan(ctx, c, r.formatKey("balance", "*"), 100).Result()
		if err != nil {
			util.Error.Println("purge balance scan", err.Error())
			break
		}
		for _, row := range keys {
			login := strings.Split(row, ":")[2]
			if _, ok := miners[login]; !ok {
				n, err := r.client.ZRemRangeByScore(ctx, r.formatKey("balance", login), "-inf", max).Result()
				if err != nil {
					util.Error.Println("purge: balance remove", err.Error(), login)
					break
				}
				miners[login] = struct{}{}
				total += n
			}
		}
		if c == 0 {
			break
		}
	}

	for {
		miners := make(map[string]struct{})
		var keys []string
		var err error
		keys, c, err = r.client.Scan(ctx, c, r.formatKey("rewards", "*"), 100).Result()
		if err != nil {
			util.Error.Println("purge rewards scan", err.Error())
			break
		}
		for _, row := range keys {
			login := strings.Split(row, ":")[2]
			if _, ok := miners[login]; !ok {
				n, err := r.client.ZRemRangeByScore(ctx, r.formatKey("rewards", login), "-inf", max).Result()
				if err != nil {
					util.Error.Println("purge: rewards remove", err.Error(), login)
					break
				}
				miners[login] = struct{}{}
				total += n
			}
		}
		if c == 0 {
			break
		}
	}

	for {
		miners := make(map[string]struct{})
		var keys []string
		var err error
		keys, c, err = r.client.Scan(ctx, c, r.formatKey("payment", "*"), 100).Result()
		if err != nil {
			util.Error.Println("purge payment scan", err.Error())
			return total, err
		}
		for _, row := range keys {
			login := strings.Split(row, ":")[2]
			if _, ok := miners[login]; !ok {
				n, err := r.client.ZRemRangeByScore(ctx, r.formatKey("payment", login), "-inf", max).Result()
				if err != nil {
					util.Error.Println("purge: payment remove", err.Error(), login)
					return total, err
				}
				miners[login] = struct{}{}
				total += n
			}
		}
		if c == 0 {
			break
		}
	}
	return total, nil
}
