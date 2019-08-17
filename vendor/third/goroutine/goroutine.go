package goroutine

import (
	//"third/goid"
	"third/goroutineid"
)

func GoroutineId() int64 {
	//return goid.Get()
	return goroutineid.GetGoID()
}
