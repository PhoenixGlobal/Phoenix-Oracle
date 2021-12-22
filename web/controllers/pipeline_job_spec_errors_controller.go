package controllers

import (
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/orm"
	"PhoenixOracle/web"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
)

type PipelineJobSpecErrorsController struct {
	App phoenix.Application
}

func (psec *PipelineJobSpecErrorsController) Destroy(c *gin.Context) {
	jobSpec := job.Job{}
	err := jobSpec.SetID(c.Param("ID"))
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	err = psec.App.JobORM().DismissError(context.Background(), jobSpec.ID)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("PipelineJobSpecError not found"))
		return
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c, nil, "job", http.StatusNoContent)
}
