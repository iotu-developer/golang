package model

import (
	"fmt"
	"time"
	"web/dbutil"
)

type Code struct {
	Id            int       `gorm:"id"`              //自增主键
	CreateUserId  string    `gorm:"create_user_id"`  //创建者ID
	ConsumeUserId string    `gorm:"consume_user_id"` //使用者ID
	ActiveCode    string    `gorm:"active_code"`     //激活码
	State         int       `gorm:"state"`           //激活码状态 0：失效；1：有效
	CreateAt      time.Time `gorm:"create_at"`       //创建时间
	ConsumeAt     time.Time `gorm:"consume_at"`      //使用时间
}

var (
	IotuDb           = "IOTU"
	CodeTable        = "code"
	AccountTable     = "user"
	RotaryChartTable = "rotary_chart"
	IotuMemberTable  = "iotu_members"
	ColumnDataTable  = "column_data"
)

//获取code的数据库名
func (code *Code) TableName() string {
	return fmt.Sprintf("%s.%s", IotuDb, CodeTable)
}

//新增一条数据
func (code *Code) Create() error {
	DB := dbutil.IOTUGormDb.Table(code.TableName())
	return DB.Create(code).Error
}

//用激活码查询记录
func (code *Code) FindByCode() error {
	DB := dbutil.IOTUGormDb.Table(code.TableName())
	return DB.Where("active_code = ?", code.ActiveCode).Find(&code).Error
}

//修改激活码状态和使用者
func (code *Code) UpdateStateAndConsumer(updateMap map[string]interface{}) error {
	DB := dbutil.IOTUGormDb.Table(code.TableName())
	return DB.Where("active_code = ?", code.ActiveCode).Update(updateMap).Error
}
