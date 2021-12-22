package controllers

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo"

	"stock_adapter/models"
	"stock_adapter/stock"
)

// GetNews ...
func GetNews(c echo.Context) (err error) {
	// Read the Body content
	var bodyBytes []byte
	if c.Request().Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request().Body)
	}
	// Restore the io.ReadCloser to its original state
	c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	// Continue to use the Body, like Binding it to a struct:
	u := new(models.Company)
	// bind the model with the context body
	er := c.Bind(u)
	// panic!
	if er != nil {
		panic(err)
	}
	// crawl with the passed in data
	r := stock.News(u.Ticker)
	// return the links
	return c.JSON(http.StatusCreated, r)
}
