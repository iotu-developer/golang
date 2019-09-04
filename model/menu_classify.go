package model

import (
	"fmt"
	"time"
	"web/dbutil"
)

type MenuClassify struct {
	Id          int       `gorm:"id" json:"id"`
	NameCn      string    `gorm:"name_cn" json:"name_cn"`
	Icon        string    `gorm:"icon" json:"icon"`
	Description string    `gorm:"description" json:"description"`
	Picture     string    `gorm:"picture" json:"picture"`
	Url         string    `gorm:"url" json:"url"`
	Classify    int       `gorm:"classify" json:"classify"`
	Keyword     string    `gorm:"keyword" json:"keyword"`
	CreateTime  time.Time `gorm:"create_time" json:"create_time"`
	UpdateTime  time.Time `gorm:"update_time" json:"update_time"`
}

func (m *MenuClassify) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, MenuClassifyTable)
}

func (m *MenuClassify) GetByClassify(classify int) (menuClassifies []MenuClassify, err error) {
	db := dbutil.IOTUGormDb.Table(m.TableName())
	err = db.Where("classify = ?", classify).Find(&menuClassifies).Error
	return
}
