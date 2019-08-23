package model

import (
	"fmt"
	"web/dbutil"
)

const (
	STATE_ON  = 1
	STATE_OFF = 0
)

type IotuMember struct {
	Id           int    `json:"id" gorm:"id"`
	Name         string `json:"name" gorm:"name"`
	ImgUrl       string `json:"img_url" gorm:"img_url"`
	Introduction string `json:"introduction" gorm:"introduction"`
}

//获取code的数据库名
func (i *IotuMember) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, IotuMemberTable)
}

func (i *IotuMember) GetAll(limit, offset int) (datalist []IotuMember, err error) {
	db := dbutil.IOTUGormDb.Table(i.TableName())
	offset -= 1
	err = db.Where("state = ?", STATE_ON).Limit(limit).Offset(offset).Find(&datalist).Error
	return datalist, err
}
