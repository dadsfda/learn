# Eino 框架小白学习教程

> 从零开始学习 Go 语言 LLM 应用开发框架 Eino

## 欢迎

欢迎来到 Eino 框架学习教程！本教程专为 Go 语言初学者设计，通过 11 个章节的渐进式学习，帮助你掌握 Eino 框架的核心概念和实战技能。

## 什么是 Eino？

**Eino**（发音 'aino'）是字节跳动开源的 Go 语言 LLM 应用开发框架，灵感来自 LangChain、Google ADK 等项目，但完全按照 Go 语言惯例设计。

### 核心特性

| 特性 | 说明 |
|------|------|
| **组件化设计** | ChatModel、Tool、Retriever 等可复用模块 |
| **智能体开发套件（ADK）** | 支持工具调用、多智能体协作、中断/恢复 |
| **图编排** | 通过 DAG 组装复杂工作流 |
| **流式处理** | 自动处理流式：拼接、装箱、合并、复制 |
| **回调切面** | 在生命周期切点注入日志、追踪、指标 |
| **中断/恢复** | 任何智能体或工具可暂停等待人工输入 |

## 前置知识

在开始学习之前，你需要具备以下基础知识：

### 必须掌握
- **Go 语言基础**：变量、函数、结构体、接口
- **Go 1.18 泛型**：类型参数、约束
- **Context**：上下文传递和取消机制
- **Goroutine 和 Channel**：并发编程基础

### 推荐学习资源
- [Go 官方教程](https://go.dev/tour/)
- [Go 泛型入门](https://go.dev/doc/tutorial/generics)
- [Go Context 详解](https://go.dev/blog/context)

## 环境准备

### 1. 安装 Go

```bash
# 检查 Go 版本（需要 >= 1.18）
go version

# 如果没有安装，请访问 https://go.dev/dl/
```

### 2. 配置 API Key

```bash
# OpenAI API Key（推荐）
export OPENAI_API_KEY="sk-your-key-here"

# 或者使用其他 LLM 提供商
export ANTHROPIC_API_KEY="your-claude-key"
```

### 3. 克隆教程代码

```bash
# 如果是从 Git 克隆
git clone <repository-url>
cd learn

# 或者直接使用已有的目录
cd e:\aiproject\learn
```

## 学习路径

本教程分为 4 个阶段，建议按顺序学习：

### 第一阶段：基础入门（1-3 天）

| 章节 | 主题 | 核心概念 | 预计时间 |
|------|------|----------|----------|
| [第 1 章](chapter01-chatmodel-message/) | ChatModel 和 Message | 对话模型、消息类型 | 2-3 小时 |
| [第 2 章](chapter02-chatmodel-agent-runner/) | ChatModelAgent 和 Runner | 智能体、运行器、事件流 | 3-4 小时 |
| [第 3 章](chapter03-memory-session/) | Memory 和 Session | 会话管理、状态持久化 | 3-4 小时 |

### 第二阶段：工具和中间件（4-7 天）

| 章节 | 主题 | 核心概念 | 预计时间 |
|------|------|----------|----------|
| [第 4 章](chapter04-tools-filesystem/) | Tools 和文件系统访问 | 工具接口、ReAct 循环 | 4-5 小时 |
| [第 5 章](chapter05-middleware/) | Middleware | 中间件模式、横切关注点 | 3-4 小时 |
| [第 6 章](chapter06-callback-trace/) | Callback 和 Trace | 回调机制、链路追踪 | 3-4 小时 |
| [第 7 章](chapter07-interrupt-resume/) | Interrupt/Resume | 人在环中、检查点 | 4-5 小时 |

### 第三阶段：编排和高级功能（7-14 天）

| 章节 | 主题 | 核心概念 | 预计时间 |
|------|------|----------|----------|
| [第 8 章](chapter08-graph-tool/) | Graph Tool | DAG 编排、工作流 | 5-6 小时 |
| [第 9 章](chapter09-skill-middleware/) | Skill Middleware | 技能管理、动态工具 | 4-5 小时 |
| [第 10 章](chapter10-a2ui-protocol/) | A2UI Protocol | 流式 UI、HTTP SSE | 4-5 小时 |
| [第 11 章](chapter11-turnloop/) | TurnLoop | 轮次循环、抢占调度 | 4-5 小时 |
| [第 12 章](chapter12-rag-retrieval-augmented-generation/) | RAG 检索增强生成 | 文档加载、向量检索、上下文增强 | 5-6 小时 |

### 第四阶段：实战项目（14+ 天）

完成所有章节后，建议进行实战项目练习：

1. **Todo 管理 Agent**：综合运用工具、会话、中间件
2. **RAG 知识问答系统**：文档加载、向量检索、上下文增强
3. **多智能体协作系统**：Agent 通信、任务分配、结果聚合

## 快速开始

### 1. 运行第一个示例

```bash
# 进入第 1 章目录
cd chapter01-chatmodel-message

# 初始化模块
go mod init chapter01

# 安装依赖
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/openai

# 运行简单对话示例
go run main.go simple
```

### 2. 体验交互式对话

```bash
# 运行交互式对话
go run main.go interactive

# 输入消息与 AI 对话
# 输入 'clear' 清空历史
# 输入 'quit' 退出
```

### 3. 查看流式输出

```bash
# 运行流式输出示例
go run main.go stream
```

## 各章节概览

### 第 1 章：ChatModel 和 Message
- 学习 ChatModel 接口
- 理解 Message 类型
- 完成第一次 LLM 调用
- **无需 API Key 的示例**：无

### 第 2 章：ChatModelAgent 和 Runner
- 构建智能体（Agent）
- 使用 Runner 执行查询
- 处理事件流
- **无需 API Key 的示例**：无

### 第 3 章：Memory 和 Session
- 实现会话持久化
- 内存和文件存储
- 跨会话状态管理
- **无需 API Key 的示例**：`memory`、`persist`

### 第 4 章：Tools 和文件系统访问
- 定义和实现工具
- 文件系统操作
- ReAct 循环
- **无需 API Key 的示例**：`basic`、`infer`、`filesystem`

### 第 5 章：Middleware
- 中间件模式
- 日志、认证、限流
- 中间件组合
- **无需 API Key 的示例**：`demo`、`onion`、`chain`、`logging`、`auth`、`ratelimit`、`safetool`

### 第 6 章：Callback 和 Trace
- 回调机制
- 链路追踪
- 性能监控
- **无需 API Key 的示例**：`basic`、`log`、`perf`、`trace`

### 第 7 章：Interrupt/Resume
- 中断和恢复
- 人工审批流程
- 检查点机制
- **无需 API Key 的示例**：`demo`、`checkpoint`、`approval`、`review`

### 第 8 章：Graph Tool
- DAG 图构建
- 条件分支和并行
- 图作为工具
- **无需 API Key 的示例**：`demo`、`basic`、`chain`、`branch`、`parallel`、`graphtool`、`simplechain`

### 第 9 章：Skill Middleware
- 技能定义和管理
- 动态工具发现
- 技能路由
- **无需 API Key 的示例**：`demo`、`skill`、`toolsearch`、`backend`、`combined`

### 第 10 章：A2UI Protocol
- HTTP SSE 服务
- 流式 UI 集成
- 前端通信
- **无需 API Key 的示例**：`sse`、`json-sse`

### 第 11 章：TurnLoop
- 轮次循环管理
- 超时和取消控制
- 抢占式调度
- **无需 API Key 的示例**：所有示例均无需 API Key

### 第 12 章：RAG 检索增强生成
- RAG 完整流程：文档加载 → 分割 → 嵌入 → 存储 → 检索 → 增强生成
- Eino RAG 组件：Document、Embedding、Indexer、Retriever、ChatTemplate
- 向量相似度计算和检索
- 构建知识问答系统
- **无需 API Key 的示例**：`demo`、`document`、`embedding`、`retriever`、`template`、`rag`

## 学习建议

### 1. 动手实践
- 每个章节的代码都要亲自运行
- 修改参数观察变化
- 完成每章的练习题

### 2. 理解概念
- 不要急于求成，理解核心概念
- 画图帮助理解流程
- 查阅官方文档深入学习

### 3. 循序渐进
- 按章节顺序学习
- 前一章没掌握不要跳到下一章
- 遇到困难多看几遍

### 4. 做笔记
- 记录重要概念
- 记录遇到的问题和解决方案
- 记录自己的理解

## 常见问题

### Q1: 需要什么 Go 版本？
A: 需要 Go 1.18 或更高版本，因为 Eino 大量使用泛型特性。

### Q2: 必须使用 OpenAI 吗？
A: 不必须。Eino 支持多种 LLM 提供商，包括 Claude、Gemini、Ollama 等。本教程默认使用 OpenAI，但你可以轻松切换。

### Q3: 没有 API Key 怎么办？
A: 很多示例无需 API Key 即可运行（已标注）。你也可以：
- 使用 Ollama 运行本地模型
- 申请 OpenAI 免费额度
- 使用其他免费 LLM 服务

### Q4: 学完能找到工作吗？
A: 本教程帮助你掌握 Eino 框架基础，但找工作还需要：
- 深入理解 AI/LLM 概念
- 积累实战项目经验
- 学习相关技术栈（向量数据库、RAG 等）

### Q5: 遇到问题怎么办？
A: 可以通过以下方式寻求帮助：
- 查阅 [官方文档](https://www.cloudwego.io/docs/eino/)
- 在 [GitHub Issues](https://github.com/cloudwego/eino/issues) 提问
- 加入 CloudWeGo 社区讨论

## 官方资源

| 资源 | 链接 | 说明 |
|------|------|------|
| **GitHub 仓库** | [github.com/cloudwego/eino](https://github.com/cloudwego/eino) | 核心框架源码 |
| **组件扩展** | [github.com/cloudwego/eino-ext](https://github.com/cloudwego/eino-ext) | OpenAI、Claude 等组件实现 |
| **示例代码** | [github.com/cloudwego/eino-examples](https://github.com/cloudwego/eino-examples) | 官方示例库 |
| **官方文档** | [cloudwego.io/docs/eino](https://www.cloudwego.io/docs/eino/) | 完整 API 文档 |
| **社区** | [cloudwego.io](https://www.cloudwego.io/) | CloudWeGo 社区 |

## 贡献

欢迎提交 Issue 和 Pull Request 来改进本教程！

## 许可证

本教程基于 Apache-2.0 许可证开源。

---

**开始学习**：[第 1 章：ChatModel 和 Message](chapter01-chatmodel-message/README.md)
