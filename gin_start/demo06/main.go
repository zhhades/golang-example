package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	router := gin.Default()
	router.GET("/json", json)
	router.GET("/proto", protobuf)

	router.Run(":8083")
}

func protobuf(c *gin.Context) {
	//c.ProtoBuf(http.StatusOK,)
}

func json(c *gin.Context) {
	var msg struct {
		Name    string `json:"user"`
		Message string `json:"message"`
		Number  int `json:"number"`
	}

	msg.Name = "zhhades"
	msg.Message = "xxxxx"
	msg.Number = 1111

	c.JSON(http.StatusOK, msg)

}
