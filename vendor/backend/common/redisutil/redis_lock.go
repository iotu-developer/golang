package redisutil

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// 基于redis的分布式锁，不是100%可靠 特别是redis集群环境下，对资源抢占非常敏感的业务慎用
type RedisLock struct {
	LockKey string
	value   string
}

//保证原子性（redis是单线程），避免del删除了，其他client获得的lock
var delScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end`

func (rl *RedisLock) Lock(cache *Cache, timeout int) error {
	//随机数
	if cache == nil {
		return errors.New("redis cache is nil")
	}
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	rl.value = base64.StdEncoding.EncodeToString(b)
	ok, err := cache.SetNxEx(rl.LockKey, rl.value, timeout)
	if err == nil && ok {
		// key已经存在
		errMsg := fmt.Sprintf("lock fail")
		return errors.New(errMsg)
	} else if err == nil {
		return nil
	} else {
		errMsg := fmt.Sprintf("redis fail err %v", err)
		return errors.New(errMsg)
	}
}

func (rl *RedisLock) Unlock(cache *Cache) error {
	if cache == nil {
		return errors.New("redis cache is nil")
	}
	_, err := cache.EvalHash(1, delScript, rl.LockKey, rl.value)
	return err
}
