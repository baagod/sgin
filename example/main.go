package main

import (
	"fmt"

	"github.com/baagod/sgin"
	"github.com/baagod/sgin/oa"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	// 传统方式需要导入具体语言包（已不推荐）：
	// "github.com/go-playground/locales/zh"
	// tzh "github.com/go-playground/validator/v10/translations/zh"
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
	Age  int    `json:"age" binding:"required,gt=18" label:"年龄"`
}

func main() {
	r := sgin.New(sgin.Config{
		Mode:    gin.DebugMode,       // 调试模式
		OpenAPI: oa.New(oa.Config{}), // 开启 OpenAPI 文档服务
		// 框架会自动查找并创建对应的翻译器
		// 目前支持的语言：中文(zh)、英文(en)、日文(ja)、韩文(ko)、法文(fr)、俄文(ru)、德文(de)、西班牙文(es)
		Locales: []language.Tag{language.Chinese, language.English},
		Recovery: func(c *sgin.Ctx, out, s string) {
			fmt.Println(out)
		},
	})

	r.POST("/i18n", func(c *sgin.Ctx, user User) gin.H {
		return gin.H{"msg": "success"}
	})

	// Panic 演示路由 - 用于测试 Recovery 功能
	r.GET("/api/users/:id", HandleAPI)

	fmt.Println("Sgin 服务器正在端口 :8080 运行...")
	fmt.Println("请访问 http://localhost:8080/docs 查看 API 文档。")
	fmt.Println("或访问 http://localhost:8080/openapi.yaml 查看原始 OpenAPI Spec。")
	fmt.Println("Panic 测试: GET http://localhost:8080/api/users/123")

	// 启动服务器
	err := r.Run(":8080")
	fmt.Println(err)
}

// UserProfile 用于演示 panic 的结构体
type UserProfile struct {
	Name    string
	Profile *ProfileInfo
}

type ProfileInfo struct {
	Name string
}

// LoadUserProfile 模型层函数 - 这里会触发 panic
func LoadUserProfile(userID string) (*UserProfile, error) {
	user := &UserProfile{Name: "测试用户", Profile: nil}
	// 加载用户详细信息
	profileName := user.Profile.Name // panic 发生在这里
	_ = profileName                  // 避免编译警告
	return user, nil
}

// GetUserProfile 业务层函数
func GetUserProfile(userID string) (*UserProfile, error) {
	// 调用模型层获取用户信息
	return LoadUserProfile(userID)
}

// HandleAPI API 层处理函数
func HandleAPI(c *sgin.Ctx) {
	userID := c.Param("id")
	profile, err := GetUserProfile(userID)
	if err != nil {
		c.Send(err)
		return
	}
	c.Send(profile)
}
