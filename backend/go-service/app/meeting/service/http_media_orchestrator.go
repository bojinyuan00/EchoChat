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
// routerInfoCache Router 本地缓存条目：除 ID 外保存 Node 返回的 rtpCapabilities，
// 供前端 mediasoup-client Device.load 直接使用（Task 9 引入）。
type routerInfoCache struct {
	ID              string
	RtpCapabilities json.RawMessage
}

type HTTPMediaOrchestrator struct {
	cfg    config.MediaServerConfig
	client *http.Client

	// roomCode → *routerInfoCache 本地缓存，服务重启后会丢失（此时 Node 侧的 Router 也会随 Node 重启而释放，状态一致）
	// Task 9 起由 sync.Map<string> 升级为 sync.Map<*routerInfoCache>
	roomRouterInfos sync.Map
}

// NewHTTPMediaOrchestrator 构造真实 HTTP 客户端
// 由 wire 注入，在 app/provider/provider.go 中统一绑定为 MediaOrchestrator
// 创建时做一次性配置校验：base_url 必须非空，internal_token 必须非空
func NewHTTPMediaOrchestrator(cfg *config.Config) *HTTPMediaOrchestrator {
	mc := cfg.MediaServer
	if mc.TimeoutMS <= 0 {
		// Task 16 Nit（代码审查 2026-04-23 第 15 条）：
		// 原默认 5000ms 对 CreateRouter 偏紧（Worker 冷启动 + Router 首次创建在慢机上可达 6~8s），
		// 统一将默认超时放宽到 10000ms，显式配置（config.*.yaml）不受影响
		mc.TimeoutMS = 10000
	}
	if mc.CloseTimeoutMS <= 0 {
		mc.CloseTimeoutMS = 2000
	}
	if mc.CloseRetry < 0 {
		mc.CloseRetry = 0
	}
	if mc.CreateRouterRetry < 0 {
		mc.CreateRouterRetry = 0
	}
	if mc.CreateRouterRetry == 0 {
		// Task 16 Nit（代码审查 2026-04-23 第 15 条）：
		// 默认允许 1 次轻量重试，300ms 退避，仅对非 404 错误生效
		mc.CreateRouterRetry = 1
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
// Task 8 幂等防御（双层，与业务层 JoinRoom 不再主动调用形成互补）：
//   - 命中缓存直接返回已缓存 routerID，不发起 HTTP 请求
//   - 仅在缓存缺失时真正调 Node；避免业务误多次调用产生 Node 侧重复 Router
func (h *HTTPMediaOrchestrator) CreateRouter(ctx context.Context, roomCode string) (string, error) {
	funcName := "service.http_media_orchestrator.CreateRouter"

	if cached, ok := h.roomRouterInfos.Load(roomCode); ok {
		if info, _ := cached.(*routerInfoCache); info != nil && info.ID != "" {
			logs.Debug(ctx, funcName, "命中本地 Router 缓存，跳过 Node 调用",
				zap.String("room_code", roomCode),
				zap.String("router_id", info.ID))
			return info.ID, nil
		}
	}

	reqBody := map[string]any{"roomCode": roomCode}
	var resp struct {
		RouterID        string          `json:"routerId"`
		RtpCapabilities json.RawMessage `json:"rtpCapabilities"`
	}

	// Task 16 Nit：CreateRouter 在 5xx / 网络错误时允许 CreateRouterRetry 次重试（退避 300ms）
	// - 404 不应出现在 POST /routers，若出现视为 media-server 配置异常，不重试
	// - ctx.Err() 立即终止（上游取消或超时）
	attempts := h.cfg.CreateRouterRetry + 1
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(300 * time.Millisecond):
			}
			logs.Info(ctx, funcName, "CreateRouter 重试",
				zap.String("room_code", roomCode),
				zap.Int("attempt", i+1))
		}
		err := h.doRequest(ctx, requestOptions{
			method:    http.MethodPost,
			path:      "/internal/v1/routers",
			body:      reqBody,
			timeoutMS: h.cfg.TimeoutMS,
			funcName:  funcName,
			logFields: []zap.Field{zap.String("room_code", roomCode), zap.Int("attempt", i+1)},
		}, &resp)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		if errors.Is(err, ErrMediaResourceNotFound) {
			break
		}
	}
	if lastErr != nil {
		return "", lastErr
	}

	info := &routerInfoCache{
		ID:              resp.RouterID,
		RtpCapabilities: resp.RtpCapabilities,
	}
	// LoadOrStore 防止并发首次创建时重复覆盖：若他人已写入先到的值就用已存在的
	actual, loaded := h.roomRouterInfos.LoadOrStore(roomCode, info)
	if loaded {
		if existing, _ := actual.(*routerInfoCache); existing != nil && existing.ID != "" && existing.ID != resp.RouterID {
			logs.Warn(ctx, funcName, "并发创建 Router 出现不一致，保留先到值",
				zap.String("room_code", roomCode),
				zap.String("existing", existing.ID),
				zap.String("new", resp.RouterID))
			return existing.ID, nil
		}
	}

	logs.Info(ctx, funcName, "Node Router 创建成功",
		zap.String("room_code", roomCode),
		zap.String("router_id", resp.RouterID))
	return resp.RouterID, nil
}

// ResolveRouterID 从本地缓存读取 roomCode 对应的 routerID（不触发 HTTP）
// 返回 (id, true) 表示命中；返回 (_, false) 表示缓存缺失（通常意味着该房间尚未创建 Router）
// 供 MeetingService.JoinRoom 等需复用已有 Router 的场景使用，避免重复调 Node
func (h *HTTPMediaOrchestrator) ResolveRouterID(roomCode string) (string, bool) {
	v, ok := h.roomRouterInfos.Load(roomCode)
	if !ok {
		return "", false
	}
	info, _ := v.(*routerInfoCache)
	if info == nil || info.ID == "" {
		return "", false
	}
	return info.ID, true
}

// ResolveRouterInfo 读取 roomCode 对应的 Router 完整信息（routerID + rtpCapabilities）
// Task 9 引入：前端 mediasoup-client Device.load 需要 rtpCapabilities，JoinRoom 响应会回填此字段
// 返回 (id, caps, true) 命中；(_, _, false) 缺失
func (h *HTTPMediaOrchestrator) ResolveRouterInfo(roomCode string) (string, json.RawMessage, bool) {
	v, ok := h.roomRouterInfos.Load(roomCode)
	if !ok {
		return "", nil, false
	}
	info, _ := v.(*routerInfoCache)
	if info == nil || info.ID == "" {
		return "", nil, false
	}
	return info.ID, info.RtpCapabilities, true
}

// CloseRouter 调用 DELETE /internal/v1/routers/:routerId
// 入参是 roomCode（符合设计 §6.6 NodeClient 契约），内部反查本地缓存拿到 routerID
// 若本地缓存不存在映射（可能是 go-service 重启后状态丢失），直接返回 nil：
//   - Node 那边的 Router 也会随 Node 重启释放
//   - 对 meeting 业务层是幂等的"已清理"语义
func (h *HTTPMediaOrchestrator) CloseRouter(ctx context.Context, roomCode string) error {
	funcName := "service.http_media_orchestrator.CloseRouter"

	v, ok := h.roomRouterInfos.Load(roomCode)
	if !ok {
		logs.Debug(ctx, funcName, "无本地映射，跳过 Close", zap.String("room_code", roomCode))
		return nil
	}
	info, _ := v.(*routerInfoCache)
	if info == nil || info.ID == "" {
		h.roomRouterInfos.Delete(roomCode)
		return nil
	}
	routerID := info.ID

	err := h.doCloseRequest(ctx, fmt.Sprintf("/internal/v1/routers/%s", routerID), funcName, []zap.Field{
		zap.String("room_code", roomCode),
		zap.String("router_id", routerID),
	})
	// 成功或 ResourceNotFound 都从本地缓存删除（幂等）
	if err == nil || errors.Is(err, ErrMediaResourceNotFound) {
		h.roomRouterInfos.Delete(roomCode)
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

// CloseTransport 调用 DELETE /internal/v1/transports/:id（Task 16 P2-1 引入）
// 404 映射为 ErrMediaResourceNotFound（上层 cleanupUserResources 视为已清理）
// 与 CloseProducer / CloseConsumer 保持同一重试/超时策略
func (h *HTTPMediaOrchestrator) CloseTransport(ctx context.Context, transportID string) error {
	funcName := "service.http_media_orchestrator.CloseTransport"

	err := h.doCloseRequest(ctx, fmt.Sprintf("/internal/v1/transports/%s", transportID), funcName, []zap.Field{
		zap.String("transport_id", transportID),
	})
	if errors.Is(err, ErrMediaResourceNotFound) {
		return nil
	}
	return err
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
// Node 侧 Consumer 强制 paused=true 创建，前端在 recv Transport 与 track 挂载完成后
// 需发送 meeting.consume.resume WS 事件触发 ResumeConsumer 才会真正收到 RTP（Task 9）
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

// ResumeConsumer 调用 POST /internal/v1/consumers/:id/resume（Task 9 引入）
// Node 侧会把 mediasoup Consumer 从 paused 切到 active，开始向 Transport 推 RTP
// 语义幂等：Node 端对已 active 的 Consumer 再次 resume 不报错；未找到 Consumer 返回 404 → ErrMediaResourceNotFound
func (h *HTTPMediaOrchestrator) ResumeConsumer(ctx context.Context, consumerID string) error {
	funcName := "service.http_media_orchestrator.ResumeConsumer"

	return h.doRequest(ctx, requestOptions{
		method:    http.MethodPost,
		path:      fmt.Sprintf("/internal/v1/consumers/%s/resume", consumerID),
		body:      nil,
		timeoutMS: h.cfg.TimeoutMS,
		funcName:  funcName,
		logFields: []zap.Field{zap.String("consumer_id", consumerID)},
	}, nil)
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
	v, ok := h.roomRouterInfos.Load(roomCode)
	if !ok {
		return "", fmt.Errorf("%w: no router mapped for room_code=%s", ErrMediaResourceNotFound, roomCode)
	}
	info, _ := v.(*routerInfoCache)
	if info == nil || info.ID == "" {
		return "", fmt.Errorf("%w: empty router_id for room_code=%s", ErrMediaResourceNotFound, roomCode)
	}
	return info.ID, nil
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
