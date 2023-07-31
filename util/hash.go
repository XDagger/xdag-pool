package util

import (
	"crypto/sha256"

	"github.com/XDagger/xdagpool/randomx"
)

func FastHash(blob []byte) []byte {
	hash := sha256.Sum256(blob)
	h := sha256.Sum256(hash[:])
	return h[:]
}

func RxHash(blob []byte) []byte {
	return randomx.Rx.CalculateHash(blob)
}
