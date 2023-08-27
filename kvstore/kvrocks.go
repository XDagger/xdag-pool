package kvstore

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/XDagger/xdagpool/pool"
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

func (r *KvClient) formatKey(args ...interface{}) string {
	return join(r.prefix, join(args...))
}

func (r *KvClient) checkPoWExist(timestamp uint64, params []string) (bool, error) {
	r.client.ZRemRangeByScore(ctx, r.formatKey("pow"), "-inf", fmt.Sprint("(", timestamp-3*64))
	val, err := r.client.ZAdd(ctx, r.formatKey("pow"), redis.Z{Score: float64(timestamp), Member: strings.Join(params, ":")}).Result()
	return val == 0, err
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
