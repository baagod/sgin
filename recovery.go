package sgin

import (
    "bufio"
    "errors"
    "fmt"
    "net"
    "net/http/httputil"
    "os"
    "runtime"
    "strings"
    "time"
)

// ANSI Color Codes
const (
    reset   = "\033[0m"
    red     = "\033[31m"
    green   = "\033[32m"
    yellow  = "\033[33m"
    blue    = "\033[34m"
    magenta = "\033[35m"
    cyan    = "\033[36m"
    white   = "\033[37m"
    bold    = "\033[1m"
    dim     = "\033[2m"
    bgRed   = "\033[41m"
    bgBlue  = "\033[44m"
)

// Recovery 是一个增强版的错误恢复中间件
// 它能打印出发生 panic 的具体源代码片段
func Recovery(c *Ctx) {
    gc := c.ctx
    defer func() {
        if r := recover(); r != nil {
            err := r.(error)

            // 检查连接是否断开 (broken pipe)
            var brokenPipe bool
            var ne *net.OpError
            if errors.As(err, &ne) {
                var se *os.SyscallError
                if errors.As(ne, &se) {
                    seStr := strings.ToLower(se.Error())
                    brokenPipe = strings.Contains(seStr, "broken pipe") ||
                        strings.Contains(seStr, "connection reset by peer")
                }
            }

            // 获取堆栈信息
            stack := stack(3)
            httpRequest, _ := httputil.DumpRequest(c.Request, false)
            if brokenPipe {
                // 如果是连接断开，通常不需要打印花哨的日志，安静记录即可
                fmt.Printf("%s[BROKEN PIPE]%s %s\n%s\n", red, reset, err, string(httpRequest))
                _ = gc.Error(err)
                gc.Abort()
                return
            }

            // --- 开始漂亮的打印 ---
            t := time.Now().Format("2006-01-02 15:04:05")
            fmt.Println()
            fmt.Printf("%sPANIC RECOVERED BEGIN%s\n", red+bold, reset)
            fmt.Printf("%sTime:%s     %s\n", green, reset, t)
            fmt.Printf("%sRequest:%s  %s %s\n", yellow, reset, c.Request.Method, c.Request.URL.Path)
            fmt.Printf("%sIP:%s       %s\n", yellow, reset, c.IP())
            fmt.Printf("%sTraceID:%s  %s\n", yellow, reset, c.traceid)
            fmt.Printf("%sError:%s    %v\n", red, reset, err)

            // 打印 Headers
            headers := "  " + strings.ReplaceAll(string(httpRequest), "\n", "\n  ")
            headers = strings.TrimSuffix(headers, "\n  ")
            fmt.Printf("%sHeaders:%s\n%s", magenta, reset, headers)

            // 打印源码上下文
            fmt.Printf("%sFile:%s %s:%d\n", cyan, reset, stack.file, stack.line)
            printSource(stack.file, stack.line)
            fmt.Printf("\n%sPANIC RECOVERED END%s\n", red+bold, reset)

            _ = c.Send(ErrInternalServerError()) // 返回 500 响应
        }
    }()

    gc.Next()
}

// source 存储堆栈的关键信息
type source struct {
    file     string
    line     int
    funcName string
}

// stack 获取调用栈中第一个由于用户代码触发的帧
func stack(skip int) *source {
    // 我们最多往上找 32 层
    for i := skip; i < 32; i++ {
        pc, file, line, ok := runtime.Caller(i)
        if !ok {
            break
        }

        // 过滤掉 Go Runtime 和 Gin 内部的代码，只找业务代码
        if !strings.Contains(file, "runtime/") &&
            !strings.Contains(file, "github.com/gin-gonic/gin") &&
            !strings.Contains(file, "sgin/recovery.go") /* 过滤自己 */ {
            return &source{
                file:     file,
                line:     line,
                funcName: runtime.FuncForPC(pc).Name(),
            }
        }
    }

    return &source{}
}

// printSource 读取文件并打印出错行及其前后几行
func printSource(filename string, line int) {
    f, err := os.Open(filename)
    if err != nil {
        return
    }
    defer f.Close()

    var lines []string
    minIndent, start, end := 1000, line-3, line+3
    scanner := bufio.NewScanner(f)

    for cur := 1; scanner.Scan() && cur <= end; cur++ {
        if cur >= start {
            text := strings.Replace(scanner.Text(), "\t", "    ", -1)
            lines = append(lines, text)
            // 计算缩进：原长度 - 去除左空格后的长度
            if trimmed := strings.TrimLeft(text, " "); trimmed != "" {
                if indent := len(text) - len(trimmed); indent < minIndent {
                    minIndent = indent
                }
            }
        }
    }

    for i, code := range lines {
        if minIndent < 1000 && len(code) >= minIndent {
            code = code[minIndent:]
        }
        if lineNum := start + i; lineNum == line {
            fmt.Printf("  %s%d > %s%s\n", red+bold, lineNum, code, reset)
        } else {
            fmt.Printf("  %s%d   %s%s\n", dim, lineNum, code, reset)
        }
    }
}
