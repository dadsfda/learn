# 第 6 章：Callback 和 Trace —— 可观测性

## 学习目标

通过本章学习，你将掌握：

1. **Callback 机制**：理解 Eino 回调系统的核心设计和生命周期
2. **Handler 接口**：学会实现自定义回调处理器
3. **Trace 链路追踪**：掌握 LLM 应用的调用链追踪方法
4. **性能监控**：实现请求耗时、Token 消耗等指标统计
5. **生产级可观测性**：集成 CozeLoop 等追踪平台

## 前置知识

- Go 语言基础语法（接口、context、goroutine）
- Eino ChatModel 基础使用（第 1 章）
- Eino Chain/Graph 编排基础（第 5 章）
- 日志和错误处理最佳实践

## 核心概念

### 1. 为什么需要 Callback？

想象你开发了一个 AI Agent，用户反馈"回答很慢"或"有时候出错"。没有可观测性，你无法知道：

- 模型调用了几次？每次耗时多少？
- 工具执行是否成功？耗时多久？
- Token 消耗了多少？成本如何？
- 错误发生在哪个环节？

**Callback 就是 Eino 的"旁路机制"**——在不干扰主流程的前提下，在固定的生命周期节点提取信息。

```
用户请求 → [OnStart] → 模型调用 → [OnEnd] → 工具调用 → [OnEnd] → 返回响应
                ↓                        ↓                      ↓
            记录开始时间            记录模型响应            记录工具结果
                                    统计 Token              统计耗时
```

### 2. Callback 生命周期

Eino 定义了 5 个回调时机，对应组件执行的不同阶段：

```go
// 回调时机常量
const (
    TimingOnStart               // 组件开始处理前
    TimingOnEnd                 // 组件成功返回后
    TimingOnError               // 组件发生错误时
    TimingOnStartWithStreamInput  // 流式输入到达时
    TimingOnEndWithStreamOutput   // 流式输出返回时
)
```

**非流式调用流程**：
```
OnStart → 组件处理 → OnEnd（成功）或 OnError（失败）
```

**流式调用流程**：
```
OnStart → 组件处理（流式）→ OnEndWithStreamOutput → 逐块返回数据
```

> **注意**：流式处理中的错误不会触发 OnError，而是包含在 StreamReader 内部。

### 3. Handler 接口

`Handler` 是回调处理器的核心接口，包含 5 个方法：

```go
type Handler interface {
    // 非流式开始
    OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
    // 非流式结束
    OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
    // 错误发生
    OnError(ctx context.Context, info *RunInfo, err error) context.Context
    // 流式输入开始
    OnStartWithStreamInput(ctx context.Context, info *RunInfo,
        input *schema.StreamReader[CallbackInput]) context.Context
    // 流式输出结束
    OnEndWithStreamOutput(ctx context.Context, info *RunInfo,
        output *schema.StreamReader[CallbackOutput]) context.Context
}
```

**关键设计**：
- 每个方法都返回 `context.Context`，用于在同一 Handler 的不同时机之间传递状态
- 不同 Handler 之间没有执行顺序保证
- Input/Output 是共享的，不要修改它们

### 4. RunInfo 结构

`RunInfo` 描述触发回调的实体信息：

```go
type RunInfo struct {
    Name      string              // 业务名称（节点名或用户指定）
    Type      string              // 实现类型（如 "OpenAI"）
    Component components.Component // 组件类型（如 ChatModel）
}
```

**示例值**：
- `Name`: "my_chat_model"（Graph 中的节点名）
- `Type`: "OpenAI"（由组件实现设置）
- `Component`: "ChatModel"（抽象组件类型）

### 5. 注册回调的方式

Eino 提供多种注册回调的方式：

#### 全局注册

```go
// 对所有后续运行生效
callbacks.AppendGlobalHandlers(handler)
```

#### Graph 运行时注入

```go
// 对 Graph 中所有节点生效
r.Invoke(ctx, input, compose.WithCallbacks(handler))

// 对指定节点生效
r.Invoke(ctx, input,
    compose.WithCallbacks(handler).DesignateNode("node_name"))

// 对嵌套 Graph 中的节点生效
r.Invoke(ctx, input,
    compose.WithCallbacks(handler).DesignateNodeWithPath(
        compose.NewNodePath("outer_graph", "inner_node")))
```

#### Graph 外部使用

```go
ctx := callbacks.InitCallbacks(ctx, &callbacks.RunInfo{...}, handler...)
```

## 代码示例

### 示例 1：基础日志回调

最简单的回调实现——记录每次组件调用的开始和结束：

```go
package main

import (
    "context"
    "log"

    "github.com/cloudwego/eino/callbacks"
)

// SimpleLogHandler 实现基础日志回调
type SimpleLogHandler struct{}

// OnStart 记录组件开始处理
func (h *SimpleLogHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    if info != nil {
        log.Printf("[CALLBACK] OnStart - Component: %s, Name: %s, Type: %s",
            info.Component, info.Name, info.Type)
    }
    return ctx
}

// OnEnd 记录组件处理完成
func (h *SimpleLogHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    if info != nil {
        log.Printf("[CALLBACK] OnEnd - Component: %s, Name: %s",
            info.Component, info.Name)
    }
    return ctx
}

// OnError 记录组件错误
func (h *SimpleLogHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
    if info != nil {
        log.Printf("[CALLBACK] OnError - Component: %s, Name: %s, Error: %v",
            info.Component, info.Name, err)
    }
    return ctx
}

// OnStartWithStreamInput 记录流式输入开始
func (h *SimpleLogHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
    input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
    if info != nil {
        log.Printf("[CALLBACK] OnStartWithStreamInput - Component: %s, Name: %s",
            info.Component, info.Name)
    }
    return ctx
}

// OnEndWithStreamOutput 记录流式输出结束
func (h *SimpleLogHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
    output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
    if info != nil {
        log.Printf("[CALLBACK] OnEndWithStreamOutput - Component: %s, Name: %s",
            info.Component, info.Name)
    }
    return ctx
}
```

### 示例 2：使用 HandlerHelper 简化实现

不需要实现全部 5 个方法，使用 `HandlerHelper` 只注册需要的回调：

```go
package main

import (
    "context"
    "log"

    "github.com/cloudwego/eino/callbacks"
)

func createLogHandler() callbacks.Handler {
    return callbacks.NewHandlerHelper().
        OnStart(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
            log.Printf("[START] %s/%s", info.Component, info.Name)
            return ctx
        }).
        OnEnd(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
            log.Printf("[END] %s/%s", info.Component, info.Name)
            return ctx
        }).
        OnError(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
            log.Printf("[ERROR] %s/%s: %v", info.Component, info.Name, err)
            return ctx
        }).
        Handler()
}

// 使用方式
func main() {
    handler := createLogHandler()
    callbacks.AppendGlobalHandlers(handler)
    // ... 后续所有组件调用都会触发这个 Handler
}
```

### 示例 3：性能监控回调

统计组件调用耗时，记录到上下文中：

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/cloudwego/eino/callbacks"
)

// 定义上下文 key 类型
type traceKey struct{}

// TraceInfo 存储追踪信息
type TraceInfo struct {
    StartTime time.Time
    Component string
    Name      string
}

// PerfHandler 性能监控回调处理器
type PerfHandler struct{}

func (h *PerfHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    if info != nil {
        trace := &TraceInfo{
            StartTime: time.Now(),
            Component: string(info.Component),
            Name:      info.Name,
        }
        // 将开始时间存入上下文，在 OnEnd 时取出计算耗时
        return context.WithValue(ctx, traceKey{}, trace)
    }
    return ctx
}

func (h *PerfHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    if trace, ok := ctx.Value(traceKey{}).(*TraceInfo); ok {
        duration := time.Since(trace.StartTime)
        log.Printf("[PERF] %s/%s took %v", trace.Component, trace.Name, duration)
    }
    return ctx
}

func (h *PerfHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
    if trace, ok := ctx.Value(traceKey{}).(*TraceInfo); ok {
        duration := time.Since(trace.StartTime)
        log.Printf("[PERF-ERROR] %s/%s failed after %v: %v",
            trace.Component, trace.Name, duration, err)
    }
    return ctx
}

func (h *PerfHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
    input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
    return h.OnStart(ctx, info, nil)
}

func (h *PerfHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
    output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
    return h.OnEnd(ctx, info, nil)
}
```

### 示例 4：ChatModel 回调类型安全解析

针对 ChatModel 组件，可以解析出具体的输入输出信息：

```go
package main

import (
    "context"
    "log"

    "github.com/cloudwego/eino/callbacks"
    "github.com/cloudwego/eino/components/model"
)

// ModelTraceHandler ChatModel 专用回调处理器
type ModelTraceHandler struct{}

func (h *ModelTraceHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    // 尝试将 input 转换为 ChatModel 的回调输入
    if modelInput, ok := input.(*model.CallbackInput); ok {
        log.Printf("[MODEL-START] Messages: %d, Tools: %d",
            len(modelInput.Messages), len(modelInput.Tools))
    }
    return ctx
}

func (h *ModelTraceHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    // 尝试将 output 转换为 ChatModel 的回调输出
    if modelOutput, ok := output.(*model.CallbackOutput); ok {
        log.Printf("[MODEL-END] Response length: %d", len(modelOutput.Message.Content))
        if modelOutput.TokenUsage != nil {
            log.Printf("[MODEL-USAGE] Prompt: %d, Completion: %d, Total: %d",
                modelOutput.TokenUsage.PromptTokens,
                modelOutput.TokenUsage.CompletionTokens,
                modelOutput.TokenUsage.TotalTokens)
        }
    }
    return ctx
}

func (h *ModelTraceHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
    log.Printf("[MODEL-ERROR] %v", err)
    return ctx
}

func (h *ModelTraceHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
    input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
    return ctx
}

func (h *ModelTraceHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
    output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
    log.Printf("[MODEL-STREAM-END] Streaming completed")
    return ctx
}
```

### 示例 5：使用 HandlerBuilder 构建多组件回调

跨越多个组件类型，只处理特定时机：

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/cloudwego/eino/callbacks"
)

// 创建一个只关心 Start 和 End 的通用 Handler
func createTimingHandler() callbacks.Handler {
    startTimes := make(map[string]time.Time)

    return callbacks.NewHandlerBuilder().
        OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
            if info != nil {
                key := string(info.Component) + "/" + info.Name
                startTimes[key] = time.Now()
                log.Printf("[TIMING] Start: %s", key)
            }
            return ctx
        }).
        OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
            if info != nil {
                key := string(info.Component) + "/" + info.Name
                if start, ok := startTimes[key]; ok {
                    log.Printf("[TIMING] End: %s, Duration: %v", key, time.Since(start))
                    delete(startTimes, key)
                }
            }
            return ctx
        }).
        Build()
}
```

### 示例 6：集成 CozeLoop 追踪平台

CozeLoop 是 Eino 官方支持的 AI 应用可观测性平台：

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
    "github.com/cloudwego/eino/callbacks"
    "github.com/coze-dev/cozeloop-go"
)

func setupCozeLoop(ctx context.Context) {
    apiToken := os.Getenv("COZELOOP_API_TOKEN")
    workspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")

    if apiToken == "" || workspaceID == "" {
        log.Println("CozeLoop tracing disabled (missing env vars)")
        return
    }

    // 创建 CozeLoop 客户端
    client, err := cozeloop.NewClient(
        cozeloop.WithAPIToken(apiToken),
        cozeloop.WithWorkspaceID(workspaceID),
    )
    if err != nil {
        log.Printf("Failed to create CozeLoop client: %v", err)
        return
    }

    // 注册回调处理器
    callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
    log.Println("CozeLoop tracing enabled")

    // 确保程序退出时刷新数据
    defer func() {
        time.Sleep(5 * time.Second)
        client.Close(ctx)
    }()
}
```

### 示例 7：在 Graph 中使用回调

```go
package main

import (
    "context"
    "log"

    "github.com/cloudwego/eino/callbacks"
    "github.com/cloudwego/eino/compose"
)

func graphWithCallbacks(ctx context.Context) {
    // 创建回调 Handler
    handler := createLogHandler()

    // 方式 1：全局注册
    callbacks.AppendGlobalHandlers(handler)

    // 方式 2：运行时注入到 Graph
    // 假设 r 是已创建的 Graph Runner
    // r.Invoke(ctx, input, compose.WithCallbacks(handler))

    // 方式 3：只对特定节点生效
    // r.Invoke(ctx, input,
    //     compose.WithCallbacks(handler).DesignateNode("chat_model"))
}
```

## 完整示例：ChatModel + 回调监控

将以上知识整合，创建一个带完整回调监控的对话程序：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/callbacks"
    "github.com/cloudwego/eino/components/model"
)

func main() {
    ctx := context.Background()

    // 1. 注册全局回调处理器
    setupCallbacks()

    // 2. 创建 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatalf("创建 ChatModel 失败: %v", err)
    }

    // 3. 构建消息
    messages := []*model.Message{
        model.SystemMessage("你是一个 helpful 的助手。"),
        model.UserMessage("用一句话介绍 Go 语言。"),
    }

    // 4. 调用模型（回调会自动触发）
    fmt.Println("正在调用 AI 模型...")
    resp, err := chatModel.Generate(ctx, messages)
    if err != nil {
        log.Fatalf("调用失败: %v", err)
    }

    fmt.Printf("\nAI 回复: %s\n", resp.Content)
}

func setupCallbacks() {
    // 创建性能监控 Handler
    perfHandler := &PerfHandler{}
    callbacks.AppendGlobalHandlers(perfHandler)

    // 创建日志 Handler
    logHandler := callbacks.NewHandlerHelper().
        OnStart(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
            if info != nil {
                log.Printf("[TRACE] %s/%s start", info.Component, info.Name)
            }
            return ctx
        }).
        OnEnd(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
            if info != nil {
                log.Printf("[TRACE] %s/%s end", info.Component, info.Name)
            }
            return ctx
        }).
        OnError(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
            if info != nil {
                log.Printf("[TRACE] %s/%s error: %v", info.Component, info.Name, err)
            }
            return ctx
        }).
        Handler()
    callbacks.AppendGlobalHandlers(logHandler)
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.18
go version

# 设置 OpenAI API Key
export OPENAI_API_KEY="your-api-key-here"

# 可选：设置 CozeLoop 追踪
export COZELOOP_WORKSPACE_ID="your_workspace_id"
export COZELOOP_API_TOKEN="your_token"
```

### 2. 初始化项目

```bash
# 进入章节目录
cd chapter06-callback-trace

# 初始化 Go 模块
go mod init chapter06

# 安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai

# 可选：安装 CozeLoop 回调
go get github.com/cloudwego/eino-ext/callbacks/cozeloop
go get github.com/coze-dev/cozeloop-go
```

### 3. 创建 main.go

将本章的完整示例代码复制到 `main.go` 文件中。

### 4. 运行程序

```bash
# 运行示例
go run main.go

# 查看回调日志输出
# [TRACE] ChatModel/my_chat_model start
# [PERF] ChatModel/my_chat_model took 1.234s
# [TRACE] ChatModel/my_chat_model end
```

## 常见问题

### Q1: 为什么我的回调没有被触发？

**可能原因**：

1. **注册时机不对**：全局 Handler 必须在组件调用之前注册
2. **RunInfo 为 nil**：顶层调用可能没有 RunInfo，需要做 nil 检查
3. **组件未实现 Checker 接口**：某些组件可能不支持回调

**解决**：
```go
// 确保在组件创建之前注册
callbacks.AppendGlobalHandlers(handler)  // 先注册
chatModel, _ := openai.NewChatModel(...) // 再创建组件
```

### Q2: 流式回调中的 StreamReader 泄漏怎么办？

**原因**：流式回调中的 StreamReader 必须被正确关闭，否则会导致 goroutine 泄漏。

**解决**：
```go
func (h *MyHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
    output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

    // 复制一份流用于处理，不影响原始流
    sr, copy := output.Copy(2)

    // 处理副本
    go func() {
        defer copy.Close() // 必须关闭！
        for {
            chunk, err := copy.Recv()
            if err != nil {
                break
            }
            // 处理 chunk
            _ = chunk
        }
    }()

    return ctx
}
```

### Q3: 如何在回调之间传递数据？

**方案 1：通过 context（同一 Handler）**
```go
func (h *MyHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    // 存储开始时间
    return context.WithValue(ctx, "startTime", time.Now())
}

func (h *MyHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    // 取出开始时间
    if start, ok := ctx.Value("startTime").(time.Time); ok {
        log.Printf("Duration: %v", time.Since(start))
    }
    return ctx
}
```

**方案 2：通过共享变量（不同 Handler）**
```go
var requestMetrics = &sync.Map{} // 请求级别的共享变量

// 注意：需要自己处理并发安全
```

### Q4: 回调会影响性能吗？

**影响很小**，但需要注意：

- 回调在主 goroutine 中同步执行，避免耗时操作
- 耗时操作应该放到 goroutine 中异步处理
- 可以实现 `TimingChecker` 接口跳过不需要的时机

```go
// 实现 TimingChecker 接口
func (h *MyHandler) TimingCheck(t callbacks.CallbackTiming) bool {
    // 只关心 OnStart 和 OnEnd
    return t == callbacks.TimingOnStart || t == callbacks.TimingOnEnd
}
```

### Q5: 如何只监控特定组件？

**方案 1：使用 DesignateNode**
```go
r.Invoke(ctx, input,
    compose.WithCallbacks(handler).DesignateNode("target_node"))
```

**方案 2：在 Handler 中过滤**
```go
func (h *MyHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    if info != nil && info.Name == "target_node" {
        // 只处理目标节点
        log.Printf("Target node started: %s", info.Name)
    }
    return ctx
}
```

## 练习题

### 练习 1：基础回调实现

创建一个简单的回调处理器，要求：
1. 实现 `Handler` 接口的 5 个方法
2. 在 OnStart 中记录开始时间
3. 在 OnEnd 中计算并输出耗时
4. 在 OnError 中输出错误信息和耗时
5. 注册为全局 Handler 并测试

### 练习 2：Token 消耗统计

扩展回调处理器，要求：
1. 解析 ChatModel 的 CallbackOutput
2. 统计每次调用的 Token 消耗
3. 维护累计 Token 统计
4. 在程序结束时输出总消耗

### 练习 3：请求链路追踪

实现一个简单的链路追踪系统，要求：
1. 为每个请求生成唯一的 Trace ID
2. 在 context 中传递 Trace ID
3. 记录请求经过的所有组件
4. 最终输出完整的调用链

### 练习 4：错误告警

创建一个错误监控回调，要求：
1. 统计错误发生次数
2. 当错误率超过阈值时输出告警
3. 记录最近 N 次错误的详细信息
4. 支持按组件类型过滤

### 练习 5：集成 CozeLoop

尝试集成 CozeLoop 追踪平台，要求：
1. 注册 CozeLoop 账号并获取 API Token
2. 配置环境变量
3. 运行示例程序并查看追踪数据
4. 在 CozeLoop 控制台分析调用链

## 高级话题

### 1. 自定义组件的回调支持

如果你开发了自定义组件，可以在组件内部触发回调：

```go
func (c *MyComponent) Process(ctx context.Context, input string) (string, error) {
    // 触发 OnStart
    ctx = callbacks.OnStart(ctx, &MyCallbackInput{Input: input})

    // 执行业务逻辑
    output, err := c.doProcess(input)
    if err != nil {
        // 触发 OnError
        _ = callbacks.OnError(ctx, err)
        return "", err
    }

    // 触发 OnEnd
    _ = callbacks.OnEnd(ctx, &MyCallbackOutput{Output: output})
    return output, nil
}
```

### 2. 与 Prometheus 集成

将回调指标导出到 Prometheus：

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "eino_request_duration_seconds",
            Help: "Request duration in seconds",
        },
        []string{"component", "name"},
    )
)

type PrometheusHandler struct{}

func (h *PrometheusHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    if trace, ok := ctx.Value(traceKey{}).(*TraceInfo); ok {
        duration := time.Since(trace.StartTime).Seconds()
        requestDuration.WithLabelValues(trace.Component, trace.Name).Observe(duration)
    }
    return ctx
}
```

### 3. 分布式追踪

在微服务架构中，通过 context 传播 Trace ID：

```go
// 发送端
ctx = context.WithValue(ctx, "trace-id", traceID)

// 接收端（从 HTTP Header 中提取）
traceID := r.Header.Get("X-Trace-ID")
ctx = context.WithValue(ctx, "trace-id", traceID)
```

## 下一步学习

完成本章后，建议继续学习：

- **第 7 章**：中断与恢复 —— 实现长时间运行任务的暂停和继续
- **第 8 章**：Graph 和 Tool —— 构建复杂的工具调用链
- **第 9 章**：Skill 中间件 —— 实现可复用的业务逻辑

## 参考资料

- [Eino 官方文档 - Callback Manual](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)
- [Eino 官方文档 - Chapter 6: Callback and Trace](https://www.cloudwego.io/docs/eino/quick_start/chapter_06_callback_and_trace/)
- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [EinoExt 回调组件](https://github.com/cloudwego/eino-ext/tree/main/callbacks)
- [CozeLoop 文档](https://github.com/cloudwego/eino-ext/blob/main/callbacks/cozeloop/README.md)
- [Go Context 使用指南](https://go.dev/blog/context)

---

**上一章**：[第 5 章：Middleware —— 中间件机制](../chapter05-middleware/README.md)

**下一章**：[第 7 章：Interrupt 和 Resume —— 中断与恢复](../chapter07-interrupt-resume/README.md)
