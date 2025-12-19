package sgin

import (
    "bufio"
    "errors"
    "fmt"
    "io"
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

// source 存储堆栈的关键信息
type source struct {
    file     string
    line     int
    funcName string
}

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

            // 如果是连接断开，通常不需要打印花哨的日志，安静记录即可。
            req, _ := httputil.DumpRequest(c.Request, false)
            if brokenPipe {
                fmt.Printf("%s[BROKEN PIPE]%s %s\n%s\n", red, reset, err, string(req))
                _ = gc.Error(err)
                gc.Abort()
                return
            }

            // --- 构建漂亮的日志 ---
            // 输出日志：如果有回调则给回调，否则打印到 Stdout。
            logStr := buildPanicLog(c, err, stack(3), req)
            if recovery := c.engine.cfg.Recovery; recovery != nil {
                recovery(c, logStr)
            } else {
                fmt.Print(logStr)
            }

            _ = c.Send(ErrInternalServerError()) // 返回 500 响应
        }
    }()

    gc.Next()
}

func buildPanicLog(c *Ctx, err error, stack *source, req []byte) string {
    var sb strings.Builder
    t := time.Now().Format("2006-01-02 15:04:05")

    sb.WriteString(fmt.Sprintf("\n%sPANIC RECOVERED BEGIN%s\n", red+bold, reset))
    sb.WriteString(fmt.Sprintf("%sTime:%s     %s\n", green, reset, t))
    sb.WriteString(fmt.Sprintf("%sRequest:%s  %s %s\n", yellow, reset, c.Request.Method, c.Request.URL.Path))
    sb.WriteString(fmt.Sprintf("%sIP:%s       %s\n", yellow, reset, c.IP()))
    sb.WriteString(fmt.Sprintf("%sTraceID:%s  %s\n", yellow, reset, c.traceid))
    sb.WriteString(fmt.Sprintf("%sError:%s    %v\n", red, reset, err))

    // 打印 Headers
    headers := "  " + strings.ReplaceAll(string(req), "\n", "\n  ")
    headers = strings.TrimSuffix(headers, "\n  ")
    sb.WriteString(fmt.Sprintf("%sHeaders:%s\n%s", magenta, reset, headers))

    // 打印源码上下文（Killer Feature）
    sb.WriteString(fmt.Sprintf("%sFile:%s %s:%d\n", cyan, reset, stack.file, stack.line))
    printSource(&sb, stack)
    sb.WriteString(fmt.Sprintf("%sPANIC RECOVERED END%s\n\n", red+bold, reset))

    return sb.String()
}

// printSource 读取文件并打印出错行及其前后几行 (注释中假设报错行是第 100 行)
func printSource(w io.Writer, s *source) {
    f, err := os.Open(s.file)
    if err != nil {
        return
    }
    defer f.Close()

    var lines []string
    minIndent, start, end := 1000, s.line-3, s.line+3
    scanner := bufio.NewScanner(f)

    for cur := 1; scanner.Scan() && cur <= end; cur++ {
        if cur < start {
            continue // 只有行号进入了视窗 (>=97) 才开始处理
        }

        // 统一格式：把 Tab 变成 4 个空格，防止排版乱掉。
        text := strings.ReplaceAll(scanner.Text(), "\t", "    ")
        lines = append(lines, text)

        // 核心逻辑：计算这行代码左边有几个空格
        if trimmed := strings.TrimLeft(text, " "); trimmed != "" {
            // indent = 原始长度 - 去掉左空格后的长度 = 左边空格的数量
            if indent := len(text) - len(trimmed); indent < minIndent {
                minIndent = indent // 更新最小缩进值
            }
        }
    }

    for i, code := range lines {
        // 裁剪动作：如果这行代码长度够长，就切掉 minIndent (4个字符)。
        if minIndent < 1000 && len(code) >= minIndent {
            code = code[minIndent:]
        }
        if row := start + i; row == s.line { // 报错行 (100)
            _, _ = fmt.Fprintf(w, "  %s%d > %s%s\n", red+bold, row, code, reset)
        } else { // 上下文行
            _, _ = fmt.Fprintf(w, "  %s%d   %s%s\n", dim, row, code, reset)
        }
    }
}

// stack 获取调用栈中第一个由于用户代码触发的帧
func stack(skip int) *source {
    for i := skip; i < 32; i++ { // 最多往上找 32 层
        pc, file, line, ok := runtime.Caller(i)
        if !ok {
            break
        }

        // 过滤掉 Go Runtime 和 Gin 内部的代码，只找业务代码。
        if !strings.Contains(file, "runtime/") &&
            !strings.Contains(file, "github.com/gin-gonic/gin") &&
            !strings.Contains(file, "github.com/baagod/sgin") &&
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
