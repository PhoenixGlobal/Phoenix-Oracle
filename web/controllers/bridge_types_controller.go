package controllers

import (
	services "PhoenixOracle/lib/validators"
	"PhoenixOracle/web"
	"fmt"
	"net/http"

	"github.com/jackc/pgconn"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/models"
	"PhoenixOracle/db/orm"
	"PhoenixOracle/web/presenters"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type BridgeTypesController struct {
	App phoenix.Application
}

func (btc *BridgeTypesController) Create(c *gin.Context) {
	btr := &models.BridgeTypeRequest{}

	if err := c.ShouldBindJSON(btr); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}
	bta, bt, err := models.NewBridgeType(btr)
	if err != nil {
		web.JsonAPIError(c, web.StatusCodeForError(err), err)
		return
	}
	if e := services.ValidateBridgeType(btr, btc.App.GetStore()); e != nil {
		web.JsonAPIError(c, http.StatusBadRequest, e)
		return
	}
	if e := services.ValidateBridgeTypeNotExist(btr, btc.App.GetStore()); e != nil {
		web.JsonAPIError(c, http.StatusBadRequest, e)
		return
	}
	if e := btc.App.GetStore().CreateBridgeType(bt); e != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, e)
		return
	}
	switch e := err.(type) {
	case *pgconn.PgError:
		var apiErr error
		if e.ConstraintName == "external_initiators_name_key" {
			apiErr = fmt.Errorf("bridge Type %v conflict", bt.Name)
		} else {
			apiErr = err
		}
		web.JsonAPIError(c, http.StatusConflict, apiErr)
		return
	default:
		resource := presenters.NewBridgeResource(*bt)
		resource.IncomingToken = bta.IncomingToken

		web.JsonAPIResponse(c, resource, "bridge")
	}
}

func (btc *BridgeTypesController) Index(c *gin.Context, size, page, offset int) {
	bridges, count, err := btc.App.GetStore().BridgeTypes(offset, size)

	var resources []presenters.BridgeResource
	for _, bridge := range bridges {
		resources = append(resources, *presenters.NewBridgeResource(bridge))
	}

	web.PaginatedResponse(c, "Bridges", size, page, resources, count, err)
}

func (btc *BridgeTypesController) Show(c *gin.Context) {
	name := c.Param("BridgeName")

	taskType, err := models.NewTaskType(name)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	bt, err := btc.App.GetStore().FindBridge(taskType)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("bridge not found"))
		return
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewBridgeResource(bt), "bridge")
}

func (btc *BridgeTypesController) Update(c *gin.Context) {
	name := c.Param("BridgeName")
	btr := &models.BridgeTypeRequest{}

	taskType, err := models.NewTaskType(name)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	bt, err := btc.App.GetStore().FindBridge(taskType)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("bridge not found"))
		return
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	if err := c.ShouldBindJSON(btr); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}
	if err := services.ValidateBridgeType(btr, btc.App.GetStore()); err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}
	if err := btc.App.GetStore().UpdateBridgeType(&bt, btr); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewBridgeResource(bt), "bridge")
}

func (btc *BridgeTypesController) Destroy(c *gin.Context) {
	name := c.Param("BridgeName")

	taskType, err := models.NewTaskType(name)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	bt, err := btc.App.GetStore().FindBridge(taskType)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("bridge not found"))
		return
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("error searching for bridge: %+v", err))
		return
	}
	jobsUsingBridge, err := btc.App.JobORM().FindJobIDsWithBridge(name)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("error searching for associated v2 jobs: %+v", err))
		return
	}
	if len(jobsUsingBridge) > 0 {
		web.JsonAPIError(c, http.StatusConflict, fmt.Errorf("can't remove the bridge because jobs %v are associated with it", jobsUsingBridge))
		return
	}
	if err = btc.App.GetStore().DeleteBridgeType(&bt); err != nil {
		web.JsonAPIError(c, web.StatusCodeForError(err), fmt.Errorf("failed to delete bridge: %+v", err))
		return
	}

	web.JsonAPIResponse(c, presenters.NewBridgeResource(bt), "bridge")
}
