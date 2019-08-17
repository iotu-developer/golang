package httputil

import (
	"backend/common/config"
	"log"
	"net/url"
	"strings"
)

const (
	DT_HTTPS = iota
	DT_HTTP
)

var domainMap map[string]string

func init() {
	addrs := []string{"http://etcdproxy.in.codoon.com:2381"}
	domainMap = map[string]string{}

	err := config.LoadCfgFromEtcd(addrs, "domain_map", &domainMap)
	if err != nil {
		log.Printf("log domain_map from etcd failed:%v", err)
	}

}

func GetHttpsUrl(oriUrl string) string {
	addr, err := url.Parse(oriUrl)
	if err != nil {
		log.Printf("parse [%s] failed:%v", err)
		return oriUrl
	}

	oldHost := strings.ToLower(addr.Host)
	if newHost, found := domainMap[oldHost]; found {
		addr.Scheme = "https"
		addr.Host = newHost
		return addr.String()
	}

	return oriUrl
}

// deprecated
func GetHttpUrl(oriUrl string) string {
	addr, err := url.Parse(oriUrl)
	if err != nil {
		log.Printf("parse [%s] failed:%v", err)
		return oriUrl
	}

	if strings.ToLower(addr.Scheme) == "http" {
		return oriUrl
	}

	oldHost := strings.ToLower(addr.Host)
	if _, found := getMappedDomain(oldHost, DT_HTTP); found {
		addr.Scheme = "http"
		// addr.Host = newHost // do not change host
		return addr.String()
	} else {
		return oriUrl
	}
}

// deprecated
func getMappedDomain(domain string, dtype int) (newDomain string, found bool) {
	http2httpsMap := map[string]string{
		"img3.codoon.com":                      "img3.codoon.com",
		"imagead.codoon.com":                   "imageads.codoon.com",
		"pili-media.codoon-live-ta.codoon.com": "live-ta.codoon.com",
		"pili-media.live.codoon.com":           "live-live.codoon.com",
		"codoon-teiba.codoon.com":              "codoon-teiba.codoon.com",
		"image7.codoon.com":                    "image7s.codoon.com",
		"archive.codoon.com":                   "archive.codoon.com",
	}

	domain = strings.ToLower(domain)

	if dtype == DT_HTTPS {
		newDomain, found = http2httpsMap[domain]
		return
	}

	for h, hs := range http2httpsMap {
		if hs == domain {
			return h, true
		}
	}

	return domain, false
}
