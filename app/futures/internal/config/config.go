package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Mode       string `json:",default=dev"`
	DataSource string
	Redis      struct{ Host, Type, Pass string }
	Chain      struct{ ChainId int64; Contracts map[string]string }
	Kafka      struct{ Brokers []string; GroupId, Topic string }
}
