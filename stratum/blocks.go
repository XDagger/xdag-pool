package stratum

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	"github.com/XDagger/xdagpool/pool"
	"github.com/XDagger/xdagpool/randomx"
	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/xdago/base58"
)

type BlockTemplate struct {
	// diffInt64 int64
	// height     int64
	timestamp uint64
	// difficulty *big.Int
	taskIndex int
	jobHash   string
	buffer    []byte
	seedHash  []byte

	// blockReward  int64
	// txTotalFee   int64
	// nextSeedHash []byte
}

func (b *BlockTemplate) nextBlob(address string, extraNonce uint32, instanceId []byte) string {
	// 32 bytes (reserved) = 20 bytes (pool owner wallet address) + 4 bytes (extraNonce) + 4 bytes (instanceId) + 4bytes (share nonce)
	blobBuff := make([]byte, len(b.buffer)*2)

	copy(blobBuff, b.buffer)

	addr, _, _ := base58.ChkDec(address)
	copy(blobBuff[len(b.buffer)+12:], addr[:len(addr)-4]) // without checksum bytes

	enonce := make([]byte, 4)
	binary.BigEndian.PutUint32(enonce, extraNonce)
	copy(blobBuff[len(b.buffer):], enonce[:])

	copy(blobBuff[len(b.buffer)+4:], instanceId[:3])
	blobBuff[len(b.buffer)+11] = instanceId[3]

	return hex.EncodeToString(blobBuff)
}

func (s *StratumServer) fetchBlockTemplate(msg json.RawMessage) bool {
	var reply pool.XdagjTask
	err := json.Unmarshal(msg, &reply)

	if err != nil {
		util.Error.Printf("Error while refreshing block template: %s", err)
		return false
	}
	t := s.currentBlockTemplate()

	if t != nil && t.jobHash == reply.Data.PreHash {
		// // Fallback to height comparison
		// if len(reply.PrevHash) == 0 && reply.Height > t.height {
		// 	util.Info.Printf("New block to mine on %s at height %v, timestamp: %v", r.Name, reply.Height, reply.Timestamp)
		// } else {
		return false
		// }
	} else {
		util.Info.Printf("New block to mine on %s at jobHash %s,  timestamp: %v", s.config.NodeName, reply.Data.PreHash, reply.Timestamp)
	}
	// s.backend.AddWaiting(reply.Data.PreHash)

	newTemplate := BlockTemplate{
		// diffInt64:  reply.Difficulty,
		// difficulty: big.NewInt(reply.Difficulty),
		// height:     reply.Height,
		jobHash:   reply.Data.PreHash,
		timestamp: reply.Timestamp,
		taskIndex: reply.Index,
		// reservedOffset: reply.ReservedOffset,
	}
	newTemplate.seedHash, _ = hex.DecodeString(reply.Data.TashSeed)
	newTemplate.buffer, _ = hex.DecodeString(reply.Data.PreHash)

	if t == nil || reply.Data.TashSeed != hex.EncodeToString(t.seedHash) {
		if s.config.RxMode == "fast" {
			randomx.Rx.NewSeed(newTemplate.seedHash)
		} else {
			randomx.Rx.NewSeedSlow(newTemplate.seedHash)
		}
	}
	// newTemplate.nextSeedHash, _ = hex.DecodeString(reply.NextSeedHash)

	// // set blockReward and txTotalFee
	// var blockTemplateBlob blocktemplate.BlockTemplateBlob
	// bytesBuf := bytes.NewBuffer(newTemplate.buffer)
	// bufReader := io.Reader(bytesBuf)
	// err = blockTemplateBlob.UnPack(bufReader)
	// if err != nil {
	// 	util.Error.Printf("unpack block template blob fail, blob hex string: %s", reply.Blob)
	// 	return false
	// }

	// if len(blockTemplateBlob.Block.MinerTx.Vout) < 1 {
	// 	util.Error.Printf("invalid block template blob (Vout count < 1), blob hex string: %s", reply.Blob)
	// 	return false
	// }
	// newTemplate.blockReward = int64(blockTemplateBlob.Block.MinerTx.Vout[0].Amount)

	// if reply.ExpectedReward < newTemplate.blockReward {
	// 	util.Error.Printf("invalid block template blob (expectedReward: %d, blockReward: %d), blob hex string: %s",
	// 		reply.ExpectedReward, newTemplate.blockReward, reply.Blob)
	// 	return false
	// }
	// newTemplate.txTotalFee = reply.ExpectedReward - newTemplate.blockReward

	s.blockTemplate.Store(&newTemplate)
	return true
}
