// by liudanking 2017.01

package cryptoutil

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var _verifyCodeSalts []int64 = []int64{915242, 421434, 444186, 1039169, 1074116, 789131, 420241, 277814, 1063115, 108299, 996009, 545435, 353360, 779209, 296199, 599991, 279730, 740932, 338413, 938707}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenMobileVerifyCode generates verify length code for mobile
func GenMobileVerifyCode(mobile string, length int) string {
	mobileI64 := mobile2Int(mobile)
	tsMinute := time.Now().Unix() / 60
	salt := _verifyCodeSalts[rand.Intn(len(_verifyCodeSalts))]
	code := (mobileI64 + tsMinute + salt) % int64(math.Pow10(length))
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, code)
}

// CheckMobileVerifyCode checks code validation within 10 minutes.
// This mechanism is not strict, only should be used when dependency service is down.
func CheckMobileVerifyCode(mobile string, code string) bool {
	mobileI64 := mobile2Int(mobile)
	codeI64, err := strconv.ParseInt(code, 10, 64)
	if err != nil {
		return false
	}
	length := len(code)
	tsMinute := time.Now().Unix() / 60
	tsMinuteBottom := tsMinute - 10
	tsMinuteUpper := tsMinute + 2
	for i := tsMinuteBottom; i <= tsMinuteUpper; i++ {
		for j := 0; j < len(_verifyCodeSalts); j++ {
			calculatedCode := (mobileI64 + i + _verifyCodeSalts[j]) % int64(math.Pow10(length))
			if codeI64 == calculatedCode {
				return true
			}
		}
	}
	return false
}

func mobile2Int(mobile string) int64 {
	mobile = strings.TrimPrefix(mobile, "+-")
	mobileI64, _ := strconv.ParseInt(mobile, 10, 64)
	return mobileI64
}
