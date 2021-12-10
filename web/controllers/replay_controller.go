package controllers

import (
	"net/http"
	"strconv"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/web"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type ReplayController struct {
	App phoenix.Application
}

func (bdc *ReplayController) ReplayFromBlock(c *gin.Context) {

	if c.Param("number") == "" {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, errors.New("missing 'number' parameter"))
		return
	}

	blockNumber, err := strconv.ParseInt(c.Param("number"), 10, 0)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}
	if blockNumber < 0 {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, errors.Errorf("block number cannot be negative: %v", blockNumber))
		return
	}
	if err := bdc.App.ReplayFromBlock(uint64(blockNumber)); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	response := ReplayResponse{
		Message: "Replay started",
	}
	web.JsonAPIResponse(c, &response, "response")
}

type ReplayResponse struct {
	Message string `json:"message"`
}

func (s ReplayResponse) GetID() string {
	return "replayID"
}

func (ReplayResponse) GetName() string {
	return "replay"
}

func (*ReplayResponse) SetID(string) error {
	return nil
}
