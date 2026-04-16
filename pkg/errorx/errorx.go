// Package errorx 定义全局统一错误码。
// 错误码分层规则：
//   - 0:          成功
//   - 1000-1999:  通用参数错误
//   - 2000-2999:  鉴权错误
//   - 3000-3999:  DEX 业务错误
//   - 4000-4999:  Lending 业务错误
//   - 5000-5999:  Futures / Options 错误
//   - 6000-6999:  Vault / Bonds 错误
//   - 7000-7999:  Governance 错误
//   - 9000-9999:  系统错误
package errorx

import "fmt"

// CodeError 自定义业务错误
type CodeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// New 创建一个新的业务错误
func New(code int, msg string) *CodeError {
	return &CodeError{Code: code, Message: msg}
}

// Newf 创建一个格式化的业务错误
func Newf(code int, format string, args ...interface{}) *CodeError {
	return &CodeError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// GetCode 从 error 中提取 code，如果不是 CodeError 则返回系统错误码
func GetCode(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*CodeError); ok {
		return e.Code
	}
	return CodeSystemBusy
}

// GetMessage 从 error 中提取 message
func GetMessage(err error) string {
	if err == nil {
		return "ok"
	}
	if e, ok := err.(*CodeError); ok {
		return e.Message
	}
	return "系统繁忙，请稍后再试"
}

// ==================== 通用参数错误 1000-1999 ====================

const (
	CodeParamMissing      = 1001 // 参数缺失
	CodeParamInvalid      = 1002 // 参数非法
	CodePageInvalid       = 1003 // 分页参数非法
	CodeAddressInvalid    = 1004 // 地址格式非法
	CodeAmountInvalid     = 1005 // 金额格式非法
	CodeRateLimited       = 1429 // 请求过于频繁
)

var (
	ErrParamMissing   = New(CodeParamMissing, "参数缺失")
	ErrParamInvalid   = New(CodeParamInvalid, "参数非法")
	ErrPageInvalid    = New(CodePageInvalid, "分页参数非法")
	ErrAddressInvalid = New(CodeAddressInvalid, "地址格式非法")
	ErrAmountInvalid  = New(CodeAmountInvalid, "金额格式非法")
	ErrRateLimited    = New(CodeRateLimited, "请求过于频繁，请稍后再试")
)

// ==================== 鉴权错误 2000-2999 ====================

const (
	CodeUnauthorized    = 2001 // 未登录
	CodeSignatureInvalid = 2002 // 签名无效
	CodeTokenExpired    = 2003 // Token 过期
	CodeAccountBanned   = 2004 // 账号被封禁
)

var (
	ErrUnauthorized     = New(CodeUnauthorized, "请先登录")
	ErrSignatureInvalid = New(CodeSignatureInvalid, "签名无效")
	ErrTokenExpired     = New(CodeTokenExpired, "Token 已过期，请重新登录")
	ErrAccountBanned    = New(CodeAccountBanned, "账号已被封禁")
)

// ==================== DEX 业务错误 3000-3999 ====================

const (
	CodePairNotFound     = 3001 // 交易对不存在
	CodeSlippageExceeded = 3002 // 滑点超限
	CodeRouteNotFound    = 3003 // 路由不可用
)

var (
	ErrPairNotFound     = New(CodePairNotFound, "交易对不存在")
	ErrSlippageExceeded = New(CodeSlippageExceeded, "滑点超出限制")
	ErrRouteNotFound    = New(CodeRouteNotFound, "未找到可用交易路由")
)

// ==================== Lending 业务错误 4000-4999 ====================

const (
	CodeMarketNotFound    = 4001 // 市场不存在
	CodeHealthInsufficient = 4002 // 健康度不足
	CodeBorrowExceeded    = 4003 // 超过可借额度
)

var (
	ErrMarketNotFound     = New(CodeMarketNotFound, "借贷市场不存在")
	ErrHealthInsufficient = New(CodeHealthInsufficient, "健康度不足，操作被拒绝")
	ErrBorrowExceeded     = New(CodeBorrowExceeded, "超过最大可借额度")
)

// ==================== Futures / Options 错误 5000-5999 ====================

const (
	CodePositionNotFound    = 5001 // 仓位不存在
	CodeMarginInsufficient  = 5002 // 保证金不足
	CodeLeverageExceeded    = 5003 // 杠杆超限
	CodeOptionExpired       = 5004 // 期权已到期
	CodeExerciseInvalid     = 5005 // 不满足行权条件
)

var (
	ErrPositionNotFound    = New(CodePositionNotFound, "仓位不存在")
	ErrMarginInsufficient  = New(CodeMarginInsufficient, "保证金不足")
	ErrLeverageExceeded    = New(CodeLeverageExceeded, "超过最大杠杆倍数")
	ErrOptionExpired       = New(CodeOptionExpired, "期权已到期")
	ErrExerciseInvalid     = New(CodeExerciseInvalid, "不满足行权条件")
)

// ==================== Vault / Bonds 错误 6000-6999 ====================

const (
	CodeVaultNotFound  = 6001 // 金库不存在
	CodeVaultPaused    = 6002 // 金库已暂停
	CodeBondNotFound   = 6003 // 债券不存在
)

var (
	ErrVaultNotFound = New(CodeVaultNotFound, "金库不存在")
	ErrVaultPaused   = New(CodeVaultPaused, "金库已暂停运营")
	ErrBondNotFound  = New(CodeBondNotFound, "债券市场不存在")
)

// ==================== Governance 错误 7000-7999 ====================

const (
	CodeProposalNotFound  = 7001 // 提案不存在
	CodeVotingEnded       = 7002 // 投票期已结束
	CodeVotePowerInsufficient = 7003 // 投票权不足
)

var (
	ErrProposalNotFound      = New(CodeProposalNotFound, "提案不存在")
	ErrVotingEnded           = New(CodeVotingEnded, "投票期已结束")
	ErrVotePowerInsufficient = New(CodeVotePowerInsufficient, "投票权不足")
)

// ==================== 系统错误 9000-9999 ====================

const (
	CodeSystemBusy       = 9001 // 系统繁忙
	CodeServiceDown      = 9002 // 服务不可用
	CodeChainNodeError   = 9003 // 链上节点异常
	CodeDatabaseError    = 9004 // 数据库异常
	CodeCacheError       = 9005 // 缓存异常
	CodeKafkaError       = 9006 // 消息队列异常
)

var (
	ErrSystemBusy     = New(CodeSystemBusy, "系统繁忙，请稍后再试")
	ErrServiceDown    = New(CodeServiceDown, "相关服务暂时不可用")
	ErrChainNodeError = New(CodeChainNodeError, "链上节点异常")
	ErrDatabaseError  = New(CodeDatabaseError, "数据库异常")
	ErrCacheError     = New(CodeCacheError, "缓存异常")
	ErrKafkaError     = New(CodeKafkaError, "消息队列异常")
)
