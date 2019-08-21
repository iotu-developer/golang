package model

import (
	"fmt"
	"golang/dbutil"
)

type ColumnData struct {
	Id          int    `json:"id" gorm:"id"`
	Title       string `json:"title" gorm:"title"`
	Description string `json:"description" gorm:"description"`
	ButtonText  string `json:"button_text" gorm:"button_text"`
	PicUrl      string `json:"pic_url" gorm:"pic_url"`
	MainColor   string `json:"main_color" gorm:"main_color"`
	ToUrl       string `json:"to_url" gorm:"to_url"`
	IsInSite    int    `json:"is_in_site" gorm:"is_in_site"`
}

//获取code的数据库名
func (c *ColumnData) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, ColumnDataTable)
}

func (c *ColumnData) GetAll(limit, offset int) (datalist []ColumnData, err error) {
	db := dbutil.IOTUGormDb.Table(c.TableName())
	offset -= 1
	err = db.Where("state = ?", STATE_ON).Limit(limit).Offset(offset).Find(&datalist).Error
	return datalist, err
}
