package sgin

import (
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

// Logger 返回一个 Gin 中间件，用于打印结构化的 JSON 请求日志。
func Logger(c *Ctx) {
	gc := c.ctx // *gin.Context

	// Start timer
	start := time.Now()
	path := c.Request.URL.Path
	if raw := c.Request.URL.RawQuery; raw != "" {
		path = path + "?" + raw
	}

	gc.Next()         // Process request
	end := time.Now() // Stop timer

	// 提取信息
	t := end.Format("2006-01-02 15:04:05")
	status := c.Writer.Status()
	ip := c.IP()
	traceid := c.traceid
	latency := fmt.Sprintf("%dms", end.Sub(start).Milliseconds())
	errMsg := gc.Errors.ByType(gin.ErrorTypePrivate).String()

	// 生成 Text 消息
	msg := fmt.Sprintf("[GIN] %s | status=%d latency=%s ip=%s method=%s path=%s traceid=%s",
		t, status, latency, ip, c.Request.Method, path, traceid,
	)
	if errMsg != "" {
		msg += " | error=" + errMsg
	}

	// 如果了 Logger 回调则执行并返回，否则输出默认日志到控制台。
	if fn := c.engine.cfg.Logger; fn != nil {
		logMap := map[string]any{ // 生成 JSON 消息
			"time":    t,
			"status":  status,
			"latency": latency,
			"ip":      ip,
			"method":  c.Request.Method,
			"path":    path,
			"traceid": traceid,
		}

		if errMsg != "" {
			logMap["error"] = errMsg
		}

		var sb strings.Builder
		enc := sonic.ConfigFastest.NewEncoder(&sb)
		enc.SetEscapeHTML(false) // 禁止 HTML 转义
		_ = enc.Encode(logMap)

		fn(c, msg, sb.String())
		return
	}

	fmt.Println(msg)
}
