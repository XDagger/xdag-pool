package kvstore

import (
	"errors"
	"strconv"
	"strings"
)

type DonateData struct {
	Timestamp int64
	Donate    float64
	TxBlock   string
}

func convertDonate(member string) (DonateData, error) {
	var donate DonateData
	fields := strings.Split(member, ":")
	if len(fields) != 4 {
		return donate, errors.New("donate data format error")
	}
	val, _ := strconv.ParseFloat(fields[0], 64)
	t, _ := strconv.ParseInt(fields[1], 10, 64)
	return DonateData{
		Donate:    val,
		Timestamp: t,
		TxBlock:   fields[3],
	}, nil
}

type PoolRewardsData struct {
	Timestamp int64
	Reward    float64
	Fee       float64
	TxBlock   string
}

func convertPoolRewards(member string) (PoolRewardsData, error) {
	var donate PoolRewardsData
	fields := strings.Split(member, ":")
	if len(fields) != 6 {
		return donate, errors.New("pool rewards data format error")
	}
	val, _ := strconv.ParseFloat(fields[0], 64)
	t, _ := strconv.ParseInt(fields[1], 10, 64)
	fee, _ := strconv.ParseFloat(fields[4], 64)
	return PoolRewardsData{
		Reward:    val,
		Timestamp: t,
		TxBlock:   fields[2],
		Fee:       fee,
	}, nil
}

type MinerRewardsData struct {
	Timestamp int64
	Reward    float64
	TxBlock   string
	Mode      string
}

func convertMinerRewards(member string) (MinerRewardsData, error) {
	var donate MinerRewardsData
	fields := strings.Split(member, ":")
	if len(fields) != 5 {
		return donate, errors.New("miner rewards data format error")
	}
	val, _ := strconv.ParseFloat(fields[0], 64)
	t, _ := strconv.ParseInt(fields[1], 10, 64)

	return MinerRewardsData{
		Reward:    val,
		Timestamp: t,
		TxBlock:   fields[2],
		Mode:      fields[4],
	}, nil
}

type MinerPaymentData struct {
	Timestamp int64
	Payment   float64
	TxBlock   string
}

func convertMinerPayment(member string) (MinerPaymentData, error) {
	var donate MinerPaymentData
	fields := strings.Split(member, ":")
	if len(fields) != 4 {
		return donate, errors.New("miner Payment data format error")
	}
	val, _ := strconv.ParseFloat(fields[0], 64)
	t, _ := strconv.ParseInt(fields[1], 10, 64)

	return MinerPaymentData{
		Payment:   val,
		Timestamp: t,
		TxBlock:   fields[2],
	}, nil
}

type MinerBalanceData struct {
	Action    string
	Timestamp int64
	Value     float64
	TxBlock   string
}

func convertMinerBalance(member string) (MinerBalanceData, error) {
	var donate MinerBalanceData
	fields := strings.Split(member, ":")
	if len(fields) != 4 {
		return donate, errors.New("miner Balance data format error")
	}
	val, _ := strconv.ParseFloat(fields[1], 64)
	t, _ := strconv.ParseInt(fields[2], 10, 64)

	return MinerBalanceData{
		Action:    fields[0],
		Value:     val,
		Timestamp: t,
		TxBlock:   fields[3],
	}, nil
}
