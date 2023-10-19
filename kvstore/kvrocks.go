package kvstore

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/util"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

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

func NewKvFailoverClient(cfg *pool.StorageConfigFailover, prefix string) *KvClient {
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    cfg.MasterName,
		SentinelAddrs: cfg.SentinelEndpoints,
		Password:      cfg.Password,
		DB:            int(cfg.Database),
		PoolSize:      cfg.PoolSize,
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
func (r *KvClient) writeShare(tx redis.Pipeliner, ms, ts int64, login, id string, diff int64, expire time.Duration) {
	tx.HIncrBy(ctx, r.formatKey("shares", "roundCurrent"), login, diff)
	tx.ZAdd(ctx, r.formatKey("hashrate"), redis.Z{Score: float64(ts), Member: join(diff, login, id, ms)})
	tx.ZAdd(ctx, r.formatKey("hashrate", login), redis.Z{Score: float64(ts), Member: join(diff, id, ms)})
	tx.Expire(ctx, r.formatKey("hashrate", login), expire) // Will delete hashrates for miners that gone
	// tx.HSet(ctx, r.formatKey("miners", login), "lastShare", strconv.FormatInt(ts, 10))
}

func (r *KvClient) WriteBlock(login, id string, params []string, diff, roundDiff int64, shareU64 uint64,
	timestamp uint64, window time.Duration, jobHash string) (bool, error) {
	// exist, err := r.checkPoWExist(login, params)
	// if err != nil {
	// 	return false, err
	// }
	sharesKey := login + ":" + strings.Join(params, ",")
	exist := util.MinedShares.ShareExist(sharesKey)

	if exist {
		return true, nil
	}
	util.MinedShares.Set(sharesKey)                // set share existence
	util.HashrateRank.IncShareByKey(login, diff)   // accumulated for hashrate rank
	util.HashrateRank.IncShareByKey("total", diff) // accumulated for total hashrate

	tx := r.client.TxPipeline()
	// defer tx.Close()

	ms := util.MakeTimestamp()
	ts := ms / 1000

	r.writeShare(tx, ms, ts, login, id, diff, window)
	tx.HSet(ctx, r.formatKey("works", login+"."+id), "lastShare", strconv.FormatInt(ts, 10))
	tx.HSet(ctx, r.formatKey("miners", login), "lastShare", strconv.FormatInt(ts, 10))
	tx.HSet(ctx, r.formatKey("stats"), "lastShare", strconv.FormatInt(ts, 10))
	// tx.HDel(ctx, r.formatKey("stats"), "roundShares")
	// tx.ZIncrBy(ctx, r.formatKey("finders"), 1, login)
	// tx.HIncrBy(ctx, r.formatKey("miners", login), "blocksFound", 1)

	// tx.Rename(ctx, r.formatKey("shares", "roundCurrent"), r.formatRound(int64(height), params[0]))
	// tx.HGetAll(ctx, r.formatRound(int64(height), params[0]))
	tx.HIncrBy(ctx, r.formatKey(jobHash), login, diff) // accumulate miners diff of the job (identified by job hash)
	tx.ZAdd(ctx, r.formatKey(jobHash), redis.Z{Score: float64(shareU64), Member: login})
	tx.ZRemRangeByRank(ctx, r.formatKey(jobHash), 1, -1) // delete bigger hash, remain min hash and its miner address

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
func (r *KvClient) SetReward(login, txHash, jobHash string, reward float64, ms, ts int64) error {
	tx := r.client.TxPipeline()
	tx.HIncrBy(ctx, r.formatKey("account", login), "reward", int64(reward*1e9))
	tx.HIncrBy(ctx, r.formatKey("account", login), "unpaid", int64(reward*1e9))
	tx.ZAdd(ctx, r.formatKey("rewards", login), redis.Z{Score: float64(ts), Member: join(reward, ms, txHash, jobHash)})
	tx.ZAdd(ctx, r.formatKey("balance", login), redis.Z{Score: float64(ts), Member: join("reward", reward, ms, txHash, jobHash)})
	_, err := tx.Exec(ctx)
	return err
}

func (r *KvClient) SetPayment(login, txHash string, payment float64, ms, ts int64) error {
	tx := r.client.TxPipeline()
	tx.HIncrBy(ctx, r.formatKey("account", login), "payment", int64(payment*1e9))
	tx.HIncrBy(ctx, r.formatKey("account", login), "unpaid", -1*int64(payment*1e9))
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "payment", int64(payment*1e9))
	tx.HIncrBy(ctx, r.formatKey("pool", "account"), "unpaid", -1*int64(payment*1e9))
	tx.ZAdd(ctx, r.formatKey("payment", login), redis.Z{Score: float64(ts), Member: join(payment, ms, txHash)})
	tx.ZAdd(ctx, r.formatKey("balance", login), redis.Z{Score: float64(ts), Member: join("payment", payment, ms, txHash)})
	_, err := tx.Exec(ctx)
	return err
}

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
func (r *KvClient) GetMinShare(jobHash string) uint64 {
	z, err := r.client.ZRangeWithScores(ctx, r.formatKey(jobHash), 0, 0).Result()
	if err != nil || len(z) < 1 {
		util.Error.Printf("Get %s min share failed %v", jobHash, err)
		util.BlockLog.Printf("Get %s min share failed %v", jobHash, err)
		return math.MaxUint64
	}

	return uint64(z[0].Score)
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
			s[i] = strconv.FormatFloat(x, 'f', 0, 64)
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
