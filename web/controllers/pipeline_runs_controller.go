package controllers

import (
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	"PhoenixOracle/web/presenters"
	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/jobs/webhook"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/web"
	"github.com/gin-gonic/gin"
)

type PipelineRunsController struct {
	App phoenix.Application
}

func (prc *PipelineRunsController) Index(c *gin.Context, size, page, offset int) {
	id := c.Param("ID")

	// Temporary: if no size is passed in, use a large page size. Remove once frontend can handle pagination
	if c.Query("size") == "" {
		size = 1000
	}

	var pipelineRuns []pipeline.Run
	var count int
	var err error

	if id == "" {
		pipelineRuns, count, err = prc.App.JobORM().PipelineRuns(offset, size)
	} else {
		jobSpec := job.Job{}
		err = jobSpec.SetID(c.Param("ID"))
		if err != nil {
			web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
			return
		}

		pipelineRuns, count, err = prc.App.JobORM().PipelineRunsByJobID(jobSpec.ID, offset, size)
	}

	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.PaginatedResponse(c, "pipelineRun", size, page, presenters.NewPipelineRunResources(pipelineRuns), count, err)
}

func (prc *PipelineRunsController) Show(c *gin.Context) {
	pipelineRun := pipeline.Run{}
	err := pipelineRun.SetID(c.Param("runID"))
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	pipelineRun, err = prc.App.PipelineORM().FindRun(pipelineRun.ID)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewPipelineRunResource(pipelineRun), "pipelineRun")
}

func (prc *PipelineRunsController) Create(c *gin.Context) {
	respondWithPipelineRun := func(jobRunID int64) {
		pipelineRun, err := prc.App.PipelineORM().FindRun(jobRunID)
		if err != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, err)
			return
		}
		web.JsonAPIResponse(c, presenters.NewPipelineRunResource(pipelineRun), "pipelineRun")
	}

	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}
	idStr := c.Param("ID")

	user, isUser := web.AuthenticatedUser(c)
	ei, _ := web.AuthenticatedEI(c)
	authorizer := webhook.NewAuthorizer(prc.App.GetStore().DB, user, ei)

	jobUUID, err := uuid.FromString(idStr)
	if err == nil {
		canRun, err2 := authorizer.CanRun(c.Request.Context(), prc.App.GetConfig(), jobUUID)
		if err2 != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, err2)
			return
		}
		if canRun {
			jobRunID, err3 := prc.App.RunWebhookJobV2(c.Request.Context(), jobUUID, string(bodyBytes), pipeline.JSONSerializable{Null: true})
			if errors.Is(err3, webhook.ErrJobNotExists) {
				web.JsonAPIError(c, http.StatusNotFound, err3)
				return
			} else if err3 != nil {
				web.JsonAPIError(c, http.StatusInternalServerError, err3)
				return
			}
			respondWithPipelineRun(jobRunID)
		} else {
			web.JsonAPIError(c, http.StatusUnauthorized, errors.Errorf("external initiator %s is not allowed to run job %s", ei.Name, jobUUID))
		}
		return
	}

	if isUser {
		var jobID int32
		jobID64, err := strconv.ParseInt(idStr, 10, 32)
		if err == nil {
			jobID = int32(jobID64)
			jobRunID, err := prc.App.RunJobV2(c.Request.Context(), jobID, nil)
			if err != nil {
				web.JsonAPIError(c, http.StatusInternalServerError, err)
				return
			}
			respondWithPipelineRun(jobRunID)
			return
		}
	}

	web.JsonAPIError(c, http.StatusUnprocessableEntity, errors.New("bad job ID"))
}

func (prc *PipelineRunsController) Resume(c *gin.Context) {
	taskID, err := uuid.FromString(c.Param("runID"))
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	bodyBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	if err := prc.App.ResumeJobV2(context.Background(), taskID, bodyBytes); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}
