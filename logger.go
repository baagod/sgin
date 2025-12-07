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
    raw := c.Request.URL.RawQuery

    // Process request
    gc.Next()

    // Stop timer
    end := time.Now()
    latency := end.Sub(start)

    if raw != "" {
        path = path + "?" + raw
    }

    // 提取错误信息
    var errMsg string
    if gc.Errors != nil && len(gc.Errors) > 0 {
        errMsg = gc.Errors.ByType(gin.ErrorTypePrivate).String()
    }

    t := end.Format("2006-01-02 15:04:05")
    status := c.Writer.Status()
    ip := c.IP()
    traceid := c.traceid
    milli := fmt.Sprintf("%dms", latency.Milliseconds())

    // 1. 生成 Text 消息
    // 格式: [GIN] time | status= latency= ip= method= path= trace=
    msg := fmt.Sprintf("[GIN] %s | status=%d latency=%s ip=%s method=%s path=%s trace=%s",
        t, status, milli, ip, c.Request.Method, path, traceid,
    )
    if errMsg != "" {
        msg += " | error=" + errMsg
    }

    // 2. 生成 JSON 消息
    logMap := map[string]any{
        "time":    t,
        "status":  status,
        "latency": milli,
        "ip":      ip,
        "method":  c.Request.Method,
        "path":    path,
        "traceid": traceid,
    }
    if errMsg != "" {
        logMap["error"] = errMsg
    }
    jsonBytes, _ := json.Marshal(logMap)
    jsonLog := string(jsonBytes)

    // 3. 执行回调或默认输出
    next := true
    if logger := c.engine.config.Logger; logger != nil {
        next = logger(c, msg, jsonLog)
    }

    if next { // 默认输出 msg，方便终端查看。
        fmt.Fprintf(gin.DefaultWriter, "%s\n", msg)
    }
}
