# 第 4 章：Tools 和文件系统访问

## 学习目标

通过本章学习，你将掌握：

1. **Tool 接口体系**：理解 Eino 中工具的五层接口设计
2. **工具定义与实现**：学会创建自定义工具
3. **工具调用流程**：理解 ReAct 循环中工具的调用机制
4. **文件系统工具**：使用 Eino 内置的文件系统后端
5. **InferTool 快捷方式**：用泛型函数快速创建工具

## 前置知识

- 第 1 章：ChatModel 和 Message
- 第 2 章：ChatModelAgent 和 Runner
- Go 接口和结构体
- Go 泛型基础（1.18+）

## 核心概念

### 1. 什么是 Tool？

在 LLM 应用中，**Tool（工具）** 是赋予 AI "手和脚"的关键机制。没有工具，AI 只能"说"；有了工具，AI 能"做"。

```
用户提问 → LLM 思考 → 决定调用工具 → 执行工具 → 返回结果 → LLM 总结
```

**生活中的类比**：
- LLM 就像一个聪明的顾问，他知道该做什么
- Tool 就像顾问手里的各种工具（计算器、文件柜、搜索引擎）
- Agent 就是顾问拿着工具在工作

### 2. Tool 接口体系

Eino 设计了五层工具接口，从简单到复杂：

```
BaseTool                    ← 只提供元数据（名称、描述、参数）
  ├── InvokableTool         ← 可执行，输入 JSON 字符串，输出字符串
  ├── StreamableTool        ← 可执行，流式输出
  ├── EnhancedInvokableTool ← 可执行，支持多模态（图片、音频等）
  └── EnhancedStreamableTool← 可执行，流式多模态输出
```

#### BaseTool —— 最基础的接口

```go
type BaseTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
}
```

`Info` 方法告诉 LLM：
- 这个工具叫什么名字（`Name`）
- 这个工具做什么（`Desc`）
- 这个工具需要什么参数（`ParamsOneOf`）

**类比**：就像工具箱上的标签，告诉你工具的名称和用途。

#### InvokableTool —— 可执行的工具

```go
type InvokableTool interface {
    BaseTool
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}
```

- `argumentsInJSON`：LLM 生成的参数，格式为 JSON 字符串
- 返回值：工具执行结果的字符串

**类比**：你按照说明书（JSON 参数）操作工具，得到结果（字符串）。

#### StreamableTool —— 流式工具

```go
type StreamableTool interface {
    BaseTool
    StreamableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (*schema.StreamReader[string], error)
}
```

适用于需要长时间执行或逐步返回结果的工具。

### 3. ToolInfo 结构

`ToolInfo` 是工具的"名片"，告诉 LLM 如何使用这个工具：

```go
type ToolInfo struct {
    Name  string           // 工具名称，如 "read_file"
    Desc  string           // 工具描述，如 "读取文件内容"
    Extra map[string]any   // 额外信息
    *ParamsOneOf           // 参数定义
}
```

#### ParameterInfo —— 参数定义

```go
type ParameterInfo struct {
    Type      DataType                // 参数类型：string, number, integer, boolean, object, array
    Desc      string                  // 参数描述
    Required  bool                    // 是否必需
    Enum      []string                // 枚举值（可选）
    SubParams map[string]*ParameterInfo // 子参数（当 Type 为 object 时）
    ElemInfo  *ParameterInfo          // 数组元素信息（当 Type 为 array 时）
}
```

**示例**：定义一个读取文件的工具参数：

```go
map[string]*schema.ParameterInfo{
    "file_path": {
        Type:     schema.String,
        Desc:     "要读取的文件路径",
        Required: true,
    },
    "encoding": {
        Type: schema.String,
        Desc: "文件编码，如 'utf-8'",
        Enum: []string{"utf-8", "gbk", "ascii"},
    },
}
```

### 4. ReAct 循环

ReAct（Reasoning + Acting）是 Agent 使用工具的核心循环：

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│   用户提问                                          │
│      ↓                                              │
│   LLM 思考（Reasoning）                             │
│      ↓                                              │
│   LLM 决定：需要调用工具？                          │
│      ├── 否 → 返回最终答案                          │
│      └── 是 ↓                                       │
│   LLM 生成工具调用（Acting）                        │
│      ↓                                              │
│   框架执行工具                                      │
│      ↓                                              │
│   工具返回结果                                      │
│      ↓                                              │
│   结果加入对话历史                                  │
│      ↓                                              │
│   回到 LLM 思考...                                  │
│                                                     │
└─────────────────────────────────────────────────────┘
```

在 Eino 中，`ChatModelAgent` 自动实现了这个循环，你只需要：
1. 创建工具
2. 把工具注册到 Agent
3. 运行 Agent

### 5. 文件系统 Backend

Eino 提供了文件系统抽象层 `filesystem.Backend`，支持：

```go
type Backend interface {
    LsInfo(ctx context.Context, req *LsInfoRequest) ([]FileInfo, error)      // 列出目录
    Read(ctx context.Context, req *ReadRequest) (*FileContent, error)         // 读取文件
    GrepRaw(ctx context.Context, req *GrepRequest) ([]GrepMatch, error)      // 搜索内容
    GlobInfo(ctx context.Context, req *GlobInfoRequest) ([]FileInfo, error)  // 模式匹配
    Write(ctx context.Context, req *WriteRequest) error                       // 写入文件
    Edit(ctx context.Context, req *EditRequest) error                         // 编辑文件
}
```

内置实现：
- `InMemoryBackend`：内存文件系统，适合测试和演示

## 代码示例

### 示例 1：手动实现一个工具

最基础的方式：实现 `InvokableTool` 接口。

```go
package main

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/schema"
)

// WeatherTool 天气查询工具
type WeatherTool struct{}

// Info 返回工具的元数据
func (w *WeatherTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "get_weather",
        Desc: "查询指定城市的当前天气信息",
        ParamsOneOf: schema.NewParamsOneOfByParams(
            map[string]*schema.ParameterInfo{
                "city": {
                    Type:     schema.String,
                    Desc:     "城市名称，如 '北京'、'上海'",
                    Required: true,
                },
            },
        ),
    }, nil
}

// InvokableRun 执行工具
func (w *WeatherTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    // 在实际应用中，这里会调用天气 API
    // 这里用模拟数据演示
    return `{"city": "北京", "temperature": "22°C", "weather": "晴", "humidity": "45%"}`, nil
}
```

### 示例 2：使用 InferTool 快速创建工具

Eino 提供了泛型工具函数 `InferTool`，可以自动从 Go 结构体推导参数 Schema：

```go
package main

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/components/tool/utils"
)

// 定义参数结构体（字段的 json tag 会成为参数名）
type ReadFileInput struct {
    FilePath string `json:"file_path" description:"要读取的文件路径"`
    Encoding string `json:"encoding" description:"文件编码，默认 utf-8"`
}

// 定义工具函数
func readFile(ctx context.Context, input ReadFileInput) (string, error) {
    // 实际的文件读取逻辑
    return fmt.Sprintf("文件 %s 的内容...", input.FilePath), nil
}

func main() {
    // 一行代码创建工具！
    readFileTool, err := utils.InferTool("read_file", "读取指定路径的文件内容", readFile)
    if err != nil {
        panic(err)
    }
    // readFileTool 已经是一个完整的 InvokableTool 了
    _ = readFileTool
}
```

### 示例 3：带工具的 Agent

将工具注册到 Agent，让 LLM 自动决定何时调用：

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

// 定义工具...

func main() {
    ctx := context.Background()

    // 1. 创建 ChatModel
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        panic(err)
    }

    // 2. 创建带工具的 Agent
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model:       chatModel,
        Instruction: "你是一个文件管理助手，可以帮助用户读取和管理文件。",
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{
                    &ReadFileTool{},
                    &ListDirTool{},
                    // ... 更多工具
                },
            },
        },
    })
    if err != nil {
        panic(err)
    }

    // 3. 运行 Agent
    runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
    iter := runner.Query(ctx, "请帮我读取 config.txt 文件的内容")

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
        }
    }
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.18
go version

# 设置 API Key（选择你使用的 LLM 提供商）
# OpenAI
export OPENAI_API_KEY="your-api-key-here"

# 或者使用其他提供商，如 Anthropic
export ANTHROPIC_API_KEY="your-api-key-here"
```

### 2. 初始化项目

```bash
# 进入章节目录
cd chapter04-tools-filesystem

# 初始化 Go 模块
go mod init chapter04

# 安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

### 3. 运行示例

```bash
# 运行基础工具示例（不需要 API Key，本地演示）
go run main.go basic

# 运行文件系统工具示例（不需要 API Key，本地演示）
go run main.go filesystem

# 运行带 Agent 的完整示例（需要 API Key）
go run main.go agent

# 运行交互式示例（需要 API Key）
go run main.go interactive
```

## 常见问题

### Q1: BaseTool 和 InvokableTool 有什么区别？

- **BaseTool**：只需要实现 `Info()` 方法，提供工具的元数据。LLM 只能看到工具的描述，但无法执行它。
- **InvokableTool**：在 BaseTool 的基础上，增加了 `InvokableRun()` 方法，LLM 可以实际调用这个工具。

**什么时候用 BaseTool？**
- 你只想把工具定义传给模型，让模型生成调用参数，但执行逻辑在别处处理。

**什么时候用 InvokableTool？**
- 你希望框架自动执行工具并返回结果给模型。

### Q2: InferTool 和手动实现有什么区别？

**手动实现**：
- 完全控制参数解析和验证
- 适合复杂逻辑
- 代码量较多

**InferTool**：
- 自动从 Go 结构体推导 JSON Schema
- 自动处理 JSON 反序列化
- 代码简洁，推荐大多数场景使用

### Q3: 如何处理工具执行错误？

```go
func (t *MyTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    // 返回错误时，框架会将错误信息传递给 LLM
    // LLM 可以根据错误信息决定重试或告知用户
    if err != nil {
        return "", fmt.Errorf("读取文件失败: %w", err)
    }
    return result, nil
}
```

### Q4: 工具的参数类型有哪些？

Eino 支持以下参数类型：

| 类型 | 说明 | 示例 |
|------|------|------|
| `schema.String` | 字符串 | `"hello"` |
| `schema.Number` | 浮点数 | `3.14` |
| `schema.Integer` | 整数 | `42` |
| `schema.Boolean` | 布尔值 | `true` |
| `schema.Object` | 对象 | `{"key": "value"}` |
| `schema.Array` | 数组 | `[1, 2, 3]` |

### Q5: 如何限制工具的执行时间？

使用 Go 的 `context` 机制：

```go
func (t *MyTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    // 创建带超时的 context
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // 执行可能耗时的操作
    select {
    case <-ctx.Done():
        return "", fmt.Errorf("工具执行超时")
    case result := <-doSomething():
        return result, nil
    }
}
```

## 练习题

### 练习 1：创建一个计算器工具

实现一个计算器工具，支持基本的数学运算（加减乘除）。要求：
1. 接收两个数字和一个运算符
2. 返回计算结果
3. 处理除零错误

**提示**：使用 `InferTool` 快速创建。

### 练习 2：创建文件管理工具集

实现以下文件操作工具：
1. `read_file`：读取文件内容
2. `write_file`：写入文件
3. `list_files`：列出目录下的文件

**提示**：使用 Go 标准库的 `os` 和 `io/ioutil` 包。

### 练习 3：集成到 Agent

将练习 1 和练习 2 的工具集成到一个 Agent 中，实现：
1. 用户可以用自然语言请求文件操作
2. 用户可以进行数学计算
3. Agent 能自动选择合适的工具

### 练习 4：使用 InMemoryBackend

使用 Eino 内置的 `InMemoryBackend` 实现一个虚拟文件系统：
1. 创建内存文件系统
2. 预先写入一些测试文件
3. 创建工具让 Agent 可以操作这个虚拟文件系统

## 下一步学习

完成本章后，建议继续学习：

- **第 5 章**：Middleware —— 拦截和增强工具调用
- **第 6 章**：Callback 和 Trace —— 监控工具执行

## 参考资料

- [Eino 官方文档 - Tool](https://github.com/cloudwego/eino/tree/main/components/tool)
- [Eino 官方文档 - ADK](https://github.com/cloudwego/eino/tree/main/adk)
- [Eino 文件系统 Backend](https://github.com/cloudwego/eino/tree/main/adk/filesystem)
- [OpenAI Function Calling 文档](https://platform.openai.com/docs/guides/function-calling)
- [Go 泛型教程](https://go.dev/doc/tutorial/generics)

---

**上一章**：[第 3 章：Memory 和 Session](../chapter03-memory-session/README.md)

**下一章**：[第 5 章：Middleware](../chapter05-middleware/README.md)
