package sgin

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

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
	white   = "\033[97m"
)

func debugInfo(format string, values ...any) {
	if gin.IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		_, _ = fmt.Fprintf(gin.DefaultWriter, "[GIN-debug] "+format, values...)
	}
}

func debugWarning(format string, values ...any) {
	_, _ = fmt.Fprintf(gin.DefaultWriter, "[GIN-WARNING] "+format, values...)
}

func debugError(err error) {
	if err != nil && gin.IsDebugging() {
		_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[GIN-debug] [ERROR] %v\n", err)
	}
}
