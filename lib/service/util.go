package service

import (
	"crypto/rand"
	"math/big"
)

const BTC_ASSET_ID   = "native-asset-bitcoin"
const BTC_ASSET_NAME = "bitcoin"

type AssetType int64
// https://lightning.engineering/api-docs/api/taproot-assets/universe/query-asset-stats#taprpcassettype
const (
	Normal      = 0
	Collectible = 1
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
