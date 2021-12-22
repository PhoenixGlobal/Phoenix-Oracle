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

type OCRKeysController struct {
	App phoenix.Application
}

func (ocrkc *OCRKeysController) Index(c *gin.Context) {
	ekbs, err := ocrkc.App.GetKeyStore().OCR().GetAll()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewOCRKeysBundleResources(ekbs), "offChainReportingKeyBundle")
}

func (ocrkc *OCRKeysController) Create(c *gin.Context) {
	key, err := ocrkc.App.GetKeyStore().OCR().Create()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewOCRKeysBundleResource(key), "offChainReportingKeyBundle")
}

func (ocrkc *OCRKeysController) Delete(c *gin.Context) {
	var err error
	id := c.Param("keyID")
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}
	key, err := ocrkc.App.GetKeyStore().OCR().Get(id)
	if err != nil {
		web.JsonAPIError(c, http.StatusNotFound, err)
		return
	}
	_, err = ocrkc.App.GetKeyStore().OCR().Delete(id)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	web.JsonAPIResponse(c, presenters.NewOCRKeysBundleResource(key), "offChainReportingKeyBundle")
}

func (ocrkc *OCRKeysController) Import(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	bytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}
	oldPassword := c.Query("oldpassword")
	encryptedOCRKeyBundle, err := ocrkc.App.GetKeyStore().OCR().Import(bytes, oldPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, encryptedOCRKeyBundle, "offChainReportingKeyBundle")
}

func (ocrkc *OCRKeysController) Export(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	stringID := c.Param("ID")
	newPassword := c.Query("newpassword")
	bytes, err := ocrkc.App.GetKeyStore().OCR().Export(stringID, newPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	c.Data(http.StatusOK, web.MediaType, bytes)
}
