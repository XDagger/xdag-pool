package kvstore

import "github.com/XDagger/xdagpool/util"

func (r *KvClient) GetTotalDonate() (float64, int64, error) {
	val, err := r.client.HGet(ctx, r.formatKey("pool", "account"), "donate").Int64()
	if err != nil {
		util.Error.Println("get pool total donate error", err)
		return 0, 0, err
	}
	count, err := r.client.ZCard(ctx, r.formatKey("pool", "donate")).Result()
	if err != nil {
		util.Error.Println("get pool donate count error", err)
		return float64(val) / 1e9, 0, err
	}
	return float64(val) / 1e9, count, nil
}

func (r *KvClient) GetDonateList(start, end int64) ([]DonateData, error) {
	val, err := r.client.ZRange(ctx, r.formatKey("pool", "donate"), start, end).Result()
	if err != nil {
		util.Error.Println("get pool donate page error", err)
		return nil, err
	}
	var list []DonateData
	for _, v := range val {
		d, err := convertDonate(v)
		if err != nil {
			continue
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *KvClient) GetPoolAccount() (float64, float64, float64, float64, error) {
	donate, err := r.client.HGet(ctx, r.formatKey("pool", "account"), "donate").Int64()
	if err != nil {
		util.Error.Println("get pool total donate error", err)
		return 0, 0, 0, 0, err
	}
	rewards, err := r.client.HGet(ctx, r.formatKey("pool", "account"), "rewards").Int64()
	if err != nil {
		util.Error.Println("get pool total rewards error", err)
		return 0, 0, 0, 0, err
	}
	payment, err := r.client.HGet(ctx, r.formatKey("pool", "account"), "payment").Int64()
	if err != nil {
		util.Error.Println("get pool total payment error", err)
		return 0, 0, 0, 0, err
	}
	unpaid, err := r.client.HGet(ctx, r.formatKey("pool", "account"), "unpaid").Int64()
	if err != nil {
		util.Error.Println("get pool total unpaid error", err)
		return 0, 0, 0, 0, err
	}
	return float64(rewards) / 1e9, float64(payment) / 1e9, float64(unpaid) / 1e9, float64(donate) / 1e9, nil
}

func (r *KvClient) GetTotalPoolRewards() (float64, int64, error) {
	val, err := r.client.HGet(ctx, r.formatKey("pool", "account"), "rewards").Int64()
	if err != nil {
		util.Error.Println("get pool total rewards error", err)
		return 0, 0, err
	}
	count, err := r.client.ZCard(ctx, r.formatKey("pool", "rewards")).Result()
	if err != nil {
		util.Error.Println("get pool rewards count error", err)
		return float64(val) / 1e9, 0, err
	}
	return float64(val) / 1e9, count, nil
}

func (r *KvClient) GetPoolRewardsList(start, end int64) ([]PoolRewardsData, error) {
	val, err := r.client.ZRange(ctx, r.formatKey("pool", "rewards"), start, end).Result()
	if err != nil {
		util.Error.Println("get pool rewards page error", err)
		return nil, err
	}
	var list []PoolRewardsData
	for _, v := range val {
		d, err := convertPoolRewards(v)
		if err != nil {
			continue
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *KvClient) GetMinerAccount(address string) (float64, float64, float64, error) {
	reward, err := r.client.HGet(ctx, r.formatKey("account", address), "reward").Int64()
	if err != nil {
		util.Error.Println("get miner total donate error", err, address)
		return 0, 0, 0, err
	}

	payment, err := r.client.HGet(ctx, r.formatKey("account", address), "payment").Int64()
	if err != nil {
		util.Error.Println("get miner total payment error", err, address)
		return 0, 0, 0, err
	}
	unpaid, err := r.client.HGet(ctx, r.formatKey("account", address), "unpaid").Int64()
	if err != nil {
		util.Error.Println("get miner total unpaid error", err, address)
		return 0, 0, 0, err
	}
	return float64(reward) / 1e9, float64(payment) / 1e9, float64(unpaid) / 1e9, nil
}

func (r *KvClient) GetMinerUnpaid(address string) float64 {

	unpaid, err := r.client.HGet(ctx, r.formatKey("account", address), "unpaid").Int64()
	if err != nil {
		util.Error.Println("get pool total unpaid error", err)
		return 0
	}
	return float64(unpaid) / 1e9
}

func (r *KvClient) MinerTotalRewards(address string) (float64, int64, error) {
	rewards, err := r.client.HGet(ctx, r.formatKey("account", address), "reward").Int64()
	if err != nil {
		util.Error.Println("get miner total rewards error", err)
		return 0, 0, err
	}
	count, err := r.client.ZCard(ctx, r.formatKey("rewards", address)).Result()
	if err != nil {
		util.Error.Println("get miner rewards count error", err)
		return float64(rewards) / 1e9, 0, err
	}
	return float64(rewards) / 1e9, count, nil
}

func (r *KvClient) MinerRewardsList(address string, start, end int64) ([]MinerRewardsData, error) {
	val, err := r.client.ZRange(ctx, r.formatKey("rewards", address), start, end).Result()
	if err != nil {
		util.Error.Println("get miner rewards page error", err)
		return nil, err
	}
	var list []MinerRewardsData
	for _, v := range val {
		d, err := convertMinerRewards(v)
		if err != nil {
			continue
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *KvClient) MinerTotalPayment(address string) (float64, int64, error) {
	payment, err := r.client.HGet(ctx, r.formatKey("account", address), "payment").Int64()
	if err != nil {
		util.Error.Println("get miner total payment error", err)
		return 0, 0, err
	}
	count, err := r.client.ZCard(ctx, r.formatKey("payment", address)).Result()
	if err != nil {
		util.Error.Println("get miner payment count error", err)
		return float64(payment) / 1e9, 0, err
	}
	return float64(payment) / 1e9, count, nil
}

func (r *KvClient) MinerPaymentList(address string, start, end int64) ([]MinerPaymentData, error) {
	val, err := r.client.ZRange(ctx, r.formatKey("payment", address), start, end).Result()
	if err != nil {
		util.Error.Println("get miner payment page error", err)
		return nil, err
	}
	var list []MinerPaymentData
	for _, v := range val {
		d, err := convertMinerPayment(v)
		if err != nil {
			continue
		}
		list = append(list, d)
	}
	return list, nil
}

func (r *KvClient) MinerBalanceList(address string, start, end int64) (int64, []MinerBalanceData, error) {

	count, err := r.client.ZCard(ctx, r.formatKey("balance", address)).Result()
	if err != nil {
		util.Error.Println("get miner balance count error", err)
		return 0, nil, err
	}

	val, err := r.client.ZRange(ctx, r.formatKey("balance", address), start, end).Result()
	if err != nil {
		util.Error.Println("get miner balance page error", err)
		return count, nil, err
	}
	var list []MinerBalanceData
	for _, v := range val {
		d, err := convertMinerBalance(v)
		if err != nil {
			continue
		}
		list = append(list, d)
	}
	return count, list, nil
}
