package cryptoutil

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
)

const (
	CODOON_REQUEST_ID   = "codoon_request_id"
	CODOON_SERVICE_CODE = "codoon_service_code"
	CODOON_CLIENT_IP    = "codoon_client_ip"
)

var Debug bool

type HttpSigner struct {
	requiredHeader []string
	excludeHeader  []string
}

func NewHttpSigner(requiredHeader, excludeHeader []string) *HttpSigner {
	return &HttpSigner{
		requiredHeader: requiredHeader,
		excludeHeader:  excludeHeader,
	}
}

// doc: https://yiqixie.com/d/home/fcACqtx6MTjSAi_pksuUfcNNo#
func (hs *HttpSigner) CalculateSignature(req *http.Request, key []byte) (string, error) {
	sep := []byte{'|'}
	mac := hmac.New(sha1.New, key)

	header := req.Header
	values := url.Values{}
	headerFields := hs.requiredHeader
	for _, field := range headerFields {
		if s := header.Get(field); s == "" {
			return "", fmt.Errorf("header field: %s is empty", field)
		} else {
			values.Set(field, s)
		}
	}
	s1 := EncodeWithoutEscape(values)
	mac.Write([]byte(s1))

	s2 := "path=" + req.URL.Path
	mac.Write(sep)
	mac.Write([]byte(s2))

	mac.Write(sep)
	mac.Write([]byte("body="))
	var data []byte
	var err error
	if req.Body != nil {
		lr := io.LimitReader(req.Body, 256*1024)
		data, err = ioutil.ReadAll(lr)
		if err != nil {
			return "", fmt.Errorf("read body failed:%v", err)
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(data))
		mac.Write(data)
	}

	excludeFieds := hs.excludeHeader
	values = req.URL.Query()
	for _, field := range excludeFieds {
		values.Del(field)
	}
	s4 := EncodeWithoutEscape(values)
	mac.Write(sep)
	mac.Write([]byte(s4))

	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if Debug {
		reqId := req.Header.Get(CODOON_REQUEST_ID)
		log.Printf("[req_id:%s][s1:%s]", reqId, s1)
		log.Printf("[req_id:%s][s2:%s]", reqId, s2)
		log.Printf("[req_id:%s][s3:%s]", reqId, "body="+string(data))
		log.Printf("[req_id:%s][s4:%s]", reqId, s4)
		log.Printf("[req_id:%s][sign:%s]", reqId, sign)
	}
	return sign, nil
}

func EncodeWithoutEscape(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := k + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(v)
		}
	}
	return buf.String()
}
