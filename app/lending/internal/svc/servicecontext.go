package svc

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/lending/internal/config"
	"github.com/reyfi/reyfi-backend/app/lending/internal/consumer"
	"github.com/reyfi/reyfi-backend/app/lending/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config             config.Config
	DB                 *sql.DB
	Redis              *redis.Redis
	MarketModel        *model.LendingMarketModel
	SnapshotModel      *model.LendingMarketSnapshotModel
	UserPositionModel  *model.LendingUserPositionModel
	LiquidationModel   *model.LendingLiquidationModel
	lendingConsumer    *consumer.LendingEventConsumer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.DataSource)
	if err != nil {
		logx.Must(err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(20)
	if err := db.Ping(); err != nil {
		logx.Must(err)
	}
	logx.Info("lending database connected")

	rds := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})

	svcCtx := &ServiceContext{
		Config:            c,
		DB:                db,
		Redis:             rds,
		MarketModel:       model.NewLendingMarketModel(db),
		SnapshotModel:     model.NewLendingMarketSnapshotModel(db),
		UserPositionModel: model.NewLendingUserPositionModel(db),
		LiquidationModel:  model.NewLendingLiquidationModel(db),
	}

	svcCtx.lendingConsumer = consumer.NewLendingEventConsumer(db, c.Chain.ChainId, rds)
	return svcCtx
}

func (s *ServiceContext) StartConsumers() {
	cfg := s.Config.Kafka
	if len(cfg.Brokers) == 0 {
		logx.Info("kafka not configured, skip lending consumer")
		return
	}
	logx.Infof("starting lending kafka consumer: topic=%s, group=%s", cfg.Topic, cfg.GroupId)
	if err := s.lendingConsumer.Start(context.Background(), cfg.Brokers, cfg.GroupId, cfg.Topic); err != nil {
		logx.Errorf("lending kafka consumer error: %v", err)
	}
}
