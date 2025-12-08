package sgin

import (
    "encoding/json"
    "fmt"
    "time"

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

    // 执行回调或默认输出
    next := true
    if logger := c.engine.config.Logger; logger != nil {
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

        jsonBytes, _ := json.Marshal(logMap)
        next = logger(c, msg, string(jsonBytes))
    }

    if next { // 默认输出 msg，方便终端查看。
        fmt.Fprintf(gin.DefaultWriter, "%s\n", msg)
    }
}
