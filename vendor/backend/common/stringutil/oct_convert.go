package stringutil

import "fmt"

func Char2Oct(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'z':
		return c - 'a' + 10
	case 'A' <= c && c <= 'Z':
		return c - 'A' + 10
	default:
		return 0
	}
}

func Oct2Char(o byte) string {
	return fmt.Sprintf("%02x", o)
}


// 将16进制字符串转为16进制码流
func Str2Oct(s string) []byte {
	l := len(s)
	if 0 != l % 2 {
		return nil
	}
	ret := make([]byte, 0)

	for i := 0; i < l; i += 2 {
		d := (Char2Oct(s[i]) << 4) + Char2Oct(s[i+1])
		ret = append(ret, d)
	}

	return ret
}

// 将16进制码流转为16进制字符串
func Oct2Str(b []byte) string {
	ret := ""
	for i := 0; i < len(b); i++ {
		ret = ret + Oct2Char(b[i])
	}

	return ret
}
