package model

import (
	"fmt"
	"web/dbutil"
)

type RotaryChart struct {
	Id          int    `json:"id" gorm:"id"`                   //ID
	PicUrl      string `json:"pic_url" gorm:"pic_url"`         //图片url地址
	ToUrl       string `json:"to_url" gorm:"to_url"`           //跳转url地址
	IsInSite    int    `json:"is_in_site" gorm:"is_in_site"`   //跳转地址是否为站内
	Title       string `json:"title" gorm:"title"`             //标题
	Description string `json:"description" gorm:"description"` //描述
	Button      string `json:"button" gorm:"button"`           //按钮
}

func (r *RotaryChart) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, RotaryChartTable)
}

func (r *RotaryChart) GetAll() (dataList []RotaryChart, err error) {
	db := dbutil.IOTUGormDb.Table(r.TableName())
	err = db.Find(&dataList).Error
	return dataList, err
}
