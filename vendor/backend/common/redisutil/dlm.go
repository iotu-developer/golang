package redisutil

import (
	"backend/common/cryptoutil"
	"log"
	"third/redigo/redis"
	"time"
)

const (
	Lockms = 10000 * time.Millisecond
)

type Mutex struct {
	cache      *Cache
	name       string
	value      string
	chHearbeat chan struct{}
}

func NewMutex(cache *Cache, name string) *Mutex {
	return &Mutex{
		cache:      cache,
		name:       name,
		chHearbeat: make(chan struct{}, 0),
	}
}

func (m *Mutex) Lock() (time.Duration, error) {
	value := cryptoutil.RandString(32)
	for {
		if m.acquire(value, Lockms) {
			m.value = value
			go m.hearbeat()
			return Lockms, nil
		}

		time.Sleep(Lockms / 2)
	}

	return Lockms, nil
}

func (m *Mutex) Unlock() bool {
	m.chHearbeat <- struct{}{}
	return m.release(m.value)
}

func (m *Mutex) hearbeat() {
	for {
		select {
		case <-m.chHearbeat:
			log.Println("stop hearbeat")
			return
		case <-time.After(Lockms / 3):
			if !m.touch(m.value, int(Lockms/time.Millisecond)) {
				log.Printf("touch error: [%s:%s] fails", m.name, m.value)
			}
		}
	}
}

func (m *Mutex) acquire(value string, lockms time.Duration) bool {
	conn := m.cache.RedisPool().Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do("SET", m.name, value, "NX", "PX", int(lockms/time.Millisecond)))
	return err == nil && reply == "OK"
}

var deleteScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

func (m *Mutex) release(value string) bool {
	conn := m.cache.RedisPool().Get()
	defer conn.Close()
	status, err := deleteScript.Do(conn, m.name, value)
	return err == nil && status != 0
}

var touchScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("SET", KEYS[1], ARGV[1], "XX", "PX", ARGV[2])
	else
		return "ERR"
	end
`)

func (m *Mutex) touch(value string, expiry int) bool {
	conn := m.cache.RedisPool().Get()
	defer conn.Close()
	status, err := redis.String(touchScript.Do(conn, m.name, value, expiry))
	return err == nil && status != "ERR"
}
