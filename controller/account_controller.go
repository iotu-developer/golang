package controller

import (
	"backend/common/clog"
	"errors"
	"golang/basic"
	"golang/model"
	"golang/redisUtil"
	"golang/util"
	"time"
)

func AccountRegister(req basic.AccountRegisterReq) (err error) {
	//检查激活码
	if checkResult := CheckCode(req.ActiveCode); !checkResult {
		clog.Errorf("激活码无效")
		return errors.New("激活码无效")
	}
	//创建账号
	account := model.Account{
		UserName: req.UserName,
	}
	isAccount, err := account.GetByUserName()
	if isAccount {
		clog.Errorf("该用户名已经存在 [UserName = %s]", req.UserName)
		return errors.New("该用户名已经存在")
	}
	//消耗掉激活码
	if result := ConsumeCode(req.UserName, req.ActiveCode); !result {
		clog.Errorf("消费激活码失败")
		return errors.New("消费激活码失败")
	}
	account.NickName = req.NickName
	account.Password = req.Password
	account.College = req.College
	account.Major = req.Major
	account.RegisterTime = time.Now()
	account.LastLoginTime = time.Now()
	account.LoginCount = 0
	err = account.Create()
	if err != nil {
		clog.Errorf("创建用户失败 [UserName = %s]", req.UserName)
		return err
	}
	return
}

func AccountLogin(req basic.AccountLoginReq) (token string, err error) {
	account := model.Account{
		UserName: req.UserName,
	}
	_, err = account.GetByUserName()
	if err != nil {
		clog.Errorf("无此用户")
		return "", errors.New("无此用户")
	}
	if account.Password != req.Password {
		clog.Errorf("密码错误")
		return "", errors.New("密码错误")
	} else {
		//生成token
		if token, err = util.GenerateToken(&account); err != nil {
			clog.Errorf("token生成失败")
			return "", errors.New("token生成失败")
		} else {
			IsSetRedisOk := redisUtil.SetString(account.UserName, token)
			if IsSetRedisOk {
				IsSetExpireOk := redisUtil.SetExpire(account.UserName, 3600)
				if IsSetExpireOk {
					return token, err
				} else {
					return "", errors.New("设置超时错误")
				}
			} else {
				return "", errors.New("设置redis错误")
			}
		}
	}
}
