package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Mode string `json:",default=dev"`

	DataSource string

	Redis RedisConfig

	JwtAuth JwtAuthConfig
}

type RedisConfig struct {
	Host string
	Type string `json:",default=node"`
	Pass string `json:",optional"`
}

type JwtAuthConfig struct {
	AccessSecret  string
	AccessExpire  int64 `json:",default=7200"`
	RefreshExpire int64 `json:",default=604800"`
}
