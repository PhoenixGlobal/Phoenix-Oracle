package controllers

import (
	"net/http"
	"strconv"

	"PhoenixOracle/core/chain/evm"
	"PhoenixOracle/core/chain/evm/types"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/util"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
)

type NodesController struct {
	App phoenix.Application
}

func (nc *NodesController) Index(c *gin.Context, size, page, offset int) {
	id := c.Param("ID")

	var nodes []types.Node
	var count int
	var err error

	if id == "" {
		nodes, count, err = nc.App.EVMORM().Nodes(offset, size)

	} else {
		chainID := utils.Big{}
		if err = chainID.UnmarshalText([]byte(id)); err != nil {
			web.JsonAPIError(c, http.StatusBadRequest, err)
			return
		}
		nodes, count, err = nc.App.EVMORM().NodesForChain(chainID, offset, size)
	}

	var resources []presenters.NodeResource
	for _, node := range nodes {
		resources = append(resources, presenters.NewNodeResource(node))
	}

	web.PaginatedResponse(c, "node", size, page, resources, count, err)
}

func (nc *NodesController) Create(c *gin.Context) {
	var request evm.NewNode

	if err := c.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	node, err := nc.App.EVMORM().CreateNode(request)

	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewNodeResource(node), "node")
}

func (nc *NodesController) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("ID"), 10, 64)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	err = nc.App.EVMORM().DeleteNode(id)

	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c, nil, "node", http.StatusNoContent)
}
