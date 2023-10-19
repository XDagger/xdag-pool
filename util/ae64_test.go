package util

import (
	"fmt"
	"testing"
)

func TestAe64Encode(t *testing.T) {
	src := []byte("0xae17a0398694c94d4f861c5aa1b215adbf0d48b5")
	src2 := []byte("")
	key := []byte("12345678")
	dst, _ := Ae64Encode(src, key)
	dst2, _ := Ae64Encode(src2, key)
	fmt.Println(dst)
	fmt.Println(dst2)
}

func TestAe64Decode(t *testing.T) {
	// src := "bg2Z2F+OA6LTR5VQjsOiLOH2YqSiFbETBQWlZ25nt51UsZrrRqWSvWJT4yX6Oz5r"
	// src2 := "nuaXMECKl3fLIRwzJyKXHA=="
	src := "ntr5fRwReQRQ7U2RAv4EzjMtXo/kazyl/bWaQyirr8sjM8B38/vZC/0+IPkRNKqg"
	src2 := "XAhBZ5thsGAzhnxJwwPgiQ=="
	key := []byte("12345678")
	orgi, _ := Ae64Decode(src, key)
	orgi2, _ := Ae64Decode(src2, key)
	fmt.Println(string(orgi))
	fmt.Println(string(orgi2))
}
