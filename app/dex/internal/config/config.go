package config

import "github.com/zeromicro/go-zero/zrpc"

// Config DEX 服务配置
type Config struct {
	zrpc.RpcServerConf
	Mode string `json:",default=dev"`

	// 数据库
	DataSource string

	// Redis
	Redis RedisConfig

	// 链上配置
	Chain ChainConfig

	// Kafka
	Kafka KafkaConsumerConfig
}

type RedisConfig struct {
	Host string
	Type string `json:",default=node"`
	Pass string `json:",optional"`
}

type ChainConfig struct {
	RpcUrl    string            `json:"rpcUrl"`
	ChainId   int64             `json:"chainId"`
	Contracts map[string]string `json:"contracts"`
}

type KafkaConsumerConfig struct {
	Brokers []string
	GroupId string
	Topic   string
}
