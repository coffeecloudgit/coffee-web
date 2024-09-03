package fil

import (
	"github.com/coffeecloudgit/filecoin-wallet-signing/chain/types"
	"github.com/coffeecloudgit/filecoin-wallet-signing/cmd/msig"
	"github.com/shopspring/decimal"
)

var decimals int32 = 18

func GetMultiAccountInfo(account string) (*msig.MultiAccountInfo, error) {
	return msig.GetMultiAccountInfo(account)
}

func GetAccountInfo(account string) (*msig.AccountInfo, error) {
	return msig.GetAccountInfo(account)
}

func BigIntToDecimals(amount *types.BigInt) decimal.Decimal {
	if amount == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(amount.Int, 0).Shift(-decimals)
}

func GetActorAddress(address string) (map[string]interface{}, error) {
	return msig.GetActorAddress(address)
}
