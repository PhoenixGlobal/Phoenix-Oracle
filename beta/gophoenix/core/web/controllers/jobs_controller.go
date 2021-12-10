package controllers

import (
	"PhoenixOracle/gophoenix/core/adapters"
	"PhoenixOracle/gophoenix/core/services"
	"PhoenixOracle/gophoenix/core/store/models"
	"fmt"
	"github.com/asdine/storm"
	"github.com/gin-gonic/gin"
)

type JobsController struct{
	App *services.Application
}

func (self *JobsController) Index(c *gin.Context) {
	var jobs []models.Job
	if err := self.App.Store.AllByIndex("CreatedAt", &jobs); err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else {
		c.JSON(200, jobs)
	}
}

func (tc *JobsController) Create(c *gin.Context) {
	fmt.Println("22222222222222222222222")
	j := models.NewJob()
	if err := c.ShouldBindJSON(&j); err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else if err = adapters.Validate(j); err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else if err = tc.App.AddJob(j); err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else {
		c.JSON(200, gin.H{"id": j.ID})
	}
}

type JobPresenter struct {
	models.Job
	Runs []models.JobRun `json:"runs,omitempty"`
}


func (tc *JobsController) Show(c *gin.Context) {
	id := c.Param("id")
	var j models.Job
	err := tc.App.Store.One("ID", id, &j)

	if err == storm.ErrNotFound {
		c.JSON(404, gin.H{
			"errors": []string{"Job not found."},
		})
	} else if err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else if runs, err := tc.App.Store.JobRunsFor(j); err != nil {
		c.JSON(500, gin.H{
			"errors": []string{err.Error()},
		})
	} else {
		c.JSON(200, JobPresenter{j, runs})
	}
}


