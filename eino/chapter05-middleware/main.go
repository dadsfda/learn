// ============================================================================
// 第 5 章：Middleware -- 横切关注点
// ============================================================================
//
// 本文件演示 Eino 框架的中间件（Middleware）模式。
//
// 由于 Eino 的 ADK 中间件需要完整的 Agent 运行环境（需要 API Key 等），
// 本示例采用"先理解原理，再看实际用法"的方式：
//
//   Part 1: 用纯 Go 模拟中间件的洋葱模型和装饰器模式（不需要 API Key）
//   Part 2: 展示如何在真实 Eino Agent 中使用中间件（需要 API Key）
//
// 运行方式：
//   go run main.go demo          - 运行完整中间件演示（模拟，不需要 API Key）
//   go run main.go chain         - 运行中间件链演示（模拟，不需要 API Key）
//   go run main.go logging       - 运行日志中间件演示
//   go run main.go auth          - 运行认证中间件演示
//   go run main.go ratelimit     - 运行限流中间件演示
//   go run main.go safetool      - 运行安全工具中间件演示
//   go run main.go eino          - 运行真实 Eino Agent 中间件示例（需要 API Key）
//
// ============================================================================

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Part 1: 模拟中间件系统（帮助理解原理）
// ============================================================================
//
// 这部分用纯 Go 代码模拟 Eino 的中间件模式。
// 不需要任何外部依赖，帮助你理解中间件的核心思想。
//
// ============================================================================

// --------------------------------------------------------------------------
// 1.1 核心类型定义
// --------------------------------------------------------------------------

// ToolCall 模拟一次工具调用的请求
// 在真实 Eino 中，对应 InvokableToolCallEndpoint 的参数
type ToolCall struct {
	ToolName string            // 工具名称，如 "read_file"
	Args     map[string]string // 工具参数，如 {"path": "/tmp/test.txt"}
}

// ToolResult 模拟工具调用的结果
type ToolResult struct {
	Output string // 工具输出
	Error  error  // 工具错误（nil 表示成功）
}

// ToolFunc 工具调用函数类型
// 这就是 InvokableToolCallEndpoint 的简化版
// 在真实 Eino 中：func(ctx, args string, opts ...tool.Option) (string, error)
type ToolFunc func(ctx context.Context, call ToolCall) ToolResult

// Middleware 中间件函数类型
// 接收一个"下一个"工具函数，返回一个包装后的新函数
// 这就是 WrapInvokableToolCall 的简化版
type Middleware func(next ToolFunc) ToolFunc

// --------------------------------------------------------------------------
// 1.2 模拟的工具实现
// --------------------------------------------------------------------------

// readFileTool 模拟读取文件的工具
func readFileTool(ctx context.Context, call ToolCall) ToolResult {
	path := call.Args["path"]
	fmt.Printf("      [工具执行] read_file(path: %q)\n", path)

	// 模拟一些文件不存在的情况
	if path == "nonexistent.txt" {
		return ToolResult{
			Output: "",
			Error:  fmt.Errorf("open %s: no such file or directory", path),
		}
	}

	// 模拟成功读取
	return ToolResult{
		Output: fmt.Sprintf("文件 %s 的内容：Hello, World!", path),
		Error:  nil,
	}
}

// listFilesTool 模拟列出文件的工具
func listFilesTool(ctx context.Context, call ToolCall) ToolResult {
	dir := call.Args["directory"]
	fmt.Printf("      [工具执行] list_files(directory: %q)\n", dir)

	return ToolResult{
		Output: "main.go\nREADME.md\ngo.mod\ngo.sum",
		Error:  nil,
	}
}

// getToolByName 根据名称获取工具实现
func getToolByName(name string) ToolFunc {
	switch name {
	case "read_file":
		return readFileTool
	case "list_files":
		return listFilesTool
	default:
		return func(ctx context.Context, call ToolCall) ToolResult {
			return ToolResult{Error: fmt.Errorf("未知工具: %s", name)}
		}
	}
}

// --------------------------------------------------------------------------
// 1.3 中间件实现
// --------------------------------------------------------------------------

// ===== 日志中间件 =====
//
// 功能：记录每个工具调用的名称、参数、结果和耗时
// 类比：就像餐厅的服务员记录每道菜的下单时间和上菜时间
//
// 在真实 Eino 中，对应实现 WrapInvokableToolCall 方法

func LoggingMiddleware() Middleware {
	return func(next ToolFunc) ToolFunc {
		// 返回的新函数就是"包装后的工具调用函数"
		return func(ctx context.Context, call ToolCall) ToolResult {
			// ---- 调用前（Before）----
			start := time.Now()
			fmt.Printf("    [LOG] >>> 开始调用工具: %s\n", call.ToolName)
			fmt.Printf("    [LOG]     参数: %v\n", call.Args)

			// ---- 调用原始工具 ----
			result := next(ctx, call)

			// ---- 调用后（After）----
			elapsed := time.Since(start)
			if result.Error != nil {
				fmt.Printf("    [LOG] <<< 工具 %s 执行失败 (%v): %v\n",
					call.ToolName, elapsed, result.Error)
			} else {
				// 只显示前 100 个字符
				output := result.Output
				if len(output) > 100 {
					output = output[:100] + "..."
				}
				fmt.Printf("    [LOG] <<< 工具 %s 执行成功 (%v): %s\n",
					call.ToolName, elapsed, output)
			}

			return result
		}
	}
}

// ===== 认证中间件 =====
//
// 功能：在工具调用前检查认证信息
// 类比：就像进公司大楼前刷卡，没卡进不去
//
// 在真实 Eino 中，通常在 BeforeAgent 方法中实现

func AuthMiddleware(validTokens map[string]bool) Middleware {
	return func(next ToolFunc) ToolFunc {
		return func(ctx context.Context, call ToolCall) ToolResult {
			// 从 context 中获取 token
			token, ok := ctx.Value("auth_token").(string)
			if !ok || token == "" {
				fmt.Printf("    [AUTH] 认证失败：缺少 auth_token\n")
				return ToolResult{
					Error: fmt.Errorf("认证失败：缺少 auth_token"),
				}
			}

			if !validTokens[token] {
				fmt.Printf("    [AUTH] 认证失败：无效的 token %q\n", token)
				return ToolResult{
					Error: fmt.Errorf("认证失败：无效的 token"),
				}
			}

			fmt.Printf("    [AUTH] 认证成功\n")
			return next(ctx, call)
		}
	}
}

// ===== 限流中间件 =====
//
// 功能：使用令牌桶算法限制工具调用频率
// 类比：就像高速收费站，每秒只放行 N 辆车
//
// 令牌桶算法原理：
//   - 桶里有 N 个令牌
//   - 每次调用消耗 1 个令牌
//   - 桶以固定速率补充令牌
//   - 桶空了就拒绝请求

type RateLimiter struct {
	mu         sync.Mutex   // 保护令牌桶的互斥锁
	tokens     float64      // 当前可用令牌数
	maxTokens  float64      // 令牌桶最大容量
	refillRate float64      // 每秒补充的令牌数
	lastRefill time.Time    // 上次补充令牌的时间
}

// NewRateLimiter 创建一个新的限流器
// maxTokens: 令牌桶最大容量（允许的突发请求数）
// refillRate: 每秒补充的令牌数（持续请求速率）
func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens, // 初始时桶是满的
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// allow 检查是否允许一次调用
func (r *RateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// 计算从上次到现在应该补充多少令牌
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.refillRate

	// 令牌数不能超过桶的容量
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	// 尝试消耗一个令牌
	if r.tokens >= 1 {
		r.tokens--
		return true
	}

	return false
}

func RateLimitMiddleware(limiter *RateLimiter) Middleware {
	return func(next ToolFunc) ToolFunc {
		return func(ctx context.Context, call ToolCall) ToolResult {
			if !limiter.allow() {
				fmt.Printf("    [RATE] 请求被限流，请稍后重试\n")
				return ToolResult{
					Error: fmt.Errorf("请求被限流，请稍后重试"),
				}
			}
			fmt.Printf("    [RATE] 限流检查通过\n")
			return next(ctx, call)
		}
	}
}

// ===== 安全工具中间件 (SafeToolMiddleware) =====
//
// 功能：捕获工具执行错误，转化为文本返回
// 效果：工具报错不会中断对话，模型会根据错误信息调整策略
//
// 这是 Eino 官方推荐的中间件！
// 真实场景：用户让 AI 读取不存在的文件
//   - 没有 SafeToolMiddleware → 对话崩溃
//   - 有 SafeToolMiddleware   → AI 收到错误信息，自动调整策略

func SafeToolMiddleware() Middleware {
	return func(next ToolFunc) ToolFunc {
		return func(ctx context.Context, call ToolCall) ToolResult {
			result := next(ctx, call)

			if result.Error != nil {
				// 将错误转化为文本，返回给模型
				// 模型可以看到 "[tool error] ..." 并据此调整回答
				fmt.Printf("    [SAFE] 捕获工具错误，转化为文本: %v\n", result.Error)
				return ToolResult{
					Output: fmt.Sprintf("[tool error] %v", result.Error),
					Error:  nil, // 注意：错误被"吞掉"了，转为 Output
				}
			}

			return result
		}
	}
}

// --------------------------------------------------------------------------
// 1.4 中间件链构建
// --------------------------------------------------------------------------

// Chain 将多个中间件组合成一条链
//
// 执行顺序（洋葱模型）：
//   请求方向: middlewares[0] → middlewares[1] → ... → middlewares[n] → 实际工具
//   响应方向: 实际工具 → middlewares[n] → ... → middlewares[1] → middlewares[0]
//
// 例如：Chain(A, B, C, tool)
//   请求: A.Before → B.Before → C.Before → tool
//   响应: tool → C.After → B.After → A.After

func Chain(tool ToolFunc, middlewares ...Middleware) ToolFunc {
	// 从最后一个中间件开始，逐个包裹
	// 最终 middlewares[0] 在最外层
	result := tool
	for i := len(middlewares) - 1; i >= 0; i-- {
		result = middlewares[i](result)
	}
	return result
}

// --------------------------------------------------------------------------
// 1.5 演示函数
// --------------------------------------------------------------------------

// demoMiddlewareChain 演示中间件链的完整执行过程
func demoMiddlewareChain() {
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("  中间件链演示 -- 洋葱模型")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println()

	// 创建限流器：每秒允许 5 个请求，桶容量 10
	limiter := NewRateLimiter(10, 5)

	// 创建中间件链（从外到内）
	handler := Chain(
		readFileTool,                                          // 实际工具
		SafeToolMiddleware(),                                  // 最内层：错误处理
		RateLimitMiddleware(limiter),                          // 第三层：限流
		AuthMiddleware(map[string]bool{"valid-token": true}),  // 第二层：认证
		LoggingMiddleware(),                                   // 最外层：日志
	)

	// ---- 测试 1: 正常请求 ----
	fmt.Println("--- 测试 1: 正常请求（带有效 token）---")
	ctx := context.WithValue(context.Background(), "auth_token", "valid-token")
	call := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "config.txt"},
	}
	result := handler(ctx, call)
	fmt.Printf("  最终结果: %q\n\n", result.Output)

	// ---- 测试 2: 工具报错（SafeToolMiddleware 生效）----
	fmt.Println("--- 测试 2: 工具报错（SafeToolMiddleware 捕获错误）---")
	call = ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "nonexistent.txt"},
	}
	result = handler(ctx, call)
	fmt.Printf("  最终结果: %q\n", result.Output)
	fmt.Printf("  错误信息: %v (应该为 nil，因为错误已被 SafeToolMiddleware 转化)\n\n", result.Error)

	// ---- 测试 3: 认证失败 ----
	fmt.Println("--- 测试 3: 认证失败 ---")
	ctxNoAuth := context.WithValue(context.Background(), "auth_token", "invalid-token")
	call = ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "config.txt"},
	}
	result = handler(ctxNoAuth, call)
	fmt.Printf("  最终结果: %q\n", result.Output)
	fmt.Printf("  错误信息: %v\n\n", result.Error)

	// ---- 测试 4: 缺少认证信息 ----
	fmt.Println("--- 测试 4: 缺少认证信息 ---")
	ctxEmpty := context.Background()
	result = handler(ctxEmpty, call)
	fmt.Printf("  最终结果: %q\n", result.Output)
	fmt.Printf("  错误信息: %v\n\n", result.Error)
}

// demoSingleMiddleware 逐个演示每个中间件的效果
func demoSingleMiddleware() {
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("  单独中间件演示")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println()

	ctx := context.WithValue(context.Background(), "auth_token", "valid-token")
	call := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "test.txt"},
	}

	// ---- 只用日志中间件 ----
	fmt.Println("--- 只用日志中间件 ---")
	handler := Chain(readFileTool, LoggingMiddleware())
	result := handler(ctx, call)
	fmt.Printf("  结果: %q\n\n", result.Output)

	// ---- 只用 SafeToolMiddleware ----
	fmt.Println("--- 只用 SafeToolMiddleware（工具报错场景）---")
	handler = Chain(readFileTool, SafeToolMiddleware())
	callError := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "nonexistent.txt"},
	}
	result = handler(ctx, callError)
	fmt.Printf("  结果: %q\n", result.Output)
	fmt.Printf("  错误: %v (应该是 nil)\n\n", result.Error)

	// ---- 只用限流中间件 ----
	fmt.Println("--- 只用限流中间件（模拟限流）---")
	limiter := NewRateLimiter(2, 0.5) // 桶容量 2，每秒补充 0.5 个
	handler = Chain(readFileTool, RateLimitMiddleware(limiter))
	for i := 0; i < 4; i++ {
		fmt.Printf("  第 %d 次调用:\n", i+1)
		result = handler(ctx, call)
		if result.Error != nil {
			fmt.Printf("    → 被限流: %v\n", result.Error)
		} else {
			fmt.Printf("    → 成功: %q\n", result.Output)
		}
	}
	fmt.Println()
}

// demoOnionModel 直观展示洋葱模型的执行顺序
func demoOnionModel() {
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("  洋葱模型执行顺序演示")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println()

	// 创建带标签的中间件，方便观察执行顺序
	makeTaggedMiddleware := func(tag string) Middleware {
		return func(next ToolFunc) ToolFunc {
			return func(ctx context.Context, call ToolCall) ToolResult {
				fmt.Printf("    [%s] >>> 请求进入（Before）\n", tag)
				result := next(ctx, call)
				fmt.Printf("    [%s] <<< 响应离开（After）\n", tag)
				return result
			}
		}
	}

	// 创建标记为 A、B、C 的中间件
	handler := Chain(
		readFileTool,
		makeTaggedMiddleware("C-安全工具"),
		makeTaggedMiddleware("B-限流"),
		makeTaggedMiddleware("A-日志"),
	)

	fmt.Println("  注册顺序: [A-日志, B-限流, C-安全工具]")
	fmt.Println("  执行顺序如下：")
	fmt.Println()

	ctx := context.WithValue(context.Background(), "auth_token", "token")
	call := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "test.txt"},
	}
	handler(ctx, call)

	fmt.Println()
	fmt.Println("  可以看到：")
	fmt.Println("  - 请求方向: A → B → C → 工具")
	fmt.Println("  - 响应方向: 工具 → C → B → A")
	fmt.Println("  - A 在最外层，最先处理请求，最后处理响应")
	fmt.Println("  - C 在最内层，最后处理请求，最先处理响应")
	fmt.Println()
}

// --------------------------------------------------------------------------
// 1.6 真实 Eino Agent 中间件用法（需要 API Key）
// --------------------------------------------------------------------------

// 以下是真实 Eino Agent 中间件的代码示例。
// 由于需要 API Key 和完整的 Eino 依赖，这里只展示代码结构，
// 不会在 demo 模式中运行。
//
// 如果你想运行真实示例，请：
//   1. 设置 OPENAI_API_KEY 环境变量
//   2. 运行 go run main.go eino

/*
// === 真实 Eino 中间件示例 ===
// 以下代码展示如何在真实 Eino Agent 中使用中间件

import (
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/adk/deep"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

// --- SafeToolMiddleware（官方推荐）---
type safeToolMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
    _ context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    _ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        result, err := endpoint(ctx, args, opts...)
        if err != nil {
            // 检查是否是中断重运行错误（需要原样传播）
            if _, ok := compose.IsInterruptRerunError(err); ok {
                return "", err
            }
            // 将错误转化为文本
            return fmt.Sprintf("[tool error] %v", err), nil
        }
        return result, nil
    }, nil
}

// --- 创建 Agent 并注册中间件 ---
func createAgentWithMiddleware(ctx context.Context) {
    apiKey := os.Getenv("OPENAI_API_KEY")

    // 创建 ChatModel
    cm, _ := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: apiKey,
    })

    // 创建 Agent，注册中间件
    agent, _ := deep.New(ctx, &deep.Config{
        Name:        "MiddlewareDemoAgent",
        ChatModel:   cm,
        Instruction: "你是一个 helpful 的助手。",

        // 中间件列表（从外到内）
        Handlers: []adk.ChatModelAgentMiddleware{
            &loggingMiddleware{},   // 日志（最外层）
            &authMiddleware{},      // 认证
            &safeToolMiddleware{},  // 安全工具（最内层）
        },

        // ChatModel 重试配置
        ModelRetryConfig: &adk.ModelRetryConfig{
            MaxRetries: 5,
            IsRetryAble: func(_ context.Context, err error) bool {
                return strings.Contains(err.Error(), "429") ||
                    strings.Contains(err.Error(), "Too Many Requests")
            },
        },
    })

    _ = agent
}
*/

// ============================================================================
// Part 2: 完整的独立中间件演示
// ============================================================================
//
// 以下函数组合使用所有中间件，展示完整的中间件处理流程。
// 不需要 API Key，所有工具调用都是模拟的。
//
// ============================================================================

// simulateAgentExecution 模拟一次 Agent 的执行过程
// 在真实 Eino 中，这个过程由 Agent 框架自动完成
func simulateAgentExecution(ctx context.Context, handler ToolFunc, calls []ToolCall) {
	for i, call := range calls {
		fmt.Printf("\n  ── 工具调用 %d/%d ──\n", i+1, len(calls))
		result := handler(ctx, call)
		if result.Error != nil {
			fmt.Printf("  最终错误: %v\n", result.Error)
		} else {
			fmt.Printf("  最终输出: %s\n", result.Output)
		}
	}
}

// runFullDemo 运行完整演示
func runFullDemo() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  Eino 中间件完整演示")
	fmt.Println("  模拟一个 AI Agent 使用中间件处理工具调用的完整过程")
	fmt.Println(strings.Repeat("=", 70))

	// 创建限流器
	limiter := NewRateLimiter(10, 5)

	// 构建中间件链
	handler := Chain(
		readFileTool,
		SafeToolMiddleware(),
		RateLimitMiddleware(limiter),
		AuthMiddleware(map[string]bool{"demo-token": true}),
		LoggingMiddleware(),
	)

	// 模拟一系列工具调用
	ctx := context.WithValue(context.Background(), "auth_token", "demo-token")

	calls := []ToolCall{
		{ToolName: "read_file", Args: map[string]string{"path": "config.txt"}},
		{ToolName: "read_file", Args: map[string]string{"path": "nonexistent.txt"}},
		{ToolName: "read_file", Args: map[string]string{"path": "data.json"}},
	}

	simulateAgentExecution(ctx, handler, calls)
	fmt.Println()
}

// ============================================================================
// main 函数
// ============================================================================

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "demo":
		runFullDemo()
	case "chain":
		demoMiddlewareChain()
	case "logging":
		demoSingleMiddleware()
	case "auth":
		demoAuthOnly()
	case "ratelimit":
		demoRateLimitOnly()
	case "safetool":
		demoSafeToolOnly()
	case "eino":
		demoEinoRealMiddleware()
	case "onion":
		demoOnionModel()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino 中间件学习示例")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run main.go demo        - 运行完整中间件演示（推荐！）")
	fmt.Println("  go run main.go onion       - 洋葱模型执行顺序演示")
	fmt.Println("  go run main.go chain       - 中间件链演示")
	fmt.Println("  go run main.go logging     - 单独日志中间件演示")
	fmt.Println("  go run main.go auth        - 单独认证中间件演示")
	fmt.Println("  go run main.go ratelimit   - 单独限流中间件演示")
	fmt.Println("  go run main.go safetool    - 单独安全工具中间件演示")
	fmt.Println("  go run main.go eino        - 真实 Eino Agent 中间件示例（需要 API Key）")
	fmt.Println()
	fmt.Println("建议学习顺序:")
	fmt.Println("  1. go run main.go demo     ← 先看完整效果")
	fmt.Println("  2. go run main.go onion    ← 理解洋葱模型")
	fmt.Println("  3. go run main.go logging  ← 逐个理解中间件")
	fmt.Println("  4. go run main.go auth")
	fmt.Println("  5. go run main.go ratelimit")
	fmt.Println("  6. go run main.go safetool")
}

// demoAuthOnly 单独演示认证中间件
func demoAuthOnly() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  认证中间件单独演示")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	handler := Chain(
		readFileTool,
		AuthMiddleware(map[string]bool{"my-secret-token": true}),
	)

	call := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "test.txt"},
	}

	// 测试 1: 有效 token
	fmt.Println("--- 测试 1: 有效 token ---")
	ctx := context.WithValue(context.Background(), "auth_token", "my-secret-token")
	result := handler(ctx, call)
	fmt.Printf("  结果: %v\n\n", result.Output)

	// 测试 2: 无效 token
	fmt.Println("--- 测试 2: 无效 token ---")
	ctx = context.WithValue(context.Background(), "auth_token", "wrong-token")
	result = handler(ctx, call)
	fmt.Printf("  错误: %v\n\n", result.Error)

	// 测试 3: 无 token
	fmt.Println("--- 测试 3: 无 token ---")
	ctx = context.Background()
	result = handler(ctx, call)
	fmt.Printf("  错误: %v\n\n", result.Error)
}

// demoRateLimitOnly 单独演示限流中间件
func demoRateLimitOnly() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  限流中间件单独演示")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	// 创建一个严格的限流器：桶容量 3，每秒补充 1 个令牌
	limiter := NewRateLimiter(3, 1)
	handler := Chain(readFileTool, RateLimitMiddleware(limiter))

	ctx := context.WithValue(context.Background(), "auth_token", "token")
	call := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "test.txt"},
	}

	fmt.Println("限流器配置: 桶容量=3, 每秒补充=1")
	fmt.Println("连续发送 6 个请求：")
	fmt.Println()

	for i := 0; i < 6; i++ {
		fmt.Printf("  请求 %d:\n", i+1)
		result := handler(ctx, call)
		if result.Error != nil {
			fmt.Printf("    → 被限流: %v\n", result.Error)
		} else {
			fmt.Printf("    → 成功\n")
		}
	}

	fmt.Println()
	fmt.Println("可以看到：前 3 个请求成功（桶里有 3 个令牌），第 4 个被限流")
}

// demoSafeToolOnly 单独演示安全工具中间件
func demoSafeToolOnly() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  安全工具中间件 (SafeToolMiddleware) 单独演示")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.WithValue(context.Background(), "auth_token", "token")

	// ---- 不使用 SafeToolMiddleware ----
	fmt.Println("--- 不使用 SafeToolMiddleware ---")
	handler := Chain(readFileTool) // 只有原始工具，没有中间件
	call := ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "nonexistent.txt"},
	}
	result := handler(ctx, call)
	fmt.Printf("  输出: %q\n", result.Output)
	fmt.Printf("  错误: %v\n", result.Error)
	fmt.Println("  → 错误会直接抛出，导致对话中断！")
	fmt.Println()

	// ---- 使用 SafeToolMiddleware ----
	fmt.Println("--- 使用 SafeToolMiddleware ---")
	handler = Chain(readFileTool, SafeToolMiddleware())
	result = handler(ctx, call)
	fmt.Printf("  输出: %q\n", result.Output)
	fmt.Printf("  错误: %v (应该是 nil)\n", result.Error)
	fmt.Println("  → 错误被转化为文本，模型可以据此调整策略！")
	fmt.Println()

	// ---- 成功的调用不受影响 ----
	fmt.Println("--- 成功的调用不受影响 ---")
	call = ToolCall{
		ToolName: "read_file",
		Args:     map[string]string{"path": "test.txt"},
	}
	result = handler(ctx, call)
	fmt.Printf("  输出: %q\n", result.Output)
	fmt.Printf("  错误: %v\n", result.Error)
}

// demoEinoRealMiddleware 展示真实 Eino 中间件的用法
func demoEinoRealMiddleware() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("此示例需要 OPENAI_API_KEY 环境变量。")
		fmt.Println()
		fmt.Println("请按以下步骤操作：")
		fmt.Println("  1. 设置环境变量: export OPENAI_API_KEY=\"your-key\"")
		fmt.Println("  2. 初始化模块: go mod init chapter05-middleware")
		fmt.Println("  3. 安装依赖:   go get github.com/cloudwego/eino github.com/cloudwego/eino-ext/components/model/openai")
		fmt.Println("  4. 取消 main.go 中真实 Eino 代码的注释")
		fmt.Println("  5. 重新运行:   go run main.go eino")
		fmt.Println()
		fmt.Println("或者运行模拟演示: go run main.go demo")
		return
	}

	fmt.Println("真实 Eino Agent 中间件示例")
	fmt.Println("请参考 main.go 中的注释代码，取消注释后运行。")
	fmt.Println("当前 API Key 已配置，长度:", len(apiKey))
}
