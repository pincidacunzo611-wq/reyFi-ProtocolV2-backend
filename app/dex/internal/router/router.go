// Package router 实现 Uniswap V2 最佳兑换路径搜索。
//
// 核心思路：将所有活跃交易对建模为无向图（代币为节点，交易对为边），
// 通过 BFS 搜索所有 ≤ maxHops 跳的路径，利用恒定乘积公式（x*y=k）
// 计算每条路径的实际输出金额，选出最优路径返回。
package router

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// ==================== 数据结构 ====================

// PairReserve 交易对储备量信息（从 DB 加载）
type PairReserve struct {
	PairAddress    string
	Token0Address  string
	Token1Address  string
	Token0Symbol   string
	Token1Symbol   string
	Token0Decimals int
	Token1Decimals int
	Reserve0       *big.Int // token0 储备量
	Reserve1       *big.Int // token1 储备量
	FeeBps         int      // 手续费（基点），默认 30 = 0.3%
}

// Edge 图中的一条有向边
type Edge struct {
	PairAddress string   // 交易对合约地址
	TokenIn     string   // 输入代币
	TokenOut    string   // 输出代币
	ReserveIn   *big.Int // tokenIn 侧的储备量
	ReserveOut  *big.Int // tokenOut 侧的储备量
	FeeBps      int      // 手续费基点
}

// Graph 代币关系图（邻接表）
type Graph struct {
	// adjacency[tokenAddress] = 从该代币出发的所有边
	adjacency map[string][]Edge
	// symbolMap[tokenAddress] = tokenSymbol
	symbolMap map[string]string
}

// Route 一条完整的兑换路径
type Route struct {
	Path        []string // 代币地址序列 [tokenIn, intermediate..., tokenOut]
	PathSymbols []string // 代币符号序列（便于前端展示）
	Pairs       []string // 经过的交易对地址
	AmountOut   *big.Int // 最终输出金额
	PriceImpact string   // 价格影响百分比（字符串，如 "0.85"）
	FeeBps      []int    // 每一跳的手续费
}

// Router 路由器
type Router struct {
	maxHops    int // 最大跳数（含端点之间的跳数）
	maxResults int // 最多返回的路径数
}

// ==================== 构造函数 ====================

// NewRouter 创建路由器
//   - maxHops: 最大跳数，推荐 3
//   - maxResults: 最多返回路径数，推荐 3
func NewRouter(maxHops, maxResults int) *Router {
	if maxHops <= 0 {
		maxHops = 3
	}
	if maxResults <= 0 {
		maxResults = 3
	}
	return &Router{
		maxHops:    maxHops,
		maxResults: maxResults,
	}
}

// ==================== 图构建 ====================

// BuildGraph 根据交易对和储备量数据构建代币关系图。
// 每个交易对生成两条有向边（A→B 和 B→A），因为 Uniswap V2 pair 支持双向兑换。
func BuildGraph(pairs []PairReserve) *Graph {
	g := &Graph{
		adjacency: make(map[string][]Edge),
		symbolMap: make(map[string]string),
	}

	for _, p := range pairs {
		// 跳过储备量为 0 的交易对（无流动性）
		if p.Reserve0 == nil || p.Reserve1 == nil {
			continue
		}
		if p.Reserve0.Sign() <= 0 || p.Reserve1.Sign() <= 0 {
			continue
		}

		t0 := strings.ToLower(p.Token0Address)
		t1 := strings.ToLower(p.Token1Address)

		// 记录符号映射
		if p.Token0Symbol != "" {
			g.symbolMap[t0] = p.Token0Symbol
		}
		if p.Token1Symbol != "" {
			g.symbolMap[t1] = p.Token1Symbol
		}

		// 正向边: token0 → token1
		g.adjacency[t0] = append(g.adjacency[t0], Edge{
			PairAddress: p.PairAddress,
			TokenIn:     t0,
			TokenOut:    t1,
			ReserveIn:   new(big.Int).Set(p.Reserve0),
			ReserveOut:  new(big.Int).Set(p.Reserve1),
			FeeBps:      p.FeeBps,
		})

		// 反向边: token1 → token0
		g.adjacency[t1] = append(g.adjacency[t1], Edge{
			PairAddress: p.PairAddress,
			TokenIn:     t1,
			TokenOut:    t0,
			ReserveIn:   new(big.Int).Set(p.Reserve1),
			ReserveOut:  new(big.Int).Set(p.Reserve0),
			FeeBps:      p.FeeBps,
		})
	}

	return g
}

// GetSymbol 获取代币符号
func (g *Graph) GetSymbol(tokenAddress string) string {
	if sym, ok := g.symbolMap[strings.ToLower(tokenAddress)]; ok {
		return sym
	}
	// 缩短地址作为 fallback
	addr := strings.ToLower(tokenAddress)
	if len(addr) > 10 {
		return addr[:6] + "..." + addr[len(addr)-4:]
	}
	return addr
}

// ==================== AMM 数学 ====================

// GetAmountOut 计算单跳输出金额（Uniswap V2 恒定乘积公式）。
//
// 公式:
//
//	amountInWithFee = amountIn * (10000 - feeBps)
//	numerator       = amountInWithFee * reserveOut
//	denominator     = reserveIn * 10000 + amountInWithFee
//	amountOut       = numerator / denominator
//
// 参数:
//   - amountIn: 输入金额
//   - reserveIn: 输入代币侧的储备量
//   - reserveOut: 输出代币侧的储备量
//   - feeBps: 手续费基点（如 30 表示 0.3%）
func GetAmountOut(amountIn, reserveIn, reserveOut *big.Int, feeBps int) (*big.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient input amount")
	}
	if reserveIn == nil || reserveIn.Sign() <= 0 || reserveOut == nil || reserveOut.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient liquidity: reserveIn=%v, reserveOut=%v", reserveIn, reserveOut)
	}

	// amountInWithFee = amountIn * (10000 - feeBps)
	feeMultiplier := big.NewInt(int64(10000 - feeBps))
	amountInWithFee := new(big.Int).Mul(amountIn, feeMultiplier)

	// numerator = amountInWithFee * reserveOut
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)

	// denominator = reserveIn * 10000 + amountInWithFee
	denominator := new(big.Int).Mul(reserveIn, big.NewInt(10000))
	denominator.Add(denominator, amountInWithFee)

	// amountOut = numerator / denominator
	amountOut := new(big.Int).Div(numerator, denominator)

	if amountOut.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient output amount")
	}

	return amountOut, nil
}

// GetAmountIn 计算要输出指定金额所需的输入金额（反向计算）。
//
// 公式:
//
//	numerator   = reserveIn * amountOut * 10000
//	denominator = (reserveOut - amountOut) * (10000 - feeBps)
//	amountIn    = numerator / denominator + 1
func GetAmountIn(amountOut, reserveIn, reserveOut *big.Int, feeBps int) (*big.Int, error) {
	if amountOut == nil || amountOut.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient output amount")
	}
	if reserveIn == nil || reserveIn.Sign() <= 0 || reserveOut == nil || reserveOut.Sign() <= 0 {
		return nil, fmt.Errorf("insufficient liquidity")
	}
	// 输出不能超过储备量
	if amountOut.Cmp(reserveOut) >= 0 {
		return nil, fmt.Errorf("insufficient liquidity: amountOut >= reserveOut")
	}

	// numerator = reserveIn * amountOut * 10000
	numerator := new(big.Int).Mul(reserveIn, amountOut)
	numerator.Mul(numerator, big.NewInt(10000))

	// denominator = (reserveOut - amountOut) * (10000 - feeBps)
	diff := new(big.Int).Sub(reserveOut, amountOut)
	feeMultiplier := big.NewInt(int64(10000 - feeBps))
	denominator := new(big.Int).Mul(diff, feeMultiplier)

	// amountIn = numerator / denominator + 1
	amountIn := new(big.Int).Div(numerator, denominator)
	amountIn.Add(amountIn, big.NewInt(1))

	return amountIn, nil
}

// ==================== 路径搜索 ====================

// searchState BFS 搜索状态
type searchState struct {
	currentToken string
	path         []string // 代币地址序列
	pairs        []string // 交易对地址序列
	feeBps       []int    // 每跳手续费
	amountOut    *big.Int // 到达当前节点时的累计输出
}

// FindBestRoute 搜索从 tokenIn 到 tokenOut 的最佳兑换路径。
//
// 使用 BFS 遍历所有 ≤ maxHops 跳的路径，对每条路径计算实际输出金额，
// 按输出金额降序排列，返回前 maxResults 条。
//
// 参数:
//   - graph: 代币关系图
//   - tokenIn: 输入代币地址
//   - tokenOut: 输出代币地址
//   - amountIn: 输入金额
//
// 返回:
//   - routes: 按输出金额降序排列的路径列表
//   - error: 找不到路径时返回错误
func (r *Router) FindBestRoute(graph *Graph, tokenIn, tokenOut string, amountIn *big.Int) ([]Route, error) {
	tokenIn = strings.ToLower(tokenIn)
	tokenOut = strings.ToLower(tokenOut)

	if tokenIn == tokenOut {
		return nil, fmt.Errorf("tokenIn and tokenOut must be different")
	}
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, fmt.Errorf("amountIn must be positive")
	}

	// 检查代币是否存在于图中
	if _, ok := graph.adjacency[tokenIn]; !ok {
		return nil, fmt.Errorf("tokenIn %s not found in any pair", tokenIn)
	}
	if _, ok := graph.adjacency[tokenOut]; !ok {
		return nil, fmt.Errorf("tokenOut %s not found in any pair", tokenOut)
	}

	// BFS 搜索所有可达路径
	var validRoutes []Route

	queue := []searchState{
		{
			currentToken: tokenIn,
			path:         []string{tokenIn},
			pairs:        nil,
			feeBps:       nil,
			amountOut:    new(big.Int).Set(amountIn),
		},
	}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		// 当前已走的跳数
		hops := len(state.path) - 1

		// 遍历当前代币的所有出边
		edges := graph.adjacency[state.currentToken]
		for _, edge := range edges {
			// 防止环路：不走访问过的代币
			if containsToken(state.path, edge.TokenOut) {
				continue
			}

			// 计算这一跳的输出
			hopAmountOut, err := GetAmountOut(state.amountOut, edge.ReserveIn, edge.ReserveOut, edge.FeeBps)
			if err != nil {
				// 流动性不足，跳过此边
				continue
			}

			newPath := append(copySlice(state.path), edge.TokenOut)
			newPairs := append(copySlice(state.pairs), edge.PairAddress)
			newFeeBps := append(copyIntSlice(state.feeBps), edge.FeeBps)

			// 如果到达目标代币，记录路径
			if edge.TokenOut == tokenOut {
				// 计算价格影响
				priceImpact := calculatePriceImpact(amountIn, hopAmountOut, graph, newPath)

				// 构造符号序列
				symbols := make([]string, len(newPath))
				for i, t := range newPath {
					symbols[i] = graph.GetSymbol(t)
				}

				validRoutes = append(validRoutes, Route{
					Path:        newPath,
					PathSymbols: symbols,
					Pairs:       newPairs,
					AmountOut:   hopAmountOut,
					PriceImpact: priceImpact,
					FeeBps:      newFeeBps,
				})
				continue
			}

			// 还没到达目标，如果未超过 maxHops 则继续搜索
			if hops+1 < r.maxHops {
				queue = append(queue, searchState{
					currentToken: edge.TokenOut,
					path:         newPath,
					pairs:        newPairs,
					feeBps:       newFeeBps,
					amountOut:    hopAmountOut,
				})
			}
		}
	}

	if len(validRoutes) == 0 {
		return nil, fmt.Errorf("no route found from %s to %s", tokenIn, tokenOut)
	}

	// 按 AmountOut 降序排列（输出越大越好）
	sort.Slice(validRoutes, func(i, j int) bool {
		return validRoutes[i].AmountOut.Cmp(validRoutes[j].AmountOut) > 0
	})

	// 限制返回数量
	if len(validRoutes) > r.maxResults {
		validRoutes = validRoutes[:r.maxResults]
	}

	return validRoutes, nil
}

// ==================== 价格影响计算 ====================

// calculatePriceImpact 计算价格影响。
//
// 价格影响 = 1 - (实际汇率 / 无滑点汇率)
// 无滑点汇率通过极小输入量的理论输出来近似。
func calculatePriceImpact(amountIn, actualAmountOut *big.Int, graph *Graph, path []string) string {
	if len(path) < 2 {
		return "0"
	}

	// 用一个极小量（1e6 wei）来近似"无滑点价格"
	tinyAmount := big.NewInt(1e6)
	tinyOut := new(big.Int).Set(tinyAmount)

	for i := 0; i < len(path)-1; i++ {
		edges := graph.adjacency[path[i]]
		found := false
		for _, edge := range edges {
			if edge.TokenOut == path[i+1] {
				out, err := GetAmountOut(tinyOut, edge.ReserveIn, edge.ReserveOut, edge.FeeBps)
				if err != nil {
					return "0"
				}
				tinyOut = out
				found = true
				break
			}
		}
		if !found {
			return "0"
		}
	}

	if tinyOut.Sign() <= 0 {
		return "0"
	}

	// idealRate = tinyOut / tinyAmount
	// actualRate = actualAmountOut / amountIn
	// priceImpact = 1 - actualRate / idealRate
	//             = 1 - (actualAmountOut * tinyAmount) / (amountIn * tinyOut)

	// 使用 10000 倍精度来避免浮点
	// impactBps = 10000 - (actualAmountOut * tinyAmount * 10000) / (amountIn * tinyOut)
	precision := big.NewInt(10000)

	numerator := new(big.Int).Mul(actualAmountOut, tinyAmount)
	numerator.Mul(numerator, precision)

	denominator := new(big.Int).Mul(amountIn, tinyOut)

	if denominator.Sign() <= 0 {
		return "0"
	}

	ratioBps := new(big.Int).Div(numerator, denominator)
	impactBps := new(big.Int).Sub(precision, ratioBps)

	// 如果影响为负（说明实际更好），取 0
	if impactBps.Sign() < 0 {
		impactBps = big.NewInt(0)
	}

	// 转换为百分比字符串（保留 2 位小数）
	// impactBps / 100 = 价格影响百分比
	whole := new(big.Int).Div(impactBps, big.NewInt(100))
	frac := new(big.Int).Mod(impactBps, big.NewInt(100))

	return fmt.Sprintf("%d.%02d", whole.Int64(), frac.Int64())
}

// ==================== 辅助函数 ====================

// containsToken 检查路径中是否已包含某代币
func containsToken(path []string, token string) bool {
	for _, t := range path {
		if t == token {
			return true
		}
	}
	return false
}

// copySlice 复制字符串切片（BFS 中避免共享底层数组）
func copySlice(s []string) []string {
	c := make([]string, len(s))
	copy(c, s)
	return c
}

// copyIntSlice 复制 int 切片
func copyIntSlice(s []int) []int {
	c := make([]int, len(s))
	copy(c, s)
	return c
}
