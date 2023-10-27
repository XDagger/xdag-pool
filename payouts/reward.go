package payouts

import (
	"github.com/XDagger/xdagpool/xdago/secp256k1"
	bip "github.com/XDagger/xdagpool/xdago/wallet"
)

var BipWallet *bip.Wallet
var BipAddress string
var BipKey *secp256k1.PrivateKey
