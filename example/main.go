package main

import (
	"fmt"

	"github.com/baagod/sgin/v2"
	"github.com/gin-gonic/gin"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	r := sgin.New(sgin.Config{
		Mode:    gin.DebugMode,
		OpenAPI: sgin.NewAPI(),
	})

	secret := []byte("secret")
	user := &User{Name: "Baago", Age: 24}

	// 1. 创建 JWT 管理器
	auth := sgin.NewJWT[*User]("user", secret, 0)

	// 2. 签发 Token
	token, _ := auth.Issue(user)
	fmt.Println("Token:", token)

	// 3. 注册中间件
	r.Use(auth.Auth(nil))

	// 4. 使用
	r.GET("/me", sgin.Ho(func(c *sgin.Ctx, _ struct{}) (r *sgin.Result) {
		// 从上下文获取 Claims
		if claims, ok := c.Get("user").(*sgin.Claims[*User]); ok {
			fmt.Printf("User: %+v\n", claims.Data)
			return r.OK(claims.Data)
		}
		return r.OK(nil)
	}))

	_ = r.Run(":8080")
}
