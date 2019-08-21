package util

import (
	"backend/common/clog"
	"github.com/dgrijalva/jwt-go"
	"golang/model"
	"golang/redisUtil"
	"time"
)

const SecretKey = "iotu"

//生成token
func GenerateToken(user *model.Account) (string, error) {
	jwtMap := jwt.MapClaims{
		"username": user.UserName,
		"exp":      time.Now().Add(time.Hour * 2).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtMap)
	tokenString, err := token.SignedString([]byte(SecretKey))
	return tokenString, err
}

//验证token
func CheckToken(tokenStr string) (result bool) {
	//根据盐值把tokenStr转换成token结构体
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (i interface{}, e error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		clog.Errorf("token异常 [err = %s]", err)
		return false
	}
	//拿到token结构体里的头部字段
	finToken := token.Claims.(jwt.MapClaims)
	//拿到userName 类型为interface{}
	userName := finToken["username"]
	//类型断言
	userNameValue, ok := userName.(string)
	if ok {
		temp := redisUtil.GetString(userNameValue)
		if temp == tokenStr {
			redisUtil.SetExpire(userNameValue, 3600)
			return true
		} else {
			return false
		}
	} else {
		clog.Errorf("从reids中获取UserName 类型断言失败")
		return false
	}
}
