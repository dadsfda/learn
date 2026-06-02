# 第 8 章：Graph Tool -- 复杂工作流

## 学习目标

通过本章学习，你将掌握：

1. **Graph（图）概念**：理解有向无环图（DAG）在工作流编排中的核心作用
2. **Node（节点）**：学会使用 Lambda 节点、ChatModel 节点、ToolsNode 等不同类型的节点
3. **Edge（边）**：理解节点之间的数据流向和连接方式
4. **Compile（编译）**：掌握图的编译过程，将图定义转化为可执行的 Runnable
5. **条件分支**：使用 GraphBranch 实现根据条件选择不同的执行路径
6. **并行执行**：使用 Parallel 让多个节点同时执行，提升效率
7. **图作为工具**：将复杂的 Graph 封装为单个 Tool，供 LLM 调用
8. **Chain 链式编排**：使用 Chain 简化线性工作流的构建

## 前置知识

- 第 1 章：ChatModel 和 Message
- 第 2 章：ChatModelAgent 和 Runner
- 第 4 章：Tools 和 FileSystem
- Go 泛型基础（`[I, O any]` 语法）
- Go 函数式编程（高阶函数、闭包）
- context.Context 使用

## 核心概念

### 1. 什么是 Graph（图）？

在 Eino 中，Graph 是一种**有向无环图（DAG, Directed Acyclic Graph）**编排系统，用于将多个处理步骤（节点）按特定的依赖关系（边）组合成复杂的工作流。

**生活中的类比**：想象一个餐厅的出餐流程：

```
点单 → 配菜 → 烹饪 → 装盘 → 上菜
              ↓
         调制酱汁 ──→ 装盘
```

- 每个步骤是一个**节点（Node）**
- 步骤之间的顺序关系是**边（Edge）**
- "配菜"和"调制酱汁"可以**并行执行**
- 整个流程**不能有循环**（不能回到已经完成的步骤）

### 2. 为什么需要 Graph？

在第 2 章我们学习了 Chain（链），它是一种简单的线性编排：

```
输入 → 步骤1 → 步骤2 → 步骤3 → 输出
```

但实际的 AI 应用往往需要更复杂的编排：

| 场景 | 需求 | 解决方案 |
|------|------|----------|
| **并行处理** | 同时搜索多个数据源 | Parallel 并行节点 |
| **条件路由** | 根据用户意图选择不同处理路径 | GraphBranch 条件分支 |
| **复杂依赖** | 某些步骤依赖多个前置步骤的结果 | Graph 边连接 |
| **子工作流** | 将一组步骤封装为可复用的单元 | Graph 作为节点嵌套 |

### 3. Graph 的核心组成部分

#### 3.1 Node（节点）

节点是 Graph 中的处理单元。每个节点接收输入，处理数据，产生输出。

Eino 支持多种节点类型：

| 节点类型 | 说明 | 创建方式 |
|----------|------|----------|
| **Lambda 节点** | 自定义函数逻辑 | `compose.InvokableLambda(fn)` |
| **ChatModel 节点** | 调用 LLM | `graph.AddChatModelNode(name, model)` |
| **ToolsNode** | 执行工具调用 | `graph.AddToolsNode(name, toolsNode)` |
| **Passthrough 节点** | 直接透传数据 | `graph.AddPassthroughNode(name)` |
| **子图节点** | 嵌套另一个 Graph | `graph.AddGraphNode(name, subGraph)` |

**Lambda 节点**是最灵活的节点类型，它可以是任何函数：

```go
// 创建一个 Lambda 节点：将输入转为大写
lambda := compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
    return strings.ToUpper(input), nil
})
```

Lambda 有四种变体：

| 类型 | 函数签名 | 说明 |
|------|----------|------|
| `InvokableLambda` | `func(ctx, I) (O, error)` | 同步调用，最常用 |
| `StreamableLambda` | `func(ctx, I) (*StreamReader[O], error)` | 同步输入，流式输出 |
| `CollectableLambda` | `func(ctx, *StreamReader[I]) (O, error)` | 流式输入，同步输出 |
| `TransformableLambda` | `func(ctx, *StreamReader[I]) (*StreamReader[O], error)` | 流式输入，流式输出 |

#### 3.2 Edge（边）

边定义了节点之间的数据流向。在 Eino 中，使用 `AddEdge` 方法连接两个节点：

```go
// 从 "start" 节点到 "process" 节点
graph.AddEdge(compose.START, "process")

// 从 "process" 节点到 "end" 节点
graph.AddEdge("process", compose.END)
```

Eino 定义了两个特殊的节点名：
- `compose.START`：图的入口节点
- `compose.END`：图的出口节点

```
START → node1 → node2 → END
```

#### 3.3 Compile（编译）

图定义完成后，需要**编译**才能执行。编译过程会：
1. 验证图的结构（无循环、类型兼容）
2. 构建执行计划
3. 返回一个可执行的 `Runnable`

```go
// 编译图
runnable, err := graph.Compile(ctx)

// 执行图
result, err := runnable.Invoke(ctx, input)
```

**重要**：编译后的图不能再修改！如果尝试修改，会返回 `ErrGraphCompiled` 错误。

#### 3.4 DAG 执行

DAG 执行遵循以下规则：
- 没有入边的节点（除了 START）不会被执行
- 节点的所有前置节点完成后，该节点才会执行
- 支持并行执行没有依赖关系的节点
- 数据从上游节点流向下游节点

```
        ┌──→ B ──┐
START ──┤        ├──→ D → END
        └──→ C ──┘
```

在这个图中：
- B 和 C 可以并行执行
- D 需要等待 B 和 C 都完成后才能执行

### 4. Graph vs Chain

| 特性 | Chain | Graph |
|------|-------|-------|
| 拓扑结构 | 线性（单链） | DAG（有向无环图） |
| 并行执行 | 不支持 | 支持 |
| 条件分支 | 不支持 | 支持 |
| 数据流 | 单向线性 | 多对多 |
| 复杂度 | 简单 | 灵活 |
| 适用场景 | 简单管道 | 复杂工作流 |
| API 风格 | 链式调用 | 显式添加节点和边 |

**选择建议**：
- 简单的线性处理 → 使用 Chain
- 需要并行或条件分支 → 使用 Graph
- 复杂的多步骤工作流 → 使用 Graph

## 代码示例

### 示例 1：基础 Graph 构建

最简单的 Graph，只包含一个 Lambda 节点：

```go
// 创建图：输入 string → 转大写 → 输出 string
graph := compose.NewGraph[string, string]()

// 添加一个 Lambda 节点
graph.AddLambdaNode("to_upper",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return strings.ToUpper(input), nil
    }),
)

// 连接边：START → to_upper → END
graph.AddEdge(compose.START, "to_upper")
graph.AddEdge("to_upper", compose.END)

// 编译并执行
runnable, _ := graph.Compile(ctx)
result, _ := runnable.Invoke(ctx, "hello world")
// result = "HELLO WORLD"
```

### 示例 2：多节点链式处理

多个 Lambda 节点串联：

```go
graph := compose.NewGraph[string, string]()

// 节点 1：去除空格
graph.AddLambdaNode("trim",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return strings.TrimSpace(input), nil
    }),
)

// 节点 2：转大写
graph.AddLambdaNode("to_upper",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return strings.ToUpper(input), nil
    }),
)

// 节点 3：添加前缀
graph.AddLambdaNode("add_prefix",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return "RESULT: " + input, nil
    }),
)

// 连接边：START → trim → to_upper → add_prefix → END
graph.AddEdge(compose.START, "trim")
graph.AddEdge("trim", "to_upper")
graph.AddEdge("to_upper", "add_prefix")
graph.AddEdge("add_prefix", compose.END)

// 编译并执行
runnable, _ := graph.Compile(ctx)
result, _ := runnable.Invoke(ctx, "  hello world  ")
// result = "RESULT: HELLO WORLD"
```

### 示例 3：条件分支（GraphBranch）

根据条件选择不同的处理路径：

```go
graph := compose.NewGraph[string, string]()

// 分析节点：判断输入是数字还是文本
graph.AddLambdaNode("analyze",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        if _, err := strconv.Atoi(input); err == nil {
            return "number", nil
        }
        return "text", nil
    }),
)

// 数字处理节点
graph.AddLambdaNode("handle_number",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        num, _ := strconv.Atoi(input)
        return fmt.Sprintf("数字的平方是: %d", num*num), nil
    }),
)

// 文本处理节点
graph.AddLambdaNode("handle_text",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return "文本的大写是: " + strings.ToUpper(input), nil
    }),
)

// 连接边
graph.AddEdge(compose.START, "analyze")

// 条件分支：根据 analyze 的输出选择路径
graph.AddBranch("analyze", compose.NewGraphBranch(
    func(ctx context.Context, input string) (string, error) {
        return input, nil // 返回 "number" 或 "text"
    },
    map[string]bool{
        "handle_number": true,
        "handle_text":   true,
    },
))

// 两个分支都汇聚到 END
graph.AddEdge("handle_number", compose.END)
graph.AddEdge("handle_text", compose.END)
```

### 示例 4：并行执行（Parallel）

多个节点同时执行，然后汇聚结果：

```go
// 使用 Chain 的 Parallel 功能
chain := compose.NewChain[string, string]()

// 创建并行组
parallel := compose.NewParallel()

// 并行节点 1：计算长度
parallel.AddLambda("length",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return fmt.Sprintf("长度: %d", len(input)), nil
    }),
)

// 并行节点 2：转大写
parallel.AddLambda("upper",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return "大写: " + strings.ToUpper(input), nil
    }),
)

// 并行节点 3：反转
parallel.AddLambda("reverse",
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        runes := []rune(input)
        for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
            runes[i], runes[j] = runes[j], runes[i]
        }
        return "反转: " + string(runes), nil
    }),
)

// 将并行组添加到 Chain
chain.AppendParallel(parallel)

// 添加汇聚节点，合并并行结果
chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, inputs map[string]string) (string, error) {
    return strings.Join([]string{inputs["length"], inputs["upper"], inputs["reverse"]}, "\n"), nil
}))
```

### 示例 5：Graph 封装为 Tool

将复杂的 Graph 封装为一个 Tool，供 LLM 调用：

```go
// 定义工具的输入结构
type TextInput struct {
    Text string `json:"text" description:"要处理的文本"`
}

// 定义工具的输出结构
type TextOutput struct {
    Result string `json:"result" description:"处理结果"`
}

// 创建一个 Graph
graph := compose.NewGraph[TextInput, TextOutput]()

// 添加处理节点
graph.AddLambdaNode("process",
    compose.InvokableLambda(func(ctx context.Context, input TextInput) (TextOutput, error) {
        result := strings.ToUpper(strings.TrimSpace(input.Text))
        return TextOutput{Result: result}, nil
    }),
)

graph.AddEdge(compose.START, "process")
graph.AddEdge("process", compose.END)

// 编译图
runnable, _ := graph.Compile(ctx)

// 封装为 Tool
tool, _ := utils.InferTool("text_processor", "处理文本：去除空格并转为大写",
    func(ctx context.Context, input TextInput) (TextOutput, error) {
        return runnable.Invoke(ctx, input)
    },
)
```

### 示例 6：Chain 链式编排

Chain 是 Graph 的简化版，适合线性处理：

```go
// 创建 Chain：string → 去空格 → 转大写 → 添加前缀
chain := compose.NewChain[string, string]()

// 添加 Lambda 节点
chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
    return strings.TrimSpace(input), nil
}))

chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
    return strings.ToUpper(input), nil
}))

chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
    return "RESULT: " + input, nil
}))

// 编译并执行
runnable, _ := chain.Compile(ctx)
result, _ := runnable.Invoke(ctx, "  hello  ")
// result = "RESULT: HELLO"
```

### 示例 7：使用 ToolsNode

在 Graph 中使用 ToolsNode 执行工具调用：

```go
// 创建工具
weatherTool, _ := utils.InferTool("get_weather", "获取天气信息",
    func(ctx context.Context, input struct {
        City string `json:"city" description:"城市名称"`
    }) (string, error) {
        return fmt.Sprintf("%s: 晴, 25°C", input.City), nil
    },
)

// 创建 ToolsNode
toolsNode, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{weatherTool},
})

// 创建 Graph
graph := compose.NewGraph[*schema.Message, []*schema.Message]()

// 添加 ToolsNode
graph.AddToolsNode("tools", toolsNode)

// 连接边
graph.AddEdge(compose.START, "tools")
graph.AddEdge("tools", compose.END)

// 编译并执行
runnable, _ := graph.Compile(ctx)

// 构造包含工具调用的消息
msg := &schema.Message{
    Role: schema.Assistant,
    ToolCalls: []schema.ToolCall{
        {
            ID:   "call_1",
            Name: "get_weather",
            Arguments: `{"city": "北京"}`,
        },
    },
}

results, _ := runnable.Invoke(ctx, msg)
// results[0].Content = "北京: 晴, 25°C"
```

## 运行步骤

### 1. 初始化 Go 模块

```bash
cd chapter08-graph-tool
go mod init chapter08-graph-tool
```

### 2. 安装依赖

```bash
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai
```

### 3. 设置环境变量（可选，部分示例需要）

```bash
# Linux/Mac
export OPENAI_API_KEY="your-api-key-here"

# Windows PowerShell
$env:OPENAI_API_KEY = "your-api-key-here"
```

### 4. 运行示例

```bash
# 运行基础 Graph 示例（纯模拟，不需要 API Key）
go run main.go basic

# 运行多节点链式处理
go run main.go chain

# 运行条件分支示例
go run main.go branch

# 运行并行执行示例
go run main.go parallel

# 运行 Graph 封装为 Tool 示例
go run main.go graphtool

# 运行 Chain 链式编排示例
go run main.go simplechain

# 运行完整演示（包含所有示例）
go run main.go demo
```

## 常见问题

### Q1: Graph 和 Chain 有什么区别？

**A**：

| 对比项 | Chain | Graph |
|--------|-------|-------|
| 拓扑结构 | 线性（单链） | DAG（有向无环图） |
| 并行执行 | 不支持 | 支持 |
| 条件分支 | 不支持 | 支持 |
| 数据流 | 单向线性 | 多对多 |
| 适用场景 | 简单管道 | 复杂工作流 |
| API 风格 | 链式调用 | 显式添加节点和边 |

简单来说：
- **Chain** = 简单的流水线，数据从头到尾依次处理
- **Graph** = 复杂的工作流，支持并行、分支、汇聚

### Q2: 什么是 Lambda 节点？

**A**：Lambda 节点是使用 Go 函数定义的自定义节点。它是最灵活的节点类型，可以实现任何逻辑：

```go
// 最简单的 Lambda 节点
lambda := compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
    return strings.ToUpper(input), nil
})
```

Lambda 有四种变体，分别对应不同的输入/输出模式：
- `InvokableLambda`：同步输入 → 同步输出（最常用）
- `StreamableLambda`：同步输入 → 流式输出
- `CollectableLambda`：流式输入 → 同步输出
- `TransformableLambda`：流式输入 → 流式输出

### Q3: 编译（Compile）的作用是什么？

**A**：编译是将图的定义转化为可执行对象的过程。编译时会：

1. **验证图结构**：检查是否有循环、是否有未连接的节点
2. **类型检查**：确保节点之间的数据类型兼容
3. **构建执行计划**：确定节点的执行顺序和并行策略
4. **返回 Runnable**：一个可以调用 `Invoke` 或 `Stream` 的可执行对象

```go
// 编译
runnable, err := graph.Compile(ctx)
if err != nil {
    // 图结构有问题，编译失败
    log.Fatal(err)
}

// 执行
result, err := runnable.Invoke(ctx, input)
```

### Q4: 如何实现条件分支？

**A**：使用 `GraphBranch` 实现条件分支：

```go
// 创建条件分支
branch := compose.NewGraphBranch(
    func(ctx context.Context, input string) (string, error) {
        // 根据输入返回目标节点名
        if condition {
            return "node_a", nil
        }
        return "node_b", nil
    },
    map[string]bool{
        "node_a": true,
        "node_b": true,
    },
)

// 将分支添加到图
graph.AddBranch("source_node", branch)
```

### Q5: 如何实现并行执行？

**A**：使用 `Parallel` 实现并行执行：

```go
// 创建并行组
parallel := compose.NewParallel()

// 添加并行节点
parallel.AddLambda("key1", lambda1)
parallel.AddLambda("key2", lambda2)
parallel.AddLambda("key3", lambda3)

// 将并行组添加到 Chain
chain.AppendParallel(parallel)
```

并行节点会同时执行，它们的输出会汇聚成一个 map，key 是添加时指定的 outputKey。

### Q6: 如何将 Graph 封装为 Tool？

**A**：使用 `utils.InferTool` 或 `utils.NewTool` 将 Graph 的执行逻辑封装为 Tool：

```go
// 编译图
runnable, _ := graph.Compile(ctx)

// 封装为 Tool
tool, _ := utils.InferTool("tool_name", "工具描述",
    func(ctx context.Context, input InputType) (OutputType, error) {
        return runnable.Invoke(ctx, input)
    },
)
```

这样，复杂的图工作流就可以作为一个简单的工具供 LLM 调用。

### Q7: AddEdge 和 AddBranch 有什么区别？

**A**：

| 对比项 | AddEdge | AddBranch |
|--------|---------|-----------|
| 用途 | 固定连接两个节点 | 根据条件选择目标节点 |
| 目标数量 | 1 个 | 多个（条件选择） |
| 使用场景 | 确定的数据流 | 动态路由 |

```go
// AddEdge：固定连接
graph.AddEdge("node_a", "node_b")  // node_a 的输出总是流向 node_b

// AddBranch：条件选择
graph.AddBranch("node_a", branch)   // node_a 的输出可能流向 node_b 或 node_c
```

### Q8: 图编译后还能修改吗？

**A**：不能。图编译后会返回 `ErrGraphCompiled` 错误。如果需要修改，必须重新创建一个新图。

```go
runnable, _ := graph.Compile(ctx)

// 尝试修改编译后的图会报错
err := graph.AddEdge("a", "b")
// err == compose.ErrGraphCompiled
```

## 练习题

### 练习 1：构建文本处理图

创建一个 Graph，实现以下文本处理流程：

```
输入 → 去空格 → 转小写 → 统计字数 → 输出
```

要求：
- 使用 Lambda 节点
- 每个步骤打印中间结果
- 输出格式：`"原文: xxx, 处理后: xxx, 字数: xxx"`

```go
// 提示：
graph := compose.NewGraph[string, string]()

// 1. 添加 trim 节点
// 2. 添加 to_lower 节点
// 3. 添加 count_words 节点（使用闭包传递中间结果）
// 4. 连接边
// 5. 编译并执行
```

### 练习 2：实现条件路由

创建一个 Graph，根据输入内容类型选择不同的处理路径：

```
输入 → 分析类型 → ┌ 数字 → 计算平方
                   ├ 文本 → 转大写
                   └ 列表 → 拼接字符串
                   └──→ 汇总输出
```

要求：
- 使用 `GraphBranch` 实现条件分支
- 支持三种输入类型：数字、文本、列表（逗号分隔）
- 所有分支汇聚到同一个输出节点

```go
// 提示：
branch := compose.NewGraphBranch(
    func(ctx context.Context, input string) (string, error) {
        // 判断输入类型
        if _, err := strconv.Atoi(input); err == nil {
            return "number_handler", nil
        }
        if strings.Contains(input, ",") {
            return "list_handler", nil
        }
        return "text_handler", nil
    },
    map[string]bool{
        "number_handler": true,
        "text_handler":   true,
        "list_handler":   true,
    },
)
```

### 练习 3：并行数据收集

创建一个 Chain，使用 Parallel 并行收集多个维度的信息：

```
输入 → ┌ 计算长度 ────────┐
       ├ 检测是否包含数字 ─┤ → 汇总报告
       ├ 统计大写字母数 ──┘
       └ 检测是否包含特殊字符
```

要求：
- 使用 `compose.NewParallel()` 创建并行组
- 每个并行节点输出一个字符串
- 汇总节点将所有结果合并为一份报告

```go
// 提示：
parallel := compose.NewParallel()
parallel.AddLambda("length", lengthLambda)
parallel.AddLambda("has_digit", hasDigitLambda)
parallel.AddLambda("upper_count", upperCountLambda)
parallel.AddLambda("has_special", hasSpecialLambda)
```

### 练习 4：Graph 封装为 Tool

创建一个 Graph 实现"智能文本分析"功能，然后将其封装为 Tool：

功能要求：
1. 接收文本输入
2. 分析文本的语言（中/英文）
3. 根据语言选择不同的处理方式
4. 返回分析结果

```go
// 提示：定义输入输出结构
type AnalysisInput struct {
    Text string `json:"text" description:"要分析的文本"`
}

type AnalysisOutput struct {
    Language string `json:"language" description:"检测到的语言"`
    Length   int    `json:"length" description:"文本长度"`
    Summary  string `json:"summary" description:"分析摘要"`
}

// 创建 Graph → 编译 → 封装为 Tool
```

### 练习 5：嵌套子图

创建两个 Graph（子图和父图），将子图作为节点嵌入父图：

```
子图: 输入 → 去空格 → 转大写 → 输出

父图: 输入 → [子图] → 添加前缀 → 输出
```

要求：
- 子图处理文本标准化
- 父图在子图基础上添加业务逻辑
- 使用 `graph.AddGraphNode()` 嵌入子图

## 参考资料

### 官方资源

- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [Eino 示例代码](https://github.com/cloudwego/eino-examples)
- [Eino 官方文档 - Compose 包](https://pkg.go.dev/github.com/cloudwego/eino/compose)
- [Eino 官方文档 - Chain and Graph 编排](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/)

### 关键 API 参考

| API | 说明 |
|-----|------|
| `compose.NewGraph[I, O]()` | 创建新的 Graph |
| `compose.NewChain[I, O]()` | 创建新的 Chain |
| `graph.AddLambdaNode(key, lambda)` | 添加 Lambda 节点 |
| `graph.AddEdge(start, end)` | 添加边 |
| `graph.AddBranch(start, branch)` | 添加条件分支 |
| `graph.Compile(ctx)` | 编译图 |
| `compose.InvokableLambda(fn)` | 创建同步 Lambda |
| `compose.NewGraphBranch(cond, ends)` | 创建条件分支 |
| `compose.NewParallel()` | 创建并行组 |
| `utils.InferTool(name, desc, fn)` | 从函数创建 Tool |

### 设计模式参考

- [DAG (有向无环图) - Wikipedia](https://en.wikipedia.org/wiki/Directed_acyclic_graph)
- [管道和过滤器模式 - Martin Fowler](https://martinfowler.com/articles/2023-pipelines-and-filters.html)
- [Go 泛型教程](https://go.dev/doc/tutorial/generics)

## 本章小结

本章学习了 Eino 框架中构建复杂工作流的核心机制 -- Graph 编排系统：

1. **Graph（图）**是一种有向无环图（DAG）编排系统，用于构建复杂的工作流
2. **Node（节点）**是图中的处理单元，包括 Lambda 节点、ChatModel 节点、ToolsNode 等
3. **Edge（边）**定义了节点之间的数据流向
4. **Compile（编译）**将图定义转化为可执行的 Runnable
5. **GraphBranch** 实现条件分支，根据条件选择不同的执行路径
6. **Parallel** 实现并行执行，让多个节点同时处理
7. **Chain** 是 Graph 的简化版，适合线性处理场景
8. **Graph 可以封装为 Tool**，供 LLM 调用复杂的工作流

下一章我们将学习 **Skill 和 Middleware**，了解如何扩展 Eino 的能力。
