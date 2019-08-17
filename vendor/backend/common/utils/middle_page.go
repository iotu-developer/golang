package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// 为了兼容老版本的跳转链接，新增一个中间页
// 客户端用中间页的参数和自身作比较，决定后续跳转地址
// >= min_ver，拦截，把p解出来，跳过去（可能是native，可能h5）
//否则跳转到中间页（升级页面）
type JumpInfo struct {
	// 最低可见版本
	// 小于该版本的跳转到中间页，否则跳转到JumpUrl
	MinVer string `json:"min_ver"`
	// 待跳转地址
	Url string `json:"url"`
}

const (
	middlePageUrl = "https://www.codoon.com/h5/middle-page/index.html"
)
func FromatMiddlePageUrl(info JumpInfo) string {
	return formatMiddlePageUrl(middlePageUrl, info)
}

func formatMiddlePageUrl(middleUrl string, info JumpInfo) string {
	p, _ := json.Marshal(info)
	return fmt.Sprintf("%s?p=%s", middleUrl, base64.StdEncoding.EncodeToString(p))
}
