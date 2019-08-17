package httputil

import (
	"backend/common/clog"
	"backend/common/stringutil"
	"fmt"
	"net/http"
	"reflect"
	"third/gin"
)

const (
	STATUS_OK        = "OK"
	STATUS_ERR       = "Error"
	MAX_LOG_DATA_LEN = 2048
)

const (
	ErrCodeParamInvalid = 2001
	ErrCodeGeneralFail  = 2002
)

type CodoonStdRsp struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
	Desc   interface{} `json:"description"`
	EData  *ErrData    `json:"err_data,omitempty"`
}

type ErrData struct {
	Code int `json:"code"`
}

func NewErrRsp(data, desc interface{}) *CodoonStdRsp {
	errCode := ErrCodeGeneralFail
	switch code := data.(type) {
	case int:
		errCode = code
	case int64:
		errCode = int(code)
	}
	return &CodoonStdRsp{
		Status: STATUS_ERR,
		Data:   data,
		Desc:   desc,
		EData:  &ErrData{Code: errCode},
	}
}

func GinErrRsp(c *gin.Context, errCode interface{}, errMsg interface{}) {
	GinRsp(c, http.StatusOK, NewErrRsp(errCode, errMsg))
}

func GinOKRsp(c *gin.Context, data interface{}, desc interface{}) {
	GinRsp(c, http.StatusOK, gin.H{"status": STATUS_OK, "data": data, "description": desc})
}

func GinSimpleOKRsp(c *gin.Context) {
	GinRsp(c, http.StatusOK, gin.H{"status": STATUS_OK, "data": "", "description": ""})
}

func GinRsp(c *gin.Context, statusCode int, obj interface{}) {
	requestData := GetRequestData(c)
	objData := fmt.Sprintf("%+v", obj)

	clientIP := c.ClientIP()
	method := c.Request.Method
	statusColor := clog.ColorForStatus(statusCode)
	methodColor := clog.ColorForMethod(method)
	resetColor := clog.ColorForReset()
	reqId := c.Request.Header.Get("codoon_request_id")
	userId := c.Request.Header.Get("user_id")
	clog.Noticef("[GIN-RSP] %s%s%s %s%d%s %s [ip:%s] [req_id:%s] [user_id:%s] [rsp:%s]",
		methodColor, method, resetColor,
		statusColor, statusCode, resetColor,
		stringutil.Cuts(requestData, MAX_LOG_DATA_LEN),
		clientIP,
		reqId,
		userId,
		stringutil.Cuts(objData, MAX_LOG_DATA_LEN),
	)

	c.JSON(statusCode, obj)
}

// WrapHandler wrap gin handler, the handler signature should be
// func(req *ReqStruct) (*OKRspStruct, *ErrorRspStruct) or
// func(c *gin.Context, req *ReqStruct) (*OKRspStruct, *ErrorRspStruct)
func WrapHandler(f interface{}) gin.HandlerFunc {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		panic("handdler should be function type")
	}

	fnumIn := t.NumIn()
	if fnumIn == 0 || fnumIn > 2 {
		panic("handler function require 1 or 2 input parameters")
	}

	if fnumIn == 2 {
		tc := reflect.TypeOf(&gin.Context{})
		if t.In(0) != tc {
			panic("handler function first paramter should by type of *gin.Context if you have 2 input parameters")
		}
	}

	if t.NumOut() != 2 {
		panic("handler function return values should contain response data & error")
	}

	// errorInterface := reflect.TypeOf((*error)(nil)).Elem()
	// if !t.Out(1).Implements(errorInterface) {
	// 	panic("handler function second return value should by type of error")

	// }

	return func(c *gin.Context) {
		var req interface{}
		if fnumIn == 1 {
			req = newReqInstance(t.In(0))
		} else {
			req = newReqInstance(t.In(1))
		}

		if !c.Bind(req) {
			err := c.LastError()
			clog.Warnf("bind parameter error:%v", err)
			GinErrRsp(c, ErrCodeParamInvalid, err.Error())
			return
		}

		var inValues []reflect.Value
		if fnumIn == 1 {
			inValues = []reflect.Value{reflect.ValueOf(req)}
		} else {
			inValues = []reflect.Value{reflect.ValueOf(c), reflect.ValueOf(req)}
		}

		ret := reflect.ValueOf(f).Call(inValues)
		if ret[1].IsNil() {
			GinRsp(c, http.StatusOK, ret[0].Interface())
		} else {
			GinRsp(c, http.StatusOK, ret[1].Interface())
		}
	}
}

func newReqInstance(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.Ptr, reflect.Interface:
		return newReqInstance(t.Elem())
	default:
		return reflect.New(t).Interface()
	}
}
