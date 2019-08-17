package util

import (
	"github.com/dgrijalva/jwt-go"
	"golang/model"
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
func CheckToken(token string) (state bool) {
	return true
}
