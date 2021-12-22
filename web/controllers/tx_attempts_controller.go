package controllers

import (
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/web/presenters"

	"PhoenixOracle/web"
	"github.com/gin-gonic/gin"
)

type TxAttemptsController struct {
	App phoenix.Application
}

func (tac *TxAttemptsController) Index(c *gin.Context, size, page, offset int) {
	attempts, count, err := tac.App.GetStore().EthTxAttempts(offset, size)
	ptxs := make([]presenters.EthTxResource, len(attempts))
	for i, attempt := range attempts {
		ptxs[i] = presenters.NewEthTxResourceFromAttempt(attempt)
	}
	web.PaginatedResponse(c, "transactions", size, page, ptxs, count, err)
}
