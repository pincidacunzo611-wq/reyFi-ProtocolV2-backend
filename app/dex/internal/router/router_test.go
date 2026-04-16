package router

import (
	"math/big"
	"testing"
)

// ==================== GetAmountOut 单元测试 ====================

func TestGetAmountOut_Basic(t *testing.T) {
	// 经典场景：储备量 1000/2000，输入 10，0.3% 手续费
	amountIn := big.NewInt(10)
	reserveIn := big.NewInt(1000)
	reserveOut := big.NewInt(2000)
	feeBps := 30

	out, err := GetAmountOut(amountIn, reserveIn, reserveOut, feeBps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 手工计算:
	// amountInWithFee = 10 * 9970 = 99700
	// numerator = 99700 * 2000 = 199400000
	// denominator = 1000 * 10000 + 99700 = 10099700
	// amountOut = 199400000 / 10099700 = 19 (整数除法)
	expected := big.NewInt(19)
	if out.Cmp(expected) != 0 {
		t.Errorf("expected %s, got %s", expected.String(), out.String())
	}
}

func TestGetAmountOut_ZeroInput(t *testing.T) {
	_, err := GetAmountOut(big.NewInt(0), big.NewInt(1000), big.NewInt(2000), 30)
	if err == nil {
		t.Error("expected error for zero input")
	}
}

func TestGetAmountOut_NilInput(t *testing.T) {
	_, err := GetAmountOut(nil, big.NewInt(1000), big.NewInt(2000), 30)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestGetAmountOut_ZeroReserve(t *testing.T) {
	_, err := GetAmountOut(big.NewInt(10), big.NewInt(0), big.NewInt(2000), 30)
	if err == nil {
		t.Error("expected error for zero reserveIn")
	}
}

func TestGetAmountOut_LargeInput(t *testing.T) {
	// 大数场景：18 位小数的代币
	amountIn, _ := new(big.Int).SetString("1000000000000000000", 10) // 1e18
	reserveIn, _ := new(big.Int).SetString("100000000000000000000", 10) // 100e18
	reserveOut, _ := new(big.Int).SetString("200000000000000000000", 10) // 200e18

	out, err := GetAmountOut(amountIn, reserveIn, reserveOut, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Sign() <= 0 {
		t.Error("expected positive output")
	}
	// 输出应该接近但略少于 2e18（因为手续费和滑点）
	twoTokens, _ := new(big.Int).SetString("2000000000000000000", 10)
	if out.Cmp(twoTokens) >= 0 {
		t.Error("output should be less than ideal rate due to fees/slippage")
	}
	t.Logf("1 token in (from 100/200 pool) -> %s out", out.String())
}

// ==================== GetAmountIn 单元测试 ====================

func TestGetAmountIn_Basic(t *testing.T) {
	amountOut := big.NewInt(19)
	reserveIn := big.NewInt(1000)
	reserveOut := big.NewInt(2000)
	feeBps := 30

	in, err := GetAmountIn(amountOut, reserveIn, reserveOut, feeBps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 反向验证：用得到的 amountIn 计算 amountOut，应该 >= 19
	out, err := GetAmountOut(in, reserveIn, reserveOut, feeBps)
	if err != nil {
		t.Fatalf("unexpected error on reverse: %v", err)
	}
	if out.Cmp(amountOut) < 0 {
		t.Errorf("reverse check failed: in=%s gives out=%s, expected >= %s", in.String(), out.String(), amountOut.String())
	}
}

func TestGetAmountIn_ExceedsReserve(t *testing.T) {
	// 要求输出超过储备量
	_, err := GetAmountIn(big.NewInt(3000), big.NewInt(1000), big.NewInt(2000), 30)
	if err == nil {
		t.Error("expected error when amountOut >= reserveOut")
	}
}

// ==================== 图构建测试 ====================

func buildTestGraph() *Graph {
	// 构建测试图：
	// A <-> B (pair AB, reserves 1000/2000)
	// B <-> C (pair BC, reserves 500/800)
	// A <-> D (pair AD, reserves 3000/1500)
	// D <-> C (pair DC, reserves 600/900)
	pairs := []PairReserve{
		{
			PairAddress:   "0xpairAB",
			Token0Address: "0xTokenA",
			Token1Address: "0xTokenB",
			Token0Symbol:  "TKA",
			Token1Symbol:  "TKB",
			Reserve0:      big.NewInt(1000),
			Reserve1:      big.NewInt(2000),
			FeeBps:        30,
		},
		{
			PairAddress:   "0xpairBC",
			Token0Address: "0xTokenB",
			Token1Address: "0xTokenC",
			Token0Symbol:  "TKB",
			Token1Symbol:  "TKC",
			Reserve0:      big.NewInt(500),
			Reserve1:      big.NewInt(800),
			FeeBps:        30,
		},
		{
			PairAddress:   "0xpairAD",
			Token0Address: "0xTokenA",
			Token1Address: "0xTokenD",
			Token0Symbol:  "TKA",
			Token1Symbol:  "TKD",
			Reserve0:      big.NewInt(3000),
			Reserve1:      big.NewInt(1500),
			FeeBps:        30,
		},
		{
			PairAddress:   "0xpairDC",
			Token0Address: "0xTokenD",
			Token1Address: "0xTokenC",
			Token0Symbol:  "TKD",
			Token1Symbol:  "TKC",
			Reserve0:      big.NewInt(600),
			Reserve1:      big.NewInt(900),
			FeeBps:        30,
		},
	}
	return BuildGraph(pairs)
}

func TestBuildGraph(t *testing.T) {
	g := buildTestGraph()

	// 检查所有代币都出现在邻接表中
	tokens := []string{"0xtokena", "0xtokenb", "0xtokenc", "0xtokend"}
	for _, tk := range tokens {
		if _, ok := g.adjacency[tk]; !ok {
			t.Errorf("token %s not found in graph", tk)
		}
	}

	// Token A 应该有 2 条出边（到 B 和 D）
	if len(g.adjacency["0xtokena"]) != 2 {
		t.Errorf("expected 2 edges from token A, got %d", len(g.adjacency["0xtokena"]))
	}
}

func TestBuildGraph_SkipZeroReserve(t *testing.T) {
	pairs := []PairReserve{
		{
			PairAddress:   "0xpairXY",
			Token0Address: "0xTokenX",
			Token1Address: "0xTokenY",
			Reserve0:      big.NewInt(0),
			Reserve1:      big.NewInt(100),
			FeeBps:        30,
		},
	}
	g := BuildGraph(pairs)

	if len(g.adjacency) != 0 {
		t.Error("expected empty graph for zero-reserve pair")
	}
}

// ==================== 路由搜索测试 ====================

func TestFindBestRoute_DirectPath(t *testing.T) {
	// A -> B 直接路径
	g := buildTestGraph()
	r := NewRouter(3, 3)

	routes, err := r.FindBestRoute(g, "0xTokenA", "0xTokenB", big.NewInt(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least one route")
	}

	best := routes[0]
	t.Logf("Best route A->B: path=%v, amountOut=%s, impact=%s", best.PathSymbols, best.AmountOut.String(), best.PriceImpact)

	// 直接路径应该是 [A, B]
	if len(best.Path) != 2 {
		t.Errorf("expected direct path of length 2, got %d", len(best.Path))
	}
}

func TestFindBestRoute_MultiHop(t *testing.T) {
	// A -> C 没有直接交易对，需要通过 B 或 D 中转
	g := buildTestGraph()
	r := NewRouter(3, 3)

	routes, err := r.FindBestRoute(g, "0xTokenA", "0xTokenC", big.NewInt(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least one route")
	}

	// 应该找到两条路径：A->B->C 和 A->D->C
	if len(routes) < 2 {
		t.Errorf("expected at least 2 routes, got %d", len(routes))
	}

	for i, route := range routes {
		t.Logf("Route %d: path=%v, amountOut=%s, impact=%s", i+1, route.PathSymbols, route.AmountOut.String(), route.PriceImpact)
	}

	// 第一条应该是输出最大的
	if len(routes) >= 2 {
		if routes[0].AmountOut.Cmp(routes[1].AmountOut) < 0 {
			t.Error("routes should be sorted by amountOut descending")
		}
	}
}

func TestFindBestRoute_NoPath(t *testing.T) {
	// E 不存在于图中
	g := buildTestGraph()
	r := NewRouter(3, 3)

	_, err := r.FindBestRoute(g, "0xTokenA", "0xTokenE", big.NewInt(100))
	if err == nil {
		t.Error("expected error for unreachable token")
	}
}

func TestFindBestRoute_SameToken(t *testing.T) {
	g := buildTestGraph()
	r := NewRouter(3, 3)

	_, err := r.FindBestRoute(g, "0xTokenA", "0xTokenA", big.NewInt(100))
	if err == nil {
		t.Error("expected error for same tokenIn and tokenOut")
	}
}

func TestFindBestRoute_MaxHopsLimit(t *testing.T) {
	// 设置 maxHops=1，只能走直接路径
	g := buildTestGraph()
	r := NewRouter(1, 3)

	// A->C 没有直接对，应该找不到路径
	_, err := r.FindBestRoute(g, "0xTokenA", "0xTokenC", big.NewInt(100))
	if err == nil {
		t.Error("expected error for no direct path with maxHops=1")
	}

	// A->B 有直接对，应该能找到
	routes, err := r.FindBestRoute(g, "0xTokenA", "0xTokenB", big.NewInt(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) == 0 {
		t.Fatal("expected at least one route for direct pair")
	}
}

func TestFindBestRoute_NoCycle(t *testing.T) {
	// 确保路径中不会出现环路
	g := buildTestGraph()
	r := NewRouter(4, 10) // 放宽限制

	routes, err := r.FindBestRoute(g, "0xTokenA", "0xTokenC", big.NewInt(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, route := range routes {
		seen := make(map[string]bool)
		for _, token := range route.Path {
			if seen[token] {
				t.Errorf("cycle detected in route: %v", route.Path)
			}
			seen[token] = true
		}
	}
}

func TestFindBestRoute_ThreeHops(t *testing.T) {
	// 构建需要三跳的场景
	// A <-> B, B <-> C, C <-> E (但没有 A<->E 或 B<->E 的直接路径)
	pairs := []PairReserve{
		{
			PairAddress:   "0xpairAB",
			Token0Address: "0xA",
			Token1Address: "0xB",
			Token0Symbol:  "A",
			Token1Symbol:  "B",
			Reserve0:      big.NewInt(10000),
			Reserve1:      big.NewInt(20000),
			FeeBps:        30,
		},
		{
			PairAddress:   "0xpairBC",
			Token0Address: "0xB",
			Token1Address: "0xC",
			Token0Symbol:  "B",
			Token1Symbol:  "C",
			Reserve0:      big.NewInt(15000),
			Reserve1:      big.NewInt(30000),
			FeeBps:        30,
		},
		{
			PairAddress:   "0xpairCE",
			Token0Address: "0xC",
			Token1Address: "0xE",
			Token0Symbol:  "C",
			Token1Symbol:  "E",
			Reserve0:      big.NewInt(25000),
			Reserve1:      big.NewInt(50000),
			FeeBps:        30,
		},
	}
	g := BuildGraph(pairs)
	r := NewRouter(3, 3)

	routes, err := r.FindBestRoute(g, "0xA", "0xE", big.NewInt(1000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(routes) == 0 {
		t.Fatal("expected to find 3-hop route")
	}

	best := routes[0]
	if len(best.Path) != 4 { // A -> B -> C -> E
		t.Errorf("expected 4 tokens in path (3 hops), got %d: %v", len(best.Path), best.PathSymbols)
	}
	t.Logf("3-hop route: %v, amountOut=%s, impact=%s", best.PathSymbols, best.AmountOut.String(), best.PriceImpact)
}

// ==================== 辅助函数测试 ====================

func TestContainsToken(t *testing.T) {
	path := []string{"a", "b", "c"}
	if !containsToken(path, "b") {
		t.Error("expected true for existing token")
	}
	if containsToken(path, "d") {
		t.Error("expected false for non-existing token")
	}
}

func TestGetSymbol(t *testing.T) {
	g := buildTestGraph()
	if sym := g.GetSymbol("0xTokenA"); sym != "TKA" {
		t.Errorf("expected TKA, got %s", sym)
	}
	// 未知代币应该返回缩短地址
	sym := g.GetSymbol("0x1234567890abcdef")
	if sym != "0x1234...cdef" {
		t.Errorf("expected shortened address, got %s", sym)
	}
}
