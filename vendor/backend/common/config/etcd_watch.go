package config

import (
	"context"
	"log"
	"os"
	"strings"
	"third/etcd-client"
	"time"
)

//raw_value 是存到etcd的value raw， 使用法根据需要进行decode
type OnConfigChanged func(raw_value string) error

func Watch(service string, onChanged OnConfigChanged, path ...string) error {
	var environment = os.Getenv("GOENV")
	if environment == "" {
		environment = "online"
	} else {
		environment = strings.ToLower(environment)
	}
	config_key := ""
	if len(path) > 0 {
		config_key = etcdPathKey(service, path[0])
	} else {
		config_key = etcdKey(service, environment)
	}
	go innerWatch(config_key, onChanged)
	return nil
}

var TTL time.Duration = 10 * time.Second
var addr []string = []string{"http://etcd.in.codoon.com:2379"}

func newClient(addrlist ...string) (client.KeysAPI, error) {
	if len(addrlist) > 0 {
		addr = addrlist
	}
	cfg := client.Config{
		Endpoints:               addr,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: TTL,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Printf("new etcd api where addr[%+v] err:%+v", addr, err)
		return nil, err
	}
	api := client.NewKeysAPI(c)
	return api, nil
}

func innerWatch(key string, onChanged OnConfigChanged) {
	c, _ := newClient(addr...)
	w := c.Watcher(key, nil)
	for {
		if resp, err := w.Next(context.Background()); err == nil {
			log.Printf("key:%+v, resp:%+v", key, resp)
			if resp != nil && resp.Node != nil {
				if err := onChanged(resp.Node.Value); err != nil {
					log.Printf("exec onChanged  key:%+v, val:%+v, err:%+v", key, resp.Node.Value, err)
				}
			}
		} else {
			log.Printf("etcd watch next failed key:%+v, err:%+v", key, err)
			var err error
			if c, err = newClient(addr...); err != nil {
				log.Printf("new etcd watch client failed, key:%+v, err:%+v", key, err)
			}
			time.Sleep(time.Second * 10)
		}
	}
}
