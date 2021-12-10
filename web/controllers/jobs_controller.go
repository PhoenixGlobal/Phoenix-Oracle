package controllers

import (
	"PhoenixOracle/web"
	"net/http"

	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/jobs/fluxmonitor"
	"PhoenixOracle/core/service/jobs/offchainreporting"
	requestPackage "PhoenixOracle/core/service/jobs/request"
	"PhoenixOracle/core/service/jobs/timer"
	"PhoenixOracle/core/service/jobs/webhook"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/core/service/vrf"
	"PhoenixOracle/db/orm"
	"PhoenixOracle/web/presenters"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type JobsController struct {
	App phoenix.Application
}

func (jc *JobsController) Index(c *gin.Context, size, page, offset int) {
	// Temporary: if no size is passed in, use a large page size. Remove once frontend can handle pagination
	if c.Query("size") == "" {
		size = 1000
	}

	jobs, count, err := jc.App.JobORM().JobsV2(offset, size)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	var resources []presenters.JobResource
	for _, job := range jobs {
		resources = append(resources, *presenters.NewJobResource(job))
	}

	web.PaginatedResponse(c, "jobs", size, page, resources, count, err)
}

func (jc *JobsController) Show(c *gin.Context) {
	jobSpec := job.Job{}
	err := jobSpec.SetID(c.Param("ID"))
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	jobSpec, err = jc.App.JobORM().FindJobTx(jobSpec.ID)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("job not found"))
		return
	}

	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewJobResource(jobSpec), "jobs")
}

type CreateJobRequest struct {
	TOML string `json:"toml"`
}

func (jc *JobsController) Create(c *gin.Context) {
	request := CreateJobRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	jobType, err := job.ValidateSpec(request.TOML)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, errors.Wrap(err, "failed to parse TOML"))
	}

	var jb job.Job
	config := jc.App.GetStore().Config
	switch jobType {
	case job.OffchainReporting:
		jb, err = offchainreporting.ValidatedOracleSpecToml(jc.App.GetEVMConfig(), request.TOML)
		if !config.Dev() && !config.FeatureOffchainReporting() {
			web.JsonAPIError(c, http.StatusNotImplemented, errors.New("The Offchain Reporting feature is disabled by configuration"))
			return
		}
	case job.DirectRequest:
		jb, err = requestPackage.ValidatedDirectRequestSpec(request.TOML)
	case job.FluxMonitor:
		jb, err = fluxmonitor.ValidatedFluxMonitorSpec(jc.App.GetStore().Config, request.TOML)
	case job.Cron:
		jb, err = timer.ValidatedCronSpec(request.TOML)
	case job.VRF:
		jb, err = vrf.ValidatedVRFSpec(request.TOML)
	case job.Webhook:
		jb, err = webhook.ValidatedWebhookSpec(request.TOML, jc.App.GetExternalInitiatorManager())
	default:
		web.JsonAPIError(c, http.StatusUnprocessableEntity, errors.Errorf("unknown job type: %s", jobType))
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}

	jb, err = jc.App.AddJobV2(c.Request.Context(), jb, jb.Name)
	if err != nil {
		if errors.Cause(err) == job.ErrNoSuchKeyBundle || errors.Cause(err) == job.ErrNoSuchPeerID || errors.Cause(err) == job.ErrNoSuchTransmitterAddress {
			web.JsonAPIError(c, http.StatusBadRequest, err)
			return
		}
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, presenters.NewJobResource(jb), jb.Type.String())
}

func (jc *JobsController) Delete(c *gin.Context) {
	jobSpec := job.Job{}
	err := jobSpec.SetID(c.Param("ID"))
	if err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	err = jc.App.DeleteJob(c.Request.Context(), jobSpec.ID)
	if errors.Cause(err) == orm.ErrorNotFound {
		web.JsonAPIError(c, http.StatusNotFound, errors.New("JobSpec not found"))
		return
	}
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c, nil, "job", http.StatusNoContent)
}
