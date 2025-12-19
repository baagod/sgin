package main

import (
    "fmt"
    "os"
    "time"

    "github.com/baagod/sgin"
    "github.com/baagod/sgin/oa"
    "github.com/gin-gonic/gin"
)

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

func level3() {
    panic(fmt.Errorf("这是一次有意的测试 Panic"))
}

func level2() {
    level3()
}

func level1() {
    level2()
}

func main() {
    r := sgin.New(sgin.Config{
        Mode:    gin.DebugMode,       // 调试模式
        OpenAPI: oa.New(oa.Config{}), // 开启 OpenAPI 文档服务
        Recovery: func(c *sgin.Ctx, out, plain string) {
            fmt.Print(out)
            f, _ := os.OpenFile("log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
            defer f.Close()
            _, _ = f.WriteString(plain)
        },
    })

    // 注册一个触发 Panic 的测试路由
    r.GET("panic", func(c *sgin.Ctx) error {
        level1()
        return nil
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

    fmt.Println("Sgin 服务器正在端口 :8080 运行...")
    fmt.Println("请访问 http://localhost:8080/docs 查看 API 文档。")
    fmt.Println("或访问 http://localhost:8080/openapi.yaml 查看原始 OpenAPI Spec。")

    // 启动服务器
    err := r.Run(":8080")
    fmt.Println(err)
}
