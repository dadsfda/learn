# 第 2 章：ChatModelAgent 和 Runner —— 多轮对话

## 学习目标

通过本章学习，你将掌握：

1. **ChatModelAgent**：理解智能体的核心概念和实现
2. **Runner**：掌握 Agent 运行器的工作原理
3. **AgentEvent**：理解事件流处理机制
4. **多轮对话管理**：实现更智能的对话系统

## 前置知识

- 第 1 章：ChatModel 和 Message
- Go 接口和结构体
- Channel 和 Goroutine

## 核心概念

### 1. ChatModelAgent

`ChatModelAgent` 是基于 ChatModel 的智能体实现，它封装了：
- 对话状态管理
- 工具调用能力
- 事件流处理

```go
type ChatModelAgent struct {
    Model   ChatModel          // 底层对话模型
    Tools   []Tool             // 可用工具列表
    Memory  Memory             // 记忆存储
    Config  ChatModelAgentConfig
}
```

### 2. Runner

`Runner` 是 Agent 的运行器，负责：
- 执行 Agent 逻辑
- 管理事件流
- 处理工具调用
- 管理会话状态

```go
type Runner struct {
    Agent   Agent
    Config  RunnerConfig
}

type RunnerConfig struct {
    MaxTurns     int           // 最大对话轮数
    Timeout      time.Duration // 超时时间
    ErrorHandler ErrorHandler  // 错误处理函数
}
```

### 3. AgentEvent

`AgentEvent` 代表 Agent 执行过程中的事件：

```go
type AgentEvent struct {
    Type     EventType  // 事件类型
    Message  *Message   // 消息内容
    ToolCall *ToolCall  // 工具调用请求
    Error    error      // 错误信息
}

type EventType int

const (
    EventMessage   EventType = iota // 消息事件
    EventToolCall                   // 工具调用事件
    EventError                      // 错误事件
    EventDone                       // 完成事件
)
```

### 4. 事件迭代器

使用迭代器模式处理事件流：

```go
type EventIterator struct {
    events chan AgentEvent
    done   chan struct{}
}

func (it *EventIterator) Next() (AgentEvent, bool) {
    select {
    case event, ok := <-it.events:
        return event, ok
    case <-it.done:
        return AgentEvent{}, false
    }
}
```

## 代码示例

### 示例 1：基础 ChatModelAgent

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/pkg/adk"
)

func main() {
    ctx := context.Background()

    // 1. 创建 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        fmt.Printf("创建 ChatModel 失败: %v\n", err)
        os.Exit(1)
    }

    // 2. 创建 ChatModelAgent
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
    })
    if err != nil {
        fmt.Printf("创建 Agent 失败: %v\n", err)
        os.Exit(1)
    }

    // 3. 创建 Runner
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent: agent,
    })

    // 4. 执行查询
    fmt.Println("正在查询 AI...")
    iter := runner.Query(ctx, "你好，请介绍一下你自己。")

    // 5. 处理事件流
    fmt.Println("\n=== AI 回复 ===")
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }

        switch event.Type {
        case adk.EventMessage:
            fmt.Print(event.Message.Content)
        case adk.EventError:
            fmt.Printf("\n错误: %v\n", event.Error)
        case adk.EventDone:
            fmt.Println("\n\n[对话完成]")
        }
    }
}
```

### 示例 2：带工具的 Agent

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/pkg/adk"
)

// 时间工具
type TimeTool struct{}

func (t *TimeTool) Info(ctx context.Context) (*tool.Info, error) {
    return &tool.Info{
        Name: "get_current_time",
        Desc: "获取当前时间",
        ParamsOneOf: tool.NewParamsOneOfByParams(
            map[string]*tool.ParameterInfo{},
        ),
    }, nil
}

func (t *TimeTool) Run(ctx context.Context, params map[string]any) (any, error) {
    return map[string]any{
        "current_time": time.Now().Format("2006-01-02 15:04:05"),
        "timezone":     "UTC+8",
    }, nil
}

// 计算器工具
type CalculatorTool struct{}

func (t *CalculatorTool) Info(ctx context.Context) (*tool.Info, error) {
    return &tool.Info{
        Name: "calculator",
        Desc: "执行数学计算",
        ParamsOneOf: tool.NewParamsOneOfByParams(
            map[string]*tool.ParameterInfo{
                "expression": {
                    Type:     "string",
                    Desc:     "数学表达式，如 '2 + 3 * 4'",
                    Required: true,
                },
            },
        ),
    }, nil
}

func (t *CalculatorTool) Run(ctx context.Context, params map[string]any) (any, error) {
    expr := params["expression"].(string)
    // 简单示例，实际应用需要实现表达式解析
    return map[string]any{
        "expression": expr,
        "result":     "42", // 模拟结果
        "note":       "这是一个示例，请在实际应用中实现表达式解析",
    }, nil
}

func main() {
    ctx := context.Background()

    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        fmt.Printf("创建 ChatModel 失败: %v\n", err)
        os.Exit(1)
    }

    // 创建带工具的 Agent
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{
                    &TimeTool{},
                    &CalculatorTool{},
                },
            },
        },
    })
    if err != nil {
        fmt.Printf("创建 Agent 失败: %v\n", err)
        os.Exit(1)
    }

    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent: agent,
    })

    // 测试工具调用
    queries := []string{
        "现在几点了？",
        "请计算 123 * 456",
        "你能做什么？",
    }

    for _, query := range queries {
        fmt.Printf("\n=== 查询: %s ===\n", query)
        iter := runner.Query(ctx, query)

        for {
            event, ok := iter.Next()
            if !ok {
                break
            }

            switch event.Type {
            case adk.EventMessage:
                fmt.Print(event.Message.Content)
            case adk.EventToolCall:
                fmt.Printf("\n[调用工具: %s]\n", event.ToolCall.Name)
            case adk.EventError:
                fmt.Printf("\n错误: %v\n", event.Error)
            case adk.EventDone:
                fmt.Println("\n")
            }
        }
    }
}
```

### 示例 3：交互式多轮对话 Agent

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/pkg/adk"
)

func main() {
    ctx := context.Background()

    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        fmt.Printf("创建 ChatModel 失败: %v\n", err)
        os.Exit(1)
    }

    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
    })
    if err != nil {
        fmt.Printf("创建 Agent 失败: %v\n", err)
        os.Exit(1)
    }

    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent: agent,
    })

    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println("=== AI Agent 对话系统 ===")
    fmt.Println("命令: 'quit' 退出, 'clear' 清空历史")
    fmt.Println()

    for {
        fmt.Print("你: ")
        if !scanner.Scan() {
            break
        }

        userInput := strings.TrimSpace(scanner.Text())
        if userInput == "quit" {
            fmt.Println("再见！")
            break
        }

        if userInput == "clear" {
            // 重新创建 Runner 以清空历史
            runner = adk.NewRunner(ctx, adk.RunnerConfig{
                Agent: agent,
            })
            fmt.Println("✓ 对话历史已清空")
            continue
        }

        if userInput == "" {
            continue
        }

        // 执行查询
        iter := runner.Query(ctx, userInput)

        fmt.Print("AI: ")
        for {
            event, ok := iter.Next()
            if !ok {
                break
            }

            switch event.Type {
            case adk.EventMessage:
                fmt.Print(event.Message.Content)
            case adk.EventToolCall:
                fmt.Printf("\n[调用工具: %s]\n", event.ToolCall.Name)
            case adk.EventError:
                fmt.Printf("\n错误: %v\n", event.Error)
            case adk.EventDone:
                fmt.Println("\n")
            }
        }
    }
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保已安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai

# 设置 API Key
export OPENAI_API_KEY="your-api-key-here"
```

### 2. 运行示例

```bash
# 进入章节目录
cd chapter02-chatmodel-agent-runner

# 运行示例 1：基础 Agent
go run main.go simple

# 运行示例 2：带工具的 Agent
go run main.go tools

# 运行示例 3：交互式对话
go run main.go interactive
```

## 常见问题

### Q1: ChatModelAgent 和直接使用 ChatModel 有什么区别？

**ChatModelAgent** 提供了更高层次的抽象：
- 自动管理对话状态
- 内置工具调用能力
- 事件流处理
- 更易扩展和组合

**直接使用 ChatModel**：
- 更底层，更灵活
- 需要手动管理状态
- 适合简单场景

### Q2: 如何自定义 Agent 行为？

可以通过以下方式：
1. **系统提示**：在 ChatModelAgentConfig 中设置 SystemPrompt
2. **工具**：添加自定义工具
3. **中间件**：使用 Middleware 拦截和修改行为
4. **回调**：使用 Callback 监听事件

### Q3: 如何处理长时间运行的工具？

```go
// 在工具实现中使用 Context
func (t *LongRunningTool) Run(ctx context.Context, params map[string]any) (any, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err() // 被取消
    case <-time.After(10 * time.Second):
        return result, nil // 正常完成
    }
}
```

## 练习题

### 练习 1：扩展工具集

创建一个包含以下工具的 Agent：
1. **天气查询工具**：查询指定城市天气
2. **新闻查询工具**：获取最新新闻
3. **翻译工具**：翻译文本

### 练习 2：Agent 对话记录

实现对话记录功能：
1. 记录所有对话历史
2. 支持导出为 JSON 文件
3. 支持从文件导入历史

### 练习 3：错误处理增强

改进错误处理：
1. 添加重试机制
2. 实现优雅降级
3. 记录错误日志

## 下一步学习

完成本章后，建议继续学习：

- **第 3 章**：Memory 和 Session —— 实现会话持久化
- **第 4 章**：Tools 和文件系统访问 —— 扩展工具能力

## 参考资料

- [Eino 官方文档 - ADK](https://www.cloudwego.io/docs/eino/adk/)
- [Eino 示例代码](https://github.com/cloudwego/eino-examples)
- [Go 并发编程](https://go.dev/blog/pipelines)

---

**上一章**：[第 1 章：ChatModel 和 Message](../chapter01-chatmodel-message/README.md)

**下一章**：[第 3 章：Memory 和 Session](../chapter03-memory-session/README.md)
