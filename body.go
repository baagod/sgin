package sgin

const (
	FmtXML      = "XML"
	FmtJSON     = "JSON"
	FmtText     = "Text"
	FmtUpload   = "Upload"
	FmtDownload = "Download"
	FmtHTML     = "HTML"
)

// Body 表示一个类型化的 HTTP 响应体，用于通过 Send 方法指定响应格式。
// 应使用 BodyJSON, BodyHTML 等工厂函数构造，而非直接初始化。
type Body struct {
	format string // 具体的格式标识
	name   string // HTML name
	data   any    // HTML data
}

func BodyXML(data any) Body {
	return Body{format: FmtXML, data: data}
}

func BodyJSON(data any) Body {
	return Body{format: FmtJSON, data: data}
}

func BodyText(data any) Body {
	return Body{format: FmtText, data: data}
}

func BodyUpload(data any) Body {
	return Body{format: FmtUpload, data: data}
}

func BodyDownload(data any) Body {
	return Body{format: FmtDownload, data: data}
}

func BodyHTML(name string, data any) Body {
	return Body{format: FmtHTML, name: name, data: data}
}
