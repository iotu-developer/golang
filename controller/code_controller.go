package controller

import (
	"backend/common/clog"
	"golang/model"

	"math/rand"
	"third/gorm"
	"time"
)

func CreatCode(userId string) (code model.Code, state int) {
	ans := CheckUserPermissions(userId)
	//检查创建User是否有权限
	if ans == false {
		return code, -1
	} else {
		ActiveCode := NewCode(userId)
		if ActiveCode == (model.Code{}) {
			return code, 0
		} else {
			return ActiveCode, 1
		}
	}
	return
}

//检查Code是否有效
func CheckCode(code string) (ans bool) {
	ActiveCode := model.Code{
		ActiveCode: code,
	}
	err := ActiveCode.FindByCode()
	//如果激活码不存在或者不存在 返回0
	if err == gorm.RecordNotFound || ActiveCode.State != 1 {
		return false
	} else {
		//如果激活码有效 返回1
		return true
	}
}

//使用激活码
func ConsumeCode(consumerId, code string) (ans bool) {
	//检查激活码是否存在
	ActiveCode := model.Code{
		ActiveCode: code,
	}
	err := ActiveCode.FindByCode()
	if err == gorm.RecordNotFound || ActiveCode.State != 1 {
		return false
	} else {
		updateMap := map[string]interface{}{
			"consume_user_id": consumerId,
			"state":           0,
			"consume_at":      time.Now(),
		}
		err = ActiveCode.UpdateStateAndConsumer(updateMap)
		if err != nil {
			clog.Errorf("使用激活码失败 [err =%s]", err)
			return false
		} else {
			return true
		}
	}
}

//检查用户有无生成激活码权限
func CheckUserPermissions(userId string) (ans bool) {

	return true
}

//向数据库生成一条激活码记录
func NewCode(userId string) (code model.Code) {
	newCode := model.Code{
		CreateUserId: userId,
		State:        1,
		CreateAt:     time.Now(),
		ConsumeAt:    time.Now(),
	}
	newCode.ActiveCode = RandString(10)
	err := newCode.Create()
	if err != nil {
		clog.Errorf("Create Code Fail [err = %s]", err)
		return
	} else {
		code = newCode
		return
	}
}

//随机生成一条激活码字符串
func RandString(len int) string {
	randNum := rand.New(rand.NewSource(time.Now().Unix()))
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := randNum.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}
