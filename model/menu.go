package model

import (
	"fmt"
	"time"
	"web/dbutil"
)

type Menu struct {
	Id         int       `gorm:"id" json:"id"`
	NameCn     string    `gorm:"name_cn" json:"name_cn"`
	NameEn     string    `gorm:"name_en" json:"name_en"`
	Classify   int       `gorm:"classify" json:"classify"`
	CreateTime time.Time `gorm:"create_time" json:"create_time"`
	UpdateTime time.Time `gorm:"update_time" json:"update_time"`
}

func (m *Menu) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, MenuTable)
}

func (m *Menu) GetByClassify(classify int) (menus []Menu, err error) {
	db := dbutil.IOTUGormDb.Table(m.TableName())
	err = db.Where("classify = ?", classify).Find(&menus).Error
	return
}
