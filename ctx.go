package sgin

import (
    "bytes"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "path/filepath"
    "strings"

    "github.com/clbanning/mxj/v2"
    "github.com/rs/xid"
    "github.com/spf13/cast"

    "github.com/bytedance/sonic"
    "github.com/gin-gonic/gin"
)

const CtxKey = "_baa/sgin/ctxkey"

const (
    FormatXML      = "XML"
    FormatJSON     = "JSON"
    FormatText     = "Text"
    FormatUpload   = "Upload"
    FormatDownload = "Download"
)

type Ctx struct {
    Request *http.Request
    Writer  gin.ResponseWriter
    Params  gin.Params
    Keys    map[string]any

    args    any
    traceid string
    engine  *Engine
    ctx     *gin.Context
}

func newCtx(ctx *gin.Context, e *Engine) *Ctx {
    c := &Ctx{
        engine:  e,
        ctx:     ctx,
        traceid: ctx.GetHeader(HeaderXRequestID),
        Request: ctx.Request,
        Writer:  ctx.Writer,
        Params:  ctx.Params,
        Keys:    ctx.Keys,
    }

    if c.traceid == "" {
        c.traceid = xid.New().String()
        c.SetHeader(HeaderXRequestID, c.traceid)
    }

    return c
}

// ------ 请求参数 ------

func (c *Ctx) Args() (args map[string]any) {
    // 已经解析过请求数据
    if args, _ = c.args.(map[string]any); args != nil {
        return
    }

    args = map[string]any{}
    ct := c.Header(HeaderContentType)

    if ct == "" || c.Request.Method == "GET" || ct == MIMEForm {
        _ = c.Request.ParseForm()
        for k, v := range c.Request.Form {
            args[k] = v[0]
        }
    } else if strings.HasPrefix(ct, MIMEMultipartForm) {
        if form, err := c.ctx.MultipartForm(); err == nil {
            for k, v := range form.Value {
                args[k] = v[0]
            }
            for k, v := range form.File {
                args[k] = v[0]
            }
        }
    }

    switch ct {
    case MIMEJSON:
        r := bytes.NewReader(c.RawBody())
        dec := sonic.ConfigDefault.NewDecoder(r)
        dec.UseNumber()
        _ = dec.Decode(&args)
    case MIMEXML:
        if m, _ := mxj.NewMapXml(c.RawBody()); m != nil {
            args = m
        }
    }

    c.args = args
    return args
}

func (c *Ctx) Arg(key string, or ...string) string {
    if v, ok := c.Args()[key]; ok {
        return fmt.Sprint(v)
    }
    return append(or, "")[0]
}

func (c *Ctx) ArgInt(key string, or ...int) int {
    v, err := cast.ToIntE(c.Arg(key))
    if err != nil && or != nil {
        return or[0]
    }
    return v
}

func (c *Ctx) ArgInt64(key string, or ...int64) int64 {
    v, err := cast.ToInt64E(c.Arg(key))
    if err != nil && or != nil {
        return or[0]
    }
    return v
}

func (c *Ctx) ArgFloat64(key string, or ...float64) float64 {
    v, err := cast.ToFloat64E(c.Arg(key))
    if err != nil && or != nil {
        return or[0]
    }
    return v
}

func (c *Ctx) ArgBool(key string) bool {
    return cast.ToBool(c.Arg(key))
}

// ------ 响应 ------

func (c *Ctx) Send(body any, format ...string) error {
    c.autoFormat(body, format...)
    return nil
}

func (c *Ctx) SendHTML(name string, data any) error {
    c.ctx.Abort()
    c.ctx.HTML(c.StatusCode(), name, data)
    return nil
}

func (c *Ctx) Next() error {
    c.ctx.Next()
    return nil
}

// ------ SET 设置 ------

func (c *Ctx) Status(code int) *Ctx {
    c.Writer.WriteHeader(code)
    return c
}

// Locals 设置或将值存储到上下文
func (c *Ctx) Locals(key string, value ...any) any {
    if value != nil {
        c.ctx.Set(key, value[0])
        return nil
    }
    v, _ := c.ctx.Get(key)
    return v
}

// ------ GET 获取 ------

func (c *Ctx) Method() string {
    return c.Request.Method
}

// Header 获取 HTTP 请求头的值，如果不存在则返回可选的默认值。
func (c *Ctx) Header(key string, value ...string) string {
    header := c.ctx.GetHeader(key)
    if header == "" && value != nil {
        return value[0]
    }
    return header
}

func (c *Ctx) SetHeader(key string, value string) {
    c.ctx.Header(key, value)
}

func (c *Ctx) RawBody() (body []byte) {
    if body, _ = c.Locals(gin.BodyBytesKey).([]byte); body == nil {
        if body, _ = io.ReadAll(c.Request.Body); body != nil {
            c.Locals(gin.BodyBytesKey, body)
        }
    }
    return body
}

func (c *Ctx) StatusCode() int {
    return c.ctx.Writer.Status()
}

func (c *Ctx) Path(full ...bool) string {
    if full != nil {
        return c.ctx.FullPath()
    }
    return c.ctx.Request.URL.Path
}

func (c *Ctx) Param(key string) string {
    return c.Params.ByName(key)
}

func (c *Ctx) IP() string {
    return c.ctx.ClientIP()
}

func (c *Ctx) Cookie(name string) (string, error) {
    return c.ctx.Cookie(name)
}

func (c *Ctx) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
    c.ctx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

func (c *Ctx) SaveFile(file *multipart.FileHeader, dst string) error {
    return c.ctx.SaveUploadedFile(file, dst)
}

// TraceID 获取当前请求的 [跟踪ID]
func (c *Ctx) TraceID() string {
    return c.traceid
}

// sendResult 消费内部归一化的响应结果
func (c *Ctx) sendResult(r *result) {
    // 1. 处理状态码 (如果有显式设置)
    if r.Status > 0 {
        c.Status(r.Status)
    }

    // 2. 先处理错误
    if r.Err != nil {
        _ = c.engine.config.ErrorHandler(c, r.Err)
        return
    }

    // 3. 发送数据
    if r.Data != nil {
        _ = c.Send(r.Data)
    }
}

// autoFormat 自动根据 Accept 头返回对应类型的数据
func (c *Ctx) autoFormat(body any, format ...string) {
    gc := c.ctx
    if gc.Abort(); body == nil { // 停止后续请求链的执行
        return
    }

    status := c.StatusCode() // HTTP 状态码

    // 1. 如果指定格式
    if len(format) > 0 {
        switch format[0] {
        case FormatJSON:
            gc.JSON(status, body)
        case FormatXML:
            gc.XML(status, body)
        case FormatText:
            gc.String(status, fmt.Sprint(body))
        case FormatUpload, FormatDownload:
            file := fmt.Sprint(body)
            if format[0] == FormatDownload {
                filename := filepath.Base(file)
                c.Header(HeaderContentDisposition, `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
            }
            gc.File(file)
        }
        return
    }

    // 2. 特殊类型处理
    if err, ok := body.(error); ok {
        _ = c.engine.config.ErrorHandler(c, err)
        return
    }

    // 3. 按 Accept 头协商
    // 如果明确只想要 XML 且不包含 HTML (排除浏览器默认行为)，才返回 XML。
    // 浏览器通常 Accept 包含 text/html 和 application/xml，这里避免误判。
    accept := c.Header(HeaderAccept)
    if (strings.Contains(accept, MIMEXML) || strings.Contains(accept, MIMETextXML)) &&
        !strings.Contains(accept, MIMETextHTML) {
        gc.XML(status, body)
        return
    }

    // 4. 默认策略 (按类型推断)
    switch v := body.(type) {
    case string:
        // 如果是字符串，默认为 text/plain
        gc.String(status, v)
    default:
        // 其他类型（包括 struct, map, []byte, int 等），默认为 JSON
        gc.JSON(status, body)
    }
}
