package orderbook

import "github.com/nikolaydubina/fpdecimal"

type Asset string

const (
	BTC Asset = "BTC"
	PKR Asset = "PKR"
)

var Assets []Asset

func IsAllowedAsset(asset string) bool {
	for _, a := range Assets {
		if asset == string(a) {
			return true
		}
	}
	return false
}

func init() {
	Assets = []Asset{BTC, PKR}
    fpdecimal.FractionDigits = 8
}
