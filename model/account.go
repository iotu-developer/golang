package model

import (
	"fmt"
	"golang/dbutil"
	"third/gorm"
	"time"
)

type Account struct {
	Uid           int       `gorm:"uid"`
	UserName      string    `gorm:"user_name"`
	NickName      string    `gorm:"nick_name"`
	Password      string    `gorm:"password"`
	Major         string    `gorm:"major"`
	College       string    `gorm:"college"`
	ImagePath     string    `gorm:"image_path"`
	RegisterTime  time.Time `gorm:"register_time"`
	LastLoginTime time.Time `gorm:"last_login_time"`
	LoginCount    int       `gorm:"login_count"`
}

func (a *Account) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, AccountTable)
}

func (a *Account) Create() (err error) {
	db := dbutil.IOTUGormDb.Table(a.TableName())
	return db.Create(a).Error
}

func (a *Account) GetByUserName() (state bool, err error) {
	db := dbutil.IOTUGormDb.Table(a.TableName())
	err = db.Where("user_name = ?", a.UserName).Find(&a).Error
	if err == gorm.RecordNotFound {
		state = false
	} else {
		state = true
	}
	return
}
