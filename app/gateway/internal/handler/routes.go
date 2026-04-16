// Package handler 注册 Gateway 路由并实现 HTTP Handler。
// 每个 handler 负责：参数解析、调用 RPC、封装统一响应。
package handler

import (
	"net/http"

	"github.com/reyfi/reyfi-backend/app/gateway/internal/svc"
	"github.com/reyfi/reyfi-backend/pkg/middleware"
	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers 注册所有路由
func RegisterHandlers(engine *rest.Server, svcCtx *svc.ServiceContext) {
	authMiddleware := middleware.AuthMiddleware(svcCtx.Config.JwtAuth.AccessSecret)

	// ==================== 健康检查 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/health",
			Handler: healthHandler(svcCtx),
		},
	})

	// ==================== DEX 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/pairs",
			Handler: dexGetPairsHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/pairs/:pairAddress",
			Handler: dexGetPairDetailHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/pairs/:pairAddress/trades",
			Handler: dexGetTradesHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/pairs/:pairAddress/candles",
			Handler: dexGetCandlesHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/overview",
			Handler: dexGetOverviewHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/route",
			Handler: dexFindRouteHandler(svcCtx),
		},
	})

	// ==================== DEX 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/dex/positions",
			Handler: authMiddleware(dexGetPositionsHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/dex/swap/build",
			Handler: authMiddleware(dexBuildSwapHandler(svcCtx)),
		},
	})

	// ==================== User Auth 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/api/user/auth/nonce",
			Handler: userGetNonceHandler(svcCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/user/auth/login",
			Handler: userLoginHandler(svcCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/user/auth/refresh",
			Handler: userRefreshTokenHandler(svcCtx),
		},
	})

	// ==================== User 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/user/portfolio",
			Handler: authMiddleware(userGetPortfolioHandler(svcCtx)),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/user/activity",
			Handler: authMiddleware(userGetActivityHandler(svcCtx)),
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/user/settings",
			Handler: authMiddleware(userUpdateSettingsHandler(svcCtx)),
		},
	})

	// ==================== Lending 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/lending/markets",
			Handler: lendingGetMarketsHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/lending/markets/:assetAddress",
			Handler: lendingGetMarketDetailHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/lending/liquidations",
			Handler: lendingGetLiquidationsHandler(svcCtx),
		},
	})

	// ==================== Lending 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/lending/position",
			Handler: authMiddleware(lendingGetUserPositionHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/lending/supply/build",
			Handler: authMiddleware(lendingBuildSupplyHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/lending/borrow/build",
			Handler: authMiddleware(lendingBuildBorrowHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/lending/repay/build",
			Handler: authMiddleware(lendingBuildRepayHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/lending/withdraw/build",
			Handler: authMiddleware(lendingBuildWithdrawHandler(svcCtx)),
		},
	})

	// ==================== Futures 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/futures/markets",
			Handler: futuresGetMarketsHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/futures/markets/:marketAddress",
			Handler: futuresGetMarketDetailHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/futures/markets/:marketAddress/funding",
			Handler: futuresGetFundingHistoryHandler(svcCtx),
		},
	})

	// ==================== Futures 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/futures/positions",
			Handler: authMiddleware(futuresGetPositionsHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/futures/position/open",
			Handler: authMiddleware(futuresBuildOpenHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/futures/position/close",
			Handler: authMiddleware(futuresBuildCloseHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/futures/position/adjust-margin",
			Handler: authMiddleware(futuresBuildAdjustMarginHandler(svcCtx)),
		},
	})

	// ==================== Options 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/options/chains",
			Handler: optionsGetChainsHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/options/vol-surface/:underlying",
			Handler: optionsGetVolSurfaceHandler(svcCtx),
		},
	})

	// ==================== Options 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/options/positions",
			Handler: authMiddleware(optionsGetPositionsHandler(svcCtx)),
		},
	})

	// ==================== Vault 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/vault/list",
			Handler: vaultGetListHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/vault/:vaultAddress",
			Handler: vaultGetDetailHandler(svcCtx),
		},
	})

	// ==================== Vault 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/vault/positions",
			Handler: authMiddleware(vaultGetPositionsHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/vault/deposit/build",
			Handler: authMiddleware(vaultBuildDepositHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/vault/withdraw/build",
			Handler: authMiddleware(vaultBuildWithdrawHandler(svcCtx)),
		},
	})

	// ==================== Bonds 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/bonds/markets",
			Handler: bondsGetMarketsHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/bonds/markets/:marketAddress",
			Handler: bondsGetMarketDetailHandler(svcCtx),
		},
	})

	// ==================== Bonds 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/bonds/positions",
			Handler: authMiddleware(bondsGetPositionsHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/bonds/purchase/build",
			Handler: authMiddleware(bondsBuildPurchaseHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/bonds/claim/build",
			Handler: authMiddleware(bondsBuildClaimHandler(svcCtx)),
		},
	})

	// ==================== Governance 公开接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/governance/proposals",
			Handler: govGetProposalsHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/governance/proposals/:proposalId",
			Handler: govGetProposalDetailHandler(svcCtx),
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/governance/proposals/:proposalId/votes",
			Handler: govGetVotesHandler(svcCtx),
		},
	})

	// ==================== Governance 登录后接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/governance/ve-lock",
			Handler: authMiddleware(govGetVeLockHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/governance/vote/build",
			Handler: authMiddleware(govBuildVoteHandler(svcCtx)),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/governance/lock/build",
			Handler: authMiddleware(govBuildLockHandler(svcCtx)),
		},
	})

	// ==================== System 接口 ====================
	engine.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/system/sync-status",
			Handler: systemSyncStatusHandler(svcCtx),
		},
	})
}
