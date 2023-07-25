package stratum

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/big"

	"github.com/XDagger/xdagpool/base58"
	"github.com/XDagger/xdagpool/blocktemplate"
	"github.com/XDagger/xdagpool/util"
)

type BlockTemplate struct {
	diffInt64  int64
	height     int64
	timestamp  int64
	difficulty *big.Int
	// reservedOffset int
	jobHash  string
	buffer   []byte
	seedHash []byte

	blockReward  int64
	txTotalFee   int64
	nextSeedHash []byte
}

func (b *BlockTemplate) nextBlob(address string, extraNonce uint32, instanceId []byte) string {
	// 32 bytes (reserved) = 24 bytes (wallet address) + 4 bytes (extraNonce) + 4 bytes (instanceId)
	blobBuff := make([]byte, len(b.buffer)*2)

	copy(blobBuff, b.buffer)

	addr, _, _ := base58.ChkDec(address)
	copy(blobBuff[len(b.buffer):], addr[:])

	enonce := make([]byte, 4)
	binary.BigEndian.PutUint32(enonce, extraNonce)
	copy(blobBuff[len(b.buffer)+len(addr):], enonce[:])

	copy(blobBuff[len(b.buffer)+len(addr)+4:], instanceId[:])

	return hex.EncodeToString(blobBuff)
}

func (s *StratumServer) fetchBlockTemplate() bool {
	r := s.rpc()
	reply, err := r.GetBlockTemplate(8, s.config.Address)
	if err != nil {
		util.Error.Printf("Error while refreshing block template: %s", err)
		return false
	}
	t := s.currentBlockTemplate()

	if t != nil && t.jobHash == reply.GetJobHash() {
		// // Fallback to height comparison
		// if len(reply.PrevHash) == 0 && reply.Height > t.height {
		// 	util.Info.Printf("New block to mine on %s at height %v, timestamp: %v", r.Name, reply.Height, reply.Timestamp)
		// } else {
		return false
		// }
	} else {
		util.Info.Printf("New block to mine on %s at height %v, diff: %v, timestamp: %v", r.Name, reply.Height, reply.Difficulty, reply.Timestamp)
	}
	newTemplate := BlockTemplate{
		diffInt64:  reply.Difficulty,
		difficulty: big.NewInt(reply.Difficulty),
		height:     reply.Height,
		jobHash:    reply.GetJobHash(),
		timestamp:  reply.Timestamp,
		// reservedOffset: reply.ReservedOffset,
	}
	newTemplate.seedHash, _ = hex.DecodeString(reply.SeedHash)
	newTemplate.buffer, _ = hex.DecodeString(reply.Blob)
	// newTemplate.nextSeedHash, _ = hex.DecodeString(reply.NextSeedHash)

	// set blockReward and txTotalFee
	var blockTemplateBlob blocktemplate.BlockTemplateBlob
	bytesBuf := bytes.NewBuffer(newTemplate.buffer)
	bufReader := io.Reader(bytesBuf)
	err = blockTemplateBlob.UnPack(bufReader)
	if err != nil {
		util.Error.Printf("unpack block template blob fail, blob hex string: %s", reply.Blob)
		return false
	}

	if len(blockTemplateBlob.Block.MinerTx.Vout) < 1 {
		util.Error.Printf("invalid block template blob (Vout count < 1), blob hex string: %s", reply.Blob)
		return false
	}
	newTemplate.blockReward = int64(blockTemplateBlob.Block.MinerTx.Vout[0].Amount)

	if reply.ExpectedReward < newTemplate.blockReward {
		util.Error.Printf("invalid block template blob (expectedReward: %d, blockReward: %d), blob hex string: %s",
			reply.ExpectedReward, newTemplate.blockReward, reply.Blob)
		return false
	}
	newTemplate.txTotalFee = reply.ExpectedReward - newTemplate.blockReward

	s.blockTemplate.Store(&newTemplate)
	return true
}
