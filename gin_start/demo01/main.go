package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func main() {
	r := gin.Default()
	r.GET("/ping", pingHandler)
	r.Run(":8083")
}
