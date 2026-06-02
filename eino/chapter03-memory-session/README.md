# 第 3 章：Memory 和 Session —— 持久化会话

## 学习目标

通过本章学习，你将掌握：

1. **Memory 概念**：理解为什么需要会话记忆以及它的工作原理
2. **Session 管理**：学会创建、获取、列出和删除会话
3. **持久化存储**：实现基于 JSONL 文件的会话持久化
4. **会话恢复**：实现跨进程的会话状态恢复
5. **并发安全**：理解如何在并发环境下安全地管理会话

## 前置知识

- 第 1 章：ChatModel 和 Message
- 第 2 章：ChatModelAgent 和 Runner
- Go 基础：接口、结构体、并发（sync.Mutex）
- JSON 处理

## 核心概念

### 1. 为什么需要 Memory？

在前面的章节中，我们使用 `[]*schema.Message` 来维护对话历史。这种方式有一个严重的问题：

```
程序启动 → 对话 → 程序退出 → 所有历史丢失！
```

**Memory（记忆）** 解决了这个问题，它提供：

- **持久化存储**：对话历史保存到文件/数据库，程序退出后不丢失
- **会话管理**：支持多个独立的对话会话
- **跨进程恢复**：可以在不同的进程/设备中恢复对话
- **历史查询**：可以搜索、列出、导出对话历史

### 2. Session（会话）

**Session** 代表一次完整的对话。它是业务层的概念，不是 Eino 框架的核心组件。

```go
type Session struct {
    ID        string             // 会话唯一标识
    CreatedAt time.Time          // 创建时间
    Title     string             // 会话标题（从第一条用户消息生成）
    messages  []*schema.Message  // 对话消息列表
    mu        sync.Mutex         // 并发安全锁
    filePath  string             // 持久化文件路径
}
```

**核心方法**：

| 方法 | 说明 |
|------|------|
| `Append(msg)` | 添加一条消息到会话，并持久化到文件 |
| `GetMessages()` | 获取所有消息的副本 |
| `Title()` | 获取会话标题 |

**设计原则**：
- Session 只负责存储消息，不负责调用 LLM
- LLM 调用由 Agent/Runner 负责
- 这种解耦使得存储策略可以灵活替换

### 3. Store（存储管理器）

**Store** 负责管理多个 Session，提供 CRUD 操作：

```go
type Store struct {
    dir   string              // 存储目录
    cache map[string]*Session // 内存缓存
    mu    sync.Mutex          // 并发安全锁
}
```

**核心方法**：

| 方法 | 说明 |
|------|------|
| `GetOrCreate(id)` | 获取或创建会话 |
| `List()` | 列出所有会话的元数据 |
| `Delete(id)` | 删除会话 |

### 4. JSONL 文件格式

我们使用 JSONL（JSON Lines）格式存储会话数据：

```
{"type":"session","id":"abc-123","created_at":"2026-05-28T10:00:00Z"}  ← 会话头（第 1 行）
{"role":"user","content":"你好"}                                        ← 用户消息（第 2 行）
{"role":"assistant","content":"你好！有什么可以帮助你的吗？"}             ← 助手消息（第 3 行）
{"role":"user","content":"什么是 Go 语言？"}                             ← 用户消息（第 4 行）
```

**JSONL 的优势**：

| 优势 | 说明 |
|------|------|
| 简单 | 每行一个 JSON 对象，易于理解和处理 |
| 可追加 | 新消息直接追加到文件末尾，无需重写整个文件 |
| 可读 | 人类可以直接阅读文件内容 |
| 容错 | 单行损坏不影响其他消息 |
| 流式处理 | 可以逐行读取，适合大文件 |

### 5. 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      业务层（你的代码）                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐                 │
│  │ Session │    │ Session │    │ Session │  ← 会话实例       │
│  └────┬────┘    └────┬────┘    └────┬────┘                 │
│       │              │              │                       │
│       └──────────────┼──────────────┘                       │
│                      │                                      │
│               ┌──────┴──────┐                               │
│               │    Store    │  ← 存储管理器                  │
│               └──────┬──────┘                               │
│                      │                                      │
│               ┌──────┴──────┐                               │
│               │   JSONL     │  ← 文件存储                    │
│               │   文件      │                               │
│               └─────────────┘                               │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                      框架层（Eino）                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                 adk.Runner                           │  │
│  │  接收消息列表 → 调用 ChatModel → 返回回复              │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              ChatModel (OpenAI/Claude/...)           │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**数据流**：

```
用户输入 → session.Append(用户消息) → session.GetMessages() → runner.Run(历史)
    → LLM 生成回复 → session.Append(助手消息) → 显示回复
```

## 代码示例

### 示例 1：内存会话管理

最简单的会话管理方式，使用内存中的消息列表：

```go
// 创建会话
sessionID := uuid.New().String()
session := NewSession(sessionID)

// 添加用户消息
userMsg := &schema.Message{
    Role:    schema.User,
    Content: "你好，我叫小明",
}
session.Append(userMsg)

// 添加助手回复
assistantMsg := &schema.Message{
    Role:    schema.Assistant,
    Content: "你好小明！很高兴认识你。",
}
session.Append(assistantMsg)

// 获取对话历史
history := session.GetMessages()
fmt.Printf("对话历史包含 %d 条消息\n", len(history))
```

### 示例 2：持久化存储

使用 JSONL 文件实现会话持久化：

```go
// 创建存储管理器
store, err := NewStore("./data/sessions")
if err != nil {
    log.Fatal(err)
}

// 创建会话
sessionID := uuid.New().String()
session, err := store.GetOrCreate(sessionID)
if err != nil {
    log.Fatal(err)
}

// 添加消息（自动持久化到文件）
session.Append(&schema.Message{
    Role:    schema.User,
    Content: "什么是 Go 语言？",
})

// 列出所有会话
sessions, err := store.List()
for _, meta := range sessions {
    fmt.Printf("会话: %s - %s\n", meta.ID, meta.Title)
}

// 恢复会话
restoredSession, err := store.GetOrCreate(sessionID)
history := restoredSession.GetMessages() // 包含之前保存的消息
```

### 示例 3：交互式对话

完整的交互式对话系统：

```go
// 创建存储管理器
store, err := NewStore("./data/sessions")

// 获取或创建会话
sessionID := "my-session-123"
session, err := store.GetOrCreate(sessionID)

// 对话循环
scanner := bufio.NewScanner(os.Stdin)
for {
    fmt.Print("你: ")
    scanner.Scan()
    userInput := scanner.Text()

    // 保存用户消息
    session.Append(&schema.Message{
        Role:    schema.User,
        Content: userInput,
    })

    // 获取历史并调用 LLM
    history := session.GetMessages()
    resp, err := chatModel.Generate(ctx, history)

    // 保存助手回复
    session.Append(&schema.Message{
        Role:    schema.Assistant,
        Content: resp.Content,
    })

    fmt.Printf("AI: %s\n", resp.Content)
}
```

### 示例 4：与 Agent 集成

将 Session 与 Eino 的 Agent/Runner 集成：

```go
// 创建 Agent
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Model: chatModel,
})

// 创建 Runner
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent: agent,
})

// 对话循环
for {
    // 获取用户输入
    userInput := getUserInput()

    // 保存用户消息
    session.Append(&schema.Message{
        Role:    schema.User,
        Content: userInput,
    })

    // 获取历史并调用 Agent
    history := session.GetMessages()
    iter := runner.Run(ctx, history)

    // 处理事件流
    var assistantContent string
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }
        switch event.Type {
        case adk.EventMessage:
            fmt.Print(event.Message.Content)
            assistantContent += event.Message.Content
        }
    }

    // 保存助手回复
    session.Append(&schema.Message{
        Role:    schema.Assistant,
        Content: assistantContent,
    })
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.18
go version

# 设置 OpenAI API Key（交互式示例需要）
export OPENAI_API_KEY="your-api-key-here"

# 可选：设置会话存储目录
export SESSION_DIR="./data/sessions"
```

### 2. 初始化项目

```bash
# 进入章节目录
cd chapter03-memory-session

# 初始化 Go 模块
go mod init chapter03

# 安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino/schema
go get github.com/google/uuid
```

### 3. 运行示例

```bash
# 运行内存会话示例
go run main.go memory

# 运行持久化会话示例
go run main.go persist

# 运行交互式对话（创建新会话）
go run main.go interactive

# 运行交互式对话（恢复已有会话）
go run main.go interactive --session <session-id>
```

### 4. 查看会话文件

```bash
# 查看会话存储目录
ls -la data/sessions/

# 查看会话文件内容
cat data/sessions/<session-id>.jsonl
```

## 常见问题

### Q1: Memory 和直接使用 `[]*schema.Message` 有什么区别？

**直接使用 `[]*schema.Message`**：
- 简单，适合快速原型
- 程序退出后历史丢失
- 无法跨进程共享
- 无法管理多个会话

**使用 Memory/Session**：
- 持久化存储，程序退出后不丢失
- 支持跨进程/跨设备恢复
- 可以管理多个独立会话
- 支持会话列表、搜索、导出

### Q2: JSONL 和 JSON 有什么区别？

**JSON**：
```json
{
  "messages": [
    {"role": "user", "content": "你好"},
    {"role": "assistant", "content": "你好！"}
  ]
}
```

**JSONL**：
```
{"role":"user","content":"你好"}
{"role":"assistant","content":"你好！"}
```

**JSONL 的优势**：
- 可以逐行追加，无需重写整个文件
- 单行损坏不影响其他数据
- 流式处理更高效

### Q3: 如何处理并发访问？

使用 `sync.Mutex` 保护共享状态：

```go
type Session struct {
    mu       sync.Mutex
    messages []*schema.Message
}

func (s *Session) Append(msg *schema.Message) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.messages = append(s.messages, msg)
    // ... 持久化到文件
    return nil
}

func (s *Session) GetMessages() []*schema.Message {
    s.mu.Lock()
    defer s.mu.Unlock()

    result := make([]*schema.Message, len(s.messages))
    copy(result, s.messages)
    return result
}
```

### Q4: 如何在生产环境中使用？

JSONL 文件存储适合单机部署。对于生产环境，建议：

| 场景 | 存储方案 |
|------|----------|
| 单机部署 | JSONL 文件 |
| 多机部署 | Redis |
| 大规模部署 | MySQL/PostgreSQL |
| 海量数据 | MongoDB |

示例：使用 Redis 存储

```go
type RedisStore struct {
    client *redis.Client
}

func (s *RedisStore) GetOrCreate(id string) (*Session, error) {
    // 从 Redis 获取会话数据
    data, err := s.client.Get(ctx, "session:"+id).Result()
    if err == redis.Nil {
        // 创建新会话
        return NewSession(id), nil
    }
    // 反序列化
    var session Session
    json.Unmarshal([]byte(data), &session)
    return &session, nil
}
```

### Q5: 如何清理过期会话？

可以添加会话过期机制：

```go
// 在 Store 中添加清理方法
func (s *Store) CleanExpired(maxAge time.Duration) error {
    sessions, err := s.List()
    if err != nil {
        return err
    }

    now := time.Now()
    for _, meta := range sessions {
        if now.Sub(meta.CreatedAt) > maxAge {
            s.Delete(meta.ID)
        }
    }

    return nil
}

// 定期清理（例如每天执行一次）
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    for range ticker.C {
        store.CleanExpired(30 * 24 * time.Hour) // 清理 30 天前的会话
    }
}()
```

## 练习题

### 练习 1：实现会话搜索

实现一个搜索功能，可以在所有会话中搜索包含特定关键词的消息：

```go
// 搜索包含 "Go 语言" 的消息
results := store.Search("Go 语言")

// 返回格式
type SearchResult struct {
    SessionID string
    Message   *schema.Message
}
```

### 练习 2：实现会话导出

将会话导出为 Markdown 格式：

```markdown
# 会话标题

创建时间: 2026-05-28 10:00:00

---

**用户**: 什么是 Go 语言？

**助手**: Go 是 Google 开发的编程语言...

---

**用户**: Go 有什么优势？

**助手**: Go 的主要优势包括...
```

### 练习 3：实现会话分享

实现会话分享功能，生成一个只读链接：

```go
// 生成分享链接
shareID := session.Share()
// 返回: https://example.com/share/<shareID>

// 通过分享链接查看会话
sharedSession := store.GetShared(shareID)
```

### 练习 4：实现消息摘要

当对话历史过长时，自动生成摘要以节省 token：

```go
// 当消息超过 50 条时，生成摘要
if len(session.GetMessages()) > 50 {
    summary := generateSummary(session.GetMessages())
    session.Clear()
    session.Append(&schema.Message{
        Role:    schema.System,
        Content: "之前的对话摘要: " + summary,
    })
}
```

### 练习 5：实现多用户会话

扩展会话模型，支持多用户：

```go
type MultiUserSession struct {
    ID       string
    Users    []User
    Messages []Message
}

type User struct {
    ID   string
    Name string
    Role string // "admin", "member", "viewer"
}
```

## 下一步学习

完成本章后，建议继续学习：

- **第 4 章**：Tools 和文件系统访问 —— 扩展 Agent 的工具能力
- **第 5 章**：Middleware —— 实现跨切面关注点（日志、监控、限流）
- **第 6 章**：Callback 和 Trace —— 实现可观测性

## 参考资料

### 官方文档

- [Eino 官方文档](https://www.cloudwego.io/docs/eino/)
- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [Eino 示例代码](https://github.com/cloudwego/eino-examples)

### 相关概念

- [JSONL 格式规范](https://jsonlines.org/)
- [Go sync.Mutex 使用指南](https://pkg.go.dev/sync#Mutex)
- [Go 并发编程](https://go.dev/blog/patterns)

### 扩展阅读

- [会话管理最佳实践](https://www.cloudwego.io/docs/eino/best_practices/)
- [分布式会话存储方案](https://redis.io/docs/manual/patterns/)
- [数据库设计模式](https://www.mongodb.com/docs/manual/data-modeling/)

---

**上一章**：[第 2 章：ChatModelAgent 和 Runner](../chapter02-chatmodel-agent-runner/README.md)

**下一章**：[第 4 章：Tools 和文件系统访问](../chapter04-tools-filesystem/README.md)
