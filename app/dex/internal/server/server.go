// Package server DEX gRPC 服务实现
package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/reyfi/reyfi-backend/app/dex/internal/router"
	"github.com/reyfi/reyfi-backend/app/dex/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
)

// 注意: 这些类型在实际项目中由 protoc 自动生成。
// 这里手动定义以展示完整的业务逻辑。

// DexServiceServer DEX 服务实现
type DexServiceServer struct {
	svcCtx *svc.ServiceContext
}

func NewDexServiceServer(svcCtx *svc.ServiceContext) *DexServiceServer {
	return &DexServiceServer{svcCtx: svcCtx}
}

// RegisterDexServiceServer 注册 gRPC 服务（占位，实际由 protoc 生成）
func RegisterDexServiceServer(s *grpc.Server, srv *DexServiceServer) {
	// 实际项目中由 protoc 生成的 RegisterDexServiceServer 函数注册
	logx.Info("dex service registered (using manual registration)")
}

// ==================== 业务逻辑（等效于 logic 层）====================

// GetPairsReq / GetPairsResp 请求/响应结构
type GetPairsReq struct {
	Page      int64  `json:"page"`
	PageSize  int64  `json:"pageSize"`
	Keyword   string `json:"keyword"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

type PairInfo struct {
	PairAddress    string `json:"pairAddress"`
	Token0Address  string `json:"token0Address"`
	Token1Address  string `json:"token1Address"`
	Token0Symbol   string `json:"token0Symbol"`
	Token1Symbol   string `json:"token1Symbol"`
	Token0Decimals int    `json:"token0Decimals"`
	Token1Decimals int    `json:"token1Decimals"`
	Reserve0       string `json:"reserve0"`
	Reserve1       string `json:"reserve1"`
	Price          string `json:"price"`
	PriceChange24h string `json:"priceChange24h"`
	Volume24h      string `json:"volume24h"`
	Tvl            string `json:"tvl"`
	FeeApr         string `json:"feeApr"`
	FeeBps         int    `json:"feeBps"`
	IsActive       bool   `json:"isActive"`
}

type GetPairsResp struct {
	List  []*PairInfo `json:"list"`
	Total int64       `json:"total"`
}

// GetPairs 获取交易对列表
func (s *DexServiceServer) GetPairs(ctx context.Context, req *GetPairsReq) (*GetPairsResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 先尝试缓存
	cacheKey := fmt.Sprintf("dex:pairs:page:%d:%s", req.Page, req.Keyword)
	if cached, err := s.svcCtx.Redis.Get(cacheKey); err == nil && cached != "" {
		var resp GetPairsResp
		if json.Unmarshal([]byte(cached), &resp) == nil {
			return &resp, nil
		}
	}

	// 查询数据库
	pairs, total, err := s.svcCtx.PairModel.ListPairs(ctx, chainId, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*PairInfo, 0, len(pairs))
	for _, p := range pairs {
		info := &PairInfo{
			PairAddress:    p.PairAddress,
			Token0Address:  p.Token0Address,
			Token1Address:  p.Token1Address,
			Token0Symbol:   p.Token0Symbol,
			Token1Symbol:   p.Token1Symbol,
			Token0Decimals: p.Token0Decimals,
			Token1Decimals: p.Token1Decimals,
			FeeBps:         p.FeeBps,
			IsActive:       p.IsActive,
		}

		// 获取最新快照
		snap, err := s.svcCtx.PairSnapshotModel.GetLatest(ctx, chainId, p.PairAddress)
		if err == nil && snap != nil {
			info.Reserve0 = snap.Reserve0
			info.Reserve1 = snap.Reserve1
			info.Price = snap.Price0
			info.Tvl = snap.TvlUsd
		}

		// 查询 24h 成交量
		var volume24h sql.NullString
		s.svcCtx.DB.QueryRowContext(ctx,
			`SELECT COALESCE(SUM(amount_usd), '0') FROM dex_trades 
			 WHERE chain_id = ? AND pair_address = ? AND block_time >= ?`,
			chainId, p.PairAddress, time.Now().UTC().Add(-24*time.Hour),
		).Scan(&volume24h)
		if volume24h.Valid {
			info.Volume24h = volume24h.String
		}

		list = append(list, info)
	}

	resp := &GetPairsResp{List: list, Total: total}

	// 写缓存 (10 秒过期)
	if data, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(data), 10)
	}

	return resp, nil
}

// GetTrades 获取成交记录
type GetTradesReq struct {
	PairAddress string `json:"pairAddress"`
	Address     string `json:"address"`
	Page        int64  `json:"page"`
	PageSize    int64  `json:"pageSize"`
}

type TradeInfo struct {
	TxHash        string `json:"txHash"`
	PairAddress   string `json:"pairAddress"`
	TraderAddress string `json:"traderAddress"`
	Direction     string `json:"direction"`
	Amount0       string `json:"amount0"`
	Amount1       string `json:"amount1"`
	AmountUsd     string `json:"amountUsd"`
	Price         string `json:"price"`
	BlockTime     string `json:"blockTime"`
}

type GetTradesResp struct {
	List  []*TradeInfo `json:"list"`
	Total int64        `json:"total"`
}

func (s *DexServiceServer) GetTrades(ctx context.Context, req *GetTradesReq) (*GetTradesResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	trades, total, err := s.svcCtx.TradeModel.ListByPair(ctx, chainId, req.PairAddress, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	list := make([]*TradeInfo, 0, len(trades))
	for _, t := range trades {
		// 确定展示金额
		amount0 := t.Amount0In
		amount1 := t.Amount1Out
		if t.Direction == "sell" {
			amount0 = t.Amount0Out
			amount1 = t.Amount1In
		}

		list = append(list, &TradeInfo{
			TxHash:        t.TxHash,
			PairAddress:   t.PairAddress,
			TraderAddress: t.TraderAddress,
			Direction:     t.Direction,
			Amount0:       amount0,
			Amount1:       amount1,
			AmountUsd:     t.AmountUsd,
			Price:         t.Price,
			BlockTime:     t.BlockTime.UTC().Format(time.RFC3339),
		})
	}

	return &GetTradesResp{List: list, Total: total}, nil
}

// GetOverview DEX 概览
type GetOverviewResp struct {
	TotalTvl       string      `json:"totalTvl"`
	TotalVolume24h string      `json:"totalVolume24h"`
	TotalFees24h   string      `json:"totalFees24h"`
	TotalPairs     int64       `json:"totalPairs"`
	TopPairs       []*PairInfo `json:"topPairs"`
}

// ==================== 最佳兑换路径 ====================

// FindBestRouteReq 路由查询请求
type FindBestRouteReq struct {
	TokenIn    string `json:"tokenIn"`
	TokenOut   string `json:"tokenOut"`
	AmountIn   string `json:"amountIn"`
	MaxHops    int    `json:"maxHops"`
	MaxResults int    `json:"maxResults"`
}

// RouteInfo 单条路由信息
type RouteInfo struct {
	Path        []string `json:"path"`
	Pairs       []string `json:"pairs"`
	PathSymbols []string `json:"pathSymbols"`
	AmountOut   string   `json:"amountOut"`
	PriceImpact string   `json:"priceImpact"`
	FeeBps      []int    `json:"feeBps"`
}

// FindBestRouteResp 路由查询响应
type FindBestRouteResp struct {
	BestRoute    *RouteInfo   `json:"bestRoute"`
	Alternatives []*RouteInfo `json:"alternatives"`
	Rate         string       `json:"rate"`
}

// FindBestRoute 搜索最佳兑换路径
func (s *DexServiceServer) FindBestRoute(ctx context.Context, req *FindBestRouteReq) (*FindBestRouteResp, error) {
	// 参数校验
	if req.TokenIn == "" || req.TokenOut == "" || req.AmountIn == "" {
		return nil, fmt.Errorf("tokenIn, tokenOut, amountIn are required")
	}

	amountIn := new(big.Int)
	if _, ok := amountIn.SetString(req.AmountIn, 10); !ok {
		return nil, fmt.Errorf("invalid amountIn: %s", req.AmountIn)
	}

	maxHops := req.MaxHops
	if maxHops <= 0 {
		maxHops = 3
	}
	maxResults := req.MaxResults
	if maxResults <= 0 {
		maxResults = 3
	}

	// 尝试缓存（短 TTL，因为储备量变化频繁）
	cacheKey := fmt.Sprintf("dex:route:%s:%s:%s", req.TokenIn, req.TokenOut, req.AmountIn)
	if cached, err := s.svcCtx.Redis.Get(cacheKey); err == nil && cached != "" {
		var resp FindBestRouteResp
		if json.Unmarshal([]byte(cached), &resp) == nil {
			return &resp, nil
		}
	}

	// 1. 构建代币关系图
	graph, err := s.buildTokenGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("build token graph: %w", err)
	}

	// 2. 使用路由器搜索最佳路径
	pathRouter := router.NewRouter(maxHops, maxResults)
	routes, err := pathRouter.FindBestRoute(graph, req.TokenIn, req.TokenOut, amountIn)
	if err != nil {
		return nil, fmt.Errorf("find route: %w", err)
	}

	// 3. 组装响应
	resp := &FindBestRouteResp{}

	// 最佳路径
	best := routes[0]
	resp.BestRoute = &RouteInfo{
		Path:        best.Path,
		Pairs:       best.Pairs,
		PathSymbols: best.PathSymbols,
		AmountOut:   best.AmountOut.String(),
		PriceImpact: best.PriceImpact,
		FeeBps:      best.FeeBps,
	}

	// 计算汇率: amountOut / amountIn（使用浮点近似）
	amountOutFloat := new(big.Float).SetInt(best.AmountOut)
	amountInFloat := new(big.Float).SetInt(amountIn)
	if amountInFloat.Sign() > 0 {
		rate := new(big.Float).Quo(amountOutFloat, amountInFloat)
		resp.Rate = rate.Text('f', 8)
	}

	// 备选路径
	if len(routes) > 1 {
		alternatives := make([]*RouteInfo, 0, len(routes)-1)
		for _, r := range routes[1:] {
			alternatives = append(alternatives, &RouteInfo{
				Path:        r.Path,
				Pairs:       r.Pairs,
				PathSymbols: r.PathSymbols,
				AmountOut:   r.AmountOut.String(),
				PriceImpact: r.PriceImpact,
				FeeBps:      r.FeeBps,
			})
		}
		resp.Alternatives = alternatives
	}

	// 缓存 10 秒
	if data, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(data), 10)
	}

	logx.Infof("route found: %s -> %s, path=%v, amountOut=%s",
		req.TokenIn, req.TokenOut, best.PathSymbols, best.AmountOut.String())

	return resp, nil
}

// buildTokenGraph 从数据库加载所有活跃交易对和储备量，构建代币关系图
func (s *DexServiceServer) buildTokenGraph(ctx context.Context) (*router.Graph, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 加载所有活跃交易对
	pairs, err := s.svcCtx.PairModel.ListAllActivePairs(ctx, chainId)
	if err != nil {
		return nil, fmt.Errorf("list active pairs: %w", err)
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("no active pairs found")
	}

	// 收集 pair 地址，批量拉取快照
	pairAddresses := make([]string, len(pairs))
	for i, p := range pairs {
		pairAddresses[i] = p.PairAddress
	}

	snapshots, err := s.svcCtx.PairSnapshotModel.GetLatestSnapshots(ctx, chainId, pairAddresses)
	if err != nil {
		return nil, fmt.Errorf("get snapshots: %w", err)
	}

	// 组装路由器所需的 PairReserve 数据
	var pairReserves []router.PairReserve
	for _, p := range pairs {
		snap, ok := snapshots[p.PairAddress]
		if !ok {
			continue // 无快照的交易对跳过
		}

		reserve0 := new(big.Int)
		reserve1 := new(big.Int)
		if _, ok := reserve0.SetString(snap.Reserve0, 10); !ok {
			continue
		}
		if _, ok := reserve1.SetString(snap.Reserve1, 10); !ok {
			continue
		}

		pairReserves = append(pairReserves, router.PairReserve{
			PairAddress:    p.PairAddress,
			Token0Address:  p.Token0Address,
			Token1Address:  p.Token1Address,
			Token0Symbol:   p.Token0Symbol,
			Token1Symbol:   p.Token1Symbol,
			Token0Decimals: p.Token0Decimals,
			Token1Decimals: p.Token1Decimals,
			Reserve0:       reserve0,
			Reserve1:       reserve1,
			FeeBps:         p.FeeBps,
		})
	}

	if len(pairReserves) == 0 {
		return nil, fmt.Errorf("no pairs with valid reserves")
	}

	return router.BuildGraph(pairReserves), nil
}

// ==================== GetPairDetail ====================

type GetPairDetailResp struct {
	Pair      *PairInfo `json:"pair"`
	Volume7d  string    `json:"volume7d"`
	Fees24h   string    `json:"fees24h"`
	Fees7d    string    `json:"fees7d"`
	High24h   string    `json:"high24h"`
	Low24h    string    `json:"low24h"`
}

func (s *DexServiceServer) GetPairDetail(ctx context.Context, pairAddress string) (*GetPairDetailResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	pair, err := s.svcCtx.PairModel.FindByAddress(ctx, chainId, pairAddress)
	if err != nil {
		return nil, fmt.Errorf("pair not found: %w", err)
	}

	info := &PairInfo{
		PairAddress:    pair.PairAddress,
		Token0Address:  pair.Token0Address,
		Token1Address:  pair.Token1Address,
		Token0Symbol:   pair.Token0Symbol,
		Token1Symbol:   pair.Token1Symbol,
		Token0Decimals: pair.Token0Decimals,
		Token1Decimals: pair.Token1Decimals,
		FeeBps:         pair.FeeBps,
		IsActive:       pair.IsActive,
	}

	// 最新快照
	snap, err := s.svcCtx.PairSnapshotModel.GetLatest(ctx, chainId, pairAddress)
	if err == nil && snap != nil {
		info.Reserve0 = snap.Reserve0
		info.Reserve1 = snap.Reserve1
		info.Price = snap.Price0
		info.Tvl = snap.TvlUsd
	}

	now := time.Now().UTC()

	// 24h 成交量
	var vol24h sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(amount_usd AS DECIMAL(65,18))), 0) FROM dex_trades 
		 WHERE chain_id = ? AND pair_address = ? AND block_time >= ?`,
		chainId, pairAddress, now.Add(-24*time.Hour)).Scan(&vol24h)
	if vol24h.Valid {
		info.Volume24h = vol24h.String
	}

	// 7d 成交量
	var vol7d sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(amount_usd AS DECIMAL(65,18))), 0) FROM dex_trades 
		 WHERE chain_id = ? AND pair_address = ? AND block_time >= ?`,
		chainId, pairAddress, now.Add(-7*24*time.Hour)).Scan(&vol7d)

	// 24h/7d 手续费 = 成交量 * feeBps / 10000
	fees24h := "0"
	fees7d := "0"
	if vol24h.Valid {
		v := new(big.Float)
		if _, ok := v.SetString(vol24h.String); ok {
			fee := new(big.Float).Mul(v, big.NewFloat(float64(pair.FeeBps)))
			fee.Quo(fee, big.NewFloat(10000))
			fees24h = fee.Text('f', 6)
		}
	}
	if vol7d.Valid {
		v := new(big.Float)
		if _, ok := v.SetString(vol7d.String); ok {
			fee := new(big.Float).Mul(v, big.NewFloat(float64(pair.FeeBps)))
			fee.Quo(fee, big.NewFloat(10000))
			fees7d = fee.Text('f', 6)
		}
	}

	// 24h 最高/最低价
	var high24h, low24h sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(CAST(price AS DECIMAL(65,18))), 0), 
		        COALESCE(MIN(CAST(price AS DECIMAL(65,18))), 0)
		 FROM dex_trades WHERE chain_id = ? AND pair_address = ? AND block_time >= ? AND price != '0'`,
		chainId, pairAddress, now.Add(-24*time.Hour)).Scan(&high24h, &low24h)

	resp := &GetPairDetailResp{
		Pair:    info,
		Fees24h: fees24h,
		Fees7d:  fees7d,
	}
	if vol7d.Valid {
		resp.Volume7d = vol7d.String
	}
	if high24h.Valid {
		resp.High24h = high24h.String
	}
	if low24h.Valid {
		resp.Low24h = low24h.String
	}

	return resp, nil
}

// ==================== GetCandles K 线聚合 ====================

type CandleInfo struct {
	Time   int64  `json:"time"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
}

type GetCandlesResp struct {
	Candles []*CandleInfo `json:"candles"`
}

func (s *DexServiceServer) GetCandles(ctx context.Context, pairAddress, interval string, from, to int64) (*GetCandlesResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 将 interval 转换为秒
	intervalSec := int64(3600) // 默认 1h
	switch interval {
	case "1m":
		intervalSec = 60
	case "5m":
		intervalSec = 300
	case "15m":
		intervalSec = 900
	case "1h":
		intervalSec = 3600
	case "4h":
		intervalSec = 14400
	case "1d":
		intervalSec = 86400
	}

	if from <= 0 {
		from = time.Now().UTC().Add(-24 * time.Hour).Unix()
	}
	if to <= 0 {
		to = time.Now().UTC().Unix()
	}

	// 使用 SQL 聚合 K 线数据
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT 
			FLOOR(UNIX_TIMESTAMP(block_time) / ?) * ? AS candle_time,
			SUBSTRING_INDEX(GROUP_CONCAT(price ORDER BY block_time ASC), ',', 1) AS open_price,
			MAX(CAST(price AS DECIMAL(65,18))) AS high_price,
			MIN(CASE WHEN price != '0' THEN CAST(price AS DECIMAL(65,18)) ELSE NULL END) AS low_price,
			SUBSTRING_INDEX(GROUP_CONCAT(price ORDER BY block_time DESC), ',', 1) AS close_price,
			COALESCE(SUM(CAST(amount_usd AS DECIMAL(65,18))), 0) AS volume
		FROM dex_trades
		WHERE chain_id = ? AND pair_address = ? 
			AND block_time >= FROM_UNIXTIME(?) AND block_time <= FROM_UNIXTIME(?)
			AND price != '0'
		GROUP BY candle_time
		ORDER BY candle_time ASC`,
		intervalSec, intervalSec, chainId, pairAddress, from, to)
	if err != nil {
		return nil, fmt.Errorf("query candles: %w", err)
	}
	defer rows.Close()

	var candles []*CandleInfo
	for rows.Next() {
		c := &CandleInfo{}
		var high, low sql.NullString
		if err := rows.Scan(&c.Time, &c.Open, &high, &low, &c.Close, &c.Volume); err != nil {
			return nil, err
		}
		if high.Valid {
			c.High = high.String
		}
		if low.Valid {
			c.Low = low.String
		} else {
			c.Low = c.Open
		}
		candles = append(candles, c)
	}

	return &GetCandlesResp{Candles: candles}, nil
}

// ==================== GetPositions (LP 仓位) ====================

type LPPosition struct {
	PairAddress   string `json:"pairAddress"`
	Token0Symbol  string `json:"token0Symbol"`
	Token1Symbol  string `json:"token1Symbol"`
	LpBalance     string `json:"lpBalance"`
	ShareRatio    string `json:"shareRatio"`
	ValueUsd      string `json:"valueUsd"`
	PendingReward string `json:"pendingReward"`
}

type GetPositionsResp struct {
	List []*LPPosition `json:"list"`
}

func (s *DexServiceServer) GetPositions(ctx context.Context, userAddress string) (*GetPositionsResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 从流动性事件聚合用户的 LP 净头寸
	rows, err := s.svcCtx.DB.QueryContext(ctx,
		`SELECT le.pair_address,
			p.token0_symbol, p.token1_symbol,
			SUM(CASE WHEN le.event_type = 'mint' THEN CAST(le.amount0 AS DECIMAL(65,18)) ELSE -CAST(le.amount0 AS DECIMAL(65,18)) END) AS net_amount0,
			SUM(CASE WHEN le.event_type = 'mint' THEN CAST(le.amount1 AS DECIMAL(65,18)) ELSE -CAST(le.amount1 AS DECIMAL(65,18)) END) AS net_amount1
		FROM dex_liquidity_events le
		INNER JOIN dex_pairs p ON p.chain_id = le.chain_id AND p.pair_address = le.pair_address
		WHERE le.chain_id = ? AND le.user_address = ?
		GROUP BY le.pair_address, p.token0_symbol, p.token1_symbol
		HAVING net_amount0 > 0 OR net_amount1 > 0`,
		chainId, userAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*LPPosition
	for rows.Next() {
		pos := &LPPosition{}
		var netAmount0, netAmount1 string
		if err := rows.Scan(&pos.PairAddress, &pos.Token0Symbol, &pos.Token1Symbol, &netAmount0, &netAmount1); err != nil {
			return nil, err
		}
		pos.LpBalance = "0"
		pos.ShareRatio = "0"
		pos.ValueUsd = "0"
		pos.PendingReward = "0"
		list = append(list, pos)
	}

	return &GetPositionsResp{List: list}, nil
}

// ==================== BuildSwap 构建 Swap 交易 ====================

type BuildSwapReq struct {
	UserAddress string `json:"userAddress"`
	TokenIn     string `json:"tokenIn"`
	TokenOut    string `json:"tokenOut"`
	AmountIn    string `json:"amountIn"`
	SlippageBps int    `json:"slippageBps"`
	Receiver    string `json:"receiver"`
}

type BuildSwapResp struct {
	To             string   `json:"to"`
	Value          string   `json:"value"`
	Data           string   `json:"data"`
	GasLimit       string   `json:"gasLimit"`
	MinAmountOut   string   `json:"minAmountOut"`
	PriceImpact    string   `json:"priceImpact"`
	Route          []string `json:"route"`
	EstimatedGasUsd string  `json:"estimatedGasUsd"`
}

func (s *DexServiceServer) BuildSwap(ctx context.Context, req *BuildSwapReq) (*BuildSwapResp, error) {
	if req.TokenIn == "" || req.TokenOut == "" || req.AmountIn == "" {
		return nil, fmt.Errorf("tokenIn, tokenOut, amountIn are required")
	}

	amountIn := new(big.Int)
	if _, ok := amountIn.SetString(req.AmountIn, 10); !ok {
		return nil, fmt.Errorf("invalid amountIn: %s", req.AmountIn)
	}

	if req.SlippageBps <= 0 {
		req.SlippageBps = 50 // 默认 0.5%
	}

	// 1. 查找最佳路由
	graph, err := s.buildTokenGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("build token graph: %w", err)
	}

	pathRouter := router.NewRouter(3, 1)
	routes, err := pathRouter.FindBestRoute(graph, req.TokenIn, req.TokenOut, amountIn)
	if err != nil {
		return nil, fmt.Errorf("no route found: %w", err)
	}

	best := routes[0]

	// 2. 计算最小输出（应用滑点保护）
	// minAmountOut = amountOut * (10000 - slippageBps) / 10000
	slippageFactor := big.NewInt(int64(10000 - req.SlippageBps))
	minOut := new(big.Int).Mul(best.AmountOut, slippageFactor)
	minOut.Div(minOut, big.NewInt(10000))

	// 3. 构造合约调用数据
	// Router 合约函数签名: swapExactTokensForTokens(uint256,uint256,address[],address,uint256)
	// function selector = keccak256("swapExactTokensForTokens(uint256,uint256,address[],address,uint256)")[:4]
	routerAddr := s.svcCtx.Config.Chain.Contracts["router"]
	if routerAddr == "" {
		routerAddr = s.svcCtx.Config.Chain.Contracts["Router"]
	}

	receiver := req.Receiver
	if receiver == "" {
		receiver = req.UserAddress
	}

	deadline := time.Now().UTC().Add(20 * time.Minute).Unix()

	// 构建 ABI 编码的 calldata
	// 简化版: 返回参数让前端使用 ethers.js 进行编码
	resp := &BuildSwapResp{
		To:              routerAddr,
		Value:           "0",
		GasLimit:        fmt.Sprintf("%d", 200000+50000*len(best.Path)),
		MinAmountOut:    minOut.String(),
		PriceImpact:     best.PriceImpact,
		Route:           best.Path,
		EstimatedGasUsd: "0",
		Data: fmt.Sprintf(`{"method":"swapExactTokensForTokens","params":{"amountIn":"%s","amountOutMin":"%s","path":%s,"to":"%s","deadline":%d}}`,
			amountIn.String(), minOut.String(), toJSONArray(best.Path), receiver, deadline),
	}

	logx.Infof("swap built: %s -> %s, amountIn=%s, minOut=%s, path=%v",
		req.TokenIn, req.TokenOut, req.AmountIn, minOut.String(), best.PathSymbols)

	return resp, nil
}

// toJSONArray 将字符串切片转换为 JSON 数组字符串
func toJSONArray(arr []string) string {
	data, _ := json.Marshal(arr)
	return string(data)
}

// GetOverview DEX 概览
func (s *DexServiceServer) GetOverview(ctx context.Context) (*GetOverviewResp, error) {
	chainId := s.svcCtx.Config.Chain.ChainId

	// 缓存
	cacheKey := "dex:overview"
	if cached, err := s.svcCtx.Redis.Get(cacheKey); err == nil && cached != "" {
		var resp GetOverviewResp
		if json.Unmarshal([]byte(cached), &resp) == nil {
			return &resp, nil
		}
	}

	// 总交易对数
	var totalPairs int64
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dex_pairs WHERE chain_id = ? AND is_active = 1`,
		chainId).Scan(&totalPairs)

	// 24h 总成交量
	var totalVolume sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(amount_usd AS DECIMAL(65,18))), 0) FROM dex_trades 
		 WHERE chain_id = ? AND block_time >= ?`,
		chainId, time.Now().UTC().Add(-24*time.Hour)).Scan(&totalVolume)

	// 总 TVL = 所有活跃交易对最新快照的 tvl_usd 之和
	var totalTvl sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(s.tvl_usd AS DECIMAL(65,18))), 0)
		 FROM dex_pair_snapshots s
		 INNER JOIN (
			SELECT pair_address, MAX(id) AS max_id
			FROM dex_pair_snapshots WHERE chain_id = ?
			GROUP BY pair_address
		 ) latest ON s.id = latest.max_id`,
		chainId).Scan(&totalTvl)

	// 24h 总手续费（通过交易量和平均费率估算）
	var totalFees sql.NullString
	s.svcCtx.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(CAST(t.amount_usd AS DECIMAL(65,18)) * p.fee_bps / 10000), 0)
		 FROM dex_trades t
		 INNER JOIN dex_pairs p ON p.chain_id = t.chain_id AND p.pair_address = t.pair_address
		 WHERE t.chain_id = ? AND t.block_time >= ?`,
		chainId, time.Now().UTC().Add(-24*time.Hour)).Scan(&totalFees)

	resp := &GetOverviewResp{
		TotalPairs:     totalPairs,
	}
	if totalTvl.Valid {
		resp.TotalTvl = totalTvl.String
	}
	if totalVolume.Valid {
		resp.TotalVolume24h = totalVolume.String
	}
	if totalFees.Valid {
		resp.TotalFees24h = totalFees.String
	}

	// 缓存 15 秒
	if data, err := json.Marshal(resp); err == nil {
		s.svcCtx.Redis.Setex(cacheKey, string(data), 15)
	}

	return resp, nil
}
