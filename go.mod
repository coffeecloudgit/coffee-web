module github.com/coffeecloudgit/coffee-web

go 1.14

require (
	github.com/coffeecloudgit/filecoin-wallet-signing v1.0.4
	github.com/dustin/go-broadcast v0.0.0-20171205050544-f664265f5a66
	github.com/gin-gonic/gin v1.10.0
	github.com/libp2p/go-libp2p-core v0.20.1 // indirect
	github.com/manucorporat/stats v0.0.0-20180402194714-3ba42d56d227
	github.com/shopspring/decimal v1.4.0
)

//replace github.com/coffeecloudgit/filecoin-wallet-signing => ../filecoin-wallet-signing
