package http_client_cluster

import (
	"backend/common/monitor"
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"third/golang.org/x/net/http2"
	"third/hystrix"
	"time"
)

/************************************************************************
 带集群功能的client
************************************************************************/
type HttpClusterClient struct {
	sync.RWMutex
	cfg           *HttpClusterConfig
	scheme        string
	host          string // domain:port
	port          int
	cltcnt        int                           // 建立连接的客户端数量，等于len(workclients)，用于快速计算，只被updateClientAddr函数更新
	endpoints     []string                      // ip列表，与workclients保持一致，用于快速计算，只被updateClientAddr函数更新
	workclients   map[string]*httpTimeoutClient // httpTimeoutClient per ip，work for HttpClientLife
	retireclients []*httpTimeoutClient          // httpTimeoutClient，等待退休，等待IdleConnTimeout。防止回收该httpclient时还有正在使用的连接
}

/*********************************************************************************
只使用一段时间的httpclient封装，目的是在使用一段时间后，进行http重连，解决k8s内网均衡问题
**********************************************************************************/
type httpTimeoutClient struct {
	client    *http.Client
	starttime time.Time
	endpoint  string // "127.0.0.1"
	port      int
	usecnt    int64
	errcnt    int32
}

func (c *HttpClusterClient) do(request *http.Request) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
	)

	service := request.URL.Host
	starttime := time.Now()
	if circuitEnabled(service) {
		err = hystrix.Do(service, func() error {
			resp, err = c.doRequest(request)
			return err
		}, nil)
	} else {
		resp, err = c.doRequest(request)
	}

	monitor.WatchHttp(starttime, request, resp, err)
	return resp, err
}

func (c *HttpClusterClient) doRequest(request *http.Request) (*http.Response, error) {
	var err error
	var retry int
	cerr := &ClusterError{}
	var resp *http.Response

	// hold body content
	var bodyBytes []byte
	if request.Body != nil {
		bodyBytes, err = ioutil.ReadAll(request.Body)
		if err != nil {
			Errf("[HttpClusterClient] read request body error: %s for %s\n", err.Error(), c.host)
			return nil, err
		}
	}

	client := c.get()
	if client == nil {
		c.updateClientAddr()
		client = c.get()
		if client == nil {
			return nil, fmt.Errorf("[HttpClusterClient] nil client for: %s", c.host)
		}
	}

	for retry = 0; retry < c.cfg.Retry; retry++ {
		if bodyBytes != nil {
			request.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		}

		c.use(client)
		resp, err = client.client.Do(request)

		if nil != err {
			c.occurErr(client)
			cerr.Errors = append(cerr.Errors, err)
			Errf("[HttpClusterClient] do %s/%s err: %s\n", client.endpoint, c.host, err.Error())
			continue
		}

		return resp, err
	}

	if retry >= c.cfg.Retry && cerr.Errors != nil {
		Errf("[HttpClusterClient] cluster call %s/%s failed after %d times\n", client.endpoint, c.host, c.cfg.Retry)
	}

	if nil == resp && nil == err {
		err = fmt.Errorf("[HttpClusterClient] Unknown Err for %s/%s", client.endpoint, c.host)
	}

	return resp, err
}

func (c *HttpClusterClient) get() *httpTimeoutClient {
	c.RLock()
	defer c.RUnlock()

	if c.cltcnt == 0 {
		return nil
	}

	// 从workclients中随机取一个，每30s会更新一次列表
	ip := c.endpoints[rand.Intn(c.cltcnt)]
	return c.workclients[ip]
}

func (c *HttpClusterClient) updateClientAddr() {
	ips := resolveIPv4ListFromHost(c.host)

	c.Lock()
	defer c.Unlock()

	//先让该退休的client退休，让出位置给新client
	c.retire()

	// 处理愉快的度过了退休时光的client
	c.die()

	disappearips := minus(c.endpoints, ips)
	newips := minus(ips, c.endpoints)

	// 处理新ip，填加到工作队列中
	c.addNewClients(newips)

	// 处理不再存在的IP, 移动到退休队列中
	for _, ip := range disappearips {
		if cli, ok := c.workclients[ip]; ok {
			if tr, ok := cli.client.Transport.(*http.Transport); ok {
				tr.DisableKeepAlives = true
				tr.IdleConnTimeout = DefaultRequestTimeout
			}

			cli.starttime = time.Now()
			c.retireclients = append(c.retireclients, cli)
			delete(c.workclients, ip)
		}
	}

	c.cltcnt = len(c.workclients)
	c.endpoints = c.endpoints[0:0]
	for ip, _ := range c.workclients {
		c.endpoints = append(c.endpoints, ip)
	}

	if c.cltcnt == 0 {
		Errf("[HttpClusterClient] cluster has no client to use\n")
	}

	//统计打印
	Debugf("[HttpClusterClient] statistics(%s/%p) -- retireLen: %d, workers: %d, endpoints: %d, cltcnt: %d", c.host, c, len(c.retireclients), len(c.workclients), len(c.endpoints), c.cltcnt)
	for k, v := range c.workclients {
		Debugf("[HttpClusterClient] clients stats. client(%s/%s/%p): %d %d %f/%d", k, c.host, v, v.usecnt, v.errcnt, time.Since(v.starttime).Seconds(), HttpClientWorkDuration/time.Second)
	}
}

// 调用者需要写锁
func (c *HttpClusterClient) addNewClients(newips []string) {
	addr := strings.Split(c.host, ":")
	for _, ip := range newips {
		client := newHTTPClient(fmt.Sprintf("%s:%d", addr[0], c.port), ip, c.port,
			c.cfg.HeaderTimeoutPerRequest, c.cfg.Redirect, c.cfg.Cert)

		httpTimeoutClient := &httpTimeoutClient{
			client:    client,
			starttime: time.Now(),
			endpoint:  ip,
		}

		Debugf("[HttpClusterClient] new client: %s, %s, %p", addr, ip, httpTimeoutClient)

		if _, ok := c.workclients[ip]; !ok {
			c.workclients[ip] = httpTimeoutClient
		}
	}
}

// 调用者需要写锁
func (c *HttpClusterClient) retire() {
	// 遍历工作队列，让该退休的client退休，如果在update期间，调用错误率大于50%，则提前让该client退休
	for ip, cli := range c.workclients {
		isTimeout := time.Since(cli.starttime) >= HttpClientWorkDuration
		errProportion := float32(cli.errcnt) / float32(cli.usecnt)
		if isTimeout || ((cli.usecnt > 20) && (errProportion > 0.5)) {
			if tr, ok := cli.client.Transport.(*http.Transport); ok {
				tr.DisableKeepAlives = true
				tr.IdleConnTimeout = DefaultRequestTimeout
			}

			cli.starttime = time.Now()
			c.retireclients = append(c.retireclients, cli)
			delete(c.workclients, ip)

			if isTimeout {
				Debugf("[HttpClusterClient] client retire for timeout: %s/%s/%p", cli.endpoint, c.host, cli)
			}

			if (cli.usecnt > 20) && (errProportion > 0.5) {
				Debugf("[HttpClusterClient] client retire for errProportion: %s/%s/%p, %f/0.5", cli.endpoint, c.host, cli, errProportion)
			}
		} else {
			cli.usecnt = 0
			cli.errcnt = 0
		}
	}

	c.cltcnt = len(c.workclients)
	c.endpoints = c.endpoints[0:0]
	for ip, _ := range c.workclients {
		c.endpoints = append(c.endpoints, ip)
	}
}

// 调用者需要写锁
func (c *HttpClusterClient) die() {
	var newretireclient []*httpTimeoutClient
	for _, cli := range c.retireclients {
		if time.Since(cli.starttime) > HttpClientRetireDuration {
			if tr, ok := cli.client.Transport.(*http.Transport); ok {
				tr.CloseIdleConnections()
			}
			// 因为golang http没有全局CancelRquest的接口，也没有主动关闭连接的接口，所以不能显示的关闭所有连接。
			// 我们的做法的让client工作一段时间后进入退休队列，退休队列里面的client不接受请求，且关闭心跳，修改超时时长，退休时长远远超过一次request的请求超时时长（10s）
			// 等到退休结束时，我们认为在该client下的所有请求都已经完成（真实完成或者超时），因此这个时候再调用CloseIdleConnections，应该可以回收所有连接，所以我们姑且可以认为可以丢弃该client
			Debugf("[HttpClusterClient] client die. client(%s/%s/%p) time: %s\n", cli.endpoint, c.host, cli, cli.starttime.Format("15:04:05.000"))
		} else {
			newretireclient = append(newretireclient, cli)
		}
	}

	c.retireclients = newretireclient
}

func (c *HttpClusterClient) use(client *httpTimeoutClient) {
	atomic.AddInt64(&client.usecnt, 1)
}

func (c *HttpClusterClient) occurErr(client *httpTimeoutClient) {
	atomic.AddInt32(&client.errcnt, 1)
}

// 求list1和list2的差集，即list1 - list2
func minus(list1 []string, list2 []string) []string {
	var result []string
	for _, e1 := range list1 {
		var exsit bool
		for _, e2 := range list2 {
			if e1 == e2 {
				exsit = true
				break
			}
		}

		if !exsit {
			result = append(result, e1)
		}
	}

	return result
}

type ClusterError struct {
	Errors []error
}

func (ce *ClusterError) Error() string {
	return ErrClusterUnavailable.Error()
}

func (ce *ClusterError) Detail() string {
	s := ""
	for i, e := range ce.Errors {
		s += fmt.Sprintf("error #%d: %s\n", i, e)
	}
	return s
}

// addr demo: authcenter.in.codoon.com:2014
func newHTTPClient(addr string, ip string, port int, headerTimeout time.Duration, redirect bool, cert tls.Certificate) *http.Client {
	dial := func(network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout:   headerTimeout,
			KeepAlive: 75 * time.Second,
		}

		if addr == address {
			// Debugf("[HttpClusterClient] dial ip: %s %d", ip, port)
			return d.Dial(network, fmt.Sprintf("%s:%d", ip, port))
		} else {
			Debugf("[HttpClusterClient] dial addr: %s", address)
			return d.Dial(network, address)
		}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	if len(cert.Certificate) > 0 {
		tlsConfig.BuildNameToCertificate()
	}

	var tr = &http.Transport{
		Dial:                  dial,
		TLSHandshakeTimeout:   headerTimeout,
		ResponseHeaderTimeout: headerTimeout,
		TLSClientConfig:       tlsConfig,
		MaxIdleConns:          100,
		IdleConnTimeout:       75 * time.Second,
		MaxIdleConnsPerHost:   30,
		DisableKeepAlives:     false,
	}

	err := http2.ConfigureTransport(tr)
	if err != nil {
		Errf("[HttpClusterClient] http2 ConfigureTransport err: %s\n", err.Error())
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   headerTimeout,
	}

	if !redirect {
		client.CheckRedirect = noCheckRedirect
	}

	return client
}

// 控制跳转次数为1次
func noCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 1 {
		Errf("[HttpClusterClient] check redirect %s %d\n", http.ErrUseLastResponse.Error(), len(via))
		return http.ErrUseLastResponse
	}
	return nil
}

func resolveIPv4ListFromHost(host string) []string {
	var ips []string
	addr := strings.Split(host, ":")
	addrs, err := net.LookupHost(addr[0])
	if nil != err {
		Errf("[HttpClusterClient] lookup host err: %s %s\n", host, err.Error())
		return ips
	}

	// only ipv4
	for _, s := range addrs {
		ip := net.ParseIP(s)
		if ip != nil && len(ip.To4()) == net.IPv4len {
			ips = append(ips, s)
		}
	}

	return ips
}
