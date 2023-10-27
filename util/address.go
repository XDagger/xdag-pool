package util

import "github.com/XDagger/xdagpool/xdago/base58"

func ValidateAddress(address string) bool {
	_, _, err := base58.ChkDec(address)

	return err == nil
}

// func ConvertBlob(blob []byte) []byte {
// 	return blob
// }
