package main

import (
	"backend/common/clog"
	"code/dbutil"
	"code/router"
	"fmt"
	"runtime"
)

//初始化
func Init() {
	var err error
	dbutil.IOTUGormDb, err = dbutil.InitGormDb(true)
	if err != nil {
		fmt.Println("init third gps data failed")
	}
}

func main() {
	clog.Logger.Info("start code service")
	Init()
	runtime.GOMAXPROCS(runtime.NumCPU())
	router.StartHttpServer()
}
