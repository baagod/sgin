package main

import "github.com/gin-gonic/gin"

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Result struct {
	Status int    `json:"status"`
	Code   string `json:"name"`
	Data   any    `json:"age"`
}

func main() {
	g := gin.Default()

	g.GET("/", func(c *gin.Context) {

	})

	_ = g.Run(":8080")
}
