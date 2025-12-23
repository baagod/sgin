package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func BindHandler[Input any](handler func(c *gin.Context, input Input)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 创建输入结构体的实例
		var input Input

		// 2. 使用gin的绑定机制绑定数据
		// 这里简化处理，实际可以根据标签绑定URI、Query、Body等
		if err := c.ShouldBind(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 3. 执行用户传入的 handler
		handler(c, input)
	}
}

func main() {
	g := gin.Default()
	g.GET("/", BindHandler(func(c *gin.Context, input LoginRequest) {

	}))

	g.Run(":8080")
}
