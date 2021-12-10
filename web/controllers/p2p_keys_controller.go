package controllers

import (
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
)

type P2PKeysController struct {
	App phoenix.Application
}

func (p2pkc *P2PKeysController) Index(c *gin.Context) {
	keys, err := p2pkc.App.GetKeyStore().P2P().GetAll()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewP2PKeyResources(keys), "p2pKey")
}

func (p2pkc *P2PKeysController) Create(c *gin.Context) {
	key, err := p2pkc.App.GetKeyStore().P2P().Create()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewP2PKeyResource(key), "p2pKey")
}

func (p2pkc *P2PKeysController) Delete(c *gin.Context) {
	var err error
	keyID := c.Param("keyID")
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}
	key, err := p2pkc.App.GetKeyStore().P2P().Get(keyID)
	if err != nil {
		web.JsonAPIError(c, http.StatusNotFound, err)
		return
	}
	_, err = p2pkc.App.GetKeyStore().P2P().Delete(key.ID())
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewP2PKeyResource(key), "p2pKey")
}

func (p2pkc *P2PKeysController) Import(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	bytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}
	oldPassword := c.Query("oldpassword")
	key, err := p2pkc.App.GetKeyStore().P2P().Import(bytes, oldPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewP2PKeyResource(key), "p2pKey")
}

func (p2pkc *P2PKeysController) Export(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	stringID := c.Param("ID")
	newPassword := c.Query("newpassword")
	bytes, err := p2pkc.App.GetKeyStore().P2P().Export(stringID, newPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, web.MediaType, bytes)
}
