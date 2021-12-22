package controllers

import (
	services "PhoenixOracle/lib/validators"
	"PhoenixOracle/web"
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/models"
	"PhoenixOracle/db/orm"
	"PhoenixOracle/lib/auth"
	"PhoenixOracle/web/presenters"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type ExternalInitiatorsController struct {
	App phoenix.Application
}

func (eic *ExternalInitiatorsController) Index(c *gin.Context, size, page, offset int) {
	eis, count, err := eic.App.GetStore().ExternalInitiatorsSorted(offset, size)
	var resources []presenters.ExternalInitiatorResource
	for _, ei := range eis {
		resources = append(resources, presenters.NewExternalInitiatorResource(ei))
	}

	web.PaginatedResponse(c, "externalInitiators", size, page, resources, count, err)
}

func (eic *ExternalInitiatorsController) Create(c *gin.Context) {
	eir := &models.ExternalInitiatorRequest{}
	if !eic.App.GetStore().Config.Dev() && !eic.App.GetStore().Config.FeatureExternalInitiators() {
		err := errors.New("The External Initiator feature is disabled by configuration")
		web.JsonAPIError(c, http.StatusMethodNotAllowed, err)
		return
	}

	eia := auth.NewToken()
	if err := c.ShouldBindJSON(eir); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	ei, err := models.NewExternalInitiator(eia, eir)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	if err := services.ValidateExternalInitiator(eir, eic.App.GetStore()); err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}
	if err := eic.App.GetStore().CreateExternalInitiator(ei); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	resp := presenters.NewExternalInitiatorAuthentication(*ei, *eia)
	web.JsonAPIResponseWithStatus(c, resp, "external initiator authentication", http.StatusCreated)
}

func (eic *ExternalInitiatorsController) Destroy(c *gin.Context) {
	name := c.Param("Name")
	exi, err := eic.App.GetStore().FindExternalInitiatorByName(name)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("external initiator not found"))
		return
	}
	if err := eic.App.GetStore().DeleteExternalInitiator(exi.Name); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c, nil, "external initiator", http.StatusNoContent)
}
