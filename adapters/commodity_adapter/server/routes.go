package server

import (
	handlers "github.com/chutommy/market-info/handlers"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetRoutes applies all the routings.
func SetRoutes(r *gin.Engine, h *handlers.Handler) {

	// ping to test server status
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	v1 := r.Group("/v1")
	{

		// comodities
		cmd := v1.Group("/commodity")
		{
			cmd.GET("/:name", h.GetCommodity)
		}

		// currencies
		crn := v1.Group("/currency")
		{
			crn.GET("/i/:name", h.GetCurrency)
			crn.GET("/c/:base/:dest", h.GetRate)
		}

		// cryptocurrencies
		crp := v1.Group("/crypto")
		{
			crp.GET("/:name", h.GetCrypto)
		}
	}

	// swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
