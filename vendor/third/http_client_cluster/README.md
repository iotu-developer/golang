# 带集群功能的http client

## TODO: 负载均衡策略优化

旧策略：域名解析IP -> 根据 `IP*2` 建立长连接 http client -> 根据调用次数，维护了一个小堆，每次取堆顶client 发起调用

新策略：域名解析IP -> 根据 `IP*4` 建立生命周期有限的 http client -> 随机取client发起调用 -> client 达到生命周期，淘汰该 client, 并生成新 client 


## 熔断配置

从etcd的`hystrix_http`加载断路器配置。默认配置格式如下:

```
{
  "authcenter.in.codoon.com:2014": {
    "timeout": 5000,
    "max_concurrent_requests": 100,
    "request_volume_threshold": 20,
    "sleep_window": 5000,
    "error_percent_threshold": 20
  }
}
```

timeout: 超时时间，单位毫秒，根据不同服务而定

max_concurrent_requests: 最大并发请求数

request_volume_threshold: 触发断路器的进行健康检查的阈值

sleep_window: 断路器打开后，进行服务恢复测试的时间间隔，单位毫秒

error_percent_threshold: 断路器开启的阈值，单位百分比。

rolling_window目前为固定的60s, 不可配置。
