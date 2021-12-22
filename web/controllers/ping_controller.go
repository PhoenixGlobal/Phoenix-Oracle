package controllers

import (
	"net/http"

	"PhoenixOracle/core/service/phoenix"

	"github.com/gin-gonic/gin"
)

type PingController struct {
	App phoenix.Application
}

func (eic *PingController) Show(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
