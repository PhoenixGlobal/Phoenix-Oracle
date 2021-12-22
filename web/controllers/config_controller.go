package controllers

import (
	"PhoenixOracle/web"
	"fmt"
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/presenters"
	"PhoenixOracle/util"

	"github.com/gin-gonic/gin"
)

type ConfigController struct {
	App phoenix.Application
}

func (cc *ConfigController) Show(c *gin.Context) {
	cw, err := presenters.NewConfigPrinter(cc.App.GetConfig())
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("failed to build config whitelist: %+v", err))
		return
	}

	web.JsonAPIResponse(c, cw, "config")
}

type configPatchRequest struct {
	EvmGasPriceDefault *utils.Big `json:"ethGasPriceDefault"`
}

type ConfigPatchResponse struct {
	EvmGasPriceDefault Change `json:"ethGasPriceDefault"`
}

type Change struct {
	From string `json:"old"`
	To   string `json:"new"`
}

func (c ConfigPatchResponse) GetID() string {
	return "configuration"
}

func (*ConfigPatchResponse) SetID(string) error {
	return nil
}

func (cc *ConfigController) Patch(c *gin.Context) {
	request := &configPatchRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	if err := cc.App.GetEVMConfig().SetEvmGasPriceDefault(request.EvmGasPriceDefault.ToInt()); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("failed to set gas price default: %+v", err))
		return
	}

	response := &ConfigPatchResponse{
		EvmGasPriceDefault: Change{
			From: cc.App.GetEVMConfig().EvmGasPriceDefault().String(),
			To:   request.EvmGasPriceDefault.String(),
		},
	}
	web.JsonAPIResponse(c, response, "config")
}
