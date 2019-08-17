// by liudanking 2016.06

package cryptoutil

import (
	"backend/common/clog"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// I know it is stupid hard-coding a const key in code.
// We want to introcude hmac into codoon asap and just fire it up.
const _hmac_key = "bes3a3ZnfHzttfkWAUGfxzXPutuRQgUE"

// _hmac_internale_api_token should only be set during init
var _hmac_internale_api_token string

func init() {
	_hmac_internale_api_token = GenMAC("8NK8wjZfJLXtWNUtETPxptNGxcRPFjQw")
}

func GenMAC(msg string) string {
	macF := hmac.New(sha256.New, []byte(_hmac_key))
	macF.Write([]byte(msg))
	mac := macF.Sum(nil)
	return hex.EncodeToString(mac)
}

func VerifyMAC(msg, signature string) bool {
	mac := GenMAC(msg)
	return mac == signature
}

func GenInternalAPIToken() string {
	return _hmac_internale_api_token
}

func VerifyInternalAPIToken(token string) bool {
	return token == _hmac_internale_api_token
}

//URL encode后mac hash
func SignUrlValue(vm map[string]string) string {
	values := url.Values{}
	for k, v := range vm {
		values.Set(k, v)
	}
	s := values.Encode()
	log.Printf("url encoded:%s", s)
	return GenMAC(s)
}

func VerifyUrlValueSignature(signature string, vm map[string]string) bool {
	return signature == SignUrlValue(vm)
}

//生成随机数组成的字符串
func RandNumStr(n int) string {
	dict := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	dictLen := len(dict)
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = dict[rand.Intn(dictLen)]
	}
	return string(b)
}

//生成随机数组成的字符串, 严格模式，非0开头
func RandNumStrStrict(n int) string {
	dict := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	dictLen := len(dict)
	b := make([]byte, n)
	b[0] = dict[1+rand.Intn(dictLen-1)]
	for i := 1; i < n; i++ {
		b[i] = dict[rand.Intn(dictLen)]
	}
	return string(b)
}

//生成随机字符串 数字 字母大小写
func RandString(n int) string {
	dict := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	}
	dictLen := len(dict)
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = dict[rand.Intn(dictLen)]
	}
	return string(b)
}

//id混淆 maybe by liaoqiang
type DecimalismConfusion struct {
	Move    int64
	StartId int64
}

func InitDecimalismConfusion(move, startId int64) (*DecimalismConfusion, error) {
	if move <= 0 {
		return nil, errors.New("move error")
	}

	d := DecimalismConfusion{
		Move:    1,
		StartId: startId,
	}

	var i int64 = 0
	for ; i < move; i++ {
		d.Move = d.Move * 10
	}

	return &d, nil
}

func (d *DecimalismConfusion) sign(id int64) int64 {
	var signId int64

	stringId := strconv.FormatInt(id, 10)
	for i := 0; i < len(stringId); i++ {
		k := id / 10
		if k == 0 {
			signId = signId + id
			break
		}
		ii := id - k*10
		signId = signId + ii
		id = k
	}

	return signId % d.Move
}

//id编码 用于返回接口时混淆id
func (d *DecimalismConfusion) EncodeId(id int64) int64 {
	if id < d.StartId {
		return id
	}

	encodeId := id * d.Move
	encodeId = encodeId + d.sign(id)
	return encodeId
}

//id解编码 用于收到接口时还原id
func (d *DecimalismConfusion) DecodeId(id int64) (int64, error) {
	if id < d.StartId {
		return id, nil
	}

	var decodeId int64
	decodeId = id / d.Move
	signId := id - decodeId*d.Move
	if signId != d.sign(decodeId) {
		fmt.Println(decodeId, signId, d.sign(decodeId))
		return id, errors.New("decode error")
	}

	return decodeId, nil
}

//进行zlib压缩
func DoZlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

//进行zlib解压缩
func DoZlibUnCompress(compressSrc []byte) ([]byte, error) {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, err := zlib.NewReader(b)
	if nil != err || r == nil {
		clog.Errorf("DoZlibUnCompress error :%v", err)
		return compressSrc, err
	}
	io.Copy(&out, r)
	return out.Bytes(), nil
}

//进行gzip压缩
func DoGzipCompress(src []byte) []byte {
	var in bytes.Buffer
	w := gzip.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

//进行gzip解压缩
func DoGzipUnCompress(compressSrc []byte) ([]byte, error) {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, err := gzip.NewReader(b)
	if nil != err || r == nil {
		clog.Errorf("DoGzipUnCompress error :%v", err)
		return compressSrc, err
	}
	io.Copy(&out, r)
	return out.Bytes(), nil
}

// addd by wuql 2016-8-26
// get rand num range [min, max]
func RandIntRange(min, max int) int {
	if min == max {
		return min
	}
	// incompatible when min and max reversed
	if min > max {
		min, max = max, min
	}
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min+1)
}

type CbcCrypter struct {
	iv_key    string
	block     cipher.Block
	init_lock sync.RWMutex
}

// padding & unpadding
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// 此函数不保证并发安全,需要在启动时初始化
func (this *CbcCrypter) Init(iv string) error {
	if iv == "" {
		return errors.New("参数错误")
	}
	this.init_lock.Lock()
	defer this.init_lock.Unlock()
	this.iv_key = iv
	var err error
	this.block, err = aes.NewCipher([]byte(iv))
	if err != nil {
		return err
	}

	return nil
}

/*
 * author:	liujun
 * brief:	加密函数
 * update:	2016-09-07 12:08
 */
func (this *CbcCrypter) EncodeData(input []byte) ([]byte, error) {
	this.init_lock.RLock()
	defer this.init_lock.RUnlock()

	if this.iv_key == "" {
		return nil, errors.New("未初始化")
	}

	enc := cipher.NewCBCEncrypter(this.block, []byte(this.iv_key))

	plain := PKCS5Padding(input, this.block.BlockSize())
	cipher := make([]byte, len(plain))
	enc.CryptBlocks(cipher, plain)
	return []byte(base64.StdEncoding.EncodeToString(cipher)), nil
}

/*
 * author:	liujun
 * brief:	解密函数
 * update:	2016-09-07 12:09
 */
func (this *CbcCrypter) DecodeData(data []byte) ([]byte, error) {
	this.init_lock.RLock()
	defer this.init_lock.RUnlock()
	if this.iv_key == "" {
		return nil, errors.New("未初始化")
	}

	dec := cipher.NewCBCDecrypter(this.block, []byte(this.iv_key))

	decode_content, base64_err := base64.StdEncoding.DecodeString(string(data))
	if base64_err != nil {
		return nil, base64_err
	}
	if (len(decode_content) % this.block.BlockSize()) != 0 {
		return nil, errors.New("错误的数据块大小")
	}
	after_text := make([]byte, len(decode_content))
	dec.CryptBlocks(after_text, decode_content)
	ret := PKCS5UnPadding(after_text)

	return ret, nil
}


//////////////////// by tp for aes + cbc + pkcs7 ////////////////////
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext) % blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}
//////////////////// by tp for aes + cbc + pkcs7 ////////////////////
