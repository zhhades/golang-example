package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func main() {
	router := gin.Default()
	router.GET("/hello", hello)
	router.POST("/add", formPost)

	router.Run(":8083")
}

func formPost(c *gin.Context) {
	json := make(map[string]interface{})
	c.BindJSON(&json)
	log.Printf("%v", &json)
	c.JSON(http.StatusOK, gin.H{
		"name":     json["name"],
		"nick": json["nick"],
	})
}

func hello(c *gin.Context) {
	firstName := c.DefaultQuery("firstName", "zhhades")
	lastName := c.Query("lastName")
	c.JSON(http.StatusOK, gin.H{
		"firstName": firstName,
		"lastName":  lastName,
	})
}
