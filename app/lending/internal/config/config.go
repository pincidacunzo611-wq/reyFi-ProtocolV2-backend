package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Mode       string `json:",default=dev"`
	DataSource string
	Redis      RedisConfig
	Chain      ChainConfig
	Kafka      KafkaConsumerConfig
}

type RedisConfig struct {
	Host string
	Type string `json:",default=node"`
	Pass string `json:",optional"`
}

type ChainConfig struct {
	ChainId   int64             `json:"chainId"`
	Contracts map[string]string `json:"contracts"`
}

type KafkaConsumerConfig struct {
	Brokers []string
	GroupId string
	Topic   string
}
