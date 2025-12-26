// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sgin

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
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
