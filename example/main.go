package main

import (
	"fmt"
	"io"

	"github.com/baagod/sgin/v2"
	"github.com/gin-gonic/gin"
)

type User struct {
	Name string `json:"name" form:"name"`
	Age  int    `json:"age" form:"age"`
	Q    string `form:"q"`
}

func main() {
	r := sgin.New(sgin.Config{
		Mode: gin.DebugMode,
	})

	r.POST("/test", sgin.Ho(func(c *sgin.Ctx, in User) any {
		// 验证 1: 检查输入参数（应包含 Body 和 Query）
		fmt.Printf("绑定结果: %+v\n", in)

		// 验证 2: 尝试再次从 Request.Body 读取数据
		// 如果使用了 ShouldBindBodyWith，Body 会被缓存在 gc 中，可以重复读取。
		body, _ := io.ReadAll(c.Request.Body)
		fmt.Printf("再次读取 Body: %s\n", string(body))

		return gin.H{
			"received_in":  in,
			"body_re-read": string(body),
		}
	}))

	_ = r.Run(":8080")
}
