package svc

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/dex/internal/config"
	"github.com/reyfi/reyfi-backend/app/dex/internal/consumer"
	"github.com/reyfi/reyfi-backend/app/dex/internal/model"
	"github.com/reyfi/reyfi-backend/app/dex/internal/router"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// ServiceContext DEX 服务依赖注入
type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
	Redis  *redis.Redis

	// Model 层
	PairModel              *model.DexPairModel
	TradeModel             *model.DexTradeModel
	PairSnapshotModel      *model.DexPairSnapshotModel
	LiquidityEventModel    *model.DexLiquidityEventModel
	LiquidityPositionModel *model.DexLiquidityPositionModel
	PairStatsDailyModel    *model.DexPairStatsDailyModel

	// 路由器（最佳兑换路径）
	Router *router.Router

	// 消费者
	swapConsumer *consumer.DexEventConsumer
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化数据库
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil {
		logx.Must(err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(20)
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	logx.Info("dex database connected")

	// 初始化 Redis
	rds := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})

	svcCtx := &ServiceContext{
		Config: c,
		DB:     db,
		Redis:  rds,

		PairModel:              model.NewDexPairModel(db),
		TradeModel:             model.NewDexTradeModel(db),
		PairSnapshotModel:      model.NewDexPairSnapshotModel(db),
		LiquidityEventModel:    model.NewDexLiquidityEventModel(db),
		LiquidityPositionModel: model.NewDexLiquidityPositionModel(db),
		PairStatsDailyModel:    model.NewDexPairStatsDailyModel(db),

		Router: router.NewRouter(3, 3),
	}

	// 初始化消费者
	svcCtx.swapConsumer = consumer.NewDexEventConsumer(svcCtx.DB, c.Chain.ChainId, rds)

	return svcCtx
}

// StartConsumers 启动 Kafka 消费者
func (s *ServiceContext) StartConsumers() {
	cfg := s.Config.Kafka
	if len(cfg.Brokers) == 0 {
		logx.Info("kafka not configured, skip consumer")
		return
	}

	logx.Infof("starting dex kafka consumer: topic=%s, group=%s", cfg.Topic, cfg.GroupId)
	if err := s.swapConsumer.Start(context.Background(), cfg.Brokers, cfg.GroupId, cfg.Topic); err != nil {
		logx.Errorf("dex kafka consumer error: %v", err)
	}
}
