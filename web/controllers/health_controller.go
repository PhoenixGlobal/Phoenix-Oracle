package controllers

import (
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/lib/health"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
)

type HealthController struct {
	App phoenix.Application
}

func (hc *HealthController) Readyz(c *gin.Context) {
	status := http.StatusOK

	checker := hc.App.GetHealthChecker()

	ready, errors := checker.IsReady()

	if !ready {
		status = http.StatusServiceUnavailable
	}

	c.Status(status)

	if _, ok := c.GetQuery("full"); !ok {
		return
	}

	checks := make([]presenters.Check, 0, len(errors))

	for name, err := range errors {
		status := health.StatusPassing
		var output string

		if err != nil {
			status = health.StatusFailing
			output = err.Error()
		}

		checks = append(checks, presenters.Check{
			JAID:   presenters.NewJAID(name),
			Name:   name,
			Status: status,
			Output: output,
		})
	}

	web.JsonAPIResponse(c, checks, "checks")
}

func (hc *HealthController) Health(c *gin.Context) {
	status := http.StatusOK

	checker := hc.App.GetHealthChecker()

	healthy, errors := checker.IsHealthy()

	if !healthy {
		status = http.StatusServiceUnavailable
	}

	c.Status(status)

	checks := make([]presenters.Check, 0, len(errors))

	for name, err := range errors {
		status := health.StatusPassing
		var output string

		if err != nil {
			status = health.StatusFailing
			output = err.Error()
		}

		checks = append(checks, presenters.Check{
			JAID:   presenters.NewJAID(name),
			Name:   name,
			Status: status,
			Output: output,
		})
	}

	web.JsonAPIResponse(c, checks, "checks")
}
