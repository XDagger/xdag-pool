package payouts

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/XDagger/xdagpool/util"
	"github.com/XDagger/xdagpool/xdago/common"
	"github.com/XDagger/xdagpool/xdago/cryptography"
	"github.com/XDagger/xdagpool/xdago/secp256k1"
	xdagoUtils "github.com/XDagger/xdagpool/xdago/utils"
)

// batch transfer awards to miners
func TransferChunkRpc(amount []int64, from string, to []string, remark string, key *secp256k1.PrivateKey) (string, error) {

	blockHexStr, total := TransactionChunkBlock(from, to, remark, amount, key)
	util.Debug.Println(blockHexStr)
	if blockHexStr == "" {
		return "", errors.New("chunk create transaction block error")
	}

	txHash := blockHash(blockHexStr)
	util.Info.Println(from, float64(total)/float64(1e9), remark, "transaction:", txHash)

	hash, err := xdagjRpc("xdag_sendRawTransaction", blockHexStr)
	if err != nil {
		return "", err
	}

	if hash == "" {
		return "", errors.New("chunk transaction rpc return empty hash")
	}

	if !ValidateXdagAddress(hash) {
		return "", errors.New(hash)
	}

	if hash != txHash {
		util.Error.Println("want", txHash, "get", hash)
		return "", errors.New("chunk transaction block hash error")
	}

	return hash, nil
}

// batch transactions block
func TransactionChunkBlock(from string, to []string, remark string, value []int64, key *secp256k1.PrivateKey) (string, int64) {
	if key == nil {
		util.Error.Println("transaction default key error")
		return "", 0
	}
	var inAddress string
	var err error

	inAddress, err = checkBase58Address(from)
	if err != nil {
		util.Error.Println("pool address in transaction error")
		return "", 0
	}

	outs := make([]string, 0, len(to))
	for _, out := range to {
		outAddress, err := checkBase58Address(out)
		if err != nil {
			util.Error.Println(err)
			return "", 0
		}
		outs = append(outs, outAddress)
	}

	var remarkBytes [common.XDAG_FIELD_SIZE]byte
	if len(remark) > 0 {
		if ValidateRemark(remark) {
			copy(remarkBytes[:], remark)
		} else {
			util.Error.Println("remark error")
			return "", 0
		}
	}

	var total int64
	for _, v := range value {
		total += v
	}

	var amount float64

	valBytes := make([][8]byte, len(value))
	for i, val := range value {
		if val > 0 {
			f := float64(val) / float64(1e9)
			transVal := xdagoUtils.Xdag2Amount(f)
			binary.LittleEndian.PutUint64(valBytes[i][:], transVal)
			amount += f
		} else {
			util.Error.Println("transaction value is zero")
			return "", 0
		}
	}

	var amountBytes [8]byte
	transVal := xdagoUtils.Xdag2Amount(amount)
	binary.LittleEndian.PutUint64(amountBytes[:], transVal)

	t := xdagoUtils.GetCurrentTimestamp()
	var timeBytes [8]byte
	binary.LittleEndian.PutUint64(timeBytes[:], t)

	var sb strings.Builder
	// header: transport
	sb.WriteString("0000000000000000")

	compKey := key.PubKey().SerializeCompressed()

	// header: field types
	sb.WriteString(FieldChunkTypes(false,
		len(remark) > 0, compKey[0] == secp256k1.PubKeyFormatCompressedEven, len(to)))

	// header: timestamp
	sb.WriteString(hex.EncodeToString(timeBytes[:]))
	// header: fee
	sb.WriteString("0000000000000000")

	// tranx_nonce
	sb.WriteString("0000000000000000000000000000000000000000000000000000000000000000")

	// input field: input address
	sb.WriteString(inAddress)
	// input field: input value
	sb.WriteString(hex.EncodeToString(amountBytes[:]))
	// output field: output address
	for i, address := range outs {
		sb.WriteString(address)
		// output field: out value
		sb.WriteString(hex.EncodeToString(valBytes[i][:]))
	}
	// remark field
	if len(remark) > 0 {
		sb.WriteString(hex.EncodeToString(remarkBytes[:]))
	}
	// public key field
	sb.WriteString(hex.EncodeToString(compKey[1:33]))

	r, s, n := ChunkTranxSign(sb.String(), key)
	// sign field: sign_r
	sb.WriteString(r)
	// sign field: sign_s
	sb.WriteString(s)
	// zero fields
	if n > 2 {
		for i := 0; i < n-2; i++ {
			sb.WriteString("0000000000000000000000000000000000000000000000000000000000000000")
		}
	}
	return sb.String(), total
}

func FieldChunkTypes(isTest, hasRemark, isPubKeyEven bool, chunkSize int) string {

	// 1/8--E--2/C--D--D--D--D--D--D--D--D--D--[9/D]--6/7--5--5
	// header(main/test)--tranx_nonce--input(old/new)--outputs--[remark]--pubKey(even/odd)--sign_r--sign_s
	fieldTypes := make([]byte, 16)

	if isTest {
		fieldTypes[0] = 0x08 // test net
	} else {
		fieldTypes[0] = 0x01 // main net
	}
	fieldTypes[1] = 0x0E // new address

	fieldTypes[2] = 0x0C // new address

	index := 3
	for i := 0; i < chunkSize; i++ {
		fieldTypes[3+i] = 0x0D
		index += 1
	}
	if hasRemark {
		fieldTypes[index] = 0x09 // with remark
		index += 1
	}
	if isPubKeyEven {
		fieldTypes[index] = 0x06 // even public key
	} else {
		fieldTypes[index] = 0x07 // odd public key
	}
	fieldTypes[index+1] = 0x05
	fieldTypes[index+2] = 0x05

	types := make([]byte, 8)
	for i := 0; i < 8; i++ {
		types[i] |= fieldTypes[i*2]
		types[i] |= fieldTypes[i*2+1] << 4
	}

	return hex.EncodeToString(types)
}

func ChunkTranxSign(block string, key *secp256k1.PrivateKey) (string, string, int) {
	var sb strings.Builder
	sb.WriteString(block)

	n := (1024 - len(block)) / 64

	for i := 0; i < n; i++ {
		sb.WriteString("0000000000000000000000000000000000000000000000000000000000000000")
	}

	pubKey := key.PubKey().SerializeCompressed()
	sb.WriteString(hex.EncodeToString(pubKey[:]))

	b, _ := hex.DecodeString(sb.String())

	hash := cryptography.HashTwice(b)

	r, s := cryptography.EcdsaSign(key, hash[:])
	util.Debug.Println("Sign")
	return hex.EncodeToString(r[:]), hex.EncodeToString(s[:]), n
}
