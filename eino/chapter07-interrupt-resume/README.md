# 第 7 章：Interrupt/Resume —— 人在环中（Human-in-the-Loop）

## 学习目标

通过本章学习，你将掌握：

1. **Interrupt 中断机制**：理解 Eino 中 Agent 执行暂停的核心原理
2. **Resume 恢复机制**：学会从断点恢复 Agent 执行
3. **Checkpoint 检查点**：掌握执行状态的持久化和恢复
4. **Human-in-the-Loop 模式**：实现人机协作的 Agent 工作流
5. **地址系统**：理解 Eino 的层级寻址机制，精准定位中断点
6. **审批流程**：实现工具调用前的人工审批
7. **审阅编辑**：实现工具参数的人工审阅和修改

## 前置知识

- Go 语言基础语法（接口、结构体、context）
- Eino ChatModel 基础使用（第 1 章）
- Eino Agent 和 Runner 基础（第 2 章）
- Eino Graph/Chain 编排基础（第 5 章）
- Eino Callback 机制（第 6 章）

## 核心概念

### 1. 为什么需要 Human-in-the-Loop？

想象以下场景：

- **订票 Agent**：AI 要帮用户订机票，但下单前需要用户确认航班信息和价格
- **代码审查 Agent**：AI 写好了代码，但提交前需要人工审查
- **敏感操作 Agent**：AI 要删除文件，但需要管理员批准
- **医疗问答 Agent**：AI 给出了诊断建议，但需要医生审核后才能告知患者

这些场景的共同点是：**AI 不能完全自主决策，需要人类在关键节点介入**。

```
传统 Agent 流程：
  用户请求 → AI 处理 → AI 处理 → 返回结果
  （全自动，无法中途干预）

Human-in-the-Loop 流程：
  用户请求 → AI 处理 → [暂停] → 人类审核 → [恢复] → AI 继续 → 返回结果
                    ↑                    ↑
                触发中断            提供人工输入
```

### 2. Interrupt（中断）

**Interrupt** 是 Eino 框架中暂停 Agent 执行的机制。当 Agent 或工具需要人工输入时，可以触发中断，将控制权交还给调用者。

#### 中断的类型

```go
// 1. 基础中断 —— 不保存状态
compose.Interrupt(ctx, info) error

// 2. 有状态中断 —— 保存状态，恢复时可以取回
compose.StatefulInterrupt(ctx, info, state) error

// 3. 组合中断 —— 多个子 Agent 的中断合并
compose.CompositeInterrupt(ctx, info, state, errs ...error) error
```

**关键区别**：
- `Interrupt`：最简单，只通知"我暂停了"
- `StatefulInterrupt`：暂停时携带状态数据，恢复时可以取回
- `CompositeInterrupt`：用于父 Agent 聚合多个子 Agent 的中断

#### 中断信息结构

当 Agent 触发中断时，会生成一个中断事件，包含以下信息：

```go
type InterruptInfo struct {
    InterruptContexts []InterruptCtx  // 所有中断点的上下文
}

type InterruptCtx struct {
    ID          string        // 唯一标识符
    Address     Address       // 中断点的层级地址
    Info        any           // 中断时携带的信息
    IsRootCause bool          // 是否是根因中断
    Parent      *InterruptCtx // 父级中断上下文
}
```

### 3. Resume（恢复）

**Resume** 是从断点恢复 Agent 执行的机制。恢复时可以携带人工输入的数据，精确路由到中断点。

#### 恢复 API

```go
// 方式 1：不携带数据恢复（仅发送信号）
ctx = compose.Resume(ctx, interruptIDs ...string)

// 方式 2：携带数据恢复（单个目标）
ctx = compose.ResumeWithData(ctx, interruptID, data)

// 方式 3：批量携带数据恢复（多个目标）
ctx = compose.BatchResumeWithData(ctx, resumeData map[string]any)
```

**使用模式**：

```go
// 第一步：运行 Agent，触发中断
result, err := graph.Invoke(ctx, input)

// 第二步：检查是否中断
if interruptInfo, ok := compose.ExtractInterruptInfo(err); ok {
    // 展示给用户，获取人工输入
    userInput := getUserInput(interruptInfo)

    // 第三步：携带数据恢复执行
    resumeCtx := compose.ResumeWithData(ctx, interruptID, userInput)
    result, err = graph.Invoke(resumeCtx, input)
}
```

### 4. Checkpoint（检查点）

**Checkpoint** 是 Agent 执行状态的快照。当 Agent 中断时，框架自动保存检查点；恢复时，框架从检查点加载状态。

```
执行流程：
  Agent 运行 → [中断] → 保存 Checkpoint → [等待人工输入]
                                               ↓
  Agent 恢复 ← 加载 Checkpoint ← [收到人工输入]
```

#### 检查点存储

```go
// 内存存储（适合开发和测试）
type InMemoryStore struct {
    store sync.Map
}

// 接口定义（可以实现 Redis、数据库等分布式存储）
type CheckPointStore interface {
    Save(ctx context.Context, checkPointID string, data []byte) error
    Load(ctx context.Context, checkPointID string) ([]byte, error)
}
```

**注意**：
- 开发环境使用内存存储即可
- 生产环境建议使用 Redis 或数据库实现分布式存储
- 同一个 `CheckPointID` 可以跨实例恢复

### 5. 地址系统（Address System）

Eino 使用层级地址系统唯一标识每个中断点。这在复杂的嵌套 Agent 中尤其重要。

#### 地址结构

```go
type Address struct {
    Segments []AddressSegment
}

type AddressSegment struct {
    Type  AddressSegmentType  // 类型：agent、tool、node 等
    ID    string              // 唯一标识
    SubID string              // 子标识（可选）
}
```

#### 地址示例

```
ADK 层级地址：
  Agent:TicketBooker → Agent:SubAgent → Tool:BookTicket:1

Compose 层级地址：
  Runnable:my_graph → Node:sub_graph → Node:tools_node → Tool:mcp_tool:1
```

**作用**：
1. **状态定位**：精确标识中断发生的位置
2. **精准恢复**：将恢复数据路由到正确的中断点
3. **用户展示**：告诉用户"在哪个环节需要你介入"

### 6. ADK 层 vs Compose 层

Eino 的中断/恢复机制在两个层面都有支持：

| 特性 | ADK 层（Agent 级别） | Compose 层（Graph 级别） |
|------|---------------------|------------------------|
| 中断 API | `adk.Interrupt()` | `compose.Interrupt()` |
| 返回类型 | `*AgentEvent` | `error` |
| 恢复 API | `runner.ResumeWithParams()` | `compose.ResumeWithData()` |
| 适用场景 | Agent 开发 | Graph/Tool 开发 |
| 地址类型 | agent、tool | runnable、node、tool |

### 7. Human-in-the-Loop 常见模式

Eino 官方提供了多种 HITL 模式：

| 模式 | 说明 | 场景 |
|------|------|------|
| **Approval（审批）** | 工具执行前需要人工批准 | 订票、删除文件、发送邮件 |
| **Review & Edit（审阅编辑）** | 人工审阅并修改工具参数 | 代码提交、内容发布 |
| **Feedback Loop（反馈循环）** | 人工提供反馈，Agent 迭代改进 | 内容生成、设计迭代 |
| **Follow-up（追问）** | Agent 请求补充信息 | 信息不完整时的澄清 |
| **Supervisor（监督）** | 上级 Agent 审核下级操作 | 多级审批流程 |

## 代码示例

### 示例 1：基础中断/恢复概念

用纯 Go 模拟中断/恢复的核心流程，不需要 API Key：

```go
package main

import (
    "context"
    "fmt"
)

// 模拟中断信息
type InterruptInfo struct {
    ID      string
    Message string
    State   map[string]any
}

// 模拟中断错误
type InterruptError struct {
    Info *InterruptInfo
}

func (e *InterruptError) Error() string {
    return fmt.Sprintf("interrupted: %s", e.Info.Message)
}

// 模拟 Agent 执行
func runAgent(ctx context.Context, input string, resumeData any) (string, error) {
    // 检查是否有恢复数据
    if resumeData != nil {
        fmt.Printf("[Agent] 收到恢复数据: %v\n", resumeData)
        return fmt.Sprintf("处理完成！输入: %s, 人工确认: %v", input, resumeData), nil
    }

    // 模拟处理后需要人工确认
    fmt.Printf("[Agent] 处理输入: %s\n", input)
    fmt.Println("[Agent] 需要人工确认，触发中断...")

    return "", &InterruptError{
        Info: &InterruptInfo{
            ID:      "confirm-001",
            Message: "请确认是否继续执行",
            State:   map[string]any{"input": input},
        },
    }
}

func main() {
    ctx := context.Background()

    // 第一次运行 —— 触发中断
    fmt.Println("=== 第一次运行 ===")
    _, err := runAgent(ctx, "帮我订一张去北京的机票", nil)
    if err != nil {
        if ie, ok := err.(*InterruptError); ok {
            fmt.Printf("[系统] Agent 暂停: %s (ID: %s)\n", ie.Info.Message, ie.Info.ID)
            fmt.Printf("[系统] 等待人工输入...\n\n")
        }
    }

    // 模拟人工确认后恢复
    fmt.Println("=== 人工确认后恢复 ===")
    result, err := runAgent(ctx, "帮我订一张去北京的机票", "approved")
    if err != nil {
        fmt.Printf("错误: %v\n", err)
        return
    }
    fmt.Printf("[结果] %s\n", result)
}
```

### 示例 2：使用 compose 包的中断/恢复

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cloudwego/eino/compose"
)

// 审批工具 —— 在执行前触发中断
func approvalTool(ctx context.Context, input string) (string, error) {
    // 检查是否是恢复执行
    isTarget, hasData, resumeData := compose.GetResumeContext[string](ctx)
    if isTarget && hasData {
        if resumeData == "approved" {
            return fmt.Sprintf("工具执行成功: %s", input), nil
        }
        return "工具执行被拒绝", nil
    }

    // 触发中断，等待审批
    return "", compose.StatefulInterrupt(ctx, "需要审批", input)
}

func main() {
    ctx := context.Background()

    // 创建 Graph
    g := compose.NewGraph[string, string]()

    // 添加节点
    g.AddLambdaNode("tool", compose.InvokableLambda(approvalTool))
    g.AddEdge(compose.START, "tool")
    g.AddEdge("tool", compose.END)

    // 编译
    r, err := g.Compile(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 第一次运行 —— 触发中断
    result, err := r.Invoke(ctx, "处理重要数据")
    if err != nil {
        // 检查是否是中断
        if info, ok := compose.ExtractInterruptInfo(err); ok {
            fmt.Printf("中断: %v\n", info)
        }
    }

    // 恢复执行
    resumeCtx := compose.ResumeWithData(ctx, "interrupt-id", "approved")
    result, err = r.Invoke(resumeCtx, "处理重要数据")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

### 示例 3：ADK Agent 审批流程

使用 Eino ADK 实现完整的审批流程（需要 API Key）：

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/compose"
)

// 定义工具输入结构
type sendEmailInput struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

// 创建发送邮件工具
func newSendEmailTool() tool.InvokableTool {
    t, err := utils.InferTool(
        "SendEmail",
        "发送邮件到指定地址",
        func(ctx context.Context, input sendEmailInput) (string, error) {
            return fmt.Sprintf("邮件已发送到 %s", input.To), nil
        })
    if err != nil {
        log.Fatal(err)
    }
    return t
}

func main() {
    ctx := context.Background()

    // 创建 Agent，使用 InvokableApprovableTool 包裹工具
    sendEmail := newSendEmailTool()

    a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "EmailAgent",
        Description: "一个可以发送邮件的 Agent",
        Instruction: "你是一个邮件助手。使用 SendEmail 工具发送邮件。",
        // Model 和 ToolsConfig 需要根据实际情况配置
    })
    if err != nil {
        log.Fatal(err)
    }

    // 创建 Runner，配置检查点存储
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           a,
        CheckPointStore: &InMemoryStore{},
    })

    // 运行 —— 可能触发中断
    iter := runner.Query(ctx, "给 john@example.com 发一封问候邮件",
        adk.WithCheckPointID("email-001"))

    var lastEvent *adk.AgentEvent
    for {
        event, ok := iter.Next()
        if !ok {
            break
        }
        if event.Err != nil {
            log.Fatal(event.Err)
        }
        lastEvent = event
    }

    // 检查是否中断
    if lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
        interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
        fmt.Printf("Agent 暂停，等待审批 (ID: %s)\n", interruptCtx.ID)

        // 获取人工输入
        fmt.Print("是否批准发送邮件？(Y/N): ")
        scanner := bufio.NewScanner(os.Stdin)
        scanner.Scan()

        var apResult *tool.ApprovalResult
        if strings.ToUpper(scanner.Text()) == "Y" {
            apResult = &tool.ApprovalResult{Approved: true}
        } else {
            apResult = &tool.ApprovalResult{
                Approved:          false,
                DisapproveReason:  strPtr("用户拒绝"),
            }
        }

        // 恢复执行
        iter, err = runner.ResumeWithParams(ctx, "email-001", &adk.ResumeParams{
            Targets: map[string]any{
                interruptCtx.ID: apResult,
            },
        })
        if err != nil {
            log.Fatal(err)
        }

        // 处理恢复后的事件
        for {
            event, ok := iter.Next()
            if !ok {
                break
            }
            if event.Err != nil {
                log.Fatal(event.Err)
            }
            fmt.Printf("事件: %+v\n", event)
        }
    }
}

func strPtr(s string) *string {
    return &s
}

// InMemoryStore 内存检查点存储
type InMemoryStore struct {
    store sync.Map
}
```

### 示例 4：审阅编辑模式

允许人工修改工具参数后再执行：

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/cloudwego/eino/compose"
)

// ReviewEditInfo 审阅编辑信息
type ReviewEditInfo struct {
    ToolName  string         // 工具名称
    Arguments map[string]any // 工具参数
    Result    *ReviewResult  // 审阅结果
}

// ReviewResult 审阅结果
type ReviewResult struct {
    Approved     bool   // 是否批准
    EditedArgs   string // 修改后的参数（JSON 格式）
    Disapproved  bool   // 是否拒绝
    DisapproveReason string // 拒绝原因
}

// reviewEditTool 带审阅编辑的工具
func reviewEditTool(ctx context.Context, input string) (string, error) {
    // 检查是否有恢复数据
    isTarget, hasData, resumeData := compose.GetResumeContext[*ReviewEditInfo](ctx)
    if isTarget && hasData && resumeData != nil && resumeData.Result != nil {
        result := resumeData.Result
        if result.Disapproved {
            return fmt.Sprintf("操作被拒绝: %s", result.DisapproveReason), nil
        }
        if result.EditedArgs != "" {
            return fmt.Sprintf("使用修改后的参数执行: %s", result.EditedArgs), nil
        }
        return fmt.Sprintf("原始参数执行成功: %s", input), nil
    }

    // 触发有状态中断，携带工具调用信息
    info := &ReviewEditInfo{
        ToolName:  "ProcessData",
        Arguments: map[string]any{"data": input},
    }
    return "", compose.StatefulInterrupt(ctx, "请审阅工具调用参数", info)
}

func main() {
    ctx := context.Background()

    // 创建 Graph
    g := compose.NewGraph[string, string]()
    g.AddLambdaNode("processor", compose.InvokableLambda(reviewEditTool))
    g.AddEdge(compose.START, "processor")
    g.AddEdge("processor", compose.END)

    r, err := g.Compile(ctx)
    if err != nil {
        panic(err)
    }

    // 第一次运行 —— 触发中断
    _, err = r.Invoke(ctx, "重要数据")
    if err != nil {
        if info, ok := compose.ExtractInterruptInfo(err); ok {
            fmt.Println("=== 工具调用需要审阅 ===")
            fmt.Printf("中断信息: %+v\n", info)

            // 模拟人工审阅
            fmt.Println("\n选项:")
            fmt.Println("  1. 直接批准（输入 'approve'）")
            fmt.Println("  2. 修改参数（输入 JSON 格式的新参数）")
            fmt.Println("  3. 拒绝（输入 'reject'）")
        }
    }
}
```

### 示例 5：检查点保存和恢复

演示检查点的持久化机制：

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
)

// CheckPoint 检查点数据
type CheckPoint struct {
    ID        string         `json:"id"`
    State     map[string]any `json:"state"`
    Timestamp int64          `json:"timestamp"`
}

// MemoryCheckPointStore 内存检查点存储
type MemoryCheckPointStore struct {
    mu    sync.RWMutex
    store map[string]*CheckPoint
}

func NewMemoryCheckPointStore() *MemoryCheckPointStore {
    return &MemoryCheckPointStore{
        store: make(map[string]*CheckPoint),
    }
}

// Save 保存检查点
func (s *MemoryCheckPointStore) Save(cp *CheckPoint) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.store[cp.ID] = cp
    fmt.Printf("[Checkpoint] 已保存: %s\n", cp.ID)
    return nil
}

// Load 加载检查点
func (s *MemoryCheckPointStore) Load(id string) (*CheckPoint, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    cp, ok := s.store[id]
    if !ok {
        return nil, fmt.Errorf("checkpoint not found: %s", id)
    }
    fmt.Printf("[Checkpoint] 已加载: %s\n", id)
    return cp, nil
}

// Delete 删除检查点
func (s *MemoryCheckPointStore) Delete(id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.store, id)
    fmt.Printf("[Checkpoint] 已删除: %s\n", id)
    return nil
}

// List 列出所有检查点
func (s *MemoryCheckPointStore) List() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    ids := make([]string, 0, len(s.store))
    for id := range s.store {
        ids = append(ids, id)
    }
    return ids
}

func main() {
    store := NewMemoryCheckPointStore()

    // 模拟 Agent 执行并保存检查点
    fmt.Println("=== 模拟 Agent 执行 ===")
    fmt.Println("[Agent] 开始处理任务...")

    // 保存检查点（模拟中断时的状态）
    cp := &CheckPoint{
        ID: "task-001",
        State: map[string]any{
            "step":        2,
            "total_steps": 5,
            "data":        "部分处理结果",
            "status":      "waiting_approval",
        },
    }
    store.Save(cp)

    // 模拟一段时间后恢复
    fmt.Println("\n=== 模拟恢复执行 ===")
    fmt.Println("[系统] 收到人工批准，恢复执行...")

    // 加载检查点
    loaded, err := store.Load("task-001")
    if err != nil {
        panic(err)
    }

    // 从检查点恢复状态
    fmt.Printf("[Agent] 从步骤 %d/%d 恢复\n",
        loaded.State["step"], loaded.State["total_steps"])
    fmt.Printf("[Agent] 之前的数据: %s\n", loaded.State["data"])

    // 继续执行
    fmt.Println("[Agent] 继续执行剩余步骤...")
    loaded.State["step"] = 5
    loaded.State["status"] = "completed"
    fmt.Printf("[Agent] 任务完成！最终状态: %v\n", loaded.State["status"])

    // 清理检查点
    store.Delete("task-001")
    fmt.Printf("\n当前检查点数量: %d\n", len(store.List()))
}
```

## 完整示例：带中断/恢复的交互式 Agent

将以上知识整合，创建一个完整的交互式 Agent 示例：

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"
    "sync"
)

// ========== 核心类型定义 ==========

// AgentTask Agent 任务
type AgentTask struct {
    ID       string
    Input    string
    Step     int
    Total    int
    Results  []string
    Status   string
}

// InterruptSignal 中断信号
type InterruptSignal struct {
    TaskID  string
    Message string
    Options []string
}

// CheckPointStore 检查点存储
type CheckPointStore struct {
    mu    sync.RWMutex
    tasks map[string]*AgentTask
}

func NewCheckPointStore() *CheckPointStore {
    return &CheckPointStore{
        tasks: make(map[string]*AgentTask),
    }
}

func (s *CheckPointStore) Save(task *AgentTask) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.tasks[task.ID] = task
}

func (s *CheckPointStore) Load(id string) (*AgentTask, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    task, ok := s.tasks[id]
    return task, ok
}

// ========== Agent 实现 ==========

// InteractiveAgent 交互式 Agent
type InteractiveAgent struct {
    store *CheckPointStore
}

func NewInteractiveAgent() *InteractiveAgent {
    return &InteractiveAgent{
        store: NewCheckPointStore(),
    }
}

// Run 执行任务
func (a *InteractiveAgent) Run(ctx context.Context, task *AgentTask, resumeInput string) (*AgentTask, *InterruptSignal, error) {
    // 如果有恢复输入，处理人工反馈
    if resumeInput != "" {
        fmt.Printf("[Agent] 收到人工反馈: %s\n", resumeInput)
        task.Results = append(task.Results, fmt.Sprintf("人工确认: %s", resumeInput))
        task.Step++
    }

    // 模拟多步骤执行
    for task.Step < task.Total {
        fmt.Printf("[Agent] 执行步骤 %d/%d...\n", task.Step+1, task.Total)

        // 模拟步骤结果
        stepResult := fmt.Sprintf("步骤 %d 完成", task.Step+1)
        task.Results = append(task.Results, stepResult)

        // 每隔一步需要人工确认
        if task.Step == 1 {
            task.Step++
            task.Status = "waiting_approval"
            a.store.Save(task) // 保存检查点

            return task, &InterruptSignal{
                TaskID:  task.ID,
                Message: fmt.Sprintf("已完成 %d/%d 步骤，是否继续？", task.Step, task.Total),
                Options: []string{"继续", "暂停", "终止"},
            }, nil
        }

        task.Step++
    }

    task.Status = "completed"
    return task, nil, nil
}

func main() {
    ctx := context.Background()
    agent := NewInteractiveAgent()
    scanner := bufio.NewScanner(os.Stdin)

    fmt.Println("=== 交互式 Agent 演示 ===")
    fmt.Println("这个 Agent 在执行过程中会暂停请求人工确认")
    fmt.Println()

    // 创建任务
    task := &AgentTask{
        ID:      "task-001",
        Input:   "处理用户数据",
        Step:    0,
        Total:   3,
        Results: []string{},
        Status:  "running",
    }

    var resumeInput string

    for {
        // 运行 Agent
        result, signal, err := agent.Run(ctx, task, resumeInput)
        if err != nil {
            fmt.Printf("错误: %v\n", err)
            return
        }

        // 检查是否需要人工输入
        if signal != nil {
            fmt.Printf("\n[系统] %s\n", signal.Message)
            fmt.Println("选项:", strings.Join(signal.Options, ", "))
            fmt.Print("请输入: ")
            scanner.Scan()
            resumeInput = scanner.Text()

            if resumeInput == "终止" {
                fmt.Println("[Agent] 任务已终止")
                return
            }

            task = result
            continue
        }

        // 任务完成
        fmt.Println("\n=== 任务完成 ===")
        fmt.Printf("状态: %s\n", result.Status)
        fmt.Println("执行结果:")
        for i, r := range result.Results {
            fmt.Printf("  %d. %s\n", i+1, r)
        }
        return
    }
}
```

## 运行步骤

### 1. 环境准备

```bash
# 确保 Go 版本 >= 1.21
go version

# 设置 OpenAI API Key（仅 agent 示例需要）
export OPENAI_API_KEY="your-api-key-here"
```

### 2. 初始化项目

```bash
# 进入章节目录
cd chapter07-interrupt-resume

# 初始化 Go 模块
go mod init chapter07

# 安装依赖
go get github.com/cloudwego/eino
```

### 3. 创建 main.go

将本章的完整示例代码复制到 `main.go` 文件中。

### 4. 运行程序

```bash
# 运行基础中断/恢复演示（不需要 API Key）
go run main.go demo

# 运行检查点演示（不需要 API Key）
go run main.go checkpoint

# 运行交互式 Agent 演示（不需要 API Key）
go run main.go interactive

# 运行审批流程演示（不需要 API Key）
go run main.go approval

# 运行 ADK Agent 示例（需要 API Key）
go run main.go agent
```

### 预期输出

```
=== 示例 1：基础中断/恢复概念 ===
模拟中断/恢复的核心流程

[Step 1] Agent 开始处理: "帮我订一张去北京的机票"
[Step 2] 处理中... 需要人工确认
[系统] Agent 暂停: 请确认是否继续执行
[系统] 等待人工输入...

[Step 3] 收到恢复信号: "approved"
[Step 4] 继续执行...
[结果] 任务完成！
```

## 常见问题

### Q1: Interrupt 和普通的错误返回有什么区别？

**Interrupt 不是错误**，而是一种控制流机制。虽然在 Go 中 `compose.Interrupt()` 返回的是 `error` 类型，但这是为了方便在函数中传播。框架会通过 `compose.ExtractInterruptInfo()` 区分中断和真正的错误。

```go
result, err := graph.Invoke(ctx, input)
if err != nil {
    // 区分中断和错误
    if info, ok := compose.ExtractInterruptInfo(err); ok {
        // 这是中断，不是错误
        handleInterrupt(info)
    } else {
        // 这是真正的错误
        log.Fatal(err)
    }
}
```

### Q2: Checkpoint 数据存在哪里？

- **开发环境**：使用内存存储（`sync.Map`），程序重启后丢失
- **生产环境**：建议实现 `CheckPointStore` 接口，使用 Redis 或数据库
- **关键点**：同一个 `CheckPointID` 可以跨实例恢复，适合分布式部署

### Q3: 多个子 Agent 都中断了怎么办？

使用 `CompositeInterrupt` 聚合多个子 Agent 的中断：

```go
// 父 Agent 收集所有子 Agent 的中断
var interruptErrs []error
for _, subAgent := range subAgents {
    err := subAgent.Run(ctx)
    if compose.IsInterrupt(err) {
        interruptErrs = append(interruptErrs, err)
    }
}

// 合并为一个组合中断
if len(interruptErrs) > 0 {
    return compose.CompositeInterrupt(ctx, "子 Agent 需要审批", nil, interruptErrs...)
}
```

### Q4: 如何实现超时自动批准？

在恢复逻辑中添加超时检测：

```go
select {
case userInput := <-inputChan:
    // 用户提供了输入
    resumeWithData(ctx, userInput)
case <-time.After(30 * time.Second):
    // 超时，自动批准
    resumeWithData(ctx, "auto-approved")
case <-ctx.Done():
    // 上下文取消
    return ctx.Err()
}
```

### Q5: 中断时的状态会丢失吗？

不会。使用 `StatefulInterrupt` 时，状态会被保存到 Checkpoint 中：

```go
// 触发中断时保存状态
compose.StatefulInterrupt(ctx, "需要审批", currentState)

// 恢复时取回状态
wasInterrupted, hasState, oldState := compose.GetInterruptState[*MyState](ctx)
if wasInterrupted && hasState {
    // oldState 就是中断时保存的状态
}
```

### Q6: 如何在 Web 服务中使用中断/恢复？

典型的 Web 服务模式：

```go
// 1. POST /run - 启动任务
func handleRun(w http.ResponseWriter, r *http.Request) {
    result, err := graph.Invoke(ctx, input)
    if err != nil {
        if info, ok := compose.ExtractInterruptInfo(err); ok {
            // 返回中断信息给前端
            json.NewEncoder(w).Encode(map[string]any{
                "status":    "interrupted",
                "interrupt": info,
            })
            return
        }
    }
    json.NewEncoder(w).Encode(map[string]any{"result": result})
}

// 2. POST /resume - 恢复任务
func handleResume(w http.ResponseWriter, r *http.Request) {
    var req struct {
        InterruptID string `json:"interrupt_id"`
        Data        any    `json:"data"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    resumeCtx := compose.ResumeWithData(ctx, req.InterruptID, req.Data)
    result, err := graph.Invoke(resumeCtx, originalInput)
    // ...
}
```

## 练习题

### 练习 1：基础中断/恢复

创建一个简单的中断/恢复流程，要求：
1. 定义一个处理函数，在处理到一半时触发中断
2. 模拟人工输入
3. 从中断点恢复执行
4. 输出最终结果

### 练习 2：审批工具包装器

实现一个 `ApprovableTool` 包装器，要求：
1. 包装任意 `InvokableTool`
2. 在工具执行前触发中断
3. 等待人工审批（批准/拒绝）
4. 批准时执行原工具，拒绝时返回拒绝信息

### 练习 3：检查点持久化

实现一个基于文件的检查点存储，要求：
1. 实现 `Save` 和 `Load` 方法
2. 使用 JSON 格式序列化
3. 支持跨进程恢复
4. 添加过期清理机制

### 练习 4：多步骤审批

创建一个需要多步审批的工作流，要求：
1. 定义 3 个步骤，每步都需要人工确认
2. 使用检查点保存当前进度
3. 支持从任意步骤恢复
4. 输出完整的执行日志

### 练习 5：Web API 集成

将中断/恢复机制封装为 HTTP API，要求：
1. POST /task - 创建并启动任务
2. GET /task/:id - 查询任务状态
3. POST /task/:id/resume - 恢复中断的任务
4. 使用内存存储检查点

## 高级话题

### 1. 分布式检查点存储

```go
// RedisCheckPointStore 基于 Redis 的检查点存储
type RedisCheckPointStore struct {
    client *redis.Client
}

func (s *RedisCheckPointStore) Save(ctx context.Context, id string, data []byte) error {
    return s.client.Set(ctx, "checkpoint:"+id, data, 24*time.Hour).Err()
}

func (s *RedisCheckPointStore) Load(ctx context.Context, id string) ([]byte, error) {
    return s.client.Get(ctx, "checkpoint:"+id).Bytes()
}
```

### 2. 异步中断处理

```go
// 在 goroutine 中处理中断
go func() {
    for event := range eventChan {
        if event.Action.Interrupted != nil {
            // 通知前端
            notifyFrontend(event.Action.Interrupted)
        }
    }
}()
```

### 3. 中断事件的序列化

```go
// 将中断信息序列化为 JSON，用于跨服务传递
type InterruptPayload struct {
    InterruptID string         `json:"interrupt_id"`
    Info        any            `json:"info"`
    CheckpointID string        `json:"checkpoint_id"`
}
```

## 下一步学习

完成本章后，建议继续学习：

- **第 8 章**：Graph 和 Tool —— 构建复杂的工具调用链
- **第 9 章**：Skill 中间件 —— 实现可复用的业务逻辑
- **第 10 章**：A2UI 协议 —— Agent 与用户界面的交互协议

## 参考资料

- [Eino 官方文档 - Agent HITL](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_hitl/)
- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [Eino 示例 - Human-in-the-Loop](https://github.com/cloudwego/eino-examples/tree/main/adk/human-in-the-loop)
- [Eino 官方文档 - Callback Manual](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)
- [Go Context 使用指南](https://go.dev/blog/context)
- [Human-in-the-Loop 模式 - LangGraph](https://langchain-ai.github.io/langgraph/concepts/human_in_the_loop/)

---

**上一章**：[第 6 章：Callback 和 Trace —— 可观测性](../chapter06-callback-trace/README.md)

**下一章**：[第 8 章：Graph 和 Tool —— 复杂工具调用链](../chapter08-graph-tool/README.md)
