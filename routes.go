package main

import (
	"fmt"
	"github.com/coffeecloudgit/coffee-web/fil"
	"github.com/coffeecloudgit/filecoin-wallet-signing/cmd/msig"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var signedTxMaps = make(map[string]string)

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

func signTxs(c *gin.Context) {
	account := c.Param("account")
	//nick := c.Query("nick")
	txs, err := fil.GetMultiSigPendingTxs(account)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{})
		return
	}

	if txs == nil {
		nullTxs := make([]msig.MultiSignTx, 0)
		txs = nullTxs
	}

	for i := 0; i < len(txs); i++ {
		key := fmt.Sprintf("%s_%d", account, txs[i].Id)
		if v, ok := signedTxMaps[key]; ok {
			txs[i].TxId = v
		}
	}
	c.IndentedJSON(http.StatusOK, txs)
}

func getMessage(c *gin.Context) {
	json := make(map[string]interface{})
	c.BindJSON(&json)
	multiAddr := json["multiAddr"]

	txId := json["txId"]
	from := json["from"]
	stxId := ""
	if fNum, ok := txId.(float64); ok {
		stxId = strconv.FormatFloat(fNum, 'f', 0, 64)
	}
	err, message := msig.GetMessage(from.(string), multiAddr.(string), stxId)

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
	c.BindJSON(&json)
	message := json["message"]
	signature := json["signature"]
	account := json["account"].(string)
	id := json["id"].(string)

	err, result := msig.PushTx(message.(string), signature.(string))

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "error",
		})
		return
	}

	key := fmt.Sprintf("%s_%s", account, id)
	signedTxMaps[key] = result

	post := gin.H{
		"message": result,
	}
	c.JSON(http.StatusOK, post)
}

func getAccountBalance(c *gin.Context) {
	//json := make(map[string]interface{})
	//c.BindJSON(&json)
	//account := json["account"]
	account := c.Param("account")

	balance, tipSet, err := msig.GetAccountBalance(account)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "error",
		})
		return
	}
	decimal := fil.BigIntToDecimals(balance)
	post := gin.H{
		"balance": decimal,
		"height":  tipSet.Height().String(),
	}
	c.JSON(http.StatusOK, post)
}
