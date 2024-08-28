package fil

import (
	"fmt"
	"github.com/coffeecloudgit/filecoin-wallet-signing/chain/types"
	"testing"
)

func TestBigIntToDecimals(t *testing.T) {
	bigInt, _ := types.BigFromString("1499851842714359995")
	decimal := BigIntToDecimals(&bigInt)

	fmt.Println(decimal)
}
