package main

import (
    "fmt"
    "strings"
    "time"

    "github.com/baagod/sgin"
    "github.com/baagod/sgin/oa"
    "github.com/gin-gonic/gin"
)

// AuthMiddleware 模拟一个鉴权中间件
func AuthMiddleware(c *sgin.Ctx) error {
    token := c.Header("Authorization")
    if !strings.HasPrefix(token, "Bearer ") {
        return c.Send(sgin.ErrUnauthorized("Missing or invalid token"))
    }
    // 假设验证通过，将用户 ID 存入 Context
    c.Get("userID", "user123")
    return c.Next()
}

// GetUserReq 定义请求结构体
type GetUserReq struct {
    ID    int    `uri:"id" binding:"required" doc:"用户ID"`
    Type  string `form:"type" default:"guest" doc:"用户类型"`
    Token string `header:"Authorization"` // 这里没有 doc tag，看是否能正常解析
    Name  string `form:"name" doc:"用户名称"`
}

// UserResp 定义响应结构体
type UserResp struct {
    ID      int       `json:"id"`
    Info    string    `json:"info"`
    Time    time.Time `json:"time"`
    IsValid *bool     `json:"is_valid"`
}

func main() {
    // 初始化 sgin 引擎，并开启 OpenAPI 文档服务
    r := sgin.New(sgin.Config{
        Mode: gin.DebugMode, // 调试模式
        OpenAPI: func(a *oa.OpenAPI) bool {
            a.Security = append(a.Security, oa.Requirement{"bearer": {}})
            return true
        },
    })

    // 注册一个 V2 智能 Handler
    r.GET("users/:id", func(c *sgin.Ctx, q GetUserReq) (UserResp, error) {
        user := UserResp{
            ID:   q.ID,
            Info: fmt.Sprintf("类型: %s, 令牌: %s, 姓名: %s", q.Type, q.Token, q.Name),
            Time: time.Now(),
        }
        return user, nil
    })

    // // 私有路由组，需要鉴权
    // secure := r.Group("/api/v1", AuthMiddleware, func(op *sgin.OAOperation) {
    //     op.Security = []sgin.OARequirement{{"bearerAuth": {}}}
    //     op.Tags = []string{"auth"}
    // })
    //
    // secure.GET("/secure", func(op *sgin.OAOperation) {
    //     // op.Summary = "#"
    //     // op.Description = ""
    // }, func(c *sgin.Ctx) (gin.H, error) {
    //     userID := c.Get("userID").(string) // 从 Context 中获取中间件设置的用户ID
    //     token := c.Header("Authorization")
    //     return gin.H{
    //         "message": "Welcome to the secure area!",
    //         "userID":  userID,
    //         "token":   token,
    //     }, nil
    // })

    // r.GET("/test", func(c *sgin.Ctx, i UserResp) *UserResp {
    //     return &UserResp{}
    // })

    // 简单的健康检查路由
    // r.GET("/health", func(c *sgin.Ctx) string {
    //     return "Service is healthy!"
    // })

    fmt.Println("Sgin 服务器正在端口 :8080 运行...")
    fmt.Println("请访问 http://localhost:8080/docs 查看 API 文档。")
    fmt.Println("或访问 http://localhost:8080/openapi.yaml 查看原始 OpenAPI Spec。")

    // 启动服务器
    err := r.Run(":8080")
    fmt.Println(err)
}
