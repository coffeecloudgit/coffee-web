package fil

import (
	"github.com/coffeecloudgit/filecoin-wallet-signing/chain/types"
	"github.com/coffeecloudgit/filecoin-wallet-signing/cmd/msig"
	"github.com/shopspring/decimal"
)

var decimals int32 = 18

func GetMultiSigPendingTxs(account string) ([]msig.MultiSignTx, error) {
	return msig.GetMultiSigPendingTxs(account)
}

func GetAccountBalance(account string) (*types.BigInt, *types.TipSet, error) {
	return msig.GetAccountBalance(account)
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
