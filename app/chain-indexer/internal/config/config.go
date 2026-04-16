package config

// Config Chain Indexer 服务配置
type Config struct {
	Name string `json:",default=chain-indexer"`
	Mode string `json:",default=dev"`

	// 数据库
	DataSource string

	// 链上配置
	Chain ChainConfig

	// 区块扫描配置
	Scanner ScannerConfig

	// Kafka 配置
	Kafka KafkaConfig

	// 日志
	Log LogConfig
}

// ChainConfig 链上配置
type ChainConfig struct {
	RpcUrl    string            `json:"rpcUrl"`
	ChainId   int64             `json:"chainId"`
	Contracts map[string]string `json:"contracts"`
}

// ScannerConfig 区块扫描器配置
type ScannerConfig struct {
	StartBlock    int64 `json:",default=0"`      // 起始扫描区块
	BatchSize     int64 `json:",default=100"`     // 每批获取区块数
	PollInterval  int64 `json:",default=2000"`    // 轮询间隔（毫秒）
	ConfirmBlocks int64 `json:",default=12"`      // 确认块数
	MaxRetries    int   `json:",default=5"`       // 最大重试次数
	RetryInterval int64 `json:",default=3000"`    // 重试间隔（毫秒）
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers []string `json:"brokers"`
}

// LogConfig 日志配置
type LogConfig struct {
	ServiceName string `json:",default=chain-indexer"`
	Mode        string `json:",default=console"`
	Level       string `json:",default=info"`
}
