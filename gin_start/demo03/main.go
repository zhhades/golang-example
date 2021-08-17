package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	engine := gin.Default()
	goodsGroup := engine.Group("/goods")
	{
		goodsGroup.GET("", goodsList)
		goodsGroup.GET("/:id/:action", goodsDetail)
		goodsGroup.POST("/add", goodsAdd)
	}

	engine.Run(":8083")
}

func goodsAdd(c *gin.Context) {

}

func goodsList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": "list",
	})
}

func goodsDetail(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"action": action,
	})
}
