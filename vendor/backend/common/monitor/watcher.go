package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	gocache "third/go-cache"
	"time"
)

const (
	EventInnerHttp EventType = "inner_http"
	EventOuterHttp EventType = "outer_http"
	EventRpc       EventType = "rpc"
	EventMysql     EventType = "mysql"
	EventRedis     EventType = "redis"
	EventPg        EventType = "pg"
	EventKafka     EventType = "kafka"
)

const defautlTimeout = 1 * time.Second

var eventTimeout = map[EventType]time.Duration{
	EventInnerHttp: 1 * time.Second,
	EventOuterHttp: 2 * time.Second,
	EventRpc:       1 * time.Second,
	EventMysql:     100 * time.Millisecond,
	EventRedis:     50 * time.Millisecond,
	EventPg:        1 * time.Second,
	EventKafka:     2 * time.Second,
}

const (
	dbgOff    = -1
	dbgNormal = 0
	dbgOn     = 1
)

type EventType string

type Event struct {
	// 单位：ns
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
	// 事件类型
	Type   EventType              `json:"type"`
	DstSrv string                 `json:"dst_srv"`
	ErrMsg string                 `json:"err_msg"`
	Tags   map[string]interface{} `json:"tags,omitempty"`
}

type ServiceCounter struct {
	*gocache.Cache
}

type CountMsg struct {
	Type   EventType `json:"type"`
	DstSrv string    `json:"dst_srv"`
	Count  int       `json:"count"`
}

func (sc *ServiceCounter) run() {
	lastCounter := map[string]uint64{}
	for {
		time.Sleep(15 * time.Second)

		items := sc.Items()
		for k, v := range items {
			last := lastCounter[k]
			current := v.Object.(uint64)
			if current > last {
				etype := EventType("")
				service := k
				if pos := strings.Index(k, "|"); pos != -1 {
					etype = EventType(k[:pos])
					service = k[pos+1:]
				}
				cmsg := &CountMsg{
					Type:   etype,
					DstSrv: service,
					Count:  int(current - last),
				}
				data, _ := json.Marshal(cmsg)
				if debug > dbgOff {
					fmt.Printf("[CDMCount] %s\n", data)
				}
			}
			lastCounter[k] = current
		}
	}
}

var counter *ServiceCounter
var debug int

func init() {
	gcache := gocache.New(gocache.NoExpiration, 10*time.Minute)
	counter = &ServiceCounter{gcache}
	if s := os.Getenv("CDM_DBG"); s != "" {
		debug, _ = strconv.Atoi(s)
	}

	go counter.run()
}

func Watch(service string, etype EventType, startTime time.Time, err error, tags map[string]interface{}) {
	if service == "" {
		return
	}

	timeout, found := eventTimeout[etype]
	if !found {
		timeout = defautlTimeout
	}

	// count it
	key := fmt.Sprintf("%s|%s", etype, service)
	if _, err := counter.IncrementUint64(key, 1); err != nil {
		counter.Add(key, uint64(1), gocache.NoExpiration)
	}

	recErr := recover()
	cost := time.Since(startTime)
	if debug != dbgOn && err == nil && cost <= timeout && recErr == nil {
		return
	}

	// check
	event := &Event{
		StartTime: startTime.UnixNano(),
		EndTime:   time.Now().UnixNano(),
		Type:      etype,
		DstSrv:    service,
		Tags:      tags,
	}
	if err != nil {
		event.ErrMsg = err.Error()
	}
	if cost > timeout {
		event.ErrMsg = fmt.Sprintf("%s|timeout error, %.03f exceed %.03fs", event.ErrMsg, cost.Seconds(), timeout.Seconds())
	}
	if recErr != nil {
		event.ErrMsg = fmt.Sprintf("%s|%v", event.ErrMsg, recErr)
		// throw out panic
		defer panic(recErr)
	}

	data, _ := json.Marshal(event)
	if debug > dbgOff {
		fmt.Printf("[CDMWatch] %s\n", data)
	}
}

func WatchHttp(startTime time.Time, req *http.Request, rsp *http.Response, rspErr error) {
	service := req.URL.Host
	api := req.URL.Path
	etype := EventInnerHttp
	if !strings.Contains(service, ".in.codoon.com") {
		etype = EventOuterHttp
	}
	tags := map[string]interface{}{
		"http_api":     api,
		"http_traceid": req.Header.Get("codoon_request_id"),
	}
	if rspErr == nil && rsp != nil && rsp.StatusCode >= 400 {
		tags["http_status"] = rsp.StatusCode
	}
	Watch(service, etype, startTime, rspErr, tags)
}
