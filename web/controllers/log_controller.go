package controllers

import (
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"
	"net/http"
	"strconv"
)

type LogController struct {
	App phoenix.Application
}

type LogPatchRequest struct {
	Level           string      `json:"level"`
	SqlEnabled      *bool       `json:"sqlEnabled"`
	ServiceLogLevel [][2]string `json:"serviceLogLevel"`
}

func (cc *LogController) Get(c *gin.Context) {
	var svcs, lvls []string
	svcs = append(svcs, "Global")
	lvls = append(lvls, cc.App.GetStore().Config.LogLevel().String())

	svcs = append(svcs, "IsSqlEnabled")
	lvls = append(lvls, strconv.FormatBool(cc.App.GetStore().Config.LogSQLStatements()))

	logSvcs := logger.GetLogServices()
	for _, svcName := range logSvcs {
		lvl, err := cc.App.GetLogger().ServiceLogLevel(svcName)
		if err != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("error getting service log level for %s service: %v", svcName, err))
			return
		}

		svcs = append(svcs, svcName)
		lvls = append(lvls, lvl)
	}

	response := &presenters.ServiceLogConfigResource{
		JAID: presenters.JAID{
			ID: "log",
		},
		ServiceName: svcs,
		LogLevel:    lvls,
	}

	web.JsonAPIResponse(c, response, "log")
}

func (cc *LogController) Patch(c *gin.Context) {
	request := &LogPatchRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	var svcs, lvls []string

	if request.Level == "" && request.SqlEnabled == nil && len(request.ServiceLogLevel) == 0 {
		web.JsonAPIError(c, http.StatusBadRequest, fmt.Errorf("please check request params, no params configured"))
		return
	}

	if request.Level != "" {
		var ll zapcore.Level
		err := ll.UnmarshalText([]byte(request.Level))
		if err != nil {
			web.JsonAPIError(c, http.StatusBadRequest, err)
			return
		}
		if err = cc.App.GetConfig().SetLogLevel(c.Request.Context(), ll.String()); err != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, err)
			return
		}
	}
	svcs = append(svcs, "Global")
	lvls = append(lvls, cc.App.GetStore().Config.LogLevel().String())

	if request.SqlEnabled != nil {
		if err := cc.App.GetConfig().SetLogSQLStatements(c.Request.Context(), *request.SqlEnabled); err != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, err)
			return
		}
		cc.App.GetStore().SetLogging(*request.SqlEnabled)
	}
	svcs = append(svcs, "IsSqlEnabled")
	lvls = append(lvls, strconv.FormatBool(cc.App.GetStore().Config.LogSQLStatements()))

	if len(request.ServiceLogLevel) > 0 {
		for _, svcLogLvl := range request.ServiceLogLevel {
			svcName := svcLogLvl[0]
			svcLvl := svcLogLvl[1]
			var level zapcore.Level
			if err := level.UnmarshalText([]byte(svcLvl)); err != nil {
				web.JsonAPIError(c, http.StatusInternalServerError, err)
				return
			}

			if err := cc.App.SetServiceLogger(c.Request.Context(), svcName, level); err != nil {
				web.JsonAPIError(c, http.StatusInternalServerError, err)
				return
			}

			ll, err := cc.App.GetLogger().ServiceLogLevel(svcName)
			if err != nil {
				web.JsonAPIError(c, http.StatusInternalServerError, err)
				return
			}

			svcs = append(svcs, svcName)
			lvls = append(lvls, ll)
		}
	}

	logger.SetLogger(cc.App.GetStore().Config.CreateProductionLogger())
	cc.App.GetLogger().SetDB(cc.App.GetStore().DB)

	response := &presenters.ServiceLogConfigResource{
		JAID: presenters.JAID{
			ID: "log",
		},
		ServiceName: svcs,
		LogLevel:    lvls,
	}

	web.JsonAPIResponse(c, response, "log")
}
