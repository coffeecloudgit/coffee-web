package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"runtime"
)

func main() {
	//ConfigRuntime()
	//StartWorkers()
	StartGin()
}

// StartGin starts gin web server with setting router.
func StartGin() {
	gin.SetMode(gin.DebugMode)
	router := gin.New()
	router.Use(rateLimit, gin.Recovery())
	//router.LoadHTMLGlob("resources/*.templ.html")
	//router.Static("/static", "resources/static")
	router.GET("/sign/:account", signGET)
	router.POST("/sign/actor/:address", signActorAddress)
	router.POST("/sign/multi-account-info/:account", multiAccountInfo)

	router.POST("/sign/pushTx", pushTx)
	router.POST("/sign/account/info/:account", getAccountInfo)

	router.POST("/sign/create-message", getCreateMessage)
	router.POST("/sign/propose-transfer-message", getProposeTransferMessage)
	router.POST("/sign/propose-withdraw-message", getProposeWithdrawMessage)
	router.POST("/sign/propose-add-signer-message", getProposeAddSignerMessage)
	router.POST("/sign/propose-remove-signer-message", getProposeRemoveSignerMessage)
	router.POST("/sign/propose-change-threshold-message", getProposeChangeThresholdMessage)
	router.POST("/sign/get-approve-message", getApproveMessage)
	router.POST("/sign/get-cancel-proposal-message", getCancelProposalMessage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}
	if err := router.Run(":" + port); err != nil {
		log.Panicf("error: %s", err)
	}
}

// ConfigRuntime sets the number of operating system threads.
func ConfigRuntime() {
	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	fmt.Printf("Running with %d CPUs\n", nuCPU)
}

// StartWorkers start starsWorker by goroutine.
func StartWorkers() {
	go statsWorker()
}
