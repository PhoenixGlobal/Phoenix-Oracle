package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"PhoenixOracle/core/service/feedmanager"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/util/crypto"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
	"gopkg.in/guregu/null.v4"
)

type FeedsManagerController struct {
	App phoenix.Application
}

type CreateFeedsManagerRequest struct {
	Name                   string           `json:"name"`
	URI                    string           `json:"uri"`
	JobTypes               []string         `json:"jobTypes"`
	PublicKey              crypto.PublicKey `json:"publicKey"`
	IsBootstrapPeer        bool             `json:"isBootstrapPeer"`
	BootstrapPeerMultiaddr null.String      `json:"bootstrapPeerMultiaddr"`
}

func (fmc *FeedsManagerController) Create(c *gin.Context) {
	request := CreateFeedsManagerRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	ms := &feedmanager.FeedsManager{
		URI:                       request.URI,
		Name:                      request.Name,
		PublicKey:                 request.PublicKey,
		JobTypes:                  request.JobTypes,
		IsOCRBootstrapPeer:        request.IsBootstrapPeer,
		OCRBootstrapPeerMultiaddr: request.BootstrapPeerMultiaddr,
	}

	feedsService := fmc.App.GetFeedsService()

	id, err := feedsService.RegisterManager(ms)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	ms, err = feedsService.GetManager(id)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c,
		presenters.NewFeedsManagerResource(*ms),
		"feeds_managers",
		http.StatusCreated,
	)
}

func (fmc *FeedsManagerController) List(c *gin.Context) {
	mss, err := fmc.App.GetFeedsService().ListManagers()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewFeedsManagerResources(mss), "feeds_managers")
}

func (fmc *FeedsManagerController) Show(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	ms, err := fmc.App.GetFeedsService().GetManager(int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			web.JsonAPIError(c, http.StatusNotFound, errors.New("feeds Manager not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewFeedsManagerResource(*ms), "feeds_managers")
}

type UpdateFeedsManagerRequest struct {
	Name                   string           `json:"name"`
	URI                    string           `json:"uri"`
	JobTypes               []string         `json:"jobTypes"`
	PublicKey              crypto.PublicKey `json:"publicKey"`
	IsBootstrapPeer        bool             `json:"isBootstrapPeer"`
	BootstrapPeerMultiaddr null.String      `json:"bootstrapPeerMultiaddr"`
}

func (fmc *FeedsManagerController) Update(c *gin.Context) {
	request := UpdateFeedsManagerRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	mgr := &feedmanager.FeedsManager{
		ID:                        id,
		URI:                       request.URI,
		Name:                      request.Name,
		PublicKey:                 request.PublicKey,
		JobTypes:                  request.JobTypes,
		IsOCRBootstrapPeer:        request.IsBootstrapPeer,
		OCRBootstrapPeerMultiaddr: request.BootstrapPeerMultiaddr,
	}

	feedsService := fmc.App.GetFeedsService()

	err = feedsService.UpdateFeedsManager(c.Request.Context(), *mgr)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	mgr, err = feedsService.GetManager(id)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c,
		presenters.NewFeedsManagerResource(*mgr),
		"feeds_managers",
		http.StatusOK,
	)
}
