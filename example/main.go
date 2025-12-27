package main

import "github.com/gin-gonic/gin"

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	g := gin.Default()

	g.GET("/", func(c *gin.Context) {

	})

	_ = g.Run(":8080")
}
