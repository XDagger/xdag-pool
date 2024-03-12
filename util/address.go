package util

import (
	"github.com/XDagger/xdagpool/xdago/base58"
)

func ValidateAddress(address string) bool {
	_, _, err := base58.ChkDec(address)

	return err == nil
}

// func ConvertBlob(blob []byte) []byte {
// 	return blob
// }

func ValidatePasswd(encrypted, pswd string) bool {
	b, err := Ae64Decode(encrypted, []byte(pswd))
	if err != nil {
		return false
	}

	// check address
	if !ValidateAddress(string(b)) {
		return false
	}

	return true
}
