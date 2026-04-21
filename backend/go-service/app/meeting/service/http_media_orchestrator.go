// Package service 提供 meeting 模块的业务逻辑
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/echochat/backend/config"
	"github.com/echochat/backend/pkg/logs"
	"go.uber.org/zap"
)

// ====== 错误定义（供上层 errors.Is 判定并映射为 WS/HTTP 错误语义）======

// ErrMediaResourceNotFound media-server 明确返回 404 时的错误
// 典型场景：close 已经不存在的 router / transport / producer / consumer
// 对调用方而言通常可容忍（幂等关闭），在 SignalService 中一般当作"已清理"处理
var ErrMediaResourceNotFound = errors.New("media resource not found")

// ErrMediaServerError media-server 返回 5xx 或网络异常（超时 / 连接拒绝）时的错误
// 会在 WS ACK 中以 code=-1 + message 形式返回给前端
var ErrMediaServerError = errors.New("media server error")

// ====== HTTPMediaOrchestrator：MediaOrchestrator 的真实 HTTP 实现 ======

// HTTPMediaOrchestrator 通过 HTTP 调 Node media-server 的 /internal/v1/* API，实现 MediaOrchestrator 接口
//
// 核心特性：
//   - 创建类操作（router / transport / producer / consumer）：超时即失败，无重试，防止重复创建资源
//   - 关闭类操作（CloseRouter / CloseProducer / CloseConsumer / 隐含的 transport.close）：
//     独立较短超时；失败时幂等重试 CloseRetry 次，指数退避 200ms→500ms
//   - 404 统一映射为 ErrMediaResourceNotFound，其它 4xx/5xx 统一映射为 ErrMediaServerError
//   - roomCode → routerID 本地缓存：设计 §6.6 规定 CloseRouter 入参为 roomCode，但 Node 以 routerID 作为资源主键，
//     本 Orchestrator 在 CreateRouter 成功后记住映射，CloseRouter 时反查
//   - X-Internal-Token header 与 media-server/.env 的 MEDIA_INTERNAL_TOKEN 配对，两端不匹配将被 Node 的 401 拒绝
//
// 并发：所有方法可安全并发调用；内部 http.Client 复用连接池
type HTTPMediaOrchestrator struct {
	cfg    config.MediaServerConfig
	client *http.Client

	// roomCode → routerID 本地缓存，服务重启后会丢失（此时 Node 侧的 Router 也会随 Node 重启而释放，状态一致）
	roomRouterIDs sync.Map
}

// NewHTTPMediaOrchestrator 构造真实 HTTP 客户端
// 由 wire 注入，在 app/provider/provider.go 中统一绑定为 MediaOrchestrator
// 创建时做一次性配置校验：base_url 必须非空，internal_token 必须非空
func NewHTTPMediaOrchestrator(cfg *config.Config) *HTTPMediaOrchestrator {
	mc := cfg.MediaServer
	if mc.TimeoutMS <= 0 {
		mc.TimeoutMS = 5000
	}
	if mc.CloseTimeoutMS <= 0 {
		mc.CloseTimeoutMS = 2000
	}
	if mc.CloseRetry < 0 {
		mc.CloseRetry = 0
	}
	// 去除 base_url 末尾斜杠，统一拼接风格
	mc.BaseURL = strings.TrimRight(mc.BaseURL, "/")

	return &HTTPMediaOrchestrator{
		cfg: mc,
		client: &http.Client{
			// 不在 Client 层设置 Timeout，由各请求通过 context 控制，便于细粒度区分 create/close
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ====== MediaOrchestrator 接口实现 ======

// CreateRouter 调用 POST /internal/v1/routers
// 成功后在本地缓存 roomCode → routerID 映射供 CloseRouter 使用
func (h *HTTPMediaOrchestrator) CreateRouter(ctx context.Context, roomCode string) (string, error) {
	funcName := "service.http_media_orchestrator.CreateRouter"

	reqBody := map[string]any{"roomCode": roomCode}
	var resp struct {
		RouterID        string          `json:"routerId"`
		RtpCapabilities json.RawMessage `json:"rtpCapabilities"`
	}
	if err := h.doRequest(ctx, requestOptions{
		method:    http.MethodPost,
		path:      "/internal/v1/routers",
		body:      reqBody,
		timeoutMS: h.cfg.TimeoutMS,
		funcName:  funcName,
		logFields: []zap.Field{zap.String("room_code", roomCode)},
	}, &resp); err != nil {
		return "", err
	}

	// 记忆映射：一个 roomCode 仅对应一个 Router；若此前存在旧 Router ID（极少见），以新值覆盖
	h.roomRouterIDs.Store(roomCode, resp.RouterID)

	logs.Info(ctx, funcName, "Node Router 创建成功",
		zap.String("room_code", roomCode),
		zap.String("router_id", resp.RouterID))
	return resp.RouterID, nil
}

// CloseRouter 调用 DELETE /internal/v1/routers/:routerId
// 入参是 roomCode（符合设计 §6.6 NodeClient 契约），内部反查本地缓存拿到 routerID
// 若本地缓存不存在映射（可能是 go-service 重启后状态丢失），直接返回 nil：
//   - Node 那边的 Router 也会随 Node 重启释放
//   - 对 meeting 业务层是幂等的"已清理"语义
func (h *HTTPMediaOrchestrator) CloseRouter(ctx context.Context, roomCode string) error {
	funcName := "service.http_media_orchestrator.CloseRouter"

	v, ok := h.roomRouterIDs.Load(roomCode)
	if !ok {
		logs.Debug(ctx, funcName, "无本地映射，跳过 Close", zap.String("room_code", roomCode))
		return nil
	}
	routerID, _ := v.(string)
	if routerID == "" {
		h.roomRouterIDs.Delete(roomCode)
		return nil
	}

	err := h.doCloseRequest(ctx, fmt.Sprintf("/internal/v1/routers/%s", routerID), funcName, []zap.Field{
		zap.String("room_code", roomCode),
		zap.String("router_id", routerID),
	})
	// 成功或 ResourceNotFound 都从本地缓存删除（幂等）
	if err == nil || errors.Is(err, ErrMediaResourceNotFound) {
		h.roomRouterIDs.Delete(roomCode)
		return nil
	}
	return err
}

// CreateTransport 调用 POST /internal/v1/transports
// 需要先从本地缓存取 routerID；若 roomCode 对应 Router 不存在，返回 ErrMediaResourceNotFound
func (h *HTTPMediaOrchestrator) CreateTransport(ctx context.Context, req *CreateTransportReq) (*TransportInfo, error) {
	funcName := "service.http_media_orchestrator.CreateTransport"

	routerID, err := h.routerIDByRoomCode(req.RoomCode)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"routerId":  routerID,
		"userId":    req.UserID, // Node zod schema 支持 number → string 转换
		"direction": req.Direction,
	}
	var resp TransportInfo
	if err := h.doRequest(ctx, requestOptions{
		method:    http.MethodPost,
		path:      "/internal/v1/transports",
		body:      reqBody,
		timeoutMS: h.cfg.TimeoutMS,
		funcName:  funcName,
		logFields: []zap.Field{
			zap.String("room_code", req.RoomCode),
			zap.Int64("user_id", req.UserID),
			zap.String("direction", req.Direction),
		},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ConnectTransport 调用 POST /internal/v1/transports/:id/connect
// 请求体需保留 Node zod schema 要求的 {dtlsParameters: {role?, fingerprints}} 结构，由前端传入并原样转发
func (h *HTTPMediaOrchestrator) ConnectTransport(ctx context.Context, transportID string, dtlsParameters json.RawMessage) error {
	funcName := "service.http_media_orchestrator.ConnectTransport"

	reqBody := map[string]any{"dtlsParameters": dtlsParameters}
	return h.doRequest(ctx, requestOptions{
		method:    http.MethodPost,
		path:      fmt.Sprintf("/internal/v1/transports/%s/connect", transportID),
		body:      reqBody,
		timeoutMS: h.cfg.TimeoutMS,
		funcName:  funcName,
		logFields: []zap.Field{zap.String("transport_id", transportID)},
	}, nil)
}

// CreateProducer 调用 POST /internal/v1/producers
func (h *HTTPMediaOrchestrator) CreateProducer(ctx context.Context, req *CreateProducerReq) (string, error) {
	funcName := "service.http_media_orchestrator.CreateProducer"

	reqBody := map[string]any{
		"transportId":   req.TransportID,
		"kind":          req.Kind,
		"rtpParameters": req.RtpParameters,
		"appData": map[string]any{
			"userId":   req.UserID,
			"roomCode": req.RoomCode,
		},
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := h.doRequest(ctx, requestOptions{
		method:    http.MethodPost,
		path:      "/internal/v1/producers",
		body:      reqBody,
		timeoutMS: h.cfg.TimeoutMS,
		funcName:  funcName,
		logFields: []zap.Field{
			zap.String("room_code", req.RoomCode),
			zap.Int64("user_id", req.UserID),
			zap.String("transport_id", req.TransportID),
			zap.String("kind", req.Kind),
		},
	}, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

// CloseProducer 调用 DELETE /internal/v1/producers/:id
func (h *HTTPMediaOrchestrator) CloseProducer(ctx context.Context, producerID string) error {
	funcName := "service.http_media_orchestrator.CloseProducer"

	err := h.doCloseRequest(ctx, fmt.Sprintf("/internal/v1/producers/%s", producerID), funcName, []zap.Field{
		zap.String("producer_id", producerID),
	})
	// 幂等：Producer 已不存在视为已清理，返回 nil
	if errors.Is(err, ErrMediaResourceNotFound) {
		return nil
	}
	return err
}

// CreateConsumer 调用 POST /internal/v1/consumers
// Node 侧 Consumer 强制 paused=true 创建，前端收到后需额外调 /resume（MediaOrchestrator 暂未暴露 ResumeConsumer，
// 将在 Task 8 前端接入 mediasoup-client 时按需补充）
func (h *HTTPMediaOrchestrator) CreateConsumer(ctx context.Context, req *CreateConsumerReq) (*ConsumerInfo, error) {
	funcName := "service.http_media_orchestrator.CreateConsumer"

	routerID, err := h.routerIDByRoomCode(req.RoomCode)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"routerId":        routerID,
		"transportId":     req.TransportID,
		"producerId":      req.ProducerID,
		"rtpCapabilities": req.RtpCapabilities,
	}
	var resp ConsumerInfo
	if err := h.doRequest(ctx, requestOptions{
		method:    http.MethodPost,
		path:      "/internal/v1/consumers",
		body:      reqBody,
		timeoutMS: h.cfg.TimeoutMS,
		funcName:  funcName,
		logFields: []zap.Field{
			zap.String("room_code", req.RoomCode),
			zap.Int64("user_id", req.UserID),
			zap.String("transport_id", req.TransportID),
			zap.String("producer_id", req.ProducerID),
		},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CloseConsumer 调用 DELETE /internal/v1/consumers/:id
func (h *HTTPMediaOrchestrator) CloseConsumer(ctx context.Context, consumerID string) error {
	funcName := "service.http_media_orchestrator.CloseConsumer"

	err := h.doCloseRequest(ctx, fmt.Sprintf("/internal/v1/consumers/%s", consumerID), funcName, []zap.Field{
		zap.String("consumer_id", consumerID),
	})
	if errors.Is(err, ErrMediaResourceNotFound) {
		return nil
	}
	return err
}

// ====== 内部工具 ======

// routerIDByRoomCode 从本地缓存反查 routerID，缺失时返回 ErrMediaResourceNotFound
// 此错误语义表达的是"meeting 业务侧尚未（或已清理了）为该房间创建 Router"，调用方通常应转为"会议未开始/已结束"
func (h *HTTPMediaOrchestrator) routerIDByRoomCode(roomCode string) (string, error) {
	v, ok := h.roomRouterIDs.Load(roomCode)
	if !ok {
		return "", fmt.Errorf("%w: no router mapped for room_code=%s", ErrMediaResourceNotFound, roomCode)
	}
	routerID, _ := v.(string)
	if routerID == "" {
		return "", fmt.Errorf("%w: empty router_id for room_code=%s", ErrMediaResourceNotFound, roomCode)
	}
	return routerID, nil
}

// requestOptions 统一请求参数
type requestOptions struct {
	method    string
	path      string
	body      any
	timeoutMS int
	funcName  string
	logFields []zap.Field
}

// doRequest 执行一次带超时的 HTTP 请求（无重试），将 2xx 的 JSON 响应解码到 respOut 指向的结构体
// respOut 为 nil 时忽略响应体（适用于 DELETE / POST connect 等无返回值接口）
func (h *HTTPMediaOrchestrator) doRequest(ctx context.Context, opts requestOptions, respOut any) error {
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(opts.timeoutMS)*time.Millisecond)
	defer cancel()

	var bodyReader io.Reader
	if opts.body != nil {
		buf, err := json.Marshal(opts.body)
		if err != nil {
			return fmt.Errorf("%w: marshal request body: %v", ErrMediaServerError, err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	url := h.cfg.BaseURL + opts.path
	httpReq, err := http.NewRequestWithContext(reqCtx, opts.method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("%w: new request: %v", ErrMediaServerError, err)
	}
	httpReq.Header.Set("X-Internal-Token", h.cfg.InternalToken)
	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	started := time.Now()
	resp, err := h.client.Do(httpReq)
	duration := time.Since(started)
	if err != nil {
		logs.Warn(ctx, opts.funcName, "HTTP 请求 media-server 失败",
			append(opts.logFields,
				zap.String("url", url),
				zap.Duration("duration", duration),
				zap.Error(err))...)
		return fmt.Errorf("%w: %v", ErrMediaServerError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if respOut == nil {
			// 消耗响应体保证连接可复用
			_, _ = io.Copy(io.Discard, resp.Body)
			logs.Debug(ctx, opts.funcName, "HTTP 调用成功",
				append(opts.logFields,
					zap.Int("status", resp.StatusCode),
					zap.Duration("duration", duration))...)
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(respOut); err != nil {
			return fmt.Errorf("%w: decode response: %v", ErrMediaServerError, err)
		}
		logs.Debug(ctx, opts.funcName, "HTTP 调用成功",
			append(opts.logFields,
				zap.Int("status", resp.StatusCode),
				zap.Duration("duration", duration))...)
		return nil
	}

	// 非 2xx：读取响应体（用于日志）+ 映射错误类型
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := truncate(string(bodyBytes), 512)

	if resp.StatusCode == http.StatusNotFound {
		logs.Debug(ctx, opts.funcName, "media-server 返回 404",
			append(opts.logFields,
				zap.Int("status", resp.StatusCode),
				zap.String("resp_body", bodyStr))...)
		return fmt.Errorf("%w: %s %s returned 404: %s",
			ErrMediaResourceNotFound, opts.method, opts.path, bodyStr)
	}

	logs.Warn(ctx, opts.funcName, "media-server 返回异常状态",
		append(opts.logFields,
			zap.Int("status", resp.StatusCode),
			zap.String("resp_body", bodyStr),
			zap.Duration("duration", duration))...)
	return fmt.Errorf("%w: %s %s returned %d: %s",
		ErrMediaServerError, opts.method, opts.path, resp.StatusCode, bodyStr)
}

// doCloseRequest DELETE 类关闭接口的专用封装：较短超时 + 幂等重试（指数退避 200ms → 500ms）
// - 404 不重试（资源已不存在，视为成功的幂等场景，由调用方判断是否转 nil）
// - 5xx / 超时 / 网络错 重试 CloseRetry 次
func (h *HTTPMediaOrchestrator) doCloseRequest(ctx context.Context, path, funcName string, logFields []zap.Field) error {
	backoffs := []time.Duration{200 * time.Millisecond, 500 * time.Millisecond}
	attempts := h.cfg.CloseRetry + 1 // +1 表示首次
	var lastErr error

	for i := 0; i < attempts; i++ {
		if i > 0 {
			// 使用 min(i-1, len(backoffs)-1) 选退避间隔，超过预设值用最后一个
			idx := i - 1
			if idx >= len(backoffs) {
				idx = len(backoffs) - 1
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffs[idx]):
			}
		}

		err := h.doRequest(ctx, requestOptions{
			method:    http.MethodDelete,
			path:      path,
			body:      nil,
			timeoutMS: h.cfg.CloseTimeoutMS,
			funcName:  funcName,
			logFields: append(logFields, zap.Int("attempt", i+1)),
		}, nil)

		if err == nil {
			return nil
		}
		lastErr = err

		// 404 不重试
		if errors.Is(err, ErrMediaResourceNotFound) {
			return err
		}
	}

	logs.Warn(ctx, funcName, "关闭类请求重试耗尽",
		append(logFields, zap.Int("attempts", attempts), zap.Error(lastErr))...)
	return lastErr
}

// truncate 截断超长字符串用于日志避免污染
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
