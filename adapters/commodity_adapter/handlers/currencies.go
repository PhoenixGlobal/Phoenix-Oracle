package handlers

import "github.com/gin-gonic/gin"

// GetCurrency provides the data about the given currency.
// @Summary returns the latest available currency's data
// @Description get currency by name or symbol
// @ID get-currency
// @Accept  json
// @Produce  json
// @Param name path string true "Name of the currency"
// @Success 200 {object} models.Currency
// @Failure 400 {json} string
// @Router /v1/currency/{name} [get]
func (h *Handler) GetCurrency(c *gin.Context) {

	name := c.Param("name")

	crn, err := h.currencySrv.GetCurrency(name)
	if err != nil {
		c.JSON(400, gin.H{"error": gin.H{"error": err.Error()}})
	}

	c.JSON(200, crn)
}

// GetRate handles the request and calculates the exchange rate
// between two currencies.
// @Summary calculates the exchange rate of two currencies
// @Description get exchange rate
// @ID get-rate
// @Accept  json
// @Produce  json
// @Param base path string true "Name of the base currency"
// @Param dest path string true "Name of the destination currency"
// @Success 200 {object} models.ExchangeRate
// @Failure 400 {json} string
// @Router /v1/currency/{base}/{dest} [post]
func (h *Handler) GetRate(c *gin.Context) {

	base := c.Param("base")
	dest := c.Param("dest")

	// calculate the rate
	rate, err := h.currencySrv.GetRate(base, dest)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
	}

	c.JSON(200, rate)
}
