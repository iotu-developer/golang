package main

import (
	"backend/common/clog"
	"backend/common/config"
	"fmt"
	"runtime"
	"web/dbutil"
	"web/router"
)

//初始化
func Init() {
	var err error
	dbutil.IOTUGormDb, err = dbutil.InitGormDb(true)
	if err != nil {
		fmt.Println("init third gps data failed")
	}
	//初始化日志
	logConfig := config.LogConfig{
		LogDir:   "/var/log/go_log",
		LogFile:  "iotuWeb.log",
		LogLevel: "DEBUG",
	}
	clog.Logger, err = clog.InitLoggerByConfig(&logConfig)
	if nil != err {
		fmt.Println("InitLogger err :", err)
		return
	}
}

func main() {
	clog.Logger.Info("start code service")
	Init()
	runtime.GOMAXPROCS(runtime.NumCPU())
	router.StartHttpServer()
}
