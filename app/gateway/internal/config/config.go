package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Mode string `json:",default=dev"`

	JwtAuth JwtAuthConfig
	Redis   RedisConfig

	// RPC 客户端配置
	DexRpc        zrpc.RpcClientConf
	UserRpc       zrpc.RpcClientConf
	LendingRpc    zrpc.RpcClientConf
	FuturesRpc    zrpc.RpcClientConf
	OptionsRpc    zrpc.RpcClientConf
	VaultRpc      zrpc.RpcClientConf
	BondsRpc      zrpc.RpcClientConf
	GovernanceRpc zrpc.RpcClientConf
}

type JwtAuthConfig struct {
	AccessSecret string
	AccessExpire int64 `json:",default=7200"`
}

type RedisConfig struct {
	Host string
	Type string `json:",default=node"`
	Pass string `json:",optional"`
}
