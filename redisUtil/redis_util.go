package redisUtil

import (
	"backend/common/clog"
	"third/redigo/redis"
)

const redisIP = "139.155.87.206:6379"

//设置key-value
func SetString(key, value string) (result bool) {
	c, err := redis.Dial("tcp", redisIP)
	if err != nil {
		clog.Errorf("redis 连接失败 [err = %s]", err)
		return false
	}
	defer c.Close()
	_, err = c.Do("SET", key, value)
	if err != nil {
		clog.Errorf("redis SET失败 [err = %s]", err)
		return false
	} else {
		return true
	}
}

//设置超时时间
func SetExpire(key string, time int) (result bool) {
	c, err := redis.Dial("tcp", redisIP)
	if err != nil {
		clog.Errorf("redis 连接失败 [err = %s]", err)
		return false
	}
	defer c.Close()
	n, err := c.Do("EXPIRE", key, time)
	if n == int64(1) {
		return true
	} else {
		return false
	}
}

//根据Key获取value
func GetString(key string) (result string) {
	c, err := redis.Dial("tcp", redisIP)
	if err != nil {
		clog.Errorf("redis 连接失败 [err = %s]", err)
		return
	}
	defer c.Close()
	value, err := redis.String(c.Do("GET", key))
	if err != nil {
		clog.Errorf("redis GET失败 [err = %s]", err)
		return
	} else {
		result = value
		return
	}
}

//根据Key判断是否存在
func IsKeyExit(key string) (result bool) {
	c, err := redis.Dial("tcp", redisIP)
	if err != nil {
		clog.Errorf("redis 连接失败 [err = %s]", err)
		return false
	}
	defer c.Close()
	isKeyExit, err := redis.Bool(c.Do("EXISTS", key))
	if err != nil {
		clog.Errorf("redis isKeyExit 失败 [err = %s]", err)
		return false
	} else {
		return isKeyExit
	}
}
