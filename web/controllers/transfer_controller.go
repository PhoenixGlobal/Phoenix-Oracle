package controllers

import (
	"PhoenixOracle/web"
	"fmt"
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/core/service/txmanager"
	"PhoenixOracle/db/models"
	"PhoenixOracle/web/presenters"

	"github.com/gin-gonic/gin"
)

// TransfersController can send PHB tokens to another address
type TransfersController struct {
	App phoenix.Application
}

func (tc *TransfersController) Create(c *gin.Context) {
	var tr models.SendEtherRequest
	if err := c.ShouldBindJSON(&tr); err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	store := tc.App.GetStore()

	etx, err := txmanager.SendEther(store.DB, tr.FromAddress, tr.DestinationAddress, tr.Amount, tc.App.GetEVMConfig().EvmGasLimitTransfer())
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, fmt.Errorf("transaction failed: %v", err))
		return
	}

	web.JsonAPIResponse(c, presenters.NewEthTxResource(etx), "eth_tx")
}
