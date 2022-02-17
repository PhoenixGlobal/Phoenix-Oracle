package handlers

import "github.com/gin-gonic/gin"

// GetCommodity serves the commodity data to the client.
// @Summary returns the latest available commodity's data
// @Description get commodity by name
// @ID get-commodity
// @Accept  json
// @Produce  json
// @Param name path string true "Name of the commodity"
// @Success 200 {object} models.Commodity
// @Failure 400 {json} string
// @Router /v1/commodity/{name} [get]
func (h *Handler) GetCommodity(c *gin.Context) {

	// get requested the commodity
	name := c.Param("name")
	cmd, err := h.commoditySrv.GetCommodity(name)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, cmd)
}
