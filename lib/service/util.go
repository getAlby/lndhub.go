package service

import (
	"crypto/rand"
	"math/big"
)

func randBytesFromStr(length int, from string) ([]byte, error) {
	b := make([]byte, length)
	fromLenBigInt := big.NewInt(int64(len(from)))
	for i := range b {
		r, err := rand.Int(rand.Reader, fromLenBigInt)
		if err != nil {
			return nil, err
		}
		b[i] = from[r.Int64()]
	}
	return b, nil
}


func allEqual(arr []int64) bool {
	for i := 1; i < len(arr); i++ {
		// compare every item to the first positioned item
		if arr[i] != arr[0] {
			return false
		}
	}
	return true
}
