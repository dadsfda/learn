# 第 1 章：ChatModel 和 Message —— 控制台基础对话

## 学习目标

通过本章学习，你将掌握：

1. **ChatModel 接口**：理解 Eino 中对话模型的核心抽象
2. **Message 类型**：掌握消息的构建和使用方式
3. **基础 API 调用**：完成第一次 LLM 对话
4. **错误处理**：学会处理 API 调用中的异常情况

## 前置知识

- Go 语言基础语法
- Go 1.18 泛型基础
- Context 使用
- 环境变量配置

## 核心概念

### 1. ChatModel 接口

`ChatModel` 是 Eino 中最核心的接口之一，它定义了与大语言模型交互的标准方式：

```go
type ChatModel interface {
    // Generate 生成对话响应
    Generate(ctx context.Context, messages []*Message, opts ...CallOption) (*Message, error)

    // Stream 生成流式响应
    Stream(ctx context.Context, messages []*Message, opts ...CallOption) (*StreamReader, error)
}
```

**关键点**：
- `Generate`：同步调用，返回完整响应
- `Stream`：流式调用，逐步返回响应内容
- `CallOption`：可选配置参数

### 2. Message 结构

`Message` 代表对话中的一条消息：

```go
type Message struct {
    Role     Role     // 消息角色：System、User、Assistant、Tool
    Content  string   // 消息内容
    Name     string   // 可选：消息发送者名称
    ToolCalls []ToolCall // 可选：工具调用请求
}
```

**消息角色**：
- `RoleSystem`：系统提示，设定 AI 行为
- `RoleUser`：用户输入
- `RoleAssistant`：AI 响应
- `RoleTool`：工具返回结果

### 3. 消息构建辅助函数

Eino 提供了便捷的消息构建函数：

```go
// 创建系统消息
msg := model.SystemMessage("你是一个 helpful 的助手")

// 创建用户消息
msg := model.UserMessage("你好，请介绍一下自己")

// 创建助手消息
msg := model.AssistantMessage("你好！我是 AI 助手。")

// 创建工具消息
msg := model.ToolMessage("tool_id", "工具执行结果")
```

## 代码示例

### 示例 1：最简单的对话

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/model"
)

func main() {
    // 1. 创建上下文
    ctx := context.Background()

    // 2. 创建 ChatModel 实例
    chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
        Model:  "gpt-4o",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        fmt.Printf("创建 ChatModel 失败: %v\n", err)
        os.Exit(1)
    }

    // 3. 构建消息列表
    messages := []*model.Message{
        model.SystemMessage("你是一个 helpful 的助手，请用简洁明了的语言回答问题。"),
        model.UserMessage("请用一句话介绍 Go 语言的优势。"),
    }

    // 4. 调用模型生成响应
    resp, err := chatModel.Generate(ctx, messages)
    if err != nil {
        fmt.Printf("调用模型失败: %v\n", err)
        os.Exit(1)
    }

    // 5. 输出结果
    fmt.Println("AI 回复:")
    fmt.Println(resp.Content)
}
```

### 示例 2：多轮对话

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/model"
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

    // 维护对话历史
    messages := []*model.Message{
        model.SystemMessage("你是一个 helpful 的助手。"),
    }

    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println("欢迎使用 AI 对话系统！输入 'quit' 退出。")

    for {
        fmt.Print("\n你: ")
        if !scanner.Scan() {
            break
        }

        userInput := strings.TrimSpace(scanner.Text())
        if userInput == "quit" {
            fmt.Println("再见！")
            break
        }

        if userInput == "" {
            continue
        }

        // 添加用户消息到历史
        messages = append(messages, model.UserMessage(userInput))

        // 调用模型
        resp, err := chatModel.Generate(ctx, messages)
        if err != nil {
            fmt.Printf("调用模型失败: %v\n", err)
            continue
        }

        // 添加助手回复到历史
        messages = append(messages, model.AssistantMessage(resp.Content))

        // 输出回复
        fmt.Printf("\nAI: %s\n", resp.Content)
    }
}
```

### 示例 3：流式输出

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/cloudwego/eino-ext/components/model/openai"
    "github.com/cloudwego/eino/components/model"
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

    messages := []*model.Message{
        model.SystemMessage("你是一个 helpful 的助手。"),
        model.UserMessage("请写一首关于编程的短诗。"),
    }

    // 使用 Stream 方法获取流式响应
    stream, err := chatModel.Stream(ctx, messages)
    if err != nil {
        fmt.Printf("创建流失败: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("AI 回复（流式输出）:")
    for {
        chunk, err := stream.Recv()
        if err != nil {
            break // 流结束
        }
        fmt.Print(chunk.Content)
    }
    fmt.Println()
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.18
go version

# 设置 OpenAI API Key
export OPENAI_API_KEY="your-api-key-here"
```

### 2. 初始化项目

```bash
# 进入章节目录
cd chapter01-chatmodel-message

# 初始化 Go 模块
go mod init chapter01

# 安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

### 3. 创建 main.go

将上面的示例代码复制到 `main.go` 文件中。

### 4. 运行程序

```bash
# 运行示例 1
go run main.go

# 运行示例 2（交互式对话）
go run main.go

# 运行示例 3（流式输出）
go run main.go
```

## 常见问题

### Q1: 为什么报错 "OPENAI_API_KEY is not set"？

**原因**：没有设置 OpenAI API Key 环境变量。

**解决**：
```bash
# Linux/Mac
export OPENAI_API_KEY="sk-xxx"

# Windows PowerShell
$env:OPENAI_API_KEY="sk-xxx"

# Windows CMD
set OPENAI_API_KEY=sk-xxx
```

### Q2: 如何使用其他 LLM 提供商？

Eino 支持多种 LLM，通过 `eino-ext` 扩展：

```go
// 使用 Claude
import "github.com/cloudwego/eino-ext/components/model/claude"

chatModel, err := claude.NewChatModel(ctx, &claude.ChatModelConfig{
    Model:  "claude-3-opus",
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
})

// 使用 Ollama（本地模型）
import "github.com/cloudwego/eino-ext/components/model/ollama"

chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
    Model: "llama2",
    BaseURL: "http://localhost:11434",
})
```

### Q3: 如何处理长文本？

Eino 会自动处理 token 限制，但建议：
- 控制单次输入长度
- 使用滑动窗口管理对话历史
- 对长文本进行分块处理

## 练习题

### 练习 1：基础对话

创建一个简单的命令行对话程序，要求：
1. 支持多轮对话
2. 记录对话历史
3. 输入 "clear" 清空历史
4. 输入 "quit" 退出程序

### 练习 2：角色扮演

创建一个角色扮演程序，要求：
1. 设定特定角色（如：老师、医生、程序员）
2. 根据角色调整系统提示
3. 支持切换角色

### 练习 3：流式输出优化

改进流式输出示例，要求：
1. 添加打字机效果（逐字显示）
2. 支持中断输出（按 Ctrl+C）
3. 统计输出速度（tokens/秒）

## 下一步学习

完成本章后，建议继续学习：

- **第 2 章**：ChatModelAgent 和 Runner —— 构建更智能的对话系统
- **第 3 章**：Memory 和 Session —— 实现会话持久化

## 参考资料

- [Eino 官方文档 - ChatModel](https://www.cloudwego.io/docs/eino/components/chatmodel/)
- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [Go Context 使用指南](https://go.dev/blog/context)

---

**下一章**：[第 2 章：ChatModelAgent 和 Runner](../chapter02-chatmodel-agent-runner/README.md)
