// Package mathx 提供链上大数计算相关的工具函数。
// 链上金额不能使用浮点数，必须使用 big.Int 进行精确运算。
package mathx

import (
	"fmt"
	"math/big"
	"strings"
)

var (
	Zero    = big.NewInt(0)
	One     = big.NewInt(1)
	Ten     = big.NewInt(10)
	Hundred = big.NewInt(100)

	// 常用精度
	Decimals6  = new(big.Int).Exp(Ten, big.NewInt(6), nil)  // 10^6  (USDC, USDT)
	Decimals8  = new(big.Int).Exp(Ten, big.NewInt(8), nil)  // 10^8  (WBTC)
	Decimals18 = new(big.Int).Exp(Ten, big.NewInt(18), nil) // 10^18 (ETH, ERC20 标准)

	// 基点: 10000 (1 bps = 0.01%)
	BasisPointsBase = big.NewInt(10000)

	// RAY 精度: 10^27 (Aave 风格)
	Ray = new(big.Int).Exp(Ten, big.NewInt(27), nil)
)

// ParseBigInt 将十进制字符串转换为 big.Int
func ParseBigInt(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0), nil
	}
	bi, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid number: %s", s)
	}
	return bi, nil
}

// MustParseBigInt 将十进制字符串转换为 big.Int，失败则 panic
func MustParseBigInt(s string) *big.Int {
	bi, err := ParseBigInt(s)
	if err != nil {
		panic(err)
	}
	return bi
}

// FormatUnits 将 raw amount 按 decimals 格式化为人类可读字符串
// 示例: FormatUnits("1500000000000000000", 18) = "1.5"
func FormatUnits(rawAmount string, decimals int) string {
	bi, err := ParseBigInt(rawAmount)
	if err != nil || bi.Sign() == 0 {
		return "0"
	}

	divisor := new(big.Int).Exp(Ten, big.NewInt(int64(decimals)), nil)
	intPart := new(big.Int).Div(bi, divisor)
	remainder := new(big.Int).Mod(bi, divisor)

	if remainder.Sign() == 0 {
		return intPart.String()
	}

	// 补零
	fracStr := remainder.String()
	if len(fracStr) < decimals {
		fracStr = strings.Repeat("0", decimals-len(fracStr)) + fracStr
	}
	// 去除尾部零
	fracStr = strings.TrimRight(fracStr, "0")

	return fmt.Sprintf("%s.%s", intPart.String(), fracStr)
}

// ParseUnits 将人类可读金额转换为 raw amount
// 示例: ParseUnits("1.5", 18) = "1500000000000000000"
func ParseUnits(amount string, decimals int) (*big.Int, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" || amount == "0" {
		return big.NewInt(0), nil
	}

	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}

	intPart, ok := new(big.Int).SetString(parts[0], 10)
	if !ok {
		return nil, fmt.Errorf("invalid integer part: %s", parts[0])
	}

	divisor := new(big.Int).Exp(Ten, big.NewInt(int64(decimals)), nil)
	result := new(big.Int).Mul(intPart, divisor)

	if len(parts) == 2 {
		fracStr := parts[1]
		if len(fracStr) > decimals {
			fracStr = fracStr[:decimals] // 截断多余精度
		}
		if len(fracStr) < decimals {
			fracStr += strings.Repeat("0", decimals-len(fracStr))
		}
		fracPart, ok := new(big.Int).SetString(fracStr, 10)
		if !ok {
			return nil, fmt.Errorf("invalid fractional part: %s", fracStr)
		}
		if intPart.Sign() < 0 {
			result.Sub(result, fracPart)
		} else {
			result.Add(result, fracPart)
		}
	}

	return result, nil
}

// CalcPrice 计算价格: reserve1 / reserve0
// 返回精度为 18 位的 string
func CalcPrice(reserve0, reserve1 string, decimals0, decimals1 int) string {
	r0, err := ParseBigInt(reserve0)
	if err != nil || r0.Sign() == 0 {
		return "0"
	}
	r1, err := ParseBigInt(reserve1)
	if err != nil || r1.Sign() == 0 {
		return "0"
	}

	// 标准化: price = (r1 * 10^decimals0) / (r0 * 10^decimals1)
	// 再乘以 10^18 获取精度
	numerator := new(big.Int).Mul(r1, new(big.Int).Exp(Ten, big.NewInt(int64(decimals0)), nil))
	numerator.Mul(numerator, Decimals18)

	denominator := new(big.Int).Mul(r0, new(big.Int).Exp(Ten, big.NewInt(int64(decimals1)), nil))

	price := new(big.Int).Div(numerator, denominator)
	return FormatUnits(price.String(), 18)
}

// CalcBasisPoints 计算基点值
// 1 bps = 0.01%, 30 bps = 0.3%
func CalcBasisPoints(amount *big.Int, bps int64) *big.Int {
	result := new(big.Int).Mul(amount, big.NewInt(bps))
	return result.Div(result, BasisPointsBase)
}

// CalcShareRatio 计算份额占比: userShares / totalShares
// 返回精度 18 的值
func CalcShareRatio(userShares, totalShares *big.Int) string {
	if totalShares.Sign() == 0 {
		return "0"
	}
	numerator := new(big.Int).Mul(userShares, Decimals18)
	ratio := new(big.Int).Div(numerator, totalShares)
	return FormatUnits(ratio.String(), 18)
}

// CalcHealthFactor 计算健康因子: collateralValue / borrowValue
// 返回精度 8 的 string
func CalcHealthFactor(collateralValue, borrowValue *big.Int) string {
	if borrowValue.Sign() == 0 {
		return "999999.99" // 无借款，健康因子无穷大
	}
	precision := new(big.Int).Exp(Ten, big.NewInt(8), nil)
	numerator := new(big.Int).Mul(collateralValue, precision)
	hf := new(big.Int).Div(numerator, borrowValue)
	return FormatUnits(hf.String(), 8)
}

// CalcUnrealizedPnL 计算未实现盈亏
// pnl = (markPrice - entryPrice) * size * direction
// direction: 1=long, -1=short
func CalcUnrealizedPnL(markPrice, entryPrice, size *big.Int, isLong bool) *big.Int {
	priceDiff := new(big.Int).Sub(markPrice, entryPrice)
	pnl := new(big.Int).Mul(priceDiff, size)
	pnl.Div(pnl, Decimals18) // 标准化

	if !isLong {
		pnl.Neg(pnl)
	}
	return pnl
}

// Min 返回两个 big.Int 中较小的
func Min(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

// Max 返回两个 big.Int 中较大的
func Max(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}
