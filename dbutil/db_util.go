package dbutil

import (
	"backend/common/clog"
	"fmt"
	_ "third/go-sql-driver/mysql"
	"third/gorm"
)

var IOTUGormDb *gorm.DB
var DatabaseConnection = "iotu:IOTU.club666@tcp(139.155.87.206:3306)/IOTU?charset=utf8&parseTime=true"

//初始化GORM数据库连接池
func InitGormDb(setLog bool) (*gorm.DB, error) {
	db, err := gorm.Open("mysql", DatabaseConnection)
	if err != nil {
		fmt.Println("DataBase Connection err.", err)
		return nil, err
	}
	db.DB().SetMaxOpenConns(10)
	db.DB().SetMaxIdleConns(10 / 2)
	if setLog {
		db.LogMode(true)
		db.SetLogger(clog.Logger)
	}
	db.SingularTable(true)
	err = db.DB().Ping()
	if err != nil {
		fmt.Println("DataBase Ping err.", err)
		return nil, err
	}
	return &db, nil
}
