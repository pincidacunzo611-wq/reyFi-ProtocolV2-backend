package config

type Config struct {
	Name       string `json:",default=bot"`
	Mode       string `json:",default=dev"`
	DataSource string
	Redis      struct{ Host, Type, Pass string }
	Chain      struct {
		RpcUrl    string
		ChainId   int64
		Contracts map[string]string
	}
	Jobs       JobsConfig
}

type JobsConfig struct {
	LiquidationMonitorInterval  int  `json:",default=10"`
	PriceUpdateInterval         int  `json:",default=30"`
	DailyAggregationCron        string `json:",default=0 1 * * *"`
	FundingSettlementInterval   int  `json:",default=3600"`
	OptionsExpiryCheckInterval  int  `json:",default=60"`
}
