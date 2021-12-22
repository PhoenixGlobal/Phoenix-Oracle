package web

import (
	"fmt"
	"net/http"

	"PhoenixOracle/db/models"
	"PhoenixOracle/db/orm"
	"github.com/gin-gonic/gin"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/pkg/errors"
)

func StatusCodeForError(err interface{}) int {
	switch err.(type) {
	case *models.ValidationError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func JsonAPIError(c *gin.Context, statusCode int, err error) {
	_ = c.Error(err).SetType(gin.ErrorTypePublic)
	switch v := err.(type) {
	case *models.JSONAPIErrors:
		c.JSON(statusCode, v)
	default:
		c.JSON(statusCode, models.NewJSONAPIErrorsWith(err.Error()))
	}
}

func paginatedResponseWithMeta(
	c *gin.Context,
	name string,
	size int,
	page int,
	resource interface{},
	count int,
	err error,
	meta map[string]interface{},
) {
	if errors.Cause(err) == orm.ErrorNotFound {
		err = nil
	}

	if err != nil {
		JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("error getting paged %s: %+v", name, err))
	} else if buffer, err := NewPaginatedResponseWithMeta(*c.Request.URL, size, page, count, resource, meta); err != nil {
		JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("failed to marshal document: %+v", err))
	} else {
		c.Data(http.StatusOK, MediaType, buffer)
	}
}

func PaginatedResponse(
	c *gin.Context,
	name string,
	size int,
	page int,
	resource interface{},
	count int,
	err error,
) {
	if errors.Cause(err) == orm.ErrorNotFound {
		err = nil
	}

	if err != nil {
		JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("error getting paged %s: %+v", name, err))
	} else if buffer, err := NewPaginatedResponse(*c.Request.URL, size, page, count, resource); err != nil {
		JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("failed to marshal document: %+v", err))
	} else {
		c.Data(http.StatusOK, MediaType, buffer)
	}
}

func PaginatedRequest(action func(*gin.Context, int, int, int)) func(*gin.Context) {
	return func(c *gin.Context) {
		size, page, offset, err := ParsePaginatedRequest(c.Query("size"), c.Query("page"))
		if err != nil {
			JsonAPIError(c, http.StatusUnprocessableEntity, err)
			return
		}
		action(c, size, page, offset)
	}
}

func JsonAPIResponseWithStatus(c *gin.Context, resource interface{}, name string, status int) {
	json, err := jsonapi.Marshal(resource)
	if err != nil {
		JsonAPIError(c, http.StatusInternalServerError, fmt.Errorf("failed to marshal %s using jsonapi: %+v", name, err))
	} else {
		c.Data(status, MediaType, json)
	}
}

func JsonAPIResponse(c *gin.Context, resource interface{}, name string) {
	JsonAPIResponseWithStatus(c, resource, name, http.StatusOK)
}
