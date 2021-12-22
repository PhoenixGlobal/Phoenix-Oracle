package controllers

import (
	"io/ioutil"
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
)

type VRFKeysController struct {
	App phoenix.Application
}

func (vrfkc *VRFKeysController) Index(c *gin.Context) {
	keys, err := vrfkc.App.GetKeyStore().VRF().GetAll()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewVRFKeyResources(keys), "vrfKey")
}

func (vrfkc *VRFKeysController) Create(c *gin.Context) {
	pk, err := vrfkc.App.GetKeyStore().VRF().Create()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewVRFKeyResource(pk), "vrfKey")
}

func (vrfkc *VRFKeysController) Delete(c *gin.Context) {
	keyID := c.Param("keyID")
	key, err := vrfkc.App.GetKeyStore().VRF().Get(keyID)
	if err != nil {
		web.JsonAPIError(c, http.StatusNotFound, err)
		return
	}
	_, err = vrfkc.App.GetKeyStore().VRF().Delete(keyID)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewVRFKeyResource(key), "vrfKey")
}

func (vrfkc *VRFKeysController) Import(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	bytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}
	oldPassword := c.Query("oldpassword")
	key, err := vrfkc.App.GetKeyStore().VRF().Import(bytes, oldPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewVRFKeyResource(key), "vrfKey")
}

func (vrfkc *VRFKeysController) Export(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	keyID := c.Param("keyID")
	newPassword := c.Query("newpassword")
	bytes, err := vrfkc.App.GetKeyStore().VRF().Export(keyID, newPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, web.MediaType, bytes)
}
