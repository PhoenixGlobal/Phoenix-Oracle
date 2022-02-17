package handlers

import "github.com/gin-gonic/gin"

// GetCrypto handles the GetCrypto request and response with the
// latest available data about the given cryptocurrency.
// @Summary returns the latest available cryptocurrency's data
// @Description get cryptocurrency by name or symbol
// @ID get-cryptocurrency
// @Accept  json
// @Produce  json
// @Param name path string true "Name of the commodity"
// @Success 200 {object} models.Crypto
// @Failure 400 {json} string
// @Router /v1/crypto/{name} [get]
func (h *Handler) GetCrypto(c *gin.Context) {

	name := c.Param("name")
	crp, err := h.cryptoSrv.GetCrypto(name)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
	}

	c.JSON(200, crp)
}
