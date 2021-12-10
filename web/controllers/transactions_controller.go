package controllers

import (
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/orm"
	"PhoenixOracle/web/presenters"

	"PhoenixOracle/web"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type TransactionsController struct {
	App phoenix.Application
}

func (tc *TransactionsController) Index(c *gin.Context, size, page, offset int) {
	txs, count, err := tc.App.GetStore().EthTransactionsWithAttempts(offset, size)
	ptxs := make([]presenters.EthTxResource, len(txs))
	for i, tx := range txs {
		tx.EthTxAttempts[0].EthTx = tx
		ptxs[i] = presenters.NewEthTxResourceFromAttempt(tx.EthTxAttempts[0])
	}
	web.PaginatedResponse(c, "transactions", size, page, ptxs, count, err)
}

func (tc *TransactionsController) Show(c *gin.Context) {
	hash := common.HexToHash(c.Param("TxHash"))

	ethTxAttempt, err := tc.App.GetStore().FindEthTxAttempt(hash)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("Transaction not found"))
		return
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewEthTxResourceFromAttempt(*ethTxAttempt), "transaction")
}
