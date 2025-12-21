package main

import (
    "fmt"

    "github.com/baagod/sgin"
    "github.com/baagod/sgin/oa"
    "github.com/gin-gonic/gin"
    "github.com/go-playground/locales/zh"
    tzh "github.com/go-playground/validator/v10/translations/zh"
    // 如需支持英文错误消息，取消注释以下导入：
    // "github.com/go-playground/locales/en"
    // uten "github.com/go-playground/validator/v10/translations/en"
)

// GetUserReq 定义请求结构体
type GetUserReq struct {
    ID    int    `uri:"id" binding:"required" doc:"用户ID"`
    Type  string `form:"type" default:"guest" doc:"用户类型"`
    Token string `header:"Authorization"` // 这里没有 doc tag，看是否能正常解析
    Name  string `form:"name" doc:"用户名称"`
}

// User 示例结构体
type User struct {
    Name string `json:"name" binding:"required" label:"姓名"`
    Age  int    `json:"age" binding:"required,gt=18"`
}

func main() {
    r := sgin.New(sgin.Config{
        Mode:    gin.DebugMode,       // 调试模式
        OpenAPI: oa.New(oa.Config{}), // 开启 OpenAPI 文档服务
        Locales: []sgin.Locale{
            {zh.New(), tzh.RegisterDefaultTranslations}, // 第一个语言为默认语言
            // 如需支持英文，取消注释以下行：
            // {en.New(), uten.RegisterDefaultTranslations},
        },
    })

    r.POST("/i18n", func(c *sgin.Ctx, user User) gin.H {
        return gin.H{"msg": "success"}
    })

    fmt.Println("Sgin 服务器正在端口 :8080 运行...")
    fmt.Println("请访问 http://localhost:8080/docs 查看 API 文档。")
    fmt.Println("或访问 http://localhost:8080/openapi.yaml 查看原始 OpenAPI Spec。")

    // 启动服务器
    err := r.Run(":8080")
    fmt.Println(err)
}
