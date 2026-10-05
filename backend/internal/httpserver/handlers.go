package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse is returned by GET /api/health.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// InfoResponse is returned by GET /api/info.
type InfoResponse struct {
	Name    string `json:"name"    example:"Kubedactyl"`
	Version string `json:"version" example:"0.1.0"`
	Mode    string `json:"mode"    example:"production" enums:"development,production"`
}

// getHealth godoc
//
//	@Summary		Health check
//	@Description	Reports whether the panel process is running.
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

// getInfo godoc
//
//	@Summary		Panel information
//	@Description	Name, version and run mode (public, so nothing else is revealed).
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	InfoResponse
//	@Router			/info [get]
func getInfo(name, version string, devMode bool) gin.HandlerFunc {
	mode := "production"
	if devMode {
		mode = "development"
	}
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, InfoResponse{Name: name, Version: version, Mode: mode})
	}
}
