package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
)

// var encode bool
// var decode bool
var help bool

var poolKey string
var address string
var wallet string
var kv string

func init() {
	flag.BoolVar(&help, "h", false, "this help")

	// flag.BoolVar(&encode, "e", false, "to encode text")
	// flag.BoolVar(&decode, "d", false, "to decode text")

	flag.StringVar(&poolKey, "p", "", "set pool password")
	flag.StringVar(&address, "a", "", "set pool address")
	flag.StringVar(&wallet, "w", "", "set pool wallet password")
	flag.StringVar(&kv, "k", "", "set kv store password")
}

func usage() {
	fmt.Fprintf(os.Stderr, `encrypt and decrypt tool for password
Usage: encrypt [-h] [-p pool password] [-a address] [-w wallet password] [-k kv store password]
Options:
`)
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	if help {
		usage()
		return
	}

	// if encode && decode {
	// 	fmt.Fprintln(os.Stderr, "Can only choose one action: encrypt or decrypt!")
	// 	return
	// }

	if poolKey == "" {
		fmt.Fprintln(os.Stderr, "Must set pool password to encrypt or decrypt!")
		return
	}

	if len(poolKey) > 16 {
		fmt.Fprintln(os.Stderr, "Pool password length must be less than or equal 16!")
		return
	}

	keyBytes := []byte(poolKey)

	if len(address) > 0 {
		addr, err := Ae64Encode([]byte(address), keyBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Encrypt address error: "+err.Error())
			return
		}
		fmt.Println("address: " + string(addr[:]))
	}

	if len(wallet) > 0 {
		wp, err := Ae64Encode([]byte(wallet), keyBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Encrypt wallet password error: "+err.Error())
			return
		}
		fmt.Println("wallet password: " + string(wp[:]))
	}

	if len(kv) > 0 {
		kp, err := Ae64Encode([]byte(kv), keyBytes)
		if err != nil {
			fmt.Fprintln(os.Stderr, "kv store password error: "+err.Error())
			return
		}
		fmt.Println("kv store password: " + string(kp[:]))
	}
}

// MODE: CBC, Key Size: 128bits, IV and Secret Key: 16 characters long( add '*' if length not enough)
func Ae64Encode(src []byte, key []byte) (string, error) {
	if len(key) <= 16 {
		paddingCount := 16 - len(key)
		for i := 0; i < paddingCount; i++ {
			key = append(key, byte('*'))
		}
	} else {
		key = key[0:16]
	}

	encryptBytes, err := encryptAES(src, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptBytes), nil
}

func padding(src []byte, blockSize int) []byte {
	padNum := blockSize - len(src)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(src, pad...)
}

func encryptAES(src []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	src = padding(src, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, key)
	blockMode.CryptBlocks(src, src)
	return src, nil
}
