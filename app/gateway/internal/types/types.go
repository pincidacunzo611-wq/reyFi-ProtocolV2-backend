// Package types 定义 Gateway 层的请求/响应结构
package types

// ==================== 通用 ====================

type PageReq struct {
	Page     int64  `form:"page,default=1"`
	PageSize int64  `form:"pageSize,default=20"`
}

// ==================== DEX ====================

type DexPairsReq struct {
	PageReq
	Keyword   string `form:"keyword,optional"`
	Token0    string `form:"token0,optional"`
	Token1    string `form:"token1,optional"`
	SortBy    string `form:"sortBy,optional"`
	SortOrder string `form:"sortOrder,optional"`
}

type DexPairDetailReq struct {
	PairAddress string `path:"pairAddress"`
}

type DexTradesReq struct {
	PairAddress string `path:"pairAddress"`
	PageReq
}

type DexCandlesReq struct {
	PairAddress string `path:"pairAddress"`
	Interval    string `form:"interval,default=1h"`
	From        int64  `form:"from,optional"`
	To          int64  `form:"to,optional"`
}

type DexSwapBuildReq struct {
	TokenIn     string `json:"tokenIn"`
	TokenOut    string `json:"tokenOut"`
	AmountIn    string `json:"amountIn"`
	SlippageBps int    `json:"slippageBps,default=50"`
	Receiver    string `json:"receiver,optional"`
}

type DexFindRouteReq struct {
	TokenIn    string `form:"tokenIn"`
	TokenOut   string `form:"tokenOut"`
	AmountIn   string `form:"amountIn"`
	MaxHops    int    `form:"maxHops,default=3"`
	MaxResults int    `form:"maxResults,default=3"`
}

// ==================== User Auth ====================

type AuthNonceReq struct {
	Address string `json:"address"`
}

type AuthNonceResp struct {
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
	ExpiresIn int64  `json:"expiresIn"`
}

type AuthLoginReq struct {
	Address   string `json:"address"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

type AuthLoginResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type AuthRefreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

// ==================== User Portfolio ====================

type UserPortfolioResp struct {
	WalletAddress string              `json:"walletAddress"`
	TotalAssetUsd string              `json:"totalAssetUsd"`
	TotalDebtUsd  string              `json:"totalDebtUsd"`
	NetValueUsd   string              `json:"netValueUsd"`
	Pnl24h        string              `json:"pnl24h"`
	Pnl24hPercent string              `json:"pnl24hPercent"`
	Allocation    []ModuleAllocation  `json:"allocation"`
	RiskSummary   RiskSummaryResp     `json:"riskSummary"`
}

type ModuleAllocation struct {
	Module   string `json:"module"`
	Label    string `json:"label"`
	ValueUsd string `json:"valueUsd"`
	Percent  string `json:"percent"`
}

type RiskSummaryResp struct {
	LendingHealthFactor string `json:"lendingHealthFactor"`
	FuturesMarginRatio  string `json:"futuresMarginRatio"`
	RiskLevel           string `json:"riskLevel"`
}

// ==================== User Activity ====================

type UserActivityReq struct {
	PageReq
	Module string `form:"module,optional"`
}

// ==================== User Settings ====================

type UserSettingsReq struct {
	Nickname        string `json:"nickname,optional"`
	Language        string `json:"language,optional"`
	Currency        string `json:"currency,optional"`
	SlippageDefault int    `json:"slippageDefault,optional"`
}

// ==================== Lending ====================

type LendingMarketsReq struct {
	PageReq
	Keyword string `form:"keyword,optional"`
}

type LendingMarketDetailReq struct {
	AssetAddress string `path:"assetAddress"`
}

type LendingLiquidationsReq struct {
	AssetAddress string `form:"assetAddress,optional"`
	UserAddress  string `form:"userAddress,optional"`
	PageReq
}

type LendingBuildSupplyReq struct {
	AssetAddress string `json:"assetAddress"`
	Amount       string `json:"amount"`
}

type LendingBuildBorrowReq struct {
	AssetAddress string `json:"assetAddress"`
	Amount       string `json:"amount"`
}

type LendingBuildRepayReq struct {
	AssetAddress string `json:"assetAddress"`
	Amount       string `json:"amount"`
}

type LendingBuildWithdrawReq struct {
	AssetAddress string `json:"assetAddress"`
	Amount       string `json:"amount"`
}

// ==================== Futures ====================

type FuturesMarketsReq struct {
	PageReq
}

type FuturesMarketDetailReq struct {
	MarketAddress string `path:"marketAddress"`
}

type FuturesFundingHistoryReq struct {
	MarketAddress string `path:"marketAddress"`
	PageReq
}

type FuturesBuildOpenReq struct {
	MarketAddress string `json:"marketAddress"`
	Side          string `json:"side"`
	Margin        string `json:"margin"`
	Leverage      string `json:"leverage"`
	LimitPrice    string `json:"limitPrice,optional"`
}

type FuturesBuildCloseReq struct {
	PositionId string `json:"positionId"`
	Amount     string `json:"amount,optional"`
}

type FuturesBuildAdjustMarginReq struct {
	PositionId string `json:"positionId"`
	Amount     string `json:"amount"`
	IsAdd      bool   `json:"isAdd"`
}

// ==================== Options ====================

type OptionsVolSurfaceReq struct {
	Underlying string `path:"underlying"`
}

// ==================== Vault ====================

type VaultListReq struct {
	PageReq
}

type VaultDetailReq struct {
	VaultAddress string `path:"vaultAddress"`
}

type VaultBuildDepositReq struct {
	VaultAddress string `json:"vaultAddress"`
	Amount       string `json:"amount"`
}

type VaultBuildWithdrawReq struct {
	VaultAddress string `json:"vaultAddress"`
	Shares       string `json:"shares"`
}

// ==================== Bonds ====================

type BondsMarketsReq struct {
	PageReq
}

type BondsMarketDetailReq struct {
	MarketAddress string `path:"marketAddress"`
}

type BondsBuildPurchaseReq struct {
	MarketAddress string `json:"marketAddress"`
	Amount        string `json:"amount"`
}

type BondsBuildClaimReq struct {
	BondId string `json:"bondId"`
}

// ==================== Governance ====================

type GovProposalsReq struct {
	PageReq
	Status string `form:"status,optional"`
}

type GovProposalDetailReq struct {
	ProposalId string `path:"proposalId"`
}

type GovVotesReq struct {
	ProposalId string `path:"proposalId"`
	PageReq
}

type GovBuildVoteReq struct {
	ProposalId string `json:"proposalId"`
	Support    int    `json:"support"`
	Reason     string `json:"reason,optional"`
}

type GovBuildLockReq struct {
	Amount   string `json:"amount"`
	Duration int64  `json:"duration"`
}

// ==================== System ====================

type SystemSyncStatusResp struct {
	ChainHeight int64               `json:"chainHeight"`
	Modules     []ModuleSyncStatus  `json:"modules"`
}

type ModuleSyncStatus struct {
	Module             string `json:"module"`
	LastScannedBlock   int64  `json:"lastScannedBlock"`
	LastConfirmedBlock int64  `json:"lastConfirmedBlock"`
	Lag                int64  `json:"lag"`
	Status             string `json:"status"`
}

type SystemReindexReq struct {
	Module    string `json:"module"`
	FromBlock int64  `json:"fromBlock"`
	ToBlock   int64  `json:"toBlock"`
}
