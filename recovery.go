package sgin

import (
    "bufio"
    "errors"
    "fmt"
    "io"
    "net"
    "os"
    "path/filepath"
    "regexp"
    "runtime"
    "strings"
    "time"
)

var plaintext = regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString

// ANSI Color Codes
const (
    reset   = "\033[0m"
    red     = "\033[31m"
    green   = "\033[32m"
    yellow  = "\033[33m"
    blue    = "\033[34m"
    magenta = "\033[35m"
    cyan    = "\033[36m"
    bold    = "\033[1m"
    bgRed   = "\033[41m"
    bgBlue  = "\033[44m"
    black   = "\033[30m"
    white   = "\033[97m"
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
            if brokenPipe {
                fmt.Printf("%s[BROKEN PIPE]%s %s\n", red, reset, err)
                _ = gc.Error(err)
                gc.Abort()
                return
            }

            // --- 构建漂亮的日志 ---
            // 输出日志：如果有回调则给回调，否则打印到 Stdout。
            out := buildPanicLog(c, err, stacks(3))
            if recovery := c.engine.cfg.Recovery; recovery != nil {
                recovery(c, out, plaintext(out, ""))
            } else {
                fmt.Print(out)
            }

            _ = c.Send(ErrInternalServerError()) // 返回 500 响应
        }
    }()

    gc.Next()
}

func buildPanicLog(c *Ctx, err error, stacks []*source) string {
    sb := &strings.Builder{}
    t := time.Now().Format("2006-01-02 15:04:05")

    printf(sb, "\n%s PANIC RECOVERED %s\n", bgRed+white+bold, reset)
    printf(sb, "%sTime:%s         %s\n", green, reset, t)
    printf(sb, "%sRequest:%s      %s %s\n", yellow, reset, c.Request.Method, c.Request.URL.Path)
    printf(sb, "%sHost:%s         %s\n", white, reset, c.Request.Host)
    printf(sb, "%sContent-Type:%s %s\n", yellow, reset, c.Header(HeaderContentType))
    printf(sb, "%sIP:%s           %s\n", yellow, reset, c.IP())
    printf(sb, "%sTraceID:%s      %s\n", yellow, reset, c.traceid)
    printf(sb, "%sError:%s        %v\n", red, reset, err)

    // 打印源码上下文 (Killer Feature)
    for i, s := range stacks {
        format := "%sFile:%s %s:%d %s%s()%s\n"
        printf(sb, format, cyan, reset, shortenPath(s.file), s.line, magenta, s.funcName, reset)
        if printSource(sb, s); i < len(stacks)-1 {
            sb.WriteString("\n")
        }
    }

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
            printf(w, "  %s%d > %s%s\n", red+bold, row, code, reset)
        } else { // 上下文行
            printf(w, "  %d   %s%s\n", row, code, reset)
        }
    }
}

// stacks 获取调用栈中前几个由于用户代码触发的帧
func stacks(skip int) []*source {
    var sources []*source
    for i := skip; i < 32; i++ { // 最多往上找 32 层
        pc, file, line, ok := runtime.Caller(i)
        if !ok {
            break
        }

        // 统一使用 / 作为分隔符，确保在 Windows 下也能正确过滤
        file = filepath.ToSlash(file)

        // 过滤掉 Go Runtime 和 Gin 内部的代码，只找业务代码。
        if !strings.Contains(file, "runtime/") &&
            !strings.Contains(file, "github.com/gin-gonic/gin") &&
            !strings.Contains(file, "sgin/recovery.go") /* 过滤自己 */ {

            funcName := runtime.FuncForPC(pc).Name()
            if index := strings.LastIndex(funcName, "."); index != -1 {
                funcName = funcName[index+1:]
            }

            // 找到 4 层就够了
            sources = append(sources, &source{file: file, line: line, funcName: funcName})
            if len(sources) >= 4 {
                break
            }
        }
    }

    return sources
}

// shortenPath 缩短文件路径
func shortenPath(file string) string {
    if file == "" {
        return ""
    }

    path := filepath.ToSlash(file)

    // 1. 第三方库：检测 /pkg/mod/
    if i := strings.Index(path, "/pkg/mod/"); i != -1 {
        path = path[i+len("/pkg/mod/"):]
        // 去除版本号 (e.g., @v1.2.3)
        // path/gin@v1.9.1/context.go -> path/gin/context.go
        if j := strings.Index(path, "@"); j != -1 {
            if index := strings.Index(path[j:], "/"); index != -1 {
                path = path[:j] + path[j+index:]
            }
        }
        return path
    }

    // 2. 标准库：检测 GOROOT
    // os.Getenv: 获取 Go 语言的安装目录 (如 c:/go)
    if godir := os.Getenv("GOROOT"); godir != "" {
        // Rel: 计算从 "c:/go/src" 到 "c:/go/src/path/../file.go" 的相对路径
        rel, err := filepath.Rel(filepath.Join(godir, "src"), file)
        if err == nil && !strings.HasPrefix(rel, "..") {
            return filepath.ToSlash(rel)
        }
    }

    // 3. 项目文件：相对于项目根目录
    if wd, err := os.Getwd(); err == nil {
        rel, err := filepath.Rel(wd, file)
        if err == nil && !strings.HasPrefix(rel, "..") {
            return filepath.ToSlash(rel)
        }
    }

    return path
}

// printf 输出辅助函数
func printf(w io.Writer, format string, a ...any) {
    _, _ = fmt.Fprintf(w, format, a...)
}
