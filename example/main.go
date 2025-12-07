package main

import (
    "errors"
    "fmt"
    "net/http"

    "github.com/baagod/sgin"
    "github.com/gin-gonic/gin"
)

// GetUserReq 定义请求结构体
type GetUserReq struct {
    ID    int    `uri:"id" binding:"required" doc:"用户ID"`
    Type  string `form:"type" default:"guest" doc:"用户类型"`
    Token string `header:"Authorization"` // 这里没有 doc tag，看是否能正常解析
    Name  string `json:"name" doc:"用户名称"`
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
        ErrorHandler: func(c *sgin.Ctx, err error) error {
            // 示例自定义错误处理
            var apiErr sgin.APIError
            if errors.As(err, &apiErr) {
                // 如果是 APIError，使用它提供的状态码和信息
                return c.Status(apiErr.Status()).Send(gin.H{"code": apiErr.Status(), "message": apiErr.Error()})
            }
            // 否则，返回通用的 500 错误
            return c.Status(http.StatusInternalServerError).Send(gin.H{"code": http.StatusInternalServerError, "message": "Internal Server Error"})
        },
    })

    // 注册一个 V2 智能 Handler
    r.POST("/api/v1/users/:id", func(c *sgin.Ctx, req GetUserReq) (UserResp, error) {
        fmt.Printf("收到请求: %+v\n", req) // 打印请求内容

        if req.ID <= 0 {
            // 使用 sgin 的标准化错误
            return UserResp{}, sgin.ErrBadRequest("用户ID无效")
        }

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
