package controllers

import (
	"PhoenixOracle/web"
	"errors"
	"net/http"

	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
)

type CSAKeysController struct {
	App phoenix.Application
}

func (ctrl *CSAKeysController) Index(c *gin.Context) {
	keys, err := ctrl.App.GetKeyStore().CSA().GetAll()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewCSAKeyResources(keys), "csaKeys")
}

func (ctrl *CSAKeysController) Create(c *gin.Context) {
	key, err := ctrl.App.GetKeyStore().CSA().Create()
	if err != nil {
		if errors.Is(err, keystore.ErrCSAKeyExists) {
			web.JsonAPIError(c, http.StatusBadRequest, err)
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewCSAKeyResource(key), "csaKeys")
}

func (ctrl *CSAKeysController) Export(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	keyID := c.Param("keyID")
	newPassword := c.Query("newpassword")

	bytes, err := ctrl.App.GetKeyStore().CSA().Export(keyID, newPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, web.MediaType, bytes)
}
