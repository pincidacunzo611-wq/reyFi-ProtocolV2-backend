package svc

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	chainspkg "github.com/reyfi/reyfi-backend/pkg/chains"
	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/config"
	"github.com/reyfi/reyfi-backend/app/chain-indexer/internal/publisher"
	"github.com/zeromicro/go-zero/core/logx"
)

// ServiceContext 依赖注入容器
type ServiceContext struct {
	Config    config.Config
	DB        *sql.DB
	Chain     *chainspkg.Client
	Publisher publisher.EventPublisher
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil {
		logx.Must(err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	logx.Info("database connected")

	// 初始化链上客户端
	chainClient, err := chainspkg.NewClient(chainspkg.ChainConfig{
		RpcUrl:    c.Chain.RpcUrl,
		ChainId:   c.Chain.ChainId,
		Contracts: c.Chain.Contracts,
	})
	if err != nil {
		logx.Must(err)
	}

	// 初始化 Kafka 发布者
	pub := publisher.NewKafkaPublisher(c.Kafka.Brokers)
	logx.Info("kafka publisher initialized")

	return &ServiceContext{
		Config:    c,
		DB:        db,
		Chain:     chainClient,
		Publisher: pub,
	}
}

// Close 关闭所有资源
func (s *ServiceContext) Close() {
	if s.DB != nil {
		s.DB.Close()
	}
	if s.Chain != nil {
		s.Chain.Close()
	}
	if s.Publisher != nil {
		s.Publisher.Close()
	}
	logx.Info("service context closed")
}
