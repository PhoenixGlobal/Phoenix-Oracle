package controllers

import (
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/web"
	"PhoenixOracle/web/presenters"
	"github.com/gin-gonic/gin"
)

type FeaturesController struct {
	App phoenix.Application
}

const (
	FeatureKeyCSA          string = "csa"
	FeatureKeyFeedsManager string = "feeds_manager"
)

func (fc *FeaturesController) Index(c *gin.Context) {
	resources := []presenters.FeatureResource{
		*presenters.NewFeatureResource(FeatureKeyCSA, fc.App.GetConfig().FeatureUICSAKeys()),
		*presenters.NewFeatureResource(FeatureKeyFeedsManager, fc.App.GetConfig().FeatureUIFeedsManager()),
	}

	web.JsonAPIResponse(c, resources, "features")
}
