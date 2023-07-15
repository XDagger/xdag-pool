package util

import "crypto/sha256"

func FastHash(blob []byte) []byte {
	hash := sha256.Sum256(blob)
	h := sha256.Sum256(hash[:])
	return h[:]
}

func RxHash(blob []byte, seedHash []byte, height int64, maxConcurrency uint) []byte {
	return blob
}
