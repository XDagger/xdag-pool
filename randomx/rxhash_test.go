package randomx

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestRx(t *testing.T) {
	Rx.NewSeed([]byte("test key 000"))
	correct, _ := hex.DecodeString("639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f")
	hash := Rx.CalculateHash([]byte("This is a test"))
	for i := 0; i < 10000; i++ {
		hash = Rx.CalculateHash([]byte("This is a test"))
	}
	if !bytes.Equal(hash, correct) {
		t.Logf("answer is incorrect: %x, %x", hash, correct)
		t.Fail()
	}
	fmt.Println("end")
}

func TestRxSlow(t *testing.T) {
	Rx.NewSeedSlow([]byte("test key 000"))
	correct, _ := hex.DecodeString("639183aae1bf4c9a35884cb46b09cad9175f04efd7684e7262a0ac1c2f0b4e3f")
	hash := Rx.CalculateHash([]byte("This is a test"))
	for i := 0; i < 10000; i++ {
		hash = Rx.CalculateHash([]byte("This is a test"))
	}
	if !bytes.Equal(hash, correct) {
		t.Logf("answer is incorrect: %x, %x", hash, correct)
		t.Fail()
	}
	fmt.Println("end")
}
