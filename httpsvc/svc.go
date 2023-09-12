package httpsvc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HttpService struct {
	BasicAuthUser     string
	BasicAuthPassword string
}

func (svc *HttpService) SetupRouter() *gin.Engine {
	router := gin.Default()
	if svc.BasicAuthUser != "" {
		router.Use(gin.BasicAuth(gin.Accounts{svc.BasicAuthUser: svc.BasicAuthPassword}))
	}
	router.GET("/api/readback", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"address": c.Request.RemoteAddr,
			"headers": c.Request.Header,
			"method":  c.Request.Method,
			"url":     c.Request.URL.String(),
		})
	})
	router.Static("/resource", "./resource")
	router.StaticFile("/", "./resource/index.html")
	return router
}
