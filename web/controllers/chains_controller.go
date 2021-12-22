package controllers

import (
	"net/http"

	"PhoenixOracle/core/chain/evm/types"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/util"
	"PhoenixOracle/web/presenters"
	"PhoenixOracle/web"
	"github.com/gin-gonic/gin"
)

type ChainsController struct {
	App phoenix.Application
}

func (cc *ChainsController) Index(c *gin.Context, size, page, offset int) {
	chains, count, err := cc.App.EVMORM().Chains(offset, size)

	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	var resources []presenters.ChainResource
	for _, chain := range chains {
		resources = append(resources, presenters.NewChainResource(chain))
	}

	web.PaginatedResponse(c, "chain", size, page, resources, count, err)
}

type CreateChainRequest struct {
	ID     utils.Big      `json:"chainID"`
	Config types.ChainCfg `json:"config"`
}

func (cc *ChainsController) Create(c *gin.Context) {
	request := &CreateChainRequest{}

	if err := c.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	chain, err := cc.App.EVMORM().CreateChain(request.ID, request.Config)

	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewChainResource(chain), "chain")
}

func (cc *ChainsController) Delete(c *gin.Context) {
	id := utils.Big{}
	err := id.UnmarshalText([]byte(c.Param("ID")))
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	err = cc.App.EVMORM().DeleteChain(id)

	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c, nil, "chain", http.StatusNoContent)
}
