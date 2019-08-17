// by liudan
package httputil

import (
	"backend/common/trace"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"third/gin"
	"third/http_client_cluster"
	"time"
)

const (
	GIN_CTX = "gin_ctx"
)

var _httpClient *http.Client

// The Client's Transport typically has internal state (cached TCP connections),
// so Clients should be reused instead of created as needed.
// Clients are safe for concurrent use by multiple goroutines.
func init() {
	tr := &http.Transport{
		Dial: func(network, addr string) (conn net.Conn, err error) {
			return net.DialTimeout(network, addr, 5*time.Second)
		},
	}
	_httpClient = &http.Client{
		Transport: tr,
		Timeout:   45 * time.Second,
	}
}

func HttpSimpleRequest(ctx context.Context, method, addr string, params map[string]string) ([]byte, error) {
	data, _, err := HttpRequestWithCode(ctx, method, addr, params)
	return data, err
}

func HttpRequestWithCode(ctx context.Context, method, addr string, params map[string]string) ([]byte, int, error) {
	return HttpRequestWithHeader(ctx, method, addr, params, nil)
}

//带额外header参数 发送httprequest
func HttpRequestWithHeader(ctx context.Context, method, addr string, params, headers map[string]string) ([]byte, int, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	var request *http.Request
	var err error = nil
	if method == "GET" || method == "DELETE" {
		if params != nil {
			addr = addr + "?" + form.Encode()
		}
		request, err = http.NewRequest(method, addr, nil)
		if err != nil {
			return nil, 0, err
		}
	} else {
		request, err = http.NewRequest(method, addr, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, 0, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// inject header to coloring call chain
	if ctx != nil {
		var spCtx trace.SpanContext
		var spCtxOK bool
		if c, ok := ctx.Value(GIN_CTX).(*gin.Context); ok {
			if spCtx, err = trace.ExtractContext(c.Request.Header); err == nil {
				spCtxOK = true
			}
		} else {
			spCtx, spCtxOK = ctx.Value(trace.CDTCtxKey).(trace.SpanContext)
		}
		if spCtxOK {
			trace.Inject(spCtx, request.Header)
			// if dst host is not a codoon service, start a child span
			if !strings.Contains(request.Host, ".in.codoon.com") {
				name := fmt.Sprintf("%s %s %s%s", request.URL.Scheme, request.Method, request.URL.Host, request.URL.Path)
				childSpCtx := trace.StartChildSpan(name, spCtx)
				defer childSpCtx.Finish()
			}
		}
	}

	// write custom headers
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	response, err := http_client_cluster.HttpClientClusterDo(request)
	if nil != err {
		log.Printf("httpRequest: Do request (%+v) error:%v", request, err)
		return nil, 0, err
	}
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("httpRequest: read response error:%v", err)
		return nil, 0, err
	}

	return data, response.StatusCode, nil
}

//带额外header参数 发送httprequest
func HttpRawRequest(method, addr string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, addr, body)
	if err != nil {
		return nil, err
	}

	if method == "POST" || method == "PUT" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	client := &http.Client{}
	rsp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	return ioutil.ReadAll(rsp.Body)
}

func HttpRequestPostWithUrl(ctx context.Context, method, addr string, params, urls map[string]string) ([]byte, int, error) {
	form1 := url.Values{}
	for k, v := range params {
		form1.Set(k, v)
	}

	form2 := url.Values{}
	for k, v := range urls {
		form2.Set(k, v)
	}

	var request *http.Request
	var err error = nil
	if method != "POST" {
		return nil, 0, err
	} else {
		request, err = http.NewRequest(method, addr+"?"+form2.Encode(), strings.NewReader(form1.Encode()))
		if err != nil {
			return nil, 0, err
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// inject header to coloring call chain
	if ctx != nil {
		var spCtx trace.SpanContext
		var spCtxOK bool
		if c, ok := ctx.Value(GIN_CTX).(*gin.Context); ok {
			if spCtx, err = trace.ExtractContext(c.Request.Header); err == nil {
				spCtxOK = true
			}
		} else {
			spCtx, spCtxOK = ctx.Value(trace.CDTCtxKey).(trace.SpanContext)
		}
		if spCtxOK {
			trace.Inject(spCtx, request.Header)
			// if dst host is not a codoon service, start a child span
			if !strings.Contains(request.Host, ".in.codoon.com") {
				name := fmt.Sprintf("%s %s %s%s", request.URL.Scheme, request.Method, request.URL.Host, request.URL.Path)
				childSpCtx := trace.StartChildSpan(name, spCtx)
				defer childSpCtx.Finish()
			}
		}
	}

	response, err := http_client_cluster.HttpClientClusterDo(request)
	if nil != err {
		log.Printf("httpRequest: Do request (%+v) error:%v", request, err)
		return nil, 0, err
	}
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("httpRequest: read response error:%v", err)
		return nil, 0, err
	}

	return data, response.StatusCode, nil
}

func HttpAPIFormRequest(ctx context.Context, method, addr string, params map[string]string, rsp interface{}) ([]byte, error) {
	data, err := HttpSimpleRequest(ctx, method, addr, params)
	if err != nil {
		return nil, err
	}
	if rsp != nil {
		err = json.Unmarshal(data, rsp)
	}
	return data, err
}

func HttpAPIJsonRequest(ctx context.Context, method, addr string, req, rsp interface{}) ([]byte, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, addr, bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}

	// inject header to coloring call chain
	if ctx != nil {
		var spCtx trace.SpanContext
		var spCtxOK bool
		if c, ok := ctx.Value(GIN_CTX).(*gin.Context); ok {
			if spCtx, err = trace.ExtractContext(c.Request.Header); err == nil {
				spCtxOK = true
			}
		} else {
			spCtx, spCtxOK = ctx.Value(trace.CDTCtxKey).(trace.SpanContext)
		}
		if spCtxOK {
			trace.Inject(spCtx, request.Header)
			// if dst host is not a codoon service, start a child span
			if !strings.Contains(request.Host, ".in.codoon.com") {
				name := fmt.Sprintf("%s %s %s%s", request.URL.Scheme, request.Method, request.URL.Host, request.URL.Path)
				childSpCtx := trace.StartChildSpan(name, spCtx)
				defer childSpCtx.Finish()
			}
		}
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http_client_cluster.HttpClientClusterDo(request)
	if nil != err {
		return nil, err
	}
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if rsp != nil {
		err = json.Unmarshal(data, rsp)
	}
	return data, err
}

func HttpAPIJsonRequestWithHeader(ctx context.Context, method, addr string, req, rsp interface{}, headers map[string]string) ([]byte, error) {
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, addr, bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}

	// inject header to coloring call chain
	if ctx != nil {
		var spCtx trace.SpanContext
		var spCtxOK bool
		if c, ok := ctx.Value(GIN_CTX).(*gin.Context); ok {
			if spCtx, err = trace.ExtractContext(c.Request.Header); err == nil {
				spCtxOK = true
			}
		} else {
			spCtx, spCtxOK = ctx.Value(trace.CDTCtxKey).(trace.SpanContext)
		}
		if spCtxOK {
			trace.Inject(spCtx, request.Header)
			// if dst host is not a codoon service, start a child span
			if !strings.Contains(request.Host, ".in.codoon.com") {
				name := fmt.Sprintf("%s %s %s%s", request.URL.Scheme, request.Method, request.URL.Host, request.URL.Path)
				childSpCtx := trace.StartChildSpan(name, spCtx)
				defer childSpCtx.Finish()
			}
		}
	}

	// write custom headers
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http_client_cluster.HttpClientClusterDo(request)
	if nil != err {
		return nil, err
	}
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if rsp != nil {
		err = json.Unmarshal(data, rsp)
	}
	return data, err
}
