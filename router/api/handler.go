package api

import (
	"fmt"

	"github.com/JamesClonk/backman/s3"
	"github.com/JamesClonk/backman/service"
	cfenv "github.com/cloudfoundry-community/go-cfenv"
	echo "github.com/labstack/echo/v4"
)

const version = "v1"

// Handler holds all objects and configurations used across API requests
type Handler struct {
	App     *cfenv.App
	S3      *s3.Client
	Service *service.Service
}

func New() *Handler {
	return &Handler{
		App:     service.Get().App,
		S3:      service.Get().S3,
		Service: service.Get(),
	}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// everything should be placed under /api/$version/
	g := e.Group(fmt.Sprintf("/api/%s", version))

	g.GET("/services", h.ListServices)
	g.GET("/backups", h.ListBackups)

	g.GET("/backup/:service_type/:service_name/:file", h.GetBackup)
	g.GET("/backup/:service_type/:service_name/:file/download", h.DownloadBackup)
	g.POST("/backup/:service_type/:service_name", h.CreateBackup)
	g.DELETE("/backup/:service_type/:service_name/:file", h.DeleteBackup)

	g.POST("/restore/:service_type/:service_name/:file", h.RestoreBackup)
}
