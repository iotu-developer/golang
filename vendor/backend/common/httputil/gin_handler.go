// by liudan
package httputil

import (
	"backend/common/clog"
	"backend/common/kafkautil"
	"backend/common/stringutil"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"third/gin"
	"third/sarama"
	"time"
)

// ReqData2Form try to parse request (from apiMiddleWare) body as json and inject user_id from header to body, if failed, deal with it as form.
// It should be called before your business logic.
func ReqData2Form() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Header.Get("MiddleWare") == "ON" || c.Request.Header.Get("WM") == "ON" {
			reqData2Form(c)
		}
	}
}

// ReqData2Form try to parse all request body as json and inject user_id from header to body
func AllReqData2Form() gin.HandlerFunc {
	return reqData2Form
}

func reqData2Form(c *gin.Context) {
	userId := c.Request.Header.Get(CODOON_USER_ID)
	data, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("read request body error:%v", err)
		return
	}
	// fmt.Printf("raw body:%s\n", data)
	var v map[string]interface{}
	if len(data) == 0 {
		v = make(map[string]interface{})
		err = nil
	} else {
		v, err = loadJson(bytes.NewReader(data))
	}
	if err != nil {
		// if request data is NOT json format, restore body
		// log.Printf("ReqData2Form parse as json failed. restore [%s] to body", string(data))
		c.Request.Body = ioutil.NopCloser(bytes.NewReader(data))
	} else {
		// if user_id in request is not empty, move it to req_user_id
		if uid, ok := v[CODOON_USER_ID]; ok {
			v["req_user_id"] = uid
		}
		// inject use_id into form
		v[CODOON_USER_ID] = userId
		form := map2Form(v)
		s := form.Encode()
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			c.Request.Body = ioutil.NopCloser(strings.NewReader(s))
		} else if c.Request.Method == "GET" || c.Request.Method == "DELETE" {
			c.Request.Header.Del("Content-Type")
			// append url values
			urlValues := c.Request.URL.Query()
			for k, vv := range urlValues {
				if _, ok := form[k]; !ok {
					form[k] = vv
				}
			}
			c.Request.URL.RawQuery = form.Encode()
		} else {
			c.Request.Body = ioutil.NopCloser(strings.NewReader(s))
		}
	}
}

func loadJson(r io.Reader) (map[string]interface{}, error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	var v map[string]interface{}
	err := decoder.Decode(&v)
	if err != nil {
		// log.Printf("loadJson decode error:%v", err)
		return nil, err
	}
	return v, nil
}

func map2Form(v map[string]interface{}) url.Values {
	form := url.Values{}
	var vStr string
	for key, value := range v {
		switch value.(type) {
		case string:
			vStr = value.(string)
		case float64, int, int64:
			vStr = fmt.Sprintf("%v", value)
		default:
			if b, err := json.Marshal(&value); err != nil {
				vStr = fmt.Sprintf("%v", value)
			} else {
				vStr = string(b)
			}
		}
		form.Set(key, vStr)
	}
	return form
}

//慢接口日志
type SlowLogger interface {
	Notice(format string, params ...interface{})
	Warning(format string, params ...interface{})
}

//慢接口日志
func GinSlowLogger(slog SlowLogger, threshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		startAt := time.Now()

		c.Next()

		endAt := time.Now()
		latency := endAt.Sub(startAt)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		if latency > threshold {
			slog.Warning("[GIN Slowlog] %v | %3d | %12v | %s | %-7s %s %s\n%s",
				endAt.Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				c.Request.URL.String(),
				c.Request.URL.Opaque,
				c.Errors.String())
		}
	}
}

const (
	CODOON_REQUEST_ID   = "codoon_request_id"
	CODOON_SERVICE_CODE = "codoon_service_code"
	CODOON_USER_ID      = "user_id"
	CODOON_DID          = "did"
	KAFKA_TOPIC         = "codoon-trace-log"
)

var (
	_srvCode          string
	_kafkaLoggerInput chan<- *sarama.ProducerMessage
)

//日志发送到卡夫卡  目前用于消息染色
type KafkaLogRecord struct {
	ReqId        string `json:"req_id"`
	ServiceCodes string `json:"service_codes"`
	Did          string `json:"did"`
	UserId       string `json:"user_id"`
	StartTime    string `json:"start_time"`
	SpendTime    int64  `json:"spend_time"` // ms
	Method       string `json:"method"`
	Host         string `json:"host"`
	Api          string `json:"api"`
	StatusCode   int    `json:"status_code"`
	Annotation   string `json:"annotation,omitempty"`
}

//消息染色日志
func GinSimpleKafkaLogger(srvCode string) gin.HandlerFunc {
	brokerList := []string{
		"kafka-trace-log-0.in.codoon.com:9092",
		"kafka-trace-log-1.in.codoon.com:9092",
		"kafka-trace-log-2.in.codoon.com:9092"}
	return GinKafkaLogger(srvCode, brokerList)
}

// GinKafkaLogger
func GinKafkaLogger(srvCode string, brockerList []string) gin.HandlerFunc {
	if srvCode == "" {
		panic("service code invalid")
	}
	// init producer
	config := sarama.NewConfig()
	config.Producer.Retry.Max = 1
	config.Version = sarama.V0_10_0_1
	config.Producer.RequiredAcks = sarama.NoResponse
	config.Producer.Flush.Frequency = 5 * time.Second
	producer, err := sarama.NewAsyncProducer(brockerList, config)
	if err != nil {
		log.Printf("create producer error:%v", err)
		return func(*gin.Context) {}
	}
	_srvCode = srvCode
	_kafkaLoggerInput = producer.Input()

	// monitor kafka error
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write kafka log entry:", err)
			time.Sleep(1 * time.Second)
		}
	}()

	return func(c *gin.Context) {
		start := time.Now()

		// colored with current service code
		parentCodes := c.Request.Header.Get(CODOON_SERVICE_CODE)
		if parentCodes == "" {
			c.Request.Header.Set(CODOON_SERVICE_CODE, _srvCode)
		} else {
			c.Request.Header.Set(CODOON_SERVICE_CODE, parentCodes+"-"+_srvCode)
		}

		c.Next()

		m := &KafkaLogRecord{
			ReqId:        c.Request.Header.Get(CODOON_REQUEST_ID),
			ServiceCodes: c.Request.Header.Get(CODOON_SERVICE_CODE),
			Did:          c.Request.Header.Get(CODOON_DID),
			UserId:       getHeaderUid(c),
			StartTime:    start.Format("2006-01-02 15:04:05"),
			SpendTime:    time.Since(start).Nanoseconds() / 1e6,
			Method:       c.Request.Method,
			Host:         c.Request.Host,
			Api:          c.Request.URL.Path,
			StatusCode:   c.Writer.Status(),
		}

		go func() {
			select {
			case _kafkaLoggerInput <- &sarama.ProducerMessage{
				Topic: KAFKA_TOPIC,
				Key:   sarama.StringEncoder(m.UserId),
				Value: &kafkautil.JSONEncoder{m},
			}:
			// pass
			case <-time.After(2 * time.Second):
				log.Printf("[GinKafkaLogger] write timeout [req_id:%s][user_id:%s]", m.ReqId, m.UserId)
			}
		}()
		// log.Printf("kafka msg send:%+v", m)
	}
}

func getHeaderUid(c *gin.Context) string {
	userId := c.Request.Header.Get(CODOON_USER_ID)
	if userId == "" {
		if uidSlice, ok := c.Request.Header[CODOON_USER_ID]; ok && len(uidSlice) == 1 {
			userId = uidSlice[0]
		}
	}
	return userId
}

// log extra trace info to kafka
func KafkaLogTrace(c *gin.Context, annotation string) {
	if _kafkaLoggerInput == nil {
		return
	}

	m := &KafkaLogRecord{
		ReqId:        c.Request.Header.Get(CODOON_REQUEST_ID),
		ServiceCodes: c.Request.Header.Get(CODOON_SERVICE_CODE),
		Did:          c.Request.Header.Get(CODOON_DID),
		UserId:       getHeaderUid(c),
		StartTime:    time.Now().Format("2006-01-02 15:04:05"),
		SpendTime:    0,
		Method:       c.Request.Method,
		Host:         c.Request.Host,
		Api:          c.Request.URL.Path,
		StatusCode:   c.Writer.Status(),
		Annotation:   annotation,
	}

	select {
	case _kafkaLoggerInput <- &sarama.ProducerMessage{
		Topic: KAFKA_TOPIC,
		Key:   sarama.StringEncoder(m.UserId),
		Value: &kafkautil.JSONEncoder{m},
	}:
	// pass
	case <-time.After(1 * time.Second):
		log.Printf("[KafkaLogTrace] write timeout [req_id:%s][user_id:%s]", m.ReqId, m.UserId)
	}
}

// func getKafkaLogRecord(c *gin.Context) *KafkaLogRecord {

// 	// colored with current service code
// 	if _srvCode != "" {
// 		parentCodes := c.Request.Header.Get(CODOON_SERVICE_CODE)
// 		if parentCodes != "" {
// 			c.Request.Header.Set(CODOON_SERVICE_CODE, _srvCode)
// 		} else {
// 			c.Request.Header.Set(CODOON_SERVICE_CODE, parentCodes+"-"+_srvCode)
// 		}
// 	}

// 	reqId := c.Request.Header.Get(CODOON_REQUEST_ID)
// 	srvCodeChain := c.Request.Header.Get(CODOON_SERVICE_CODE)
// 	userId := c.Request.Header.Get(CODOON_USER_ID)
// 	method := c.Request.Method
// 	host := c.Request.Host
// 	api := c.Request.RequestURI

// 	return &KafkaLogRecord{
// 		ReqId:       reqId,
// 		ServiceCode: srvCodeChain,
// 		UserId:      userId,
// 		StartTime:   start.Format("2006-01-02 15:04:05"),
// 		Method:      method,
// 		Host:        host,
// 		Api:         api,
// 		StatusCode:  c.Writer.Status(),
// 	}
// }

// gin panic-recovery
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			err := recover()
			if err != nil {
				switch err.(type) {
				case error:
					fmt.Printf("%s",err.(error))
				default:
					err := errors.New(fmt.Sprint(err))
					fmt.Printf("%s",err.(error))
				}

				stack := stack(3)
				clog.Logger.Error("PANIC: %s\n%s", err, stack)
				log.Printf("PANIC: %s\n%s", err, stack) // for maintainers

				c.Writer.WriteHeader(http.StatusInternalServerError)
			}

		}()

		c.Next()
	}
}

func MyRecovery() {

	err := recover()
	if err != nil {
		switch err.(type) {
		case error:
			fmt.Printf("%s",err.(error))
		default:
			err := errors.New(fmt.Sprint(err))
			fmt.Printf("%s",err.(error))
		}

		stack := stack(3)
		clog.Logger.Error("PANIC: %s\n%s", err, stack)
		log.Printf("PANIC: %s\n%s", err, stack) // for maintainers
	}

}

// gin请求日志
func GinLogger() gin.HandlerFunc {

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		statusColor := clog.ColorForStatus(statusCode)
		methodColor := clog.ColorForMethod(method)
		reqId := c.Request.Header.Get(CODOON_REQUEST_ID)
		userId := c.Request.Header.Get(CODOON_USER_ID)

		requestData := GetRequestData(c)

		clog.Logger.Notice("[GIN] %s%s%s %s%s %s%d%s %.03f [%s] [req_id:%s] [user_id:%s] %s",
			methodColor, method, reset,
			c.Request.Host,
			stringutil.Cuts(requestData, 2048),
			statusColor, statusCode, reset,
			latency.Seconds(),
			clientIP,
			reqId,
			userId,
			stringutil.Cuts(c.Errors.String(), 2048),
		)

	}
}

type LogExtender func(c *gin.Context) string

func GinLoggerExt(extender LogExtender) gin.HandlerFunc {

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Stop timer
		end := time.Now()
		latency := end.Sub(start)

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		statusColor := clog.ColorForStatus(statusCode)
		methodColor := clog.ColorForMethod(method)
		reqId := c.Request.Header.Get(CODOON_REQUEST_ID)
		userId := c.Request.Header.Get(CODOON_USER_ID)

		requestData := GetRequestData(c)

		clog.Logger.Notice("[GIN] %s%s%s %s%s %s%d%s %.03f [%s] [req_id:%s] [user_id:%s] %s [ext:%s]",
			methodColor, method, reset,
			c.Request.Host,
			stringutil.Cuts(requestData, 2048),
			statusColor, statusCode, reset,
			latency.Seconds(),
			clientIP,
			reqId,
			userId,
			stringutil.Cuts(c.Errors.String(), 2048),
			extender(c),
		)

	}
}

func GetRequestData(c *gin.Context) string {
	var requestData string
	method := c.Request.Method
	if method == "GET" || method == "DELETE" {
		requestData = c.Request.RequestURI
	} else {
		c.Request.ParseForm()
		requestData = fmt.Sprintf("%s [%s]", c.Request.RequestURI, c.Request.Form.Encode())
	}
	return requestData
}
