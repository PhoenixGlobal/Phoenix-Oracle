package controllers

import (
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type JobProposalsController struct {
	App phoenix.Application
}

func (jpc *JobProposalsController) Index(c *gin.Context) {
	feedsSvc := jpc.App.GetFeedsService()

	jps, err := feedsSvc.ListJobProposals()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewJobProposalResources(jps), "job_proposals")
}

func (jpc *JobProposalsController) Show(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		web.JsonAPIError(c, http.StatusNotFound, err)
		return
	}

	feedsSvc := jpc.App.GetFeedsService()

	jp, err := feedsSvc.GetJobProposal(id)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewJobProposalResource(*jp), "job_proposals")
}

func (jpc *JobProposalsController) Approve(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	feedsSvc := jpc.App.GetFeedsService()

	err = feedsSvc.ApproveJobProposal(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

			web.JsonAPIError(c, http.StatusNotFound, errors.New("job proposal not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	jp, err := feedsSvc.GetJobProposal(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			web.JsonAPIError(c, http.StatusNotFound, errors.New("job proposal not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c,
		presenters.NewJobProposalResource(*jp),
		"job_proposals",
		http.StatusOK,
	)
}

func (jpc *JobProposalsController) Reject(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	feedsSvc := jpc.App.GetFeedsService()

	err = feedsSvc.RejectJobProposal(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			web.JsonAPIError(c, http.StatusNotFound, errors.New("job proposal not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	jp, err := feedsSvc.GetJobProposal(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			web.JsonAPIError(c, http.StatusNotFound, errors.New("job proposal not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c,
		presenters.NewJobProposalResource(*jp),
		"job_proposals",
		http.StatusOK,
	)
}

type UpdateSpecRequest struct {
	Spec string `json:"spec"`
}

func (jpc *JobProposalsController) UpdateSpec(c *gin.Context) {
	request := UpdateSpecRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	feedsSvc := jpc.App.GetFeedsService()

	err = feedsSvc.UpdateJobProposalSpec(c.Request.Context(), id, request.Spec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			web.JsonAPIError(c, http.StatusNotFound, errors.New("job proposal not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	jp, err := feedsSvc.GetJobProposal(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			web.JsonAPIError(c, http.StatusNotFound, errors.New("job proposal not found"))
			return
		}

		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c,
		presenters.NewJobProposalResource(*jp),
		"job_proposals",
		http.StatusOK,
	)
}
