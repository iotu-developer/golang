package http_client_cluster

import (
	"backend/common/config"
	"log"
	"third/hystrix"
)

var hystrixConfig = map[string]hystrix.CommandConfig{}

func init() {
	var err error
	hystrixConfig, err = loadCircurtConfigFromEtcd()
	if err != nil {
		log.Printf("http_client_cluster circuit load config from etcd no success:%v", err)
		return
	}

	hystrix.Configure(hystrixConfig)
}

func loadCircurtConfigFromEtcd() (map[string]hystrix.CommandConfig, error) {
	hConfig := map[string]hystrix.CommandConfig{}
	err := config.LoadCfgFromEtcd(config.DefaultEtcdAddrs, "hystrix_http", &hConfig)
	return hConfig, err
}

func circuitEnabled(service string) bool {
	_, found := hystrixConfig[service]
	return found
}
