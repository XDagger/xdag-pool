package payouts

import (
	"fmt"
	"strconv"
	"testing"
)

func TestChunk(t *testing.T) {

	miners := make(map[string]int64)
	for i := 1; i <= 11; i++ {
		miners[strconv.Itoa(i)] = int64(i)
	}

	remark := "hello"
	var chunkSize int
	if len(remark) > 0 {
		chunkSize = 10
	} else {
		chunkSize = 11
	}
	batchAddress := make([]string, 0, chunkSize)
	batchAmount := make([]int64, 0, chunkSize)
	for address, amount := range miners {

		if amount > 0 {
			batchAddress = append(batchAddress, address)
			batchAmount = append(batchAmount, amount)
		}
		if len(batchAddress) == chunkSize {
			chunkReset(&batchAddress, &batchAmount)
		}
	}

	if len(batchAddress) > 0 {
		chunkReset(&batchAddress, &batchAmount)
	}
}

func chunkReset(batchAddress *[]string, batchAmount *[]int64) {
	fmt.Println("=========================")
	fmt.Println(batchAmount)
	fmt.Println(batchAddress)
	*batchAddress = (*batchAddress)[:0]
	*batchAmount = (*batchAmount)[:0]
}
