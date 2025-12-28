package sgin

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

// Stack 存储堆栈的关键信息
type Stack struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Func        string `json:"func"`
	Source      string `json:"source"`
	colorSource string
}

type RecoverInfo struct {
	Time        string   `json:"time"`
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Host        string   `json:"host"`
	IP          string   `json:"ip"`
	ContentType string   `json:"content"`
	Accept      string   `json:"accept"`
	Traceid     string   `json:"traceid"`
	Error       string   `json:"error"`
	Sources     []*Stack `json:"stack"`
}

func (r *RecoverInfo) String() string {
	sb := &strings.Builder{}

	printf(sb, "\n%s PANIC RECOVERED %s\n", bgRed+white+bold, reset)
	printf(sb, "%sTime:%s         %s\n", green, reset, r.Time)
	printf(sb, "%sRequest:%s      %s %s\n", yellow, reset, r.Method, r.Path)
	printf(sb, "%sIP:%s           %s\n", yellow, reset, r.IP)
	printf(sb, "%sHost:%s         %s\n", white, reset, r.Host)
	printf(sb, "%sContent-Type:%s %s\n", yellow, reset, r.ContentType)
	printf(sb, "%sAccept:%s %s\n", yellow, reset, r.Accept)
	printf(sb, "%sTraceID:%s      %s\n", yellow, reset, r.Traceid)
	printf(sb, "%sError:%s        %v\n", red, reset, r.Error)

	// 打印源码上下文 (Killer Feature)
	for i, s := range r.Sources {
		format := "%sFile:%s %s:%d %s%s()%s\n"
		printf(sb, format, cyan, reset, s.File, s.Line, magenta, s.Func, reset)
		sb.WriteString(s.colorSource)

		if i < len(r.Sources)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (r *RecoverInfo) JSON() string {
	var sb strings.Builder
	enc := sonic.ConfigFastest.NewEncoder(&sb)
	enc.SetEscapeHTML(false) // 禁止 HTML 转义
	// enc.SetIndent("", "    ")
	_ = enc.Encode(r)
	return sb.String()
}

// Recovery 是一个增强版的错误恢复中间件，它能打印出发生 panic 的具体源代码片段。
var Recovery = He(func(c *Ctx) error {
	gc := c.Gin()
	defer func() {
		if recovered := recover(); recovered != nil {
			err, _ := recovered.(error)
			if err == nil {
				err = fmt.Errorf("%v", recovered)
			}

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

			info := RecoverInfo{
				Time:        time.Now().Format("2006-01-02 15:04:05"),
				Path:        c.Request.URL.Path,
				Method:      c.Request.Method,
				Host:        c.Request.Host,
				IP:          c.IP(),
				ContentType: c.GetHeader(HeaderContentType),
				Accept:      c.GetHeader(HeaderAccept),
				Traceid:     c.traceid,
				Error:       err.Error(),
				Sources:     stack(3),
			}

			// --- 构建漂亮的日志 ---
			if fn := c.engine.cfg.Recovery; fn != nil {
				fn(c, info.String(), info.JSON())
			} else {
				fmt.Print(info.String())
			}

			_ = c.Send(ErrInternalServerError()) // 返回 500 响应
		}
	}()

	return c.Next()
})

// readSource 读取文件及其前后几行 (注释中假设报错行是第 100 行)
func readSource(file string, line int) (string, string) {
	f, err := os.Open(file)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	var lines []string
	minIndent, start, end := 1000, line-3, line+3
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

	source := &strings.Builder{}
	colorSource := &strings.Builder{}

	for i, code := range lines {
		// 裁剪动作：如果这行代码长度够长，就切掉 minIndent (4个字符)。
		if minIndent < 1000 && len(code) >= minIndent {
			code = code[minIndent:]
		}

		if row := start + i; row == line { // 报错行 (100)
			printf(source, "%d > %s\n", row, code)
			printf(colorSource, "  %s%d > %s%s\n", red+bold, row, code, reset)
		} else { // 上下文行
			printf(source, "%d   %s\n", row, code)
			printf(colorSource, "  %d   %s%s\n", row, code, reset)
		}
	}

	return source.String(), colorSource.String()
}

// stack 获取调用栈中前几个由于用户代码触发的帧
func stack(skip int) (sources []*Stack) {
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
			!strings.Contains(file, "github.com/baagod/sgin") &&
			!strings.Contains(file, "sgin/recovery.go") /* 过滤自己 */ {

			funcName := runtime.FuncForPC(pc).Name()
			if index := strings.LastIndex(funcName, "."); index != -1 {
				funcName = funcName[index+1:]
			}

			source, colorSource := readSource(file, line)
			sources = append(sources, &Stack{
				File:        shortenPath(file),
				Line:        line,
				Func:        funcName,
				Source:      source,
				colorSource: colorSource,
			})

			if len(sources) >= 4 { // 找到 4 层就够了
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
		// gin@v1.9.1/context.go -> gin/context.go
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
