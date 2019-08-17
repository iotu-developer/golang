package http_client_cluster

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// per host to a client
	ClientMap    map[string]ConfigClient = make(map[string]ConfigClient, 1)
	ClientRWLock *sync.RWMutex           = new(sync.RWMutex)
	cfg          *HttpClusterConfig
)

type HttpClusterConfig struct {
	// A HeaderTimeoutPerRequest of zero means no timeout.
	HeaderTimeoutPerRequest time.Duration
	// `try` times for a request, should great or equal to 1
	Retry int
	// whether to allowed redirect
	Redirect bool
	// cert for request https
	Cert tls.Certificate
}

type ConfigClient struct {
	client *HttpClusterClient
	config *HttpClusterConfig
}

// the Response.Body has closed after reading into body.
func HttpClientClusterDo(request *http.Request) (*http.Response, error) {
	var resp *http.Response
	client, err := getClient(request.URL.Scheme, request.URL.Host)
	if nil != err {
		Errf("[HttpClusterClient] new http cluster client err: %v\n", err)
		return nil, err
	}

	if nil == client {
		err = fmt.Errorf("nil client")
		return nil, err
	}

	resp, err = client.do(request)
	return resp, err
}

func defaultConfig() *HttpClusterConfig {
	return &HttpClusterConfig{
		HeaderTimeoutPerRequest: DefaultRequestTimeout,
		Retry:    DefaultRetry,
		Redirect: Redirect,
	}
}

// eg SetDefaultConfig(1, 120*time.Second)
func SetDefaultConfig(defaultRetry int, defaultRequestTimeout time.Duration) error {
	DefaultRetry = defaultRetry
	DefaultRequestTimeout = defaultRequestTimeout
	return nil
}

func SetRedirect(ok bool) {
	Redirect = ok
}

func getClient(scheme, host string) (*HttpClusterClient, error) {
	ClientRWLock.RLock()
	config_client, ok := ClientMap[formatSchemeHost(scheme, host)]
	ClientRWLock.RUnlock()
	if ok {
		return config_client.client, nil
	}

	var clientCfg *HttpClusterConfig
	if nil == cfg {
		clientCfg = defaultConfig()
	} else {
		clientCfg = cfg
	}

	c, err := newHttpClusterClient(scheme, host, clientCfg)
	return c, err
}

func formatSchemeHost(scheme, host string) string {
	return fmt.Sprintf("%s://%s", scheme, host)
}

func newHttpClusterClient(scheme, host string, cfg *HttpClusterConfig) (*HttpClusterClient, error) {
	c := &HttpClusterClient{
		cfg:         cfg,
		scheme:      scheme,
		host:        host,
		port:        getHostPort(scheme, host),
		workclients: make(map[string]*httpTimeoutClient),
	}

	c.updateClientAddr()
	go func(c *HttpClusterClient) {
		timer := time.NewTicker(10 * time.Second) // 每10supdate一次
		for {
			select {
			case <-timer.C:
				c.updateClientAddr()
			}
		}
	}(c)

	ClientRWLock.Lock()
	defer ClientRWLock.Unlock()
	cofig_client := ConfigClient{
		client: c,
		config: cfg,
	}
	ClientMap[formatSchemeHost(scheme, host)] = cofig_client

	return c, nil
}

func getHostPort(scheme, host string) int {
	addr := strings.Split(host, ":")
	if len(addr) > 1 {
		port, _ := strconv.Atoi(addr[1])
		return port
	} else {
		switch scheme {
		case "http", "HTTP":
			return 80
		case "https", "HTTPS":
			return 443
		}
	}
	return 80
}

type Log interface {
	Error(format string, args ...interface{})
	Info(format string, args ...interface{})
	Notice(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

var (
	gLogger Log
	logMode bool
)

func Debugf(format string, args ...interface{}) {
	if logMode {
		gLogger.Info(format, args...)
	}
}

func Errf(format string, args ...interface{}) {
	if logMode {
		gLogger.Error(format, args...)
	} else {
		fmt.Printf(format, args...)
	}
}

func RegLog(logger Log) {
	gLogger = logger
	logMode = true

	gLogger.Info("[HttpClusterClient] RegLog")
}

var DefaultRequestTimeout = 10 * time.Second
var DefaultRetry = 1
var Redirect = true
var HttpClientWorkDuration = 1 * time.Minute
var HttpClientRetireDuration = 30 * time.Second

var (
	ErrNoEndpoints           = errors.New("client: no endpoints available")
	ErrTooManyRedirects      = errors.New("client: too many redirects")
	ErrClusterUnavailable    = errors.New("client: cluster is unavailable or misconfigured")
	ErrNoLeaderEndpoint      = errors.New("client: no leader endpoint available")
	errTooManyRedirectChecks = errors.New("client: too many redirect checks")
)
