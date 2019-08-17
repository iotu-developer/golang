/*
用简易的环形队列来存储M个sync.Map实现的local storage（大于等于3个）
假设g为当前时刻local storage的索引
如果local storage[g]没有查找到当前goid的数据，则尝试local storage[g-1]，甚至继续往前
定时重置并删除local storage[g-M+1]
*/

package gls

import (
	"fmt"
	"sync"
	"time"
)

const (
	MAX_GRIDS        = 3 //local storage的个数，需要大于等于3
	MINUTES_PER_GRID = 5 //一个local storage包含的分钟数
)

var (
	glsRegistry = make([]*sync.Map, MAX_GRIDS)
	glsInit     = false
)

type Values map[interface{}]interface{}

func InitGls() {
	fmt.Println("gls init")

	go func() {
		now := time.Now()
		y, m, d := now.Date()
		hour := now.Hour()
		minute := now.Minute()
		nextMinute := (minute/MINUTES_PER_GRID + 1) * MINUTES_PER_GRID

		//把时间对齐到下个grid
		var nextTime time.Time
		if nextMinute == 60 {
			nextTime = time.Date(y, m, d, hour+1, 0, 0, 0, time.Local)
		} else {
			nextTime = time.Date(y, m, d, hour, nextMinute, 0, 0, time.Local)
		}
		dur := nextTime.Sub(now)
		time.Sleep(dur)

		ticker := time.NewTicker(time.Minute * MINUTES_PER_GRID)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				clear()
			}
		}
	}()

	glsInit = true
}

//返回当前时刻在local storage列表的索引
func grid() int {
	minute := time.Now().Minute()
	return (minute / MINUTES_PER_GRID) % MAX_GRIDS
}

//保存，id典型为goid
func Set(goid func() int64, vs Values) {
	if !glsInit {
		panic("gls not init")
	}

	id := goid()
	g := grid()

	if m := glsRegistry[g]; m != nil {
		m.Store(id, vs)
	} else {
		glsRegistry[g] = &sync.Map{}
		glsRegistry[g].Store(id, vs)
	}
}

//获取，id典型为goid
//失败则返回nil
func Get(id int64) (vs Values) {
	if !glsInit {
		panic("gls not init")
	}

	g := grid()

	find := false
	if vs, find = try(id, g); vs == nil || find == false {
		//尝试找上一个storage
		g -= 1
		if g < 0 {
			g += MAX_GRIDS
		}
		vs, _ = try(id, g)
	}

	return vs
}

func try(id int64, g int) (Values, bool) {
	if m := glsRegistry[g]; m != nil {
		if v, ok := m.Load(id); !ok {
			return nil, false
		} else {
			switch t := v.(type) {
			case Values:
				return t, true
			default:
				return nil, false
			}
		}
	}

	return nil, false
}

//定时删除最早的一个Map
func clear() {
	fmt.Println("clear grid")

	g := grid()
	g -= (MAX_GRIDS - 1)
	if g < 0 {
		g += MAX_GRIDS
	}
	if m := glsRegistry[g]; m != nil {
		glsRegistry[g] = nil
	}
}

//替换go
//输入为原函数、获取goid的函数、以及context
func Go(f func(), goid func() int64, vs Values) {
	if !glsInit {
		panic("gls not init")
	}

	newf := func(f func(), goid func() int64, vs Values) func() {
		return func() {
			Set(goid, vs)
			f()
		}
	}(f, goid, vs)

	go newf()
}
