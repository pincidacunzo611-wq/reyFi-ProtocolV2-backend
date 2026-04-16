package svc

import (
	"github.com/reyfi/reyfi-backend/app/bonds/rpc/bondsclient"
	"github.com/reyfi/reyfi-backend/app/dex/rpc/dexclient"
	"github.com/reyfi/reyfi-backend/app/futures/rpc/futuresclient"
	"github.com/reyfi/reyfi-backend/app/gateway/internal/config"
	"github.com/reyfi/reyfi-backend/app/governance/rpc/governanceclient"
	"github.com/reyfi/reyfi-backend/app/lending/rpc/lendingclient"
	"github.com/reyfi/reyfi-backend/app/options/rpc/optionsclient"
	"github.com/reyfi/reyfi-backend/app/user/rpc/userclient"
	"github.com/reyfi/reyfi-backend/app/vault/rpc/vaultclient"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext Gateway 服务依赖
type ServiceContext struct {
	Config config.Config
	Redis  *redis.Redis

	// RPC 客户端
	DexRpc        dexclient.DexService
	UserRpc       userclient.UserService
	LendingRpc    lendingclient.LendingService
	FuturesRpc    futuresclient.FuturesService
	OptionsRpc    optionsclient.OptionsService
	VaultRpc      vaultclient.VaultService
	BondsRpc      bondsclient.BondsService
	GovernanceRpc governanceclient.GovernanceService
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Type: c.Redis.Type,
		Pass: c.Redis.Pass,
	})

	return &ServiceContext{
		Config: c,
		Redis:  rds,

		DexRpc:        dexclient.NewDexService(zrpc.MustNewClient(c.DexRpc)),
		UserRpc:       userclient.NewUserService(zrpc.MustNewClient(c.UserRpc)),
		LendingRpc:    lendingclient.NewLendingService(zrpc.MustNewClient(c.LendingRpc)),
		FuturesRpc:    futuresclient.NewFuturesService(zrpc.MustNewClient(c.FuturesRpc)),
		OptionsRpc:    optionsclient.NewOptionsService(zrpc.MustNewClient(c.OptionsRpc)),
		VaultRpc:      vaultclient.NewVaultService(zrpc.MustNewClient(c.VaultRpc)),
		BondsRpc:      bondsclient.NewBondsService(zrpc.MustNewClient(c.BondsRpc)),
		GovernanceRpc: governanceclient.NewGovernanceService(zrpc.MustNewClient(c.GovernanceRpc)),
	}
}
