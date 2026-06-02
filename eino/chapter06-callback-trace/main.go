package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ============================================================================
// 第 6 章：Callback 和 Trace —— 可观测性
// ============================================================================
//
// 本示例演示 Eino 框架的回调机制，包括：
// 1. basic    - 基础回调实现
// 2. log      - 日志记录回调
// 3. perf     - 性能监控回调
// 4. trace    - 链路追踪示例
// 5. chat     - 带回调监控的完整对话
//
// 运行方式：
//   go run main.go basic
//   go run main.go log
//   go run main.go perf
//   go run main.go trace
//   go run main.go chat
//
// ============================================================================

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "basic":
			basicCallback()
		case "log":
			logCallback()
		case "perf":
			perfCallback()
		case "trace":
			traceCallback()
		case "chat":
			chatWithCallback()
		default:
			printUsage()
		}
	} else {
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino Callback 和 Trace 示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go basic  - 基础回调实现")
	fmt.Println("  go run main.go log    - 日志记录回调")
	fmt.Println("  go run main.go perf   - 性能监控回调")
	fmt.Println("  go run main.go trace  - 链路追踪示例")
	fmt.Println("  go run main.go chat   - 带回调监控的完整对话")
	fmt.Println("")
	fmt.Println("环境变量:")
	fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥（必需）")
}

// ============================================================================
// 示例 1：基础回调实现
// ============================================================================
// 演示最简单的回调处理器，实现 Handler 接口的 5 个方法
// ============================================================================

// SimpleHandler 是一个最基础的回调处理器
// 它实现了 Eino 的 callbacks.Handler 接口
type SimpleHandler struct{}

// OnStart 在组件开始处理前被调用
// 参数说明：
//   - ctx: 上下文，可以携带请求级别的数据
//   - info: 描述触发回调的实体信息（组件类型、名称等）
//   - input: 组件的输入数据（类型取决于具体组件）
// 返回值：新的上下文，可以在后续回调中使用
func (h *SimpleHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info != nil {
		fmt.Printf("[Basic] OnStart  - 组件: %s, 名称: %s\n", info.Component, info.Name)
	}
	return ctx
}

// OnEnd 在组件成功处理完成后被调用
func (h *SimpleHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info != nil {
		fmt.Printf("[Basic] OnEnd    - 组件: %s, 名称: %s\n", info.Component, info.Name)
	}
	return ctx
}

// OnError 在组件处理出错时被调用
func (h *SimpleHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info != nil {
		fmt.Printf("[Basic] OnError  - 组件: %s, 名称: %s, 错误: %v\n", info.Component, info.Name, err)
	}
	return ctx
}

// OnStartWithStreamInput 在流式输入到达时被调用
// 注意：流式回调接收的是 StreamReader，需要正确处理
func (h *SimpleHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if info != nil {
		fmt.Printf("[Basic] OnStartStreamIn  - 组件: %s, 名称: %s\n", info.Component, info.Name)
	}
	return ctx
}

// OnEndWithStreamOutput 在流式输出返回时被调用
// 重要：StreamReader 必须被正确关闭，否则会导致 goroutine 泄漏！
func (h *SimpleHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if info != nil {
		fmt.Printf("[Basic] OnEndStreamOut  - 组件: %s, 名称: %s\n", info.Component, info.Name)
	}
	return ctx
}

func basicCallback() {
	fmt.Println("=== 示例 1：基础回调实现 ===")
	fmt.Println("演示 Handler 接口的基本用法")
	fmt.Println()

	// 创建回调处理器
	handler := &SimpleHandler{}

	// 注册为全局处理器
	// 注意：必须在组件调用之前注册
	callbacks.AppendGlobalHandlers(handler)
	fmt.Println("已注册全局回调处理器")
	fmt.Println("当调用 ChatModel 时，回调会自动触发")
	fmt.Println()
	fmt.Println("提示：运行 'go run main.go chat' 查看实际效果")
}

// ============================================================================
// 示例 2：日志记录回调
// ============================================================================
// 使用 HandlerHelper 简化实现，只注册需要的回调方法
// HandlerHelper 提供了链式调用的方式，代码更简洁
// ============================================================================

// createLogHandler 创建一个日志记录回调处理器
// 使用 HandlerHelper 只实现需要的方法，不需要实现全部 5 个
func createLogHandler() callbacks.Handler {
	return callbacks.NewHandlerHelper().
		OnStart(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info != nil {
				log.Printf("[LOG] ▶ 开始 - %s/%s (类型: %s)",
					info.Component, info.Name, info.Type)
			}
			return ctx
		}).
		OnEnd(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info != nil {
				log.Printf("[LOG] ■ 完成 - %s/%s", info.Component, info.Name)
			}
			return ctx
		}).
		OnError(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if info != nil {
				log.Printf("[LOG] ✖ 错误 - %s/%s: %v", info.Component, info.Name, err)
			}
			return ctx
		}).
		Handler()
}

func logCallback() {
	fmt.Println("=== 示例 2：日志记录回调 ===")
	fmt.Println("使用 HandlerHelper 简化实现")
	fmt.Println()

	// 创建日志 Handler
	handler := createLogHandler()

	// 注册为全局处理器
	callbacks.AppendGlobalHandlers(handler)
	fmt.Println("已注册日志回调处理器")
	fmt.Println()
	fmt.Println("输出格式说明：")
	fmt.Println("  [LOG] ▶ 开始 - 组件/名称 (类型: 实现类型)")
	fmt.Println("  [LOG] ■ 完成 - 组件/名称")
	fmt.Println("  [LOG] ✖ 错误 - 组件/名称: 错误信息")
	fmt.Println()
	fmt.Println("提示：运行 'go run main.go chat' 查看实际效果")
}

// ============================================================================
// 示例 3：性能监控回调
// ============================================================================
// 演示如何在回调中传递数据（通过 context）
// 统计每个组件调用的耗时
// ============================================================================

// traceKey 是 context 中存储追踪信息的 key
// 使用空结构体作为 key 类型，避免与其他包冲突
type traceKey struct{}

// TraceInfo 存储单次调用的追踪信息
type TraceInfo struct {
	StartTime time.Time // 调用开始时间
	Component string    // 组件类型
	Name      string    // 组件名称
}

// PerfHandler 性能监控回调处理器
type PerfHandler struct{}

// OnStart 记录开始时间，存入 context
// 这样在 OnEnd 时可以取出计算耗时
func (h *PerfHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info != nil {
		trace := &TraceInfo{
			StartTime: time.Now(),
			Component: string(info.Component),
			Name:      info.Name,
		}
		// 将追踪信息存入 context
		// context.WithValue 返回一个新的 context，包含这个值
		return context.WithValue(ctx, traceKey{}, trace)
	}
	return ctx
}

// OnEnd 计算耗时并输出
func (h *PerfHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	// 从 context 中取出追踪信息
	if trace, ok := ctx.Value(traceKey{}).(*TraceInfo); ok {
		duration := time.Since(trace.StartTime)
		fmt.Printf("[PERF] %s/%s 耗时: %v\n", trace.Component, trace.Name, duration)
	}
	return ctx
}

// OnError 计算错误发生时的耗时
func (h *PerfHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if trace, ok := ctx.Value(traceKey{}).(*TraceInfo); ok {
		duration := time.Since(trace.StartTime)
		fmt.Printf("[PERF-ERROR] %s/%s 失败，耗时: %v, 错误: %v\n",
			trace.Component, trace.Name, duration, err)
	}
	return ctx
}

// OnStartWithStreamInput 流式输入开始（复用 OnStart 逻辑）
func (h *PerfHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return h.OnStart(ctx, info, nil)
}

// OnEndWithStreamOutput 流式输出结束（复用 OnEnd 逻辑）
func (h *PerfHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	return h.OnEnd(ctx, info, nil)
}

func perfCallback() {
	fmt.Println("=== 示例 3：性能监控回调 ===")
	fmt.Println("演示通过 context 传递数据，统计调用耗时")
	fmt.Println()

	// 创建性能监控 Handler
	handler := &PerfHandler{}
	callbacks.AppendGlobalHandlers(handler)
	fmt.Println("已注册性能监控回调处理器")
	fmt.Println()
	fmt.Println("工作原理：")
	fmt.Println("  1. OnStart 中记录开始时间，存入 context")
	fmt.Println("  2. OnEnd 中从 context 取出开始时间，计算耗时")
	fmt.Println("  3. 使用 context.WithValue/Value 实现数据传递")
	fmt.Println()
	fmt.Println("提示：运行 'go run main.go chat' 查看实际效果")
}

// ============================================================================
// 示例 4：链路追踪示例
// ============================================================================
// 演示如何追踪一个请求经过的所有组件
// 使用 sync.Map 存储请求级别的追踪数据
// ============================================================================

// RequestTracker 追踪单个请求的调用链
type RequestTracker struct {
	mu       sync.Mutex         // 保护并发访问
	traceID  string             // 追踪 ID
	entries  []TraceEntry       // 调用链条目
}

// TraceEntry 单个调用链条目
type TraceEntry struct {
	Timestamp time.Time // 发生时间
	Component string    // 组件类型
	Name      string    // 组件名称
	Event     string    // 事件类型（start/end/error）
	Duration  time.Duration // 耗时（仅 end/error 时有效）
}

// traceIDKey context 中存储 trace ID 的 key
type traceIDKey struct{}

// trackerKey context 中存储 tracker 的 key
type trackerKey struct{}

// TraceHandler 链路追踪回调处理器
type TraceHandler struct {
	trackers sync.Map // 存储所有请求的 tracker，key 是 traceID
}

// NewTraceHandler 创建新的链路追踪处理器
func NewTraceHandler() *TraceHandler {
	return &TraceHandler{}
}

// StartTrace 开始一个新的追踪
func (h *TraceHandler) StartTrace(ctx context.Context) (context.Context, string) {
	// 生成简单的追踪 ID（实际项目中应该使用 UUID）
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())

	// 创建追踪器
	tracker := &RequestTracker{
		traceID: traceID,
		entries: make([]TraceEntry, 0),
	}

	// 存储追踪器
	h.trackers.Store(traceID, tracker)

	// 将 traceID 和 tracker 存入 context
	ctx = context.WithValue(ctx, traceIDKey{}, traceID)
	ctx = context.WithValue(ctx, trackerKey{}, tracker)

	return ctx, traceID
}

// GetTracker 从 context 中获取追踪器
func (h *TraceHandler) GetTracker(ctx context.Context) *RequestTracker {
	if tracker, ok := ctx.Value(trackerKey{}).(*RequestTracker); ok {
		return tracker
	}
	return nil
}

// PrintTrace 输出完整的调用链
func (h *TraceHandler) PrintTrace(traceID string) {
	if v, ok := h.trackers.Load(traceID); ok {
		tracker := v.(*RequestTracker)
		tracker.mu.Lock()
		defer tracker.mu.Unlock()

		fmt.Printf("\n=== 调用链详情 (Trace ID: %s) ===\n", tracker.traceID)
		fmt.Printf("共 %d 个调用\n\n", len(tracker.entries))

		for i, entry := range tracker.entries {
			indent := strings.Repeat("  ", 0) // 可以根据嵌套层级调整
			switch entry.Event {
			case "start":
				fmt.Printf("%s[%d] ▶ %s/%s 开始\n", indent, i+1, entry.Component, entry.Name)
			case "end":
				fmt.Printf("%s[%d] ■ %s/%s 完成 (耗时: %v)\n", indent, i+1, entry.Component, entry.Name, entry.Duration)
			case "error":
				fmt.Printf("%s[%d] ✖ %s/%s 失败 (耗时: %v)\n", indent, i+1, entry.Component, entry.Name, entry.Duration)
			}
		}
		fmt.Println()
	}
}

// OnStart 记录开始事件
func (h *TraceHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info != nil {
		tracker := h.GetTracker(ctx)
		if tracker != nil {
			tracker.mu.Lock()
			tracker.entries = append(tracker.entries, TraceEntry{
				Timestamp: time.Now(),
				Component: string(info.Component),
				Name:      info.Name,
				Event:     "start",
			})
			tracker.mu.Unlock()
		}

		// 同时记录开始时间，用于计算耗时
		return h.PerfHandler().OnStart(ctx, info, input)
	}
	return ctx
}

// OnEnd 记录结束事件
func (h *TraceHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info != nil {
		tracker := h.GetTracker(ctx)
		if tracker != nil {
			// 计算耗时
			var duration time.Duration
			tracker.mu.Lock()
			// 查找对应的 start 事件
			for i := len(tracker.entries) - 1; i >= 0; i-- {
				if tracker.entries[i].Component == string(info.Component) &&
					tracker.entries[i].Name == info.Name &&
					tracker.entries[i].Event == "start" {
					duration = time.Since(tracker.entries[i].Timestamp)
					break
				}
			}
			tracker.entries = append(tracker.entries, TraceEntry{
				Timestamp: time.Now(),
				Component: string(info.Component),
				Name:      info.Name,
				Event:     "end",
				Duration:  duration,
			})
			tracker.mu.Unlock()
		}
	}
	return ctx
}

// OnError 记录错误事件
func (h *TraceHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info != nil {
		tracker := h.GetTracker(ctx)
		if tracker != nil {
			var duration time.Duration
			tracker.mu.Lock()
			for i := len(tracker.entries) - 1; i >= 0; i-- {
				if tracker.entries[i].Component == string(info.Component) &&
					tracker.entries[i].Name == info.Name &&
					tracker.entries[i].Event == "start" {
					duration = time.Since(tracker.entries[i].Timestamp)
					break
				}
			}
			tracker.entries = append(tracker.entries, TraceEntry{
				Timestamp: time.Now(),
				Component: string(info.Component),
				Name:      info.Name,
				Event:     "error",
				Duration:  duration,
			})
			tracker.mu.Unlock()
		}
	}
	return ctx
}

// OnStartWithStreamInput 流式输入开始
func (h *TraceHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return h.OnStart(ctx, info, nil)
}

// OnEndWithStreamOutput 流式输出结束
func (h *TraceHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	return h.OnEnd(ctx, info, nil)
}

// PerfHandler 返回一个性能监控处理器（用于计算耗时）
func (h *TraceHandler) PerfHandler() *PerfHandler {
	return &PerfHandler{}
}

func traceCallback() {
	fmt.Println("=== 示例 4：链路追踪示例 ===")
	fmt.Println("演示如何追踪请求经过的所有组件")
	fmt.Println()

	// 创建链路追踪处理器
	traceHandler := NewTraceHandler()
	callbacks.AppendGlobalHandlers(traceHandler)

	// 模拟一个请求
	ctx := context.Background()
	ctx, traceID := traceHandler.StartTrace(ctx)
	fmt.Printf("开始追踪，Trace ID: %s\n", traceID)

	// 模拟组件调用
	fmt.Println("\n模拟组件调用...")

	// 模拟 ChatModel 调用
	ctx = callbacks.OnStart(ctx, &callbacks.RunInfo{
		Name:      "chat_model",
		Type:      "OpenAI",
		Component: "ChatModel",
	}, nil)
	time.Sleep(100 * time.Millisecond) // 模拟处理时间
	_ = callbacks.OnEnd(ctx, nil, nil)

	// 模拟 Tool 调用
	ctx = callbacks.OnStart(ctx, &callbacks.RunInfo{
		Name:      "search_tool",
		Type:      "Custom",
		Component: "Tool",
	}, nil)
	time.Sleep(50 * time.Millisecond) // 模拟处理时间
	_ = callbacks.OnEnd(ctx, nil, nil)

	// 输出调用链
	traceHandler.PrintTrace(traceID)
}

// ============================================================================
// 示例 5：带回调监控的完整对话
// ============================================================================
// 整合以上所有回调，创建一个带完整监控的对话程序
// ============================================================================

// TokenStats Token 消耗统计
type TokenStats struct {
	mu              sync.Mutex
	TotalRequests   int   // 总请求数
	TotalTokens     int   // 总 Token 数
	PromptTokens    int   // 输入 Token 数
	CompletionTokens int  // 输出 Token 数
}

// tokenStatsKey context 中存储 Token 统计的 key
type tokenStatsKey struct{}

// TokenStatsHandler Token 统计回调处理器
type TokenStatsHandler struct {
	stats *TokenStats
}

// NewTokenStatsHandler 创建 Token 统计处理器
func NewTokenStatsHandler() *TokenStatsHandler {
	return &TokenStatsHandler{
		stats: &TokenStats{},
	}
}

// OnStart 记录请求开始
func (h *TokenStatsHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info != nil && string(info.Component) == "ChatModel" {
		h.stats.mu.Lock()
		h.stats.TotalRequests++
		h.stats.mu.Unlock()
	}
	return ctx
}

// OnEnd 解析 Token 使用情况
// 注意：这里演示如何解析 ChatModel 的回调输出
// 实际的 TokenUsage 解析需要根据具体的组件实现
func (h *TokenStatsHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	// 注意：实际项目中，output 会被转换为具体的类型（如 *model.CallbackOutput）
	// 这里为了演示，我们只是简单记录
	if info != nil && string(info.Component) == "ChatModel" {
		// 在实际使用中，可以这样解析：
		// if modelOutput, ok := output.(*model.CallbackOutput); ok {
		//     if modelOutput.TokenUsage != nil {
		//         h.stats.PromptTokens += modelOutput.TokenUsage.PromptTokens
		//         h.stats.CompletionTokens += modelOutput.TokenUsage.CompletionTokens
		//         h.stats.TotalTokens += modelOutput.TokenUsage.TotalTokens
		//     }
		// }
	}
	return ctx
}

// OnError 记录错误
func (h *TokenStatsHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	return ctx
}

// OnStartWithStreamInput 流式输入
func (h *TokenStatsHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return h.OnStart(ctx, info, nil)
}

// OnEndWithStreamOutput 流式输出
func (h *TokenStatsHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	return h.OnEnd(ctx, info, nil)
}

// GetStats 获取统计数据
func (h *TokenStatsHandler) GetStats() *TokenStats {
	return h.stats
}

// PrintStats 输出统计信息
func (h *TokenStatsHandler) PrintStats() {
	h.stats.mu.Lock()
	defer h.stats.mu.Unlock()

	fmt.Println("\n=== Token 使用统计 ===")
	fmt.Printf("总请求数: %d\n", h.stats.TotalRequests)
	fmt.Printf("总 Token 数: %d\n", h.stats.TotalTokens)
	fmt.Printf("输入 Token: %d\n", h.stats.PromptTokens)
	fmt.Printf("输出 Token: %d\n", h.stats.CompletionTokens)
	if h.stats.TotalRequests > 0 {
		avgTokens := h.stats.TotalTokens / h.stats.TotalRequests
		fmt.Printf("平均每次请求: %d tokens\n", avgTokens)
	}
}

func chatWithCallback() {
	fmt.Println("=== 示例 5：带回调监控的完整对话 ===")
	fmt.Println("整合所有回调，创建带监控的对话程序")
	fmt.Println()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		fmt.Println("  export OPENAI_API_KEY=\"your-api-key-here\"")
		os.Exit(1)
	}

	ctx := context.Background()

	// ========== 注册回调处理器 ==========

	// 1. 日志处理器
	logHandler := createLogHandler()
	callbacks.AppendGlobalHandlers(logHandler)

	// 2. 性能监控处理器
	perfHandler := &PerfHandler{}
	callbacks.AppendGlobalHandlers(perfHandler)

	// 3. Token 统计处理器
	tokenHandler := NewTokenStatsHandler()
	callbacks.AppendGlobalHandlers(tokenHandler)

	fmt.Println("已注册回调处理器：日志、性能监控、Token 统计")
	fmt.Println()

	// ========== 创建 ChatModel ==========

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o",
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 维护对话历史
	messages := []*model.Message{
		model.SystemMessage("你是一个 helpful 的助手。"),
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("=== 带监控的 AI 对话系统 ===")
	fmt.Println("命令: 'stats' 查看统计, 'clear' 清空历史, 'quit' 退出")
	fmt.Println()

	for {
		fmt.Print("你: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// 处理命令
		switch userInput {
		case "quit":
			fmt.Println("\n再见！")
			tokenHandler.PrintStats()
			return
		case "stats":
			tokenHandler.PrintStats()
			continue
		case "clear":
			messages = []*model.Message{
				model.SystemMessage("你是一个 helpful 的助手。"),
			}
			fmt.Println("✓ 对话历史已清空")
			continue
		case "":
			continue
		}

		// 添加用户消息到历史
		messages = append(messages, model.UserMessage(userInput))

		// 调用模型（回调会自动触发）
		fmt.Print("AI: ")
		startTime := time.Now()

		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			fmt.Printf("调用失败: %v\n", err)
			continue
		}

		duration := time.Since(startTime)

		// 添加助手回复到历史
		messages = append(messages, model.AssistantMessage(resp.Content))

		// 输出回复和统计
		fmt.Println(resp.Content)
		fmt.Printf("[统计] 本次调用耗时: %v\n", duration)
		fmt.Println()
	}
}
