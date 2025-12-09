package main

import (
    "errors"
    "fmt"
    "net/http"

    "github.com/baagod/sgin"
    "github.com/gin-gonic/gin"
)

// Address 定义嵌套结构体
type Address struct {
    City string `form:"city" binding:"required" failtip:"城市不能为空"`
}

// GetUserReq 定义请求结构体
type GetUserReq struct {
    ID    int    `uri:"id" binding:"required" doc:"用户ID"`
    Type  string `form:"type" default:"guest" doc:"用户类型"`
    Token string `header:"Authorization"` // 这里没有 doc tag，看是否能正常解析
    Name  string `form:"name" doc:"用户名称"`
    // City  string `form:"city" binding:"required" failtip:"城市不能为空"`
    Address Address
}

// UserResp 定义响应结构体
type UserResp struct {
    ID   int    `json:"id"`
    Info string `json:"info"`
}

func main() {
    // 初始化 sgin 引擎，并开启 OpenAPI 文档服务
    r := sgin.New(sgin.Config{
        Mode:    gin.DebugMode, // 调试模式
        OpenAPI: true,          // 开启 OpenAPI 文档
        ErrorHandler: func(c *sgin.Ctx, err error) error { // 示例自定义错误处理
            var apiErr *sgin.Error
            if errors.As(err, &apiErr) {
                // 如果是 APIError，使用它提供的状态码和信息
                return c.Status(apiErr.Code).Send(gin.H{"code": apiErr.Code, "message": apiErr.Error()})
            }
            // 否则，返回通用的 500 错误
            return c.Status(http.StatusInternalServerError).Send(gin.H{"code": http.StatusInternalServerError, "message": "Internal Server Error"})
        },
    })

    // 制造一个 Panic 来测试 Recovery
    r.GET("/panic", func(c *sgin.Ctx) any {
        // 模拟空指针异常
        var user *UserResp
        return user.ID // 这里会 Panic
    })

    // 注册一个 V2 智能 Handler
    r.GET("users/:id", func(c *sgin.Ctx, req GetUserReq) (UserResp, error) {
        fmt.Printf("收到请求: %+v\n", req) // 打印请求内容

        // 模拟业务逻辑
        user := UserResp{
            ID:   req.ID,
            Info: fmt.Sprintf("类型: %s, 令牌: %s, 姓名: %s", req.Type, req.Token, req.Name),
        }
        return user, nil
    })

    // 简单的健康检查路由
    r.GET("/health", func(c *sgin.Ctx) string {
        return "Service is healthy!"
    })

    fmt.Println("Sgin 服务器正在端口 :8080 运行...")
    fmt.Println("请访问 http://localhost:8080/docs 查看 API 文档。")
    fmt.Println("或访问 http://localhost:8080/openapi.json 查看原始 OpenAPI Spec。")

    // 启动服务器
    err := r.Run(":8080")
    fmt.Println(err)
}
