// Package response 提供统一的 HTTP 响应结构。
// 所有 Gateway 层的接口都应使用此包返回响应，确保前端对接一致性。
package response

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go.opentelemetry.io/otel/trace"
)

// Body 统一响应体结构
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceId string      `json:"traceId,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int64 `json:"page"`
	PageSize   int64 `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

// PagedData 分页响应数据
type PagedData struct {
	List       interface{} `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

// Success 返回成功响应
func Success(ctx context.Context, w http.ResponseWriter, data interface{}) {
	body := &Body{
		Code:    0,
		Message: "ok",
		Data:    data,
		TraceId: getTraceId(ctx),
	}
	httpx.OkJson(w, body)
}

// SuccessWithPage 返回分页成功响应
func SuccessWithPage(ctx context.Context, w http.ResponseWriter, list interface{}, page, pageSize, total int64) {
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}
	data := &PagedData{
		List: list,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
	Success(ctx, w, data)
}

// Error 返回错误响应
func Error(ctx context.Context, w http.ResponseWriter, code int, msg string) {
	body := &Body{
		Code:    code,
		Message: msg,
		TraceId: getTraceId(ctx),
	}
	httpx.OkJson(w, body)
}

// ErrorWithHttpStatus 返回带 HTTP 状态码的错误响应
func ErrorWithHttpStatus(ctx context.Context, w http.ResponseWriter, httpStatus int, code int, msg string) {
	body := &Body{
		Code:    code,
		Message: msg,
		TraceId: getTraceId(ctx),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	httpx.OkJson(w, body)
}

// getTraceId 从 context 中提取链路追踪 ID
func getTraceId(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}
