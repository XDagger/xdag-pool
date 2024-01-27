package cryptography

import (
	"github.com/XDagger/xdagpool/util"

	"github.com/XDagger/xdagpool/xdago/common"
	"github.com/XDagger/xdagpool/xdago/secp256k1"
	"github.com/XDagger/xdagpool/xdago/secp256k1/ecdsa"
)

func EcdsaSign(key *secp256k1.PrivateKey, hash []byte) (r, s common.Field) {
	signature := ecdsa.Sign(key, hash)
	// serial := signature.Serialize()
	// rLen := int(serial[3])
	// serial = serial[4:]
	// if rLen >= common.XDAG_FIELD_SIZE {
	// 	copy(r[:], serial[rLen-common.XDAG_FIELD_SIZE:rLen])
	// } else {
	// 	copy(r[:rLen], serial[:rLen])
	// }

	// sLen := int(serial[rLen+1])
	// serial = serial[rLen+2:]
	// if sLen >= common.XDAG_FIELD_SIZE {
	// 	copy(s[:], serial[sLen-common.XDAG_FIELD_SIZE:sLen])
	// } else {
	// 	copy(s[:sLen], serial[:sLen])
	// }

	util.Debug.Println("Sign")
	return signature.Bytes()
}

func EcdsaVerify(key *secp256k1.PublicKey, hash, r, s []byte) bool {
	var scalarR, scalarS secp256k1.ModNScalar
	if overflow := scalarR.SetByteSlice(r); overflow {
		util.Error.Println("ecdsa verify error:set scalar R overflow")
		return false
	}
	if overflow := scalarS.SetByteSlice(s); overflow {
		util.Error.Println("ecdsa verify error:set scalar S overflow")
		return false
	}
	signature := ecdsa.NewSignature(&scalarR, &scalarS)

	return signature.Verify(hash, key)
}
