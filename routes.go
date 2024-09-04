package main

import (
	"fmt"
	"github.com/coffeecloudgit/coffee-web/fil"
	"github.com/coffeecloudgit/filecoin-wallet-signing/cmd/msig"
	"html"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var signedTxMaps = make(map[string]string)
var minFil = 0.0001

func rateLimit(c *gin.Context) {
	ip := c.ClientIP()
	value := int(ips.Add(ip, 1))
	if value%50 == 0 {
		fmt.Printf("ip: %s, count: %d\n", ip, value)
	}
	if value >= 200 {
		if value%200 == 0 {
			fmt.Println("ip blocked")
		}
		c.Abort()
		c.String(http.StatusServiceUnavailable, "you were automatically banned :)")
	}
}

func index(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/room/hn")
}

func roomGET(c *gin.Context) {
	roomid := c.Param("roomid")
	nick := c.Query("nick")
	if len(nick) < 2 {
		nick = ""
	}
	if len(nick) > 13 {
		nick = nick[0:12] + "..."
	}
	c.HTML(http.StatusOK, "room_login.templ.html", gin.H{
		"roomid":    roomid,
		"nick":      nick,
		"timestamp": time.Now().Unix(),
	})

}

func roomPOST(c *gin.Context) {
	roomid := c.Param("roomid")
	nick := c.Query("nick")
	message := c.PostForm("message")
	message = strings.TrimSpace(message)

	validMessage := len(message) > 1 && len(message) < 200
	validNick := len(nick) > 1 && len(nick) < 14
	if !validMessage || !validNick {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "failed",
			"error":  "the message or nickname is too long",
		})
		return
	}

	post := gin.H{
		"nick":    html.EscapeString(nick),
		"message": html.EscapeString(message),
	}
	messages.Add("inbound", 1)
	room(roomid).Submit(post)
	c.JSON(http.StatusOK, post)
}

func streamRoom(c *gin.Context) {
	roomid := c.Param("roomid")
	listener := openListener(roomid)
	ticker := time.NewTicker(1 * time.Second)
	users.Add("connected", 1)
	defer func() {
		closeListener(roomid, listener)
		ticker.Stop()
		users.Add("disconnected", 1)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg := <-listener:
			messages.Add("outbound", 1)
			c.SSEvent("message", msg)
		case <-ticker.C:
			c.SSEvent("stats", Stats())
		}
		return true
	})
}

func signGET(c *gin.Context) {
	account := c.Param("account")
	nick := c.Query("nick")
	if len(nick) < 2 {
		nick = ""
	}
	if len(nick) > 13 {
		nick = nick[0:12] + "..."
	}
	c.HTML(http.StatusOK, "sign.templ.html", gin.H{
		"account":   account,
		"nick":      nick,
		"timestamp": time.Now().Unix(),
	})
}

func signActorAddress(c *gin.Context) {
	address := c.Param("address")
	actorAddress, err := fil.GetActorAddress(address)
	if err != nil {
		fmt.Println(err)
	}
	c.IndentedJSON(http.StatusOK, actorAddress)
}

func multiAccountInfo(c *gin.Context) {
	account := c.Param("account")
	//nick := c.Query("nick")
	accountInfo, err := fil.GetMultiAccountInfo(account)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{})
		return
	}

	if accountInfo.MultiSignTxs == nil {
		nullTxs := make([]msig.MultiSignTx, 0)
		accountInfo.MultiSignTxs = nullTxs
	}

	for i := 0; i < len(accountInfo.MultiSignTxs); i++ {
		key := fmt.Sprintf("%s_%d", account, accountInfo.MultiSignTxs[i].Id)
		if v, ok := signedTxMaps[key]; ok {
			accountInfo.MultiSignTxs[i].TxId = v
		}
	}

	sort.Slice(accountInfo.MultiSignTxs, func(i, j int) bool {
		return accountInfo.MultiSignTxs[i].Id > accountInfo.MultiSignTxs[j].Id
	})
	c.IndentedJSON(http.StatusOK, accountInfo)
}

func getCreateMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "参数错误",
		})
		return
	}
	from := json["from"]
	addresses := json["addresses"]
	threshold := json["threshold"]

	var fromStr = ""
	var addressesStr = ""
	var thresholdStr = ""
	var ok = true

	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from error",
		})
		return
	}

	if addressesStr, ok = addresses.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "addresses error",
		})
		return
	}

	if thresholdStr, ok = threshold.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "threshold error",
		})
		return
	}

	err, message := msig.CreateMessage(fromStr, addressesStr, thresholdStr)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getApproveMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "参数错误",
		})
		return
	}

	multiAddr := json["multiAddr"]
	txId := json["txId"]
	from := json["from"]
	stxId := ""
	fromStr := ""
	multiAddrStr := ""
	ok := false
	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from参数错误",
		})
		return
	}
	if multiAddrStr, ok = multiAddr.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "multiAddr参数错误",
		})
		return
	}

	if fNum, ok := txId.(float64); ok {
		stxId = strconv.FormatFloat(fNum, 'f', 0, 64)
	}
	err, message := msig.GetMessage(fromStr, multiAddrStr, stxId)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getCancelProposalMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "参数错误",
		})
		return
	}

	multiAddr := json["multiAddr"]
	txId := json["txId"]
	from := json["from"]
	stxId := ""
	fromStr := ""
	multiAddrStr := ""
	ok := false
	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from参数错误",
		})
		return
	}
	if multiAddrStr, ok = multiAddr.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "multiAddr参数错误",
		})
		return
	}

	if fNum, ok := txId.(float64); ok {
		stxId = strconv.FormatFloat(fNum, 'f', 0, 64)
	}
	err, message := msig.GetCancelMessage(fromStr, multiAddrStr, stxId)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getProposeTransferMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}
	from := json["from"]
	mts := json["mts"]
	accept := json["accept"]
	filAmount := json["fil"]
	strFil := ""
	if fNum, ok := filAmount.(float64); ok {
		if fNum < minFil {
			c.JSON(http.StatusBadGateway, gin.H{
				"message": fmt.Sprintf("最小转账金额：%f", minFil),
			})
			return
		}
		strFil = strconv.FormatFloat(fNum, 'f', -1, 64)
	}
	err, message := msig.ProposeTransferMessage(from.(string), mts.(string), accept.(string), strFil)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getProposeAddSignerMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}
	from := json["from"]
	mts := json["mts"]
	address := json["address"]

	var fromStr = ""
	var mtsStr = ""
	var addressStr = ""
	var ok = true

	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from error",
		})
		return
	}

	if mtsStr, ok = mts.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "MultiAddr error",
		})
		return
	}

	if addressStr, ok = address.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "edite address error",
		})
		return
	}

	err, message := msig.ProposeAddSignerMessage(fromStr, mtsStr, addressStr, false)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getProposeRemoveSignerMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}
	from := json["from"]
	mts := json["mts"]
	address := json["address"]

	var fromStr = ""
	var mtsStr = ""
	var addressStr = ""
	var ok = true

	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from error",
		})
		return
	}

	if mtsStr, ok = mts.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "MultiAddr error",
		})
		return
	}

	if addressStr, ok = address.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "remove address error",
		})
		return
	}

	err, message := msig.ProposeRemoveSignerMessage(fromStr, mtsStr, addressStr, false)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getProposeChangeThresholdMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}
	from := json["from"]
	mts := json["mts"]
	threshold := json["threshold"]

	var fromStr = ""
	var mtsStr = ""
	var thresholdStr = ""
	var thresholdInt = uint64(0)
	var ok = true

	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from error",
		})
		return
	}

	if mtsStr, ok = mts.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "MultiAddr error",
		})
		return
	}

	if thresholdStr, ok = threshold.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "threshold error",
		})
		return
	}

	if thresholdInt, err = strconv.ParseUint(thresholdStr, 10, 64); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "threshold error",
		})
		return
	}

	err, message := msig.ProposeChangeThresholdMessage(fromStr, mtsStr, thresholdInt)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func getProposeWithdrawMessage(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}
	from := json["from"]
	mts := json["mts"]
	miner := json["miner"]
	filAmount := json["fil"]
	strFil := ""
	if fNum, ok := filAmount.(float64); ok {
		if fNum < minFil {
			c.JSON(http.StatusBadGateway, gin.H{
				"message": fmt.Sprintf("最小提现金额：%f", minFil),
			})
			return
		}
		strFil = strconv.FormatFloat(fNum, 'f', -1, 64)
	}
	err, message := msig.ProposeWithdrawMessage(from.(string), mts.(string), miner.(string), strFil)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	post := gin.H{
		"message": message,
	}
	c.JSON(http.StatusOK, post)
}

func pushTx(c *gin.Context) {
	json := make(map[string]interface{})
	err := c.BindJSON(&json)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": err.Error(),
		})
		return
	}

	message := json["message"]
	signature := json["signature"]
	account := json["account"]
	id := json["id"]
	from := json["from"]

	var messageStr = ""
	var signatureStr = ""
	var accountStr = ""
	var idStr = ""
	var fromStr = ""
	var ok = true

	if messageStr, ok = message.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "message error",
		})
		return
	}
	if signatureStr, ok = signature.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "signature error",
		})
		return
	}
	if accountStr, ok = account.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "account error",
		})
		return
	}

	if idStr, ok = id.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "id error",
		})
		return
	}

	if fromStr, ok = from.(string); !ok {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "from error",
		})
		return
	}

	err, result := msig.PushTx(messageStr, signatureStr)

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "error",
		})
		return
	}

	key := fmt.Sprintf("%s_%s", accountStr, idStr)
	signedTxMaps[key] = fmt.Sprintf("%s_%s", fromStr, result)

	post := gin.H{
		"message": result,
	}
	c.JSON(http.StatusOK, post)
}

func getAccountInfo(c *gin.Context) {
	account := c.Param("account")

	accountInfo, err := msig.GetAccountInfo(account)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "error:" + err.Error(),
		})
		return
	}
	decimal := fil.BigIntToDecimals(&accountInfo.Balance)
	post := gin.H{
		"id":      accountInfo.Id,
		"address": account,
		"balance": decimal,
		"height":  accountInfo.Height.String(),
	}
	c.JSON(http.StatusOK, post)
}
