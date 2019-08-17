package httputil

import (
	"backend/common/gls"
	"backend/common/trace"
	"fmt"
	"os"
	"third/gin"
)

func TraceHttpRoot() gin.HandlerFunc {
	hn, _ := os.Hostname()
	return func(c *gin.Context) {
		reqId := c.Request.Header.Get(CODOON_REQUEST_ID)
		if reqId == "" {
			return
		}

		// create root span
		name := fmt.Sprintf("HTTP %s %s%s", c.Request.Method, c.Request.Host, c.Request.URL.Path)
		// TODO: implement sample rate
		sp := trace.StartSpanWithTraceID(name, reqId, true)

		trace.Inject(sp.Context(), c.Request.Header)

		c.Next()

		userId := getHeaderUid(c)
		statusCode := c.Writer.Status()
		sp.SetTag("common.user_id", userId)
		sp.SetTag("common.status_code", fmt.Sprintf("%d", statusCode))
		sp.SetTag("hostname", hn)

		sp.Finish()
	}
}

func TraceHttpSpan() gin.HandlerFunc {
	return func(c *gin.Context) {

		name := fmt.Sprintf("HTTP %s %s%s", c.Request.Method, c.Request.Host, c.Request.URL.Path)
		sp, err := trace.ExtractChildSpan(name, c.Request.Header)
		if err != nil {
			return
		}

		trace.Inject(sp.Context(), c.Request.Header)

		c.Next()

		userId := getHeaderUid(c)
		statusCode := c.Writer.Status()
		sp.SetTag("common.user_id", userId)
		sp.SetTag("common.status_code", fmt.Sprintf("%d", statusCode))

		sp.Finish()
	}
}

func TraceHttpSpanGls(goid func() int64) gin.HandlerFunc {
	return func(c *gin.Context) {

		name := fmt.Sprintf("HTTP %s %s%s", c.Request.Method, c.Request.Host, c.Request.URL.Path)
		sp, err := trace.ExtractChildSpan(name, c.Request.Header)
		if err != nil {
			return
		}

		trace.Inject(sp.Context(), c.Request.Header)
		gls.Set(goid, gls.Values{"trace": sp.Context()})

		c.Next()

		userId := getHeaderUid(c)
		statusCode := c.Writer.Status()
		sp.SetTag("common.user_id", userId)
		sp.SetTag("common.status_code", fmt.Sprintf("%d", statusCode))

		sp.Finish()
	}
}
