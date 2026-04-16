package svc

import (
	"context"; "database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/reyfi/reyfi-backend/app/bonds/internal/config"
	"github.com/reyfi/reyfi-backend/app/bonds/internal/consumer"
	"github.com/reyfi/reyfi-backend/app/bonds/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config         config.Config; DB *sql.DB; Redis *redis.Redis
	MarketModel    *model.BondMarketModel; PositionModel *model.BondPositionModel
	RedemptionModel *model.BondRedemptionModel; bondsConsumer *consumer.BondsEventConsumer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := sql.Open("mysql", c.DataSource); if err != nil { logx.Must(err) }
	db.SetMaxOpenConns(50); db.SetMaxIdleConns(20)
	if err := db.Ping(); err != nil { logx.Must(err) }
	rds := redis.MustNewRedis(redis.RedisConf{Host: c.Redis.Host, Type: c.Redis.Type, Pass: c.Redis.Pass})
	s := &ServiceContext{Config: c, DB: db, Redis: rds,
		MarketModel: model.NewBondMarketModel(db), PositionModel: model.NewBondPositionModel(db),
		RedemptionModel: model.NewBondRedemptionModel(db),
	}
	s.bondsConsumer = consumer.NewBondsEventConsumer(db, c.Chain.ChainId, rds)
	return s
}

func (s *ServiceContext) StartConsumers() {
	cfg := s.Config.Kafka; if len(cfg.Brokers) == 0 { return }
	logx.Infof("starting bonds kafka consumer: topic=%s", cfg.Topic)
	if err := s.bondsConsumer.Start(context.Background(), cfg.Brokers, cfg.GroupId, cfg.Topic); err != nil {
		logx.Errorf("bonds consumer error: %v", err)
	}
}
