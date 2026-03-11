package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		// json response
		c.JSON(http.StatusOK, gin.H{
			"message": "response",
		})
	})

	r.Run()
}
