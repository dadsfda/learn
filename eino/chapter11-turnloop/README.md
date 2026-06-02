# 第 11 章：TurnLoop —— 抢占、中止与多轮生命周期

## 学习目标

完成本章学习后，你将能够：

1. **理解 TurnLoop 的核心概念**：掌握轮次循环（Turn Loop）在 Agent 多轮对话中的作用
2. **实现抢占式调度**：学会在多个任务之间进行优先级调度和资源抢占
3. **掌握优雅取消机制**：使用 Go 的 `context` 实现任务的超时控制和主动取消
4. **管理多轮生命周期**：正确管理 Agent 对话的创建、执行、暂停、恢复和销毁
5. **处理并发场景**：在多轮对话中安全地处理并发请求

---

## 核心概念

### 1. 什么是 TurnLoop？

**TurnLoop**（轮次循环）是 Eino 框架中管理 Agent 对话生命周期的核心机制。想象你在和一个 AI 助手对话：

```
用户: 你好，帮我查一下天气        ← 第 1 轮（Turn 1）
AI: 好的，正在查询...
用户: 等等，先帮我订个餐          ← 第 2 轮（Turn 2），打断了第 1 轮
AI: 好的，先处理订餐...
用户: 订完了，继续查天气吧        ← 回到第 1 轮
AI: 今天北京晴天，25度
```

在这个过程中：
- **轮次（Turn）**：一次完整的用户输入到 AI 响应的过程
- **轮次循环（TurnLoop）**：管理多个轮次的调度和执行
- **抢占（Preemption）**：高优先级任务打断低优先级任务
- **中止（Abort）**：取消正在执行的任务

### 2. 为什么需要 TurnLoop？

在简单的单轮对话中，我们不需要 TurnLoop。但在以下场景中，TurnLoop 是必需的：

| 场景 | 问题 | TurnLoop 的解决方案 |
|------|------|---------------------|
| 用户快速发送多条消息 | 消息堆积，响应混乱 | 队列管理 + 优先级调度 |
| 长时间运行的任务 | 用户不想等了 | 支持取消和超时 |
| 需要打断当前任务 | 新任务更重要 | 抢占式调度 |
| 对话状态管理 | 多轮对话需要保持状态 | 生命周期管理 |

### 3. Go 语言的 Context 机制

TurnLoop 的核心依赖 Go 的 `context` 包。如果你还不熟悉，先来快速了解：

```go
// context 就像一个"遥控器"，可以控制任务的生命周期
ctx, cancel := context.WithCancel(context.Background())
// ctx: 传递给执行任务的函数，告诉它"你可能被打断"
// cancel: 调用它来"按下停止按钮"

// 在另一个地方调用 cancel()，任务就会收到信号并停止
cancel()
```

**关键点**：
- `context.Background()`：创建一个"根"上下文，永远不会被取消
- `context.WithCancel()`：创建一个可以被手动取消的上下文
- `context.WithTimeout()`：创建一个会自动超时的上下文
- `context.WithDeadline()`：创建一个在指定时间点取消的上下文

### 4. 抢占式调度 vs 协作式调度

**协作式调度**（Cooperative）：
```
任务 A: "我要运行 10 分钟，中途不检查取消信号"
任务 B: "我优先级更高，但我得等 A 运行完"
结果：任务 B 饿死了
```

**抢占式调度**（Preemptive）：
```
任务 A: "我运行一会儿就检查一下是否被取消"
任务 B: "我优先级更高，我要抢占资源"
结果：任务 A 被中断，任务 B 立即执行
```

Eino 的 TurnLoop 使用**协作式抢占**：任务需要主动检查 `ctx.Done()` 信号。

---

## 代码示例

### 示例 1：基础轮次循环

```go
// Turn 表示一轮对话
type Turn struct {
    ID        int               // 轮次 ID
    Input     string            // 用户输入
    Output    string            // AI 输出
    Status    string            // 状态：pending/running/completed/cancelled
    CreatedAt time.Time         // 创建时间
    Context   context.Context   // 该轮次的上下文
    Cancel    context.CancelFunc // 取消函数
}

// TurnLoop 管理多轮对话
type TurnLoop struct {
    turns    []*Turn           // 所有轮次
    current  *Turn             // 当前正在执行的轮次
    mu       sync.Mutex        // 保护并发访问
    wg       sync.WaitGroup    // 等待所有任务完成
}

// NewTurnLoop 创建一个新的轮次循环
func NewTurnLoop() *TurnLoop {
    return &TurnLoop{
        turns: make([]*Turn, 0),
    }
}
```

### 示例 2：超时控制

```go
// 使用 WithTimeout 控制任务执行时间
func (tl *TurnLoop) ExecuteWithTimeout(turn *Turn, timeout time.Duration) error {
    // 创建带超时的上下文
    ctx, cancel := context.WithTimeout(turn.Context, timeout)
    defer cancel() // 确保资源被释放

    turn.Context = ctx
    turn.Status = "running"

    // 执行任务
    select {
    case result := <-tl.runTask(ctx, turn):
        turn.Output = result
        turn.Status = "completed"
        return nil
    case <-ctx.Done():
        turn.Status = "cancelled"
        return ctx.Err() // 返回 context.DeadlineExceeded
    }
}
```

### 示例 3：取消机制

```go
// 主动取消正在执行的轮次
func (tl *TurnLoop) CancelTurn(turnID int) error {
    tl.mu.Lock()
    defer tl.mu.Unlock()

    for _, turn := range tl.turns {
        if turn.ID == turnID {
            if turn.Cancel != nil {
                turn.Cancel() // 发送取消信号
                turn.Status = "cancelled"
                return nil
            }
            return fmt.Errorf("turn %d has no cancel function", turnID)
        }
    }
    return fmt.Errorf("turn %d not found", turnID)
}
```

### 示例 4：抢占式调度

```go
// PreemptTurn 抢占当前任务，执行新任务
func (tl *TurnLoop) PreemptTurn(newTurn *Turn) error {
    tl.mu.Lock()
    defer tl.mu.Unlock()

    // 如果有正在运行的任务，先取消它
    if tl.current != nil && tl.current.Status == "running" {
        tl.current.Cancel()
        tl.current.Status = "preempted"
        log.Printf("Turn %d preempted by Turn %d", tl.current.ID, newTurn.ID)
    }

    // 设置新任务为当前任务
    newTurn.Status = "running"
    tl.current = newTurn
    tl.turns = append(tl.turns, newTurn)

    return nil
}
```

### 示例 5：多轮状态管理

```go
// TurnState 管理单个轮次的状态
type TurnState struct {
    History   []string          // 历史消息
    Variables map[string]string // 临时变量
    Checkpoint []byte           // 检查点数据
}

// SaveCheckpoint 保存当前状态（用于恢复）
func (ts *TurnState) SaveCheckpoint() error {
    data, err := json.Marshal(ts)
    if err != nil {
        return err
    }
    ts.Checkpoint = data
    return nil
}

// RestoreCheckpoint 恢复之前的状态
func (ts *TurnState) RestoreCheckpoint() error {
    if ts.Checkpoint == nil {
        return fmt.Errorf("no checkpoint available")
    }
    return json.Unmarshal(ts.Checkpoint, ts)
}
```

---

## 运行步骤

### 前置条件

1. **安装 Go**：确保已安装 Go 1.21 或更高版本
   ```bash
   go version
   # 输出：go version go1.21.0 windows/amd64
   ```

2. **创建项目目录**（如果还没有）：
   ```bash
   mkdir -p E:\aiproject\learn\chapter11-turnloop
   cd E:\aiproject\learn\chapter11-turnloop
   ```

### 步骤 1：初始化 Go 模块

```bash
cd E:\aiproject\learn\chapter11-turnloop
go mod init chapter11-turnloop
```

### 步骤 2：运行示例代码

```bash
go run main.go
```

### 步骤 3：查看输出

程序会依次演示：
1. 基本轮次循环的创建和执行
2. 超时控制（任务会在超时后被取消）
3. 主动取消（手动取消正在执行的任务）
4. 抢占式调度（高优先级任务打断低优先级任务）
5. 优雅关闭（等待所有任务完成或超时后关闭）

---

## 常见问题

### Q1: 为什么我的任务没有响应取消信号？

**问题**：调用了 `cancel()`，但任务还在继续运行。

**原因**：Go 的 context 取消是**协作式**的，任务必须主动检查 `ctx.Done()` 通道。

**解决方案**：
```go
// 错误：不检查取消信号
func badTask(ctx context.Context) {
    for i := 0; i < 1000000; i++ {
        // 这里从不检查 ctx.Done()，即使调用了 cancel() 也不会停止
        heavyComputation()
    }
}

// 正确：定期检查取消信号
func goodTask(ctx context.Context) error {
    for i := 0; i < 1000000; i++ {
        // 每次循环都检查
        select {
        case <-ctx.Done():
            return ctx.Err() // 收到取消信号，立即返回
        default:
            // 继续执行
        }
        heavyComputation()
    }
    return nil
}
```

### Q2: 如何在取消时执行清理操作？

**问题**：任务被取消时，需要释放资源（如关闭文件、断开连接）。

**解决方案**：
```go
func taskWithCleanup(ctx context.Context) error {
    // 获取资源
    conn := getConnection()
    defer conn.Close() // 使用 defer 确保资源被释放

    // 执行任务
    for {
        select {
        case <-ctx.Done():
            // 取消时的清理
            log.Println("Task cancelled, cleaning up...")
            conn.Rollback() // 回滚未完成的操作
            return ctx.Err()
        default:
            // 继续执行
        }
        // ... 业务逻辑
    }
}
```

### Q3: WithTimeout 和 WithDeadline 的区别？

**解答**：

| 特性 | WithTimeout | WithDeadline |
|------|-------------|--------------|
| 参数 | 相对时间（持续时间） | 绝对时间点 |
| 用途 | "这个任务最多运行 5 秒" | "这个任务在 15:30:00 取消" |
| 示例 | `context.WithTimeout(ctx, 5*time.Second)` | `context.WithDeadline(ctx, time.Now().Add(5*time.Second))` |

实际上，`WithTimeout` 内部就是调用 `WithDeadline`：
```go
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
    return WithDeadline(parent, time.Now().Add(timeout))
}
```

### Q4: 如何处理多个并发的轮次？

**问题**：多个轮次同时运行，如何管理？

**解决方案**：使用 `sync.WaitGroup` 等待所有任务完成：
```go
func (tl *TurnLoop) RunAll(turns []*Turn) {
    var wg sync.WaitGroup

    for _, turn := range turns {
        wg.Add(1)
        go func(t *Turn) {
            defer wg.Done()
            tl.ExecuteTurn(t)
        }(turn)
    }

    wg.Wait() // 等待所有任务完成
}
```

### Q5: 抢占会导致数据丢失吗？

**解答**：如果任务被抢占时没有保存状态，可能会丢失中间结果。

**最佳实践**：
1. 定期保存检查点
2. 使用事务确保数据一致性
3. 在取消时执行清理操作

```go
func resilientTask(ctx context.Context, state *TurnState) error {
    for i := 0; i < steps; i++ {
        select {
        case <-ctx.Done():
            // 保存当前状态，以便恢复
            state.SaveCheckpoint()
            return ctx.Err()
        default:
            // 执行一步
            doStep(i)
            // 每 10 步保存一次检查点
            if i%10 == 0 {
                state.SaveCheckpoint()
            }
        }
    }
    return nil
}
```

---

## 练习题

### 练习 1：实现基本的 TurnLoop（简单）

创建一个 `TurnLoop` 结构体，实现以下功能：
1. 添加新的轮次
2. 执行单个轮次
3. 取消单个轮次
4. 获取轮次状态

**提示**：
- 使用 `sync.Mutex` 保护共享数据
- 使用 `context.WithCancel` 创建可取消的上下文

### 练习 2：实现超时控制（中等）

扩展练习 1 的代码，添加超时控制功能：
1. 为每个轮次设置默认超时时间
2. 允许自定义单个轮次的超时时间
3. 超时后自动取消并记录日志

**提示**：
- 使用 `context.WithTimeout`
- 在 goroutine 中监听 `ctx.Done()`

### 练习 3：实现抢占式调度（困难）

实现一个支持抢占的调度器：
1. 每个轮次有一个优先级（1-10，10 最高）
2. 当高优先级任务到来时，抢占低优先级任务
3. 被抢占的任务可以被恢复

**提示**：
- 使用优先级队列（可以用 `container/heap`）
- 保存被抢占任务的状态
- 实现任务恢复机制

### 练习 4：实现优雅关闭（中等）

实现一个优雅关闭机制：
1. 收到关闭信号时，停止接受新任务
2. 等待正在执行的任务完成
3. 如果等待超时，强制取消剩余任务
4. 保存所有任务的最终状态

**提示**：
- 使用 `os.Signal` 监听系统信号（如 SIGINT、SIGTERM）
- 使用 `context.WithTimeout` 设置等待超时
- 使用 `sync.WaitGroup` 等待任务完成

### 练习 5：实现完整的 Agent 对话系统（综合）

结合前面所有知识，实现一个完整的 Agent 对话系统：
1. 支持多轮对话
2. 支持任务抢占和恢复
3. 支持超时控制和取消
4. 支持优雅关闭
5. 记录对话历史和状态

---

## 参考资料

### 官方文档

1. **Go Context 包**
   - 官方文档：https://pkg.go.dev/context
   - 博客文章：https://go.dev/blog/context

2. **Go 并发模式**
   - 官方博客：https://go.dev/blog/pipelines
   - 高级并发：https://go.dev/blog/advanced-go-concurrency

3. **Eino 框架**
   - GitHub 仓库：https://github.com/cloudwego/eino
   - 官方文档：https://www.cloudwego.io/docs/eino/

### 推荐阅读

1. **《Go 并发编程实战》**
   - 深入讲解 Go 的并发模型
   - 包含大量实际案例

2. **《Go 语言高级编程》**
   - 涵盖 context、goroutine、channel 等高级主题
   - 适合有一定基础的开发者

3. **Go 官方博客：Context 取消模式**
   - https://go.dev/blog/context
   - 详细讲解 context 的使用场景和最佳实践

### 相关包

1. **sync 包**
   - `sync.Mutex`：互斥锁
   - `sync.WaitGroup`：等待组
   - `sync.Once`：单次执行

2. **context 包**
   - `context.Background()`：根上下文
   - `context.WithCancel()`：可取消的上下文
   - `context.WithTimeout()`：带超时的上下文
   - `context.WithDeadline()`：带截止时间的上下文

3. **time 包**
   - `time.After()`：定时器
   - `time.NewTimer()`：自定义定时器
   - `time.NewTicker()`：周期性定时器

### 视频教程

1. **Go Context 详解** - GopherCon 演讲
2. **Go 并发模式** - Rob Pike 经典演讲
3. **构建生产级 Go 应用** - 包含 context 最佳实践

---

## 下一步学习

完成本章后，建议继续学习：

1. **第 12 章：Tool Integration** - 工具集成与函数调用
2. **第 13 章：Memory Management** - 对话记忆与状态持久化
3. **第 14 章：Error Handling** - 错误处理与重试机制

---

## 总结

本章学习了 Eino 框架中 TurnLoop 的核心概念：

1. **TurnLoop** 是管理 Agent 多轮对话生命周期的核心机制
2. **Context** 是实现取消和超时控制的基础
3. **抢占式调度**允许高优先级任务打断低优先级任务
4. **优雅关闭**确保资源被正确释放

关键要点：
- 始终使用 `context` 传递取消信号
- 任务必须主动检查 `ctx.Done()` 信号
- 使用 `defer` 确保资源被释放
- 使用 `sync.WaitGroup` 等待并发任务完成

掌握了这些概念，你就能构建出健壮的、支持多轮对话的 AI Agent 系统了！
