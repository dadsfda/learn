# 第 9 章：Skill Middleware -- 技能中间件

## 学习目标

通过本章学习，你将掌握：

1. **Skill 概念**：理解什么是 Skill（技能），以及它与普通 Tool 的区别
2. **渐进式披露**：掌握 Skill 的三阶段加载模式（发现 -> 激活 -> 执行）
3. **SKILL.md 规范**：学会编写技能描述文件，包含元数据和执行指令
4. **Backend 抽象**：理解技能存储后端的设计，学会使用文件系统后端
5. **上下文模式**：掌握 inline、fork、fork_with_context 三种执行模式
6. **ToolSearch 中间件**：理解动态工具发现机制，解决工具过多导致的上下文溢出问题
7. **工具组合**：学会将多个工具和技能组合成强大的 Agent 能力

## 前置知识

- 第 1 章：ChatModel 和 Message
- 第 2 章：ChatModelAgent 和 Runner
- 第 4 章：Tools 和 FileSystem
- 第 5 章：Middleware 横切关注点
- Go 接口和结构体
- Go 文件操作基础

## 核心概念

### 1. 什么是 Skill？

在前面的章节中，我们学习了 Tool（工具）—— 赋予 AI "手和脚"的机制。但随着 Agent 能力的增长，我们会遇到一个新问题：**工具太多，上下文放不下**。

想象一下：
- 一个客服 Agent 可能需要 50 个工具（查询订单、退款、修改地址...）
- 一个开发 Agent 可能需要 100 个工具（读文件、写代码、运行测试...）
- 一个全能 Agent 可能需要 500 个工具

如果把所有工具的描述都塞进 LLM 的上下文窗口：
- 上下文会被工具描述占满，留给对话内容的空间变少
- LLM 在大量工具中选择的准确率会下降
- 每次请求都要传输大量 token，成本增加

**Skill（技能）** 就是解决这个问题的方案。它是一种更高层的抽象：

```
普通 Tool：  一个工具 = 一个函数（如 read_file）
Skill：     一组指令 + 脚本 + 资源 = 一个完整的能力（如 "PDF 处理"）
```

**生活中的类比**：
- Tool 就像一把螺丝刀 —— 功能单一，直接使用
- Skill 就像一个工具箱 —— 包含多种工具和使用说明，按需打开

### 2. Skill 的三阶段渐进式披露

Skill 采用**渐进式披露（Progressive Disclosure）**模式，分三个阶段加载：

```
阶段 1: 发现（Discovery）
    Agent 启动时，只加载每个 Skill 的名称和描述
    ┌─────────────────────────────────────┐
    │  可用技能列表：                       │
    │  - pdf-processing: 处理 PDF 文件      │
    │  - web-scraping: 网页数据抓取         │
    │  - data-analysis: 数据分析和可视化    │
    └─────────────────────────────────────┘
    占用上下文：很少（只有名称+描述）

阶段 2: 激活（Activation）
    当 Agent 判断需要某个 Skill 时，加载其完整内容
    ┌─────────────────────────────────────┐
    │  激活技能: pdf-processing             │
    │  ┌─────────────────────────────────┐│
    │  │ SKILL.md 完整内容：              ││
    │  │ - 详细的执行步骤                 ││
    │  │ - 可用的脚本列表                 ││
    │  │ - 参考文档                       ││
    │  └─────────────────────────────────┘│
    └─────────────────────────────────────┘
    占用上下文：按需加载，只加载需要的

阶段 3: 执行（Execution）
    Agent 按照 Skill 的指令执行任务
    ┌─────────────────────────────────────┐
    │  执行步骤：                          │
    │  1. 调用 scripts/extract_text.sh    │
    │  2. 解析输出结果                     │
    │  3. 返回结构化数据                   │
    └─────────────────────────────────────┘
```

**核心优势**：
- 上下文高效利用：不使用的 Skill 几乎不占空间
- 按需加载：只在需要时才消耗 token
- 能力可扩展：可以随时添加新 Skill，不影响已有能力

### 3. SKILL.md 规范

每个 Skill 的核心是一个 `SKILL.md` 文件，包含 YAML 前置元数据和正文内容：

```markdown
---
name: pdf-processing
description: 处理 PDF 文件，包括文本提取、页面分割、合并等操作
context: inline
agent: ""
model: ""
---

# PDF 处理技能

## 能力说明
本技能提供 PDF 文件的处理能力，包括：
- 提取 PDF 中的文本内容
- 按页码分割 PDF
- 合并多个 PDF 文件
- 提取 PDF 中的图片

## 使用方法
1. 使用 extract_text 脚本提取文本
2. 使用 split_pdf 脚本分割页面
3. 使用 merge_pdfs 脚本合并文件

## 可用脚本
- scripts/extract_text.sh - 提取文本
- scripts/split_pdf.py - 分割页面
- scripts/merge_pdfs.py - 合并文件
```

#### FrontMatter 元数据字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 技能的唯一标识符，如 `pdf-processing` |
| `description` | string | 是 | 技能描述，用于 Agent 判断是否使用该技能 |
| `context` | string | 否 | 上下文模式：`inline`（默认）、`fork`、`fork_with_context` |
| `agent` | string | 否 | 指定使用的 Agent 名称（通过 AgentHub 解析） |
| `model` | string | 否 | 指定使用的模型名称（通过 ModelHub 解析） |

#### Skill 目录结构

```
my-skill/
├── SKILL.md          # 必需：指令 + 元数据
├── scripts/          # 可选：可执行脚本
├── references/       # 可选：参考文档
└── assets/           # 可选：模板和资源
```

### 4. 上下文模式（ContextMode）

Skill 支持三种执行模式，决定了 Skill 在哪个上下文中运行：

#### Inline 模式（默认）

```
用户消息 → 当前 Agent → 激活 Skill → 在当前上下文中执行 → 返回结果
```

- Skill 内容作为工具调用的返回值，直接注入当前对话
- 当前 Agent 继续处理，可以看到 Skill 的完整指令
- 适用于：简单任务，不需要隔离上下文

#### Fork 模式

```
用户消息 → 当前 Agent → 激活 Skill → 创建新 Agent（无历史）→ 执行 → 返回结果
```

- 创建一个全新的 Agent，不携带任何对话历史
- 新 Agent 只有 Skill 的内容作为输入
- 适用于：独立任务，不依赖对话上下文

#### ForkWithContext 模式

```
用户消息 → 当前 Agent → 激活 Skill → 创建新 Agent（复制历史）→ 执行 → 返回结果
```

- 创建一个新 Agent，但复制当前对话历史
- 新 Agent 既能看懂上下文，又能独立执行
- 适用于：需要理解对话背景的复杂任务

```
三种模式对比：
┌─────────────────┬──────────┬──────────┬──────────────────┐
│     特性         │  Inline  │   Fork   │ ForkWithContext  │
├─────────────────┼──────────┼──────────┼──────────────────┤
│ 执行 Agent       │  当前    │   新建   │      新建        │
│ 对话历史         │  保留    │   无     │      复制        │
│ 上下文隔离       │  否      │   是     │      是          │
│ Token 消耗       │  低      │   低     │      高          │
│ 适用场景         │  简单    │   独立   │      复杂        │
└─────────────────┴──────────┴──────────┴──────────────────┘
```

### 5. Backend 抽象

Skill 的存储和检索通过 `Backend` 接口解耦：

```go
// Backend 是技能存储后端的接口
type Backend interface {
    // List 列出所有可用技能的元数据（用于发现阶段）
    List(ctx context.Context) ([]FrontMatter, error)

    // Get 获取指定技能的完整内容（用于激活阶段）
    Get(ctx context.Context, name string) (Skill, error)
}
```

**内置实现：文件系统后端**

```go
// 从文件系统创建 Backend
backend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
    Backend: fsBackend,  // 文件系统后端
    BaseDir: "/path/to/skills",  // 技能目录
})
```

文件系统后端会扫描 `BaseDir` 下的一级子目录，查找每个子目录中的 `SKILL.md` 文件并解析其 YAML 前置元数据。

**自定义 Backend**：你也可以实现自己的 Backend，从数据库、远程服务或任何其他来源加载技能。

### 6. ToolSearch 中间件

ToolSearch 是另一个解决"工具过多"问题的中间件。它与 Skill 中间件互补：

```
Skill 中间件：  解决"能力太多"的问题 —— 将能力组织为 Skill，按需加载
ToolSearch 中间件：解决"工具太多"的问题 —— 让 Agent 搜索需要的工具
```

#### ToolSearch 的三种模式

**模式 1：默认模式（客户端控制）**

```
Round 1: Agent 只看到 tool_search + 静态工具
    Agent: "我需要查天气" → 调用 tool_search(query="weather")
    返回: [weather_forecast, weather_alert]

Round 2: Agent 看到 tool_search + 静态工具 + weather_forecast + weather_alert
    Agent: 调用 weather_forecast(city="Beijing")
    返回: "北京今天晴，25°C"
```

**模式 2：模型原生服务端检索**

```
动态工具放入 DeferredToolInfos，由模型在服务端检索
适用于支持原生工具搜索的模型（如 Claude）
```

**模式 3：模型原生客户端代理**

```
模型调用 tool_search 工具，客户端执行搜索并返回结果
模型根据返回的 ToolInfo 决定使用哪些工具
```

#### tool_search 的查询语法

| 查询类型 | 示例 | 说明 |
|----------|------|------|
| 关键词搜索 | `"weather forecast"` | 匹配工具名称和描述 |
| 直接选择 | `"select:tool_a,tool_b"` | 精确匹配工具名 |
| 强制匹配 | `"+slack send message"` | `+` 前缀强制包含该关键词 |

#### 评分算法

关键词搜索使用以下评分规则：

| 规则 | 分数 |
|------|------|
| 工具名拆分部分**精确匹配**关键词 | 10 分 |
| 工具名拆分部分**包含**关键词 | 5 分 |
| 完整工具名包含关键词 | 3 分 |
| 工具描述包含关键词 | 2 分 |

工具名按下划线、双下划线（MCP 分隔符）和驼峰边界拆分。例如 `mcp__slack__send_message` 拆分为 `["mcp", "slack", "send", "message"]`。

### 7. 工具组合模式

在实际应用中，我们经常需要将多个工具和技能组合在一起：

```
┌─────────────────────────────────────────┐
│              Agent                       │
│  ┌─────────────────────────────────┐    │
│  │  静态工具（始终可用）             │    │
│  │  - read_file                    │    │
│  │  - write_file                   │    │
│  │  - search_web                   │    │
│  └─────────────────────────────────┘    │
│  ┌─────────────────────────────────┐    │
│  │  Skill 中间件（按需加载能力）     │    │
│  │  - pdf-processing               │    │
│  │  - data-analysis                │    │
│  │  - code-review                  │    │
│  └─────────────────────────────────┘    │
│  ┌─────────────────────────────────┐    │
│  │  ToolSearch 中间件（动态发现）    │    │
│  │  - weather_api                  │    │
│  │  - stock_api                    │    │
│  │  - currency_api                 │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

## 代码示例

### 示例 1：基础 Skill 定义

创建一个简单的 Skill 目录结构：

```
skills/
└── greeting/
    └── SKILL.md
```

`skills/greeting/SKILL.md` 内容：

```markdown
---
name: greeting
description: 生成个性化的问候语，支持多种语言和风格
context: inline
---

# 问候语生成技能

## 功能说明
根据用户的需求生成个性化的问候语。

## 使用方法
1. 分析用户想要的语言和风格
2. 生成合适的问候语
3. 如果需要，提供翻译和发音说明

## 示例
- 中文正式：尊敬的先生/女士，您好！
- 英文 casual：Hey there! What's up?
- 日文礼貌：お世話になっております。
```

### 示例 2：使用文件系统 Backend

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/adk/middlewares/dynamictool/skill"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 1. 准备技能目录
    skillDir := "./skills"
    ensureSkillDirectory(skillDir)

    // 2. 创建文件系统 Backend
    // 注意：这里需要先创建 filesystem.Backend
    // 具体实现取决于 Eino 的 filesystem 包
    fsBackend := createFilesystemBackend()

    backend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
        Backend: fsBackend,
        BaseDir: skillDir,
    })
    if err != nil {
        log.Fatalf("创建 Backend 失败: %v", err)
    }

    // 3. 创建 Skill 中间件
    handler, err := skill.NewMiddleware(ctx, &skill.Config{
        Backend: backend,
    })
    if err != nil {
        log.Fatalf("创建 Skill 中间件失败: %v", err)
    }

    // 4. 创建 Agent 并注册中间件
    chatModel := createChatModel() // 你需要实现这个函数
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
        Handlers: []adk.ChatModelAgentMiddleware{handler},
    })
    if err != nil {
        log.Fatalf("创建 Agent 失败: %v", err)
    }

    fmt.Printf("Agent 创建成功，已注册 Skill 中间件\n")
    fmt.Printf("Agent: %+v\n", agent)
}
```

### 示例 3：ToolSearch 中间件配置

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

// weatherTool 模拟天气查询工具
type weatherTool struct{}

func (t *weatherTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "weather_forecast",
        Desc: "查询指定城市的天气预报",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "city": {
                Type:     "string",
                Desc:     "城市名称",
                Required: true,
            },
        }),
    }, nil
}

func (t *weatherTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
    return fmt.Sprintf("北京今天晴，25°C，微风"), nil
}

func main() {
    ctx := context.Background()

    // 1. 创建动态工具
    dynamicTools := []tool.BaseTool{
        &weatherTool{},
        // 可以添加更多工具...
    }

    // 2. 创建 ToolSearch 中间件（默认模式）
    middleware, err := toolsearch.New(ctx, &toolsearch.Config{
        DynamicTools:       dynamicTools,
        UseModelToolSearch: false, // 使用默认的客户端控制模式
    })
    if err != nil {
        log.Fatalf("创建 ToolSearch 中间件失败: %v", err)
    }

    // 3. 创建 Agent 并注册中间件
    chatModel := createChatModel()
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model:    chatModel,
        Handlers: []adk.ChatModelAgentMiddleware{middleware},
    })
    if err != nil {
        log.Fatalf("创建 Agent 失败: %v", err)
    }

    fmt.Printf("Agent 创建成功，已注册 ToolSearch 中间件\n")
    fmt.Printf("动态工具数量: %d\n", len(dynamicTools))
    fmt.Printf("Agent: %+v\n", agent)
}
```

### 示例 4：Skill 和 ToolSearch 组合使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/adk/middlewares/dynamictool/skill"
    "github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

func main() {
    ctx := context.Background()

    // 1. 创建 Skill 中间件
    skillBackend := createSkillBackend()
    skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
        Backend: skillBackend,
    })
    if err != nil {
        log.Fatalf("创建 Skill 中间件失败: %v", err)
    }

    // 2. 创建 ToolSearch 中间件
    dynamicTools := createDynamicTools()
    toolSearchMiddleware, err := toolsearch.New(ctx, &toolsearch.Config{
        DynamicTools: dynamicTools,
    })
    if err != nil {
        log.Fatalf("创建 ToolSearch 中间件失败: %v", err)
    }

    // 3. 创建 Agent，组合多个中间件
    chatModel := createChatModel()
    agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Model: chatModel,
        Handlers: []adk.ChatModelAgentMiddleware{
            skillMiddleware,       // Skill 中间件
            toolSearchMiddleware,  // ToolSearch 中间件
        },
    })
    if err != nil {
        log.Fatalf("创建 Agent 失败: %v", err)
    }

    fmt.Printf("Agent 创建成功\n")
    fmt.Printf("已注册中间件: Skill, ToolSearch\n")
    fmt.Printf("Agent: %+v\n", agent)
}
```

### 示例 5：自定义 Backend 实现

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/cloudwego/eino/adk/middlewares/dynamictool/skill"
)

// MemoryBackend 基于内存的技能后端
// 适用于测试场景或动态生成技能的场景
type MemoryBackend struct {
    mu     sync.RWMutex
    skills map[string]skill.Skill
}

// NewMemoryBackend 创建内存后端
func NewMemoryBackend() *MemoryBackend {
    return &MemoryBackend{
        skills: make(map[string]skill.Skill),
    }
}

// Register 注册一个技能
func (b *MemoryBackend) Register(s skill.Skill) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.skills[s.Name] = s
}

// List 列出所有技能的元数据
func (b *MemoryBackend) List(ctx context.Context) ([]skill.FrontMatter, error) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    result := make([]skill.FrontMatter, 0, len(b.skills))
    for _, s := range b.skills {
        result = append(result, s.FrontMatter)
    }
    return result, nil
}

// Get 获取指定技能的完整内容
func (b *MemoryBackend) Get(ctx context.Context, name string) (skill.Skill, error) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    s, ok := b.skills[name]
    if !ok {
        return skill.Skill{}, fmt.Errorf("skill not found: %s", name)
    }
    return s, nil
}

func main() {
    // 使用内存后端
    backend := NewMemoryBackend()

    // 注册技能
    backend.Register(skill.Skill{
        FrontMatter: skill.FrontMatter{
            Name:        "greeting",
            Description: "生成个性化问候语",
            Context:     "inline",
        },
        Content: `# 问候语生成
根据用户需求生成个性化的问候语。
支持中文、英文、日文等多种语言。`,
    })

    backend.Register(skill.Skill{
        FrontMatter: skill.FrontMatter{
            Name:        "translation",
            Description: "多语言翻译服务",
            Context:     "inline",
        },
        Content: `# 翻译技能
提供高质量的多语言翻译。
支持中英日韩法德西等主要语言。`,
    })

    // 列出所有技能
    ctx := context.Background()
    skills, _ := backend.List(ctx)
    fmt.Println("已注册技能:")
    for _, s := range skills {
        fmt.Printf("  - %s: %s\n", s.Name, s.Description)
    }
}
```

## 运行步骤

### 1. 初始化 Go 模块

```bash
cd chapter09-skill-middleware
go mod init chapter09-skill-middleware
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

### 4. 创建技能目录

```bash
mkdir -p skills/greeting
mkdir -p skills/translation
```

### 5. 运行示例

```bash
# 运行完整演示（模拟，不需要 API Key）
go run main.go demo

# 运行 Skill 演示
go run main.go skill

# 运行 ToolSearch 演示
go run main.go toolsearch

# 运行组合演示
go run main.go combined

# 运行自定义 Backend 演示
go run main.go backend
```

## 常见问题

### Q1: Skill 和 Tool 有什么区别？

**A**：

| 对比项 | Tool（工具） | Skill（技能） |
|--------|-------------|---------------|
| 粒度 | 单一函数 | 一组指令 + 脚本 + 资源 |
| 复杂度 | 简单（输入→输出） | 复杂（多步骤、多工具） |
| 上下文 | 无状态 | 可携带上下文和历史 |
| 适用场景 | 原子操作 | 复杂工作流 |
| 示例 | `read_file` | "PDF 处理"（包含提取、分割、合并） |

简单来说：Tool 是"一把锤子"，Skill 是"一个工具箱加使用说明书"。

### Q2: 什么时候用 Skill，什么时候用 ToolSearch？

**A**：

| 场景 | 推荐方案 |
|------|----------|
| 工具数量少（< 20） | 直接使用 Tool，不需要中间件 |
| 工具数量多（20-100），需要按类别组织 | 使用 Skill 中间件 |
| 工具数量很多（> 100），需要动态搜索 | 使用 ToolSearch 中间件 |
| 既有复杂能力组织，又有大量动态工具 | 两者组合使用 |

### Q3: 渐进式披露会增加延迟吗？

**A**：会有一点点，但通常可以忽略：
- 发现阶段：Agent 启动时执行一次，加载所有 Skill 的元数据
- 激活阶段：每次激活一个 Skill 时，需要读取 SKILL.md 文件
- 执行阶段：与普通 Tool 调用相同

相比把所有内容都塞进上下文导致的 token 浪费和准确率下降，这点延迟是值得的。

### Q4: 如何选择上下文模式？

**A**：

- **Inline**：大多数场景的默认选择。Skill 在当前上下文中执行，简单高效。
- **Fork**：当 Skill 需要独立运行，不希望受对话历史干扰时使用。例如：代码审查、文档生成。
- **ForkWithContext**：当 Skill 需要理解对话背景，但又需要隔离执行时使用。例如：基于对话内容的数据分析。

### Q5: ToolSearch 的评分算法如何工作？

**A**：评分算法按关键词累加，每个关键词取其最高分：

```
工具: mcp__slack__send_message
描述: "Send a message to a Slack channel"

查询: "slack message"

关键词 "slack":
  - 工具名拆分 "slack" 精确匹配 → 10 分

关键词 "message":
  - 工具名拆分 "message" 精确匹配 → 10 分

总分: 20 分
```

### Q6: 如何调试 Skill 和 ToolSearch？

**A**：

1. **日志中间件**：在 Skill/ToolSearch 中间件之前添加日志中间件，记录工具调用
2. **查看工具列表**：打印 Agent 可见的工具列表，确认哪些工具被加载
3. **模拟测试**：使用内存 Backend 进行单元测试，不依赖文件系统
4. **逐步激活**：先测试单个 Skill，再测试组合场景

### Q7: 中间件的执行顺序重要吗？

**A**：非常重要！中间件按洋葱模型执行：

```
注册顺序: [Skill, ToolSearch]
请求执行: Skill.Before → ToolSearch.Before → 实际调用
响应执行: 实际调用 → ToolSearch.After → Skill.After
```

建议的顺序：
1. 日志/追踪中间件（最外层）
2. Skill 中间件
3. ToolSearch 中间件
4. 错误处理中间件（最内层）

## 练习题

### 练习 1：创建一个代码审查 Skill

创建一个 `code-review` 技能，包含：
- SKILL.md 描述文件
- 支持多种编程语言的审查规则
- 自动检测常见问题（未处理错误、硬编码值等）

```
skills/
└── code-review/
    ├── SKILL.md
    └── scripts/
        └── lint_check.sh
```

### 练习 2：实现自定义 Backend

实现一个从 HTTP API 加载技能的 Backend：

```go
type HTTPBackend struct {
    baseURL string
    client  *http.Client
}

// 提示：
// 1. List 方法调用 GET /api/skills 获取技能列表
// 2. Get 方法调用 GET /api/skills/{name} 获取技能内容
// 3. 处理网络错误和超时
```

### 练习 3：实现工具注册表

创建一个工具注册表，支持动态注册和发现工具：

```go
type ToolRegistry struct {
    mu    sync.RWMutex
    tools map[string]tool.BaseTool
}

// 提示：
// 1. Register 方法注册工具
// 2. Unregister 方法注销工具
// 3. Search 方法支持关键词搜索
// 4. ListByCategory 方法按类别列出工具
```

### 练习 4：组合 Skill 和 ToolSearch

创建一个 Agent，同时使用 Skill 和 ToolSearch 中间件：
- Skill 用于组织复杂能力（如"数据分析"、"报告生成"）
- ToolSearch 用于动态发现简单工具（如"天气查询"、"汇率转换"）
- 验证两者可以协同工作

### 练习 5：实现 Skill 热加载

实现一个支持热加载的 Backend：
- 监控技能目录的变化（使用 fsnotify 或轮询）
- 当 SKILL.md 文件更新时，自动重新加载
- 当新技能目录创建时，自动发现并注册
- 当技能目录删除时，自动注销

## 参考资料

### 官方资源

- [Eino GitHub 仓库](https://github.com/cloudwego/eino)
- [Eino 示例代码](https://github.com/cloudwego/eino-examples)
- [Eino 官方文档 - Skill 中间件](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_skill/)
- [Eino 官方文档 - ToolSearch 中间件](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_toolsearch/)
- [Eino 官方文档 - ChatModelAgentMiddleware](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/)

### 设计模式参考

- [渐进式披露 - Nielsen Norman Group](https://www.nngroup.com/articles/progressive-disclosure/)
- [策略模式 - Refactoring Guru](https://refactoring.guru/design-patterns/strategy)
- [中间件模式 - Martin Fowler](https://martinfowler.com/articles/middleware-oriented-composition.html)

### Go 语言相关

- [Go 接口最佳实践](https://go.dev/doc/effective_go#interfaces)
- [Go 并发模式](https://go.dev/doc/effective_go#concurrency)
- [sync 包文档](https://pkg.go.dev/sync)
- [Go 文件系统操作](https://pkg.go.dev/os)

### LLM 工具使用相关

- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [Anthropic Tool Use](https://docs.anthropic.com/en/docs/build-with-claude/tool-use)
- [LangChain Tools](https://python.langchain.com/docs/modules/tools/)

## 本章小结

本章学习了 Eino 框架中两个重要的中间件 —— Skill 中间件和 ToolSearch 中间件：

1. **Skill（技能）** 是比 Tool 更高层的抽象，将一组指令、脚本和资源组织为一个完整的能力
2. **渐进式披露** 是 Skill 的核心设计，分三阶段加载：发现 → 激活 → 执行
3. **SKILL.md** 是技能的描述文件，包含 YAML 前置元数据和正文内容
4. **Backend** 接口解耦了技能的存储和使用，支持文件系统、内存、远程服务等多种实现
5. **上下文模式** 决定 Skill 在哪个上下文中执行：inline（当前）、fork（新建）、fork_with_context（复制历史）
6. **ToolSearch** 中间件解决工具过多的问题，让 Agent 动态搜索需要的工具
7. **工具组合** 可以将 Skill 和 ToolSearch 组合使用，构建强大的 Agent 能力

下一章我们将学习 **A2UI 协议**，了解 Agent 如何与用户界面交互。
