# 第 5 章：Middleware -- 横切关注点

## 学习目标

通过本章学习，你将掌握：

1. **Middleware 模式**：理解中间件（拦截器）的核心思想和设计模式
2. **ChatModelAgentMiddleware 接口**：掌握 Eino Agent 中间件的标准接口
3. **横切关注点处理**：实现日志、认证、限流、错误处理等通用功能
4. **中间件链组合**：理解洋葱模型，学会组合多个中间件
5. **SafeToolMiddleware**：将工具错误转化为模型可理解的格式
6. **ModelRetryConfig**：为 ChatModel 配置自动重试策略

## 前置知识

- 第 1 章：ChatModel 和 Message
- 第 2 章：ChatModelAgent 和 Runner
- 第 4 章：Tools 和 FileSystem
- Go 接口和结构体嵌入
- Go 函数式编程（高阶函数、闭包）
- context.Context 使用

## 核心概念

### 1. 什么是横切关注点？

在软件开发中，有些功能不属于某个特定的业务模块，而是"横切"贯穿整个系统：

```
          +-------+-------+-------+
          | 模块A | 模块B | 模块C |
          +-------+-------+-------+
     =====|=======|=======|=======|=====  <-- 日志
     =====|=======|=======|=======|=====  <-- 认证
     =====|=======|=======|=======|=====  <-- 限流
          +-------+-------+-------+
```

常见的横切关注点包括：

| 关注点 | 说明 | 示例 |
|--------|------|------|
| **日志记录** | 记录请求/响应信息 | 打印工具调用参数和返回值 |
| **认证鉴权** | 验证调用者身份 | 检查 API Key、Token |
| **限流控制** | 限制请求频率 | 每秒最多 N 次工具调用 |
| **错误处理** | 统一处理异常 | 将工具错误转为文本返回给模型 |
| **重试机制** | 自动重试失败操作 | API 429 限流时指数退避重试 |
| **可观测性** | 追踪和监控 | OpenTelemetry 集成 |

**核心思想**：这些功能不应该和业务逻辑耦合，而是通过中间件"织入"到调用链中。

### 2. Middleware 模式详解

Eino 使用**装饰器模式（Decorator Pattern）**实现中间件。每个中间件包裹原始调用，可以在调用前、调用后、出错时插入自定义逻辑。

#### 洋葱模型（Onion Model）

中间件的执行顺序像洋葱一样层层嵌套：

```
请求方向 (外 → 内)
    ┌─────────────────────────────────┐
    │  Middleware A (Before)           │
    │  ┌─────────────────────────────┐│
    │  │  Middleware B (Before)       ││
    │  │  ┌─────────────────────────┐││
    │  │  │  Middleware C (Before)   │││
    │  │  │  ┌─────────────────────┐│││
    │  │  │  │  实际工具执行        ││││
    │  │  │  └─────────────────────┘│││
    │  │  │  Middleware C (After)    │││
    │  │  └─────────────────────────┘││
    │  │  Middleware B (After)        ││
    │  └─────────────────────────────┘│
    │  Middleware A (After)            │
    └─────────────────────────────────┘
响应方向 (内 → 外)
```

#### 执行顺序

```go
// 注册顺序决定执行顺序
Handlers: []adk.ChatModelAgentMiddleware{
    &loggingMiddleware{},   // 最外层：最先拦截请求，最后处理响应
    &authMiddleware{},      // 第二层
    &rateLimitMiddleware{}, // 第三层
    &safeToolMiddleware{},  // 最内层：最后拦截请求，最先处理响应
}
```

**请求流**：`logging → auth → rateLimit → safeTool → 实际工具`
**响应流**：`实际工具 → safeTool → rateLimit → auth → logging`

### 3. ChatModelAgentMiddleware 接口

Eino ADK 定义了 `ChatModelAgentMiddleware` 接口，这是 Agent 级别中间件的核心：

```go
type ChatModelAgentMiddleware interface {
    // ---- Agent 生命周期钩子 ----

    // BeforeAgent：Agent 执行前调用
    // 用途：认证检查、请求日志、注入上下文信息
    BeforeAgent(ctx context.Context, runCtx *ChatModelAgentContext) (
        context.Context, *ChatModelAgentContext, error)

    // ---- Model 调用钩子 ----

    // BeforeModelRewriteState：模型调用前，可以修改状态
    BeforeModelRewriteState(ctx context.Context, state *ChatModelAgentState,
        mc *ModelContext) (context.Context, *ChatModelAgentState, error)

    // AfterModelRewriteState：模型调用后，可以修改状态
    AfterModelRewriteState(ctx context.Context, state *ChatModelAgentState,
        mc *ModelContext) (context.Context, *ChatModelAgentState, error)

    // ---- Tool 调用包装 ----

    // WrapInvokableToolCall：包装同步工具调用
    // 这是最常用的中间件方法！
    WrapInvokableToolCall(ctx context.Context, endpoint InvokableToolCallEndpoint,
        tCtx *ToolContext) (InvokableToolCallEndpoint, error)

    // WrapStreamableToolCall：包装流式工具调用
    WrapStreamableToolCall(ctx context.Context, endpoint StreamableToolCallEndpoint,
        tCtx *ToolContext) (StreamableToolCallEndpoint, error)

    // WrapEnhancedInvokableToolCall：包装增强版同步工具调用
    WrapEnhancedInvokableToolCall(ctx context.Context,
        endpoint EnhancedInvokableToolCallEndpoint,
        tCtx *ToolContext) (EnhancedInvokableToolCallEndpoint, error)

    // WrapEnhancedStreamableToolCall：包装增强版流式工具调用
    WrapEnhancedStreamableToolCall(ctx context.Context,
        endpoint EnhancedStreamableToolCallEndpoint,
        tCtx *ToolContext) (EnhancedStreamableToolCallEndpoint, error)

    // ---- Model 包装 ----

    // WrapModel：包装 ChatModel 调用
    WrapModel(ctx context.Context, m model.BaseChatModel,
        mc *ModelContext) (model.BaseChatModel, error)
}
```

**关键理解**：

- `WrapInvokableToolCall` 是最重要的方法 -- 它接收原始的工具调用函数（endpoint），返回一个新的函数
- 新函数可以在调用原始函数**之前**和**之后**执行自定义逻辑
- 这就是装饰器模式的核心：**用一个新函数包裹旧函数**

### 4. BaseChatModelAgentMiddleware

Eino 提供了 `BaseChatModelAgentMiddleware` 基础结构体，它实现了接口的所有方法（默认是直接透传）。你只需要嵌入它，然后**只覆写你关心的方法**：

```go
// 嵌入 BaseChatModelAgentMiddleware，所有方法都有默认实现
type myMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

// 只覆写你关心的方法，其他方法自动透传
func (m *myMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // 返回一个新的函数，包裹原始调用
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        // 1. 调用前的逻辑（Before）
        fmt.Println("工具即将被调用")

        // 2. 调用原始工具
        result, err := endpoint(ctx, args, opts...)

        // 3. 调用后的逻辑（After）
        fmt.Println("工具调用完成")

        return result, err
    }, nil
}
```

### 5. InvokableToolCallEndpoint 类型

理解 `InvokableToolCallEndpoint` 是理解中间件的关键：

```go
// InvokableToolCallEndpoint 就是一个工具调用函数的类型
type InvokableToolCallEndpoint func(ctx context.Context, args string, opts ...tool.Option) (string, error)
```

它代表"一个可以被调用的工具"。中间件的工作就是：
1. 接收一个原始的 `endpoint`（原始工具调用函数）
2. 返回一个新的函数（包装后的工具调用函数）
3. 新函数内部可以修改输入、输出、错误处理

```
原始 endpoint:  参数 → 工具执行 → 结果/错误
包装后函数:     参数 → [前置逻辑] → 工具执行 → [后置逻辑] → 结果/错误
```

### 6. 中间件注册

通过 Agent 配置的 `Handlers` 字段注册中间件：

```go
agent, err := deep.New(ctx, &deep.Config{
    Name:     "MyAgent",
    ChatModel: chatModel,
    // ... 其他配置 ...

    // 注册中间件列表
    Handlers: []adk.ChatModelAgentMiddleware{
        &loggingMiddleware{},   // 日志中间件
        &authMiddleware{},      // 认证中间件
        &rateLimitMiddleware{}, // 限流中间件
        &safeToolMiddleware{},  // 安全工具中间件
    },
})
```

### 7. ModelRetryConfig -- ChatModel 重试

除了中间件，Eino 还提供了内置的重试配置，用于处理 ChatModel 的瞬时故障：

```go
agent, err := deep.New(ctx, &deep.Config{
    // ... 其他配置 ...

    ModelRetryConfig: &adk.ModelRetryConfig{
        MaxRetries: 5,  // 最大重试次数
        IsRetryAble: func(_ context.Context, err error) bool {
            // 判断错误是否可重试
            return strings.Contains(err.Error(), "429") ||
                strings.Contains(err.Error(), "Too Many Requests")
        },
    },
})
```

**重试策略**：指数退避（exponential backoff），每次重试等待时间翻倍。

## 代码示例

### 示例 1：日志中间件

记录 Agent 和工具调用的详细日志：

```go
// LoggingMiddleware 日志中间件
// 功能：记录 Agent 启动、工具调用参数、工具返回结果
type LoggingMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

// BeforeAgent 在 Agent 执行前记录日志
func (m *LoggingMiddleware) BeforeAgent(
    ctx context.Context,
    runCtx *adk.ChatModelAgentContext,
) (context.Context, *adk.ChatModelAgentContext, error) {
    fmt.Printf("[LOG] Agent 开始执行\n")
    return ctx, runCtx, nil
}

// WrapInvokableToolCall 包裹工具调用，记录调用详情
func (m *LoggingMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    toolName := tCtx.ToolName // 获取被调用的工具名称

    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        // 调用前：记录工具名称和参数
        fmt.Printf("[LOG] 调用工具: %s, 参数: %s\n", toolName, args)

        start := time.Now()

        // 调用原始工具
        result, err := endpoint(ctx, args, opts...)

        // 调用后：记录耗时和结果
        elapsed := time.Since(start)
        if err != nil {
            fmt.Printf("[LOG] 工具 %s 执行失败 (%v): %v\n", toolName, elapsed, err)
        } else {
            // 只显示前 200 个字符，避免日志过长
            displayResult := result
            if len(displayResult) > 200 {
                displayResult = displayResult[:200] + "..."
            }
            fmt.Printf("[LOG] 工具 %s 执行成功 (%v): %s\n", toolName, elapsed, displayResult)
        }

        return result, err
    }, nil
}
```

### 示例 2：认证中间件

在 Agent 执行前检查认证信息：

```go
// AuthMiddleware 认证中间件
// 功能：在 Agent 执行前检查 API Token
type AuthMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    ValidTokens map[string]bool // 合法的 Token 列表
}

// BeforeAgent 在 Agent 执行前检查认证
func (m *AuthMiddleware) BeforeAgent(
    ctx context.Context,
    runCtx *adk.ChatModelAgentContext,
) (context.Context, *adk.ChatModelAgentContext, error) {
    // 从 context 中获取 token
    token, ok := ctx.Value("auth_token").(string)
    if !ok || token == "" {
        return ctx, runCtx, fmt.Errorf("认证失败：缺少 auth_token")
    }

    // 验证 token 是否合法
    if !m.ValidTokens[token] {
        return ctx, runCtx, fmt.Errorf("认证失败：无效的 token")
    }

    fmt.Printf("[AUTH] 认证成功\n")
    return ctx, runCtx, nil
}
```

### 示例 3：限流中间件

限制工具调用的频率，防止 API 被限流：

```go
// RateLimitMiddleware 限流中间件
// 功能：使用令牌桶算法限制工具调用频率
type RateLimitMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    mu         sync.Mutex
    tokens     float64   // 当前可用令牌数
    maxTokens  float64   // 令牌桶容量
    refillRate float64   // 每秒补充的令牌数
    lastRefill time.Time // 上次补充时间
}

// allow 检查是否允许一次调用（令牌桶算法）
func (m *RateLimitMiddleware) allow() bool {
    m.mu.Lock()
    defer m.mu.Unlock()

    now := time.Now()
    // 计算从上次补充到现在应该补充多少令牌
    elapsed := now.Sub(m.lastRefill).Seconds()
    m.tokens += elapsed * m.refillRate
    if m.tokens > m.maxTokens {
        m.tokens = m.maxTokens
    }
    m.lastRefill = now

    // 尝试消耗一个令牌
    if m.tokens >= 1 {
        m.tokens--
        return true
    }
    return false
}

// WrapInvokableToolCall 包裹工具调用，执行限流检查
func (m *RateLimitMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        if !m.allow() {
            return "", fmt.Errorf("请求被限流，请稍后重试")
        }
        return endpoint(ctx, args, opts...)
    }, nil
}
```

### 示例 4：SafeToolMiddleware（安全工具中间件）

这是官方推荐的中间件，将工具运行时错误转化为模型可理解的文本：

```go
// SafeToolMiddleware 安全工具中间件
// 功能：捕获工具执行错误，转化为文本返回给模型
// 效果：工具报错不会中断对话，模型会根据错误信息调整策略
type SafeToolMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
}

func (m *SafeToolMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        result, err := endpoint(ctx, args, opts...)
        if err != nil {
            // 检查是否是中断重运行错误（需要原样传播）
            if _, ok := compose.IsInterruptRerunError(err); ok {
                return "", err
            }
            // 将错误转化为文本，返回给模型
            return fmt.Sprintf("[tool error] %v", err), nil
        }
        return result, nil
    }, nil
}
```

**关键点**：
- 普通错误 → 转化为 `[tool error] ...` 文本，模型可以看到并调整策略
- 中断重运行错误（InterruptRerun）→ 原样传播，不吞掉

### 示例 5：中间件组合

将多个中间件组合在一起：

```go
agent, err := deep.New(ctx, &deep.Config{
    Name:        "MiddlewareDemoAgent",
    ChatModel:   chatModel,
    Instruction: "你是一个 helpful 的助手。",
    // ... 其他配置 ...

    // 中间件列表：按洋葱模型从外到内排列
    Handlers: []adk.ChatModelAgentMiddleware{
        &LoggingMiddleware{},                          // 最外层：日志
        &AuthMiddleware{ValidTokens: validTokens},     // 认证
        &RateLimitMiddleware{                          // 限流
            maxTokens:  10,
            refillRate: 2,
            lastRefill: time.Now(),
            tokens:     10,
        },
        &SafeToolMiddleware{},                         // 最内层：错误处理
    },
})
```

**执行流程示例**：

```
用户: "帮我读取 config.txt 文件"
    │
    ▼
LoggingMiddleware.BeforeAgent    → [LOG] Agent 开始执行
    │
    ▼
AuthMiddleware.BeforeAgent       → [AUTH] 认证成功
    │
    ▼
Agent 分析意图，决定调用 read_file 工具
    │
    ▼
LoggingMiddleware.WrapInvokableToolCall
    → [LOG] 调用工具: read_file, 参数: {"file_path": "config.txt"}
    │
    ▼
RateLimitMiddleware.WrapInvokableToolCall
    → 检查令牌桶，允许通过
    │
    ▼
SafeToolMiddleware.WrapInvokableToolCall
    → 包裹实际工具调用
    │
    ▼
read_file 工具执行
    │
    ├── 成功: 返回文件内容
    │   → SafeToolMiddleware: 透传结果
    │   → RateLimitMiddleware: 透传结果
    │   → LoggingMiddleware: [LOG] 工具 read_file 执行成功 (15ms)
    │   → Agent: "文件内容如下..."
    │
    └── 失败: "file not found"
        → SafeToolMiddleware: 转为 "[tool error] file not found"
        → RateLimitMiddleware: 透传
        → LoggingMiddleware: [LOG] 工具 read_file 执行失败
        → Agent: "抱歉，config.txt 文件不存在..."
```

## 运行步骤

### 1. 初始化 Go 模块

```bash
cd chapter05-middleware
go mod init chapter05-middleware
```

### 2. 安装依赖

```bash
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

### 3. 设置环境变量

```bash
# Linux/Mac
export OPENAI_API_KEY="your-api-key-here"

# Windows PowerShell
$env:OPENAI_API_KEY = "your-api-key-here"
```

### 4. 运行示例

```bash
# 运行完整示例（纯模拟，不需要 API Key）
go run main.go demo

# 运行中间件链演示
go run main.go chain

# 运行独立中间件演示
go run main.go logging
go run main.go auth
go run main.go ratelimit
go run main.go safetool
```

## 常见问题

### Q1: 中间件和回调（Callback）有什么区别？

**A**：两者都是处理横切关注点的机制，但层次不同：

| 对比项 | 中间件 (Middleware) | 回调 (Callback) |
|--------|---------------------|-----------------|
| 层次 | Agent 级别 | 组件/Graph 级别 |
| 接口 | `ChatModelAgentMiddleware` | `callbacks.Handler` |
| 能力 | 可修改输入/输出/错误 | 只能观察，不能修改 |
| 适用场景 | Agent 行为控制 | 可观测性（日志、追踪） |
| 执行方式 | 洋葱模型 | 无序（不保证执行顺序） |

简单来说：
- **中间件** = 能修改行为的拦截器（可以改输入、改输出、吞掉错误）
- **回调** = 只能观察的监听器（记录日志、上报指标，不能改数据）

### Q2: 中间件的执行顺序是怎样的？

**A**：中间件按注册顺序从外到内执行（洋葱模型）：

```
注册顺序: [A, B, C]
请求执行: A.Before → B.Before → C.Before → 实际调用
响应执行: 实际调用 → C.After → B.After → A.After
```

**建议**：
- 日志中间件放最外层（最先拦截请求，最后处理响应）
- 错误处理中间件放最内层（紧贴实际调用）

### Q3: BaseChatModelAgentMiddleware 是必须的吗？

**A**：不是必须的，但强烈推荐。它提供了接口所有方法的默认实现（直接透传），你只需要覆写关心的方法。如果不嵌入它，你必须实现接口的**所有**方法。

### Q4: 如何在中间件中传递数据？

**A**：有几种方式：

1. **通过 context**：使用 `context.WithValue` 传递数据
2. **通过闭包**：中间件结构体本身就是闭包环境
3. **通过共享变量**：使用 sync.Mutex 保护的共享状态

```go
// 通过闭包传递配置
type myMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    config map[string]string  // 闭包环境中的配置
}
```

### Q5: 如何处理流式工具调用的错误？

**A**：流式工具调用需要使用 `WrapStreamableToolCall` 方法。错误处理有两种情况：

1. **调用前就出错**（如工具不存在）：直接返回错误转成的单 chunk 流
2. **流式传输中出错**：需要包裹 StreamReader，在读取时捕获错误

```go
func (m *SafeToolMiddleware) WrapStreamableToolCall(
    ctx context.Context,
    endpoint adk.StreamableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
    return func(ctx context.Context, args string, opts ...tool.Option) (
        *schema.StreamReader[string], error) {
        sr, err := endpoint(ctx, args, opts...)
        if err != nil {
            // 调用前出错：返回包含错误信息的单 chunk 流
            return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
        }
        // 调用成功：包裹流以捕获传输中的错误
        return safeWrapReader(sr), nil
    }, nil
}
```

### Q6: 中间件中的错误处理有哪些模式？

**A**：常见的错误处理模式：

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **吞掉错误** | 将错误转为文本返回 | SafeToolMiddleware |
| **传播错误** | 原样返回错误 | 认证失败 |
| **重试** | 捕获错误后重试 | 网络超时、429 限流 |
| **降级** | 返回默认值 | 缓存 miss 时查数据库 |
| **记录后传播** | 记录日志后返回原始错误 | 通用错误监控 |

## 练习题

### 练习 1：实现超时中间件

创建一个 `TimeoutMiddleware`，为每个工具调用设置超时时间：

```go
type TimeoutMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    Timeout time.Duration
}

// 提示：使用 context.WithTimeout 包裹调用
func (m *TimeoutMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // TODO: 实现超时控制
    // 1. 创建带超时的 context
    // 2. 用 goroutine 执行原始工具调用
    // 3. 等待结果或超时
    panic("请实现这个方法")
}
```

### 练习 2：实现缓存中间件

创建一个 `CacheMiddleware`，缓存工具调用的结果：

```go
type CacheMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    cache sync.Map // 缓存存储
}

// 提示：用工具名+参数作为 key，结果作为 value
func (m *CacheMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // TODO: 实现缓存逻辑
    // 1. 构造缓存 key: toolName + args
    // 2. 检查缓存是否命中
    // 3. 命中则返回缓存结果
    // 4. 未命中则调用原始工具并缓存结果
    panic("请实现这个方法")
}
```

### 练习 3：实现统计中间件

创建一个 `MetricsMiddleware`，收集工具调用的统计信息：

```go
type MetricsMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    mu          sync.Mutex
    callCount   map[string]int           // 每个工具的调用次数
    totalTime   map[string]time.Duration // 每个工具的总耗时
    errorCount  map[string]int           // 每个工具的错误次数
}

// 提示：在 WrapInvokableToolCall 中记录调用前后的时间差
func (m *MetricsMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // TODO: 实现统计收集
    panic("请实现这个方法")
}

// PrintReport 打印统计报告
func (m *MetricsMiddleware) PrintReport() {
    // TODO: 打印每个工具的调用次数、平均耗时、错误率
    panic("请实现这个方法")
}
```

### 练习 4：实现重试中间件

创建一个 `RetryMiddleware`，在工具调用失败时自动重试：

```go
type RetryMiddleware struct {
    *adk.BaseChatModelAgentMiddleware
    MaxRetries int
    IsRetryable func(err error) bool
}

// 提示：在闭包函数中使用 for 循环重试
func (m *RetryMiddleware) WrapInvokableToolCall(
    ctx context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // TODO: 实现重试逻辑
    // 1. 调用原始工具
    // 2. 如果失败且可重试，等待后重试
    // 3. 使用指数退避策略
    // 4. 超过最大重试次数后返回错误
    panic("请实现这个方法")
}
```

## 参考资料

### 官方资源

- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [Eino 示例代码](https://github.com/cloudwego/eino-examples)
- [Eino 官方文档 - 第 5 章：Middleware](https://www.cloudwego.io/docs/eino/quick_start/chapter_05_middleware/)
- [Eino 官方文档 - Callback 手册](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)
- [Eino ADK 中间件文档](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/)

### 设计模式参考

- [装饰器模式 - Refactoring Guru](https://refactoring.guru/design-patterns/decorator)
- [中间件模式 - Martin Fowler](https://martinfowler.com/articles/middleware-oriented-composition.html)
- [Go Context 包文档](https://pkg.go.dev/context)

### Go 语言相关

- [Go 并发模式](https://go.dev/doc/effective_go#concurrency)
- [Go 接口最佳实践](https://go.dev/doc/effective_go#interfaces)
- [sync 包文档](https://pkg.go.dev/sync)

## 本章小结

本章学习了 Eino 框架中处理横切关注点的核心机制 -- 中间件模式：

1. **横切关注点**是贯穿整个系统的通用功能（日志、认证、限流等）
2. **ChatModelAgentMiddleware** 是 Eino Agent 级别的中间件接口
3. **洋葱模型**决定了中间件的执行顺序：请求从外到内，响应从内到外
4. **装饰器模式**是中间件的核心设计模式：用新函数包裹旧函数
5. **BaseChatModelAgentMiddleware** 提供默认实现，减少样板代码
6. **SafeToolMiddleware** 将工具错误转为文本，防止对话中断
7. **ModelRetryConfig** 提供 ChatModel 级别的自动重试能力
8. 中间件可以自由组合，形成强大的处理管道

下一章我们将学习 **Callback 和 Trace**，了解 Eino 的可观测性机制。
