package main

import (
	"errors"
	"fmt"

	"github.com/baagod/sgin"
	"github.com/gin-gonic/gin"
)

type Person struct {
	Name string `form:"name" json:"name"`
	Age  int    `form:"age" json:"age"`
}

func main() {
	r := sgin.New(sgin.Config{
		Mode: gin.DebugMode, // 默认值
		Recovery: func(ctx *sgin.Ctx, s string) {
			fmt.Println("[Recovery]", s)
		},
		ErrorHandler: func(c *sgin.Ctx, err error) error {
			var e *sgin.Error
			statusCode := sgin.StatusInternalServerError

			if errors.As(err, &e) && e.Code > 0 { // 如果是 *Error 错误
				statusCode = e.Code
			} else if stc := c.StatusCode(); stc != 200 && stc != 0 {
				statusCode = stc
			}

			c.Header(sgin.HeaderContentType, sgin.MIMETextPlainCharsetUTF8)
			return c.Status(statusCode).Send(err.Error())
		},
	})

	r.Use(func(c *sgin.Ctx) error {
		var err error
		if err != nil {
			return err
		}
		return c.Next()
	})

	r.GET("test", func(c *sgin.Ctx, person Person) (r *sgin.Response) {
		// person := Person{Name: "n", Age: 10}
		// var err error
		// var data any
		// var action string
		// if action == "add" {
		// 	// 创建逻辑
		// 	r.Error(db.create(&data).Error).Data(data)
		// } else if action == "update" {
		// 	// 更新逻辑
		// 	r.Error(db.updates(&data).Error)
		// }
		return r.OK(person)
	})
}
