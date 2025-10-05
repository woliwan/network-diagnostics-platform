package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/ping", PingHandler)
		api.POST("/tcping", TCPingHandler)
		api.POST("/traceroute", TracerouteHandler)
		api.POST("/dns", DNSHandler)
		api.POST("/speed", SpeedHandler)
		api.POST("/bulk/ping", BulkPingHandler)
		api.POST("/bulk/html", BulkHTMLHandler)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Printf("starting server on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}