package service

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ServiceStats struct {
	Count    int `json:"count"`
	Pinned   int `json:"pinned"`
	Unpinned int `json:"unpinned"`
	Errors   int `json:"errors"`
}

type ServerInfo struct {
	Current ServiceStats `json:"current"`
	Last    ServiceStats `json:"last"`
}

func HttpServe(service *Service, port int) {
	r := gin.Default()

	r.GET("/stats", func(c *gin.Context) {
		c.JSON(200, ServerInfo{service.stats, service.laststats})
	})
	r.Run(fmt.Sprintf(":%v", port))
}
