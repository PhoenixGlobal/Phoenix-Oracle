package controllers

import (
	"PhoenixOracle/gophoenix/core/logger"
	"PhoenixOracle/gophoenix/core/services"
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"github.com/asdine/storm"
	"github.com/gin-gonic/gin"
)

type JobRunsController struct{
	App *services.Application
}

func (self *JobRunsController) Index(c *gin.Context) {
	id := c.Param("id")
	jobRuns := []models.JobRun{}

	err := self.App.Store.Where("JobID", id, &jobRuns)

	if err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else {
		c.JSON(200, gin.H{"runs": jobRuns})
	}
}

func (self *JobRunsController) Create(c *gin.Context) {
	id := c.Param("jobID")
	j:= models.Job{}
	if err:= self.App.Store.One("ID", id, &j); err == storm.ErrNotFound{
		c.JSON(404, gin.H{
			"errors": []string{"Job not found"},
		})
	}else if err != nil{
		c.JSON(500, gin.H{
			"errors":[]string{err.Error()},
		})
	} else if !j.WebAuthorized() {
		c.JSON(403, gin.H{
			"errors": []string{"Job not available on web API. Recreate with web initiator."},
		})
	}else {
		jr := startJob(j, self.App.Store)
		c.JSON(200, gin.H{"id": jr.ID})
	}


}

func startJob(j models.Job, s *store.Store) models.JobRun {
	jr := j.NewRun()
	go func() {
		if _, err := services.StartJob(jr, s); err != nil {
			logger.Panic(err)
		}
	}()
	return jr
}

