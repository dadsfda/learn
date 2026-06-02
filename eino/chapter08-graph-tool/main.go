// ============================================================================
// 第 8 章：Graph Tool -- 复杂工作流
// ============================================================================
//
// 本文件演示 Eino 框架的 Graph 编排系统。
//
// 由于 Eino 的 Graph 编排需要完整的组件环境，本示例采用"先理解原理，再看实际用法"的方式：
//
//   Part 1: 用纯 Go 模拟 Graph 的核心概念（不需要 API Key）
//   Part 2: 展示如何在真实 Eino 中使用 Graph（需要 API Key）
//
// 运行方式：
//   go run main.go demo          - 运行完整演示（模拟，不需要 API Key）
//   go run main.go basic         - 基础 Graph 构建示例
//   go run main.go chain         - 多节点链式处理示例
//   go run main.go branch        - 条件分支示例
//   go run main.go parallel      - 并行执行示例
//   go run main.go graphtool     - Graph 封装为 Tool 示例
//   go run main.go simplechain   - Chain 链式编排示例
//   go run main.go eino          - 真实 Eino Graph 示例（需要 API Key）
//
// ============================================================================

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Part 1: 模拟 Graph 编排系统（帮助理解原理）
// ============================================================================
//
// 这部分用纯 Go 代码模拟 Eino 的 Graph 编排系统。
// 不需要任何外部依赖，帮助你理解 Graph 的核心思想。
//
// ============================================================================

// --------------------------------------------------------------------------
// 1.1 核心类型定义
// --------------------------------------------------------------------------

// NodeFunc 节点函数类型
// 在真实 Eino 中，对应 Lambda 节点的函数签名
type NodeFunc func(ctx context.Context, input any) (any, error)

// Node 节点定义
type Node struct {
	Name string   // 节点名称
	Fn   NodeFunc // 节点执行函数
}

// Edge 边定义
type Edge struct {
	From string // 起始节点
	To   string // 目标节点
}

// Branch 条件分支定义
type Branch struct {
	SourceNode string                              // 源节点
	Condition  func(ctx context.Context, input any) (string, error) // 条件函数，返回目标节点名
	EndNodes   map[string]bool                     // 可能的目标节点列表
}

// Graph 图定义
type Graph struct {
	Nodes    map[string]*Node   // 所有节点
	Edges    []Edge             // 所有边
	Branches map[string]*Branch // 条件分支（key 是源节点名）
	Start    string             // 起始节点
	End      string             // 结束节点
}

// GraphResult 图执行结果
type GraphResult struct {
	Output any   // 最终输出
	Error  error // 错误信息
}

// --------------------------------------------------------------------------
// 1.2 Graph 构建器
// --------------------------------------------------------------------------

// NewGraph 创建一个新的图
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		Edges:    []Edge{},
		Branches: make(map[string]*Branch),
		Start:    "START",
		End:      "END",
	}
}

// AddNode 添加节点
func (g *Graph) AddNode(name string, fn NodeFunc) *Graph {
	g.Nodes[name] = &Node{
		Name: name,
		Fn:   fn,
	}
	return g
}

// AddEdge 添加边
func (g *Graph) AddEdge(from, to string) *Graph {
	g.Edges = append(g.Edges, Edge{From: from, To: to})
	return g
}

// AddBranch 添加条件分支
func (g *Graph) AddBranch(sourceNode string, condition func(ctx context.Context, input any) (string, error), endNodes map[string]bool) *Graph {
	g.Branches[sourceNode] = &Branch{
		SourceNode: sourceNode,
		Condition:  condition,
		EndNodes:   endNodes,
	}
	return g
}

// --------------------------------------------------------------------------
// 1.3 Graph 执行器
// --------------------------------------------------------------------------

// Execute 执行图
func (g *Graph) Execute(ctx context.Context, input any) GraphResult {
	// 记录每个节点的输入和输出
	// 在真实 Eino 中，数据流由边的类型决定
	// 这里简化处理：分支条件使用节点输出，下游节点使用节点输入
	nodeInputs := make(map[string]any)
	nodeOutputs := make(map[string]any)
	nodeInputs[g.Start] = input
	nodeOutputs[g.Start] = input

	// 构建邻接表（用于快速查找下一个节点）
	adjacency := make(map[string][]string)
	for _, edge := range g.Edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}

	// 拓扑排序执行
	// 简化版：按 BFS 顺序执行
	visited := make(map[string]bool)
	queue := []string{g.Start}
	visited[g.Start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// 如果是 END 节点，跳过执行
		if current == g.End {
			continue
		}

		// 获取当前节点的输入
		inputData, exists := nodeInputs[current]
		if !exists {
			return GraphResult{
				Error: fmt.Errorf("节点 %s 没有输入数据", current),
			}
		}

		// 执行当前节点
		node, exists := g.Nodes[current]
		if !exists {
			// START 节点不需要执行函数
			if current == g.Start {
				// 直接传递输入到下一个节点
			} else {
				return GraphResult{
					Error: fmt.Errorf("节点 %s 不存在", current),
				}
			}
		} else {
			output, err := node.Fn(ctx, inputData)
			if err != nil {
				return GraphResult{Error: err}
			}
			nodeOutputs[current] = output
		}

		// 检查是否有条件分支
		if branch, hasBranch := g.Branches[current]; hasBranch {
			// 执行条件函数，选择下一个节点
			// 注意：条件函数接收节点的输出（用于判断分支）
			nextNode, err := branch.Condition(ctx, nodeOutputs[current])
			if err != nil {
				return GraphResult{Error: err}
			}

			// 验证目标节点是否合法
			if !branch.EndNodes[nextNode] {
				return GraphResult{
					Error: fmt.Errorf("条件分支返回了无效的节点: %s", nextNode),
				}
			}

			// 将当前节点的输入传递给目标节点
			// 在真实 Eino 中，分支后的节点接收原始输入
			nodeInputs[nextNode] = nodeInputs[current]
			// 如果目标节点是 END，直接设置其输出
			if nextNode == g.End {
				nodeOutputs[nextNode] = nodeInputs[current]
			}
			if !visited[nextNode] {
				queue = append(queue, nextNode)
				visited[nextNode] = true
			}
		} else {
			// 没有分支，按边传递到下一个节点
			for _, next := range adjacency[current] {
				nodeInputs[next] = nodeOutputs[current]
				// 如果下一个节点是 END，直接设置其输出
				if next == g.End {
					nodeOutputs[next] = nodeOutputs[current]
				}
				if !visited[next] {
					queue = append(queue, next)
					visited[next] = true
				}
			}
		}
	}

	// 返回 END 节点的输出
	if output, exists := nodeOutputs[g.End]; exists {
		return GraphResult{Output: output}
	}

	return GraphResult{
		Error: fmt.Errorf("图执行完成，但没有输出"),
	}
}

// --------------------------------------------------------------------------
// 1.4 演示函数
// --------------------------------------------------------------------------

// demoBasicGraph 基础 Graph 构建示例
func demoBasicGraph() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  基础 Graph 构建示例")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.Background()

	// 创建图
	graph := NewGraph()

	// 添加节点：将输入转为大写
	graph.AddNode("to_upper", func(ctx context.Context, input any) (any, error) {
		text, ok := input.(string)
		if !ok {
			return nil, fmt.Errorf("输入必须是字符串")
		}
		result := strings.ToUpper(text)
		fmt.Printf("  [节点 to_upper] 输入: %q → 输出: %q\n", text, result)
		return result, nil
	})

	// 连接边：START → to_upper → END
	graph.AddEdge("START", "to_upper")
	graph.AddEdge("to_upper", "END")

	// 执行图
	fmt.Println("执行图: START → to_upper → END")
	fmt.Println()
	result := graph.Execute(ctx, "hello world")
	if result.Error != nil {
		fmt.Printf("执行失败: %v\n", result.Error)
	} else {
		fmt.Printf("最终结果: %q\n", result.Output)
	}
	fmt.Println()
}

// demoChainGraph 多节点链式处理示例
func demoChainGraph() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  多节点链式处理示例")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.Background()

	// 创建图
	graph := NewGraph()

	// 节点 1：去除空格
	graph.AddNode("trim", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		result := strings.TrimSpace(text)
		fmt.Printf("  [节点 trim] 输入: %q → 输出: %q\n", text, result)
		return result, nil
	})

	// 节点 2：转小写
	graph.AddNode("to_lower", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		result := strings.ToLower(text)
		fmt.Printf("  [节点 to_lower] 输入: %q → 输出: %q\n", text, result)
		return result, nil
	})

	// 节点 3：添加前缀
	graph.AddNode("add_prefix", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		result := "RESULT: " + text
		fmt.Printf("  [节点 add_prefix] 输入: %q → 输出: %q\n", text, result)
		return result, nil
	})

	// 连接边：START → trim → to_lower → add_prefix → END
	graph.AddEdge("START", "trim")
	graph.AddEdge("trim", "to_lower")
	graph.AddEdge("to_lower", "add_prefix")
	graph.AddEdge("add_prefix", "END")

	// 执行图
	fmt.Println("执行图: START → trim → to_lower → add_prefix → END")
	fmt.Println()
	result := graph.Execute(ctx, "  HELLO World  ")
	if result.Error != nil {
		fmt.Printf("执行失败: %v\n", result.Error)
	} else {
		fmt.Printf("最终结果: %q\n", result.Output)
	}
	fmt.Println()
}

// demoBranchGraph 条件分支示例
func demoBranchGraph() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  条件分支示例")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.Background()

	// 创建图
	graph := NewGraph()

	// 分析节点：判断输入是数字还是文本
	graph.AddNode("analyze", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		if _, err := strconv.Atoi(text); err == nil {
			fmt.Printf("  [节点 analyze] 输入: %q → 类型: 数字\n", text)
			return "number", nil
		}
		fmt.Printf("  [节点 analyze] 输入: %q → 类型: 文本\n", text)
		return "text", nil
	})

	// 数字处理节点
	graph.AddNode("handle_number", func(ctx context.Context, input any) (any, error) {
		// 从原始输入获取数字
		text := input.(string)
		num, _ := strconv.Atoi(text)
		result := fmt.Sprintf("数字 %d 的平方是 %d", num, num*num)
		fmt.Printf("  [节点 handle_number] 计算完成\n")
		return result, nil
	})

	// 文本处理节点
	graph.AddNode("handle_text", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		result := fmt.Sprintf("文本 %q 的大写是 %q", text, strings.ToUpper(text))
		fmt.Printf("  [节点 handle_text] 处理完成\n")
		return result, nil
	})

	// 连接边
	graph.AddEdge("START", "analyze")

	// 添加条件分支
	graph.AddBranch("analyze",
		func(ctx context.Context, input any) (string, error) {
			// input 是 analyze 节点的输出（"number" 或 "text"）
			branchType := input.(string)
			if branchType == "number" {
				return "handle_number", nil
			}
			return "handle_text", nil
		},
		map[string]bool{
			"handle_number": true,
			"handle_text":   true,
		},
	)

	// 两个分支都汇聚到 END
	graph.AddEdge("handle_number", "END")
	graph.AddEdge("handle_text", "END")

	// 测试 1：数字输入
	fmt.Println("--- 测试 1: 数字输入 ---")
	result := graph.Execute(ctx, "42")
	if result.Error != nil {
		fmt.Printf("执行失败: %v\n", result.Error)
	} else {
		fmt.Printf("最终结果: %q\n", result.Output)
	}
	fmt.Println()

	// 测试 2：文本输入
	fmt.Println("--- 测试 2: 文本输入 ---")
	result = graph.Execute(ctx, "hello")
	if result.Error != nil {
		fmt.Printf("执行失败: %v\n", result.Error)
	} else {
		fmt.Printf("最终结果: %q\n", result.Output)
	}
	fmt.Println()
}

// demoParallelGraph 并行执行示例
func demoParallelGraph() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  并行执行示例")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.Background()

	// 模拟并行执行
	// 在真实 Eino 中，Parallel 会自动并行执行节点
	// 这里我们用 goroutine 模拟并行行为

	type ParallelResult struct {
		Key    string
		Output any
		Error  error
	}

	// 定义并行节点
	parallelNodes := map[string]NodeFunc{
		"length": func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			time.Sleep(100 * time.Millisecond) // 模拟耗时
			result := fmt.Sprintf("长度: %d", len(text))
			fmt.Printf("  [并行节点 length] 完成\n")
			return result, nil
		},
		"upper": func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			time.Sleep(150 * time.Millisecond) // 模拟耗时
			result := "大写: " + strings.ToUpper(text)
			fmt.Printf("  [并行节点 upper] 完成\n")
			return result, nil
		},
		"reverse": func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			time.Sleep(80 * time.Millisecond) // 模拟耗时
			runes := []rune(text)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			result := "反转: " + string(runes)
			fmt.Printf("  [并行节点 reverse] 完成\n")
			return result, nil
		},
		"has_digit": func(ctx context.Context, input any) (any, error) {
			text := input.(string)
			time.Sleep(120 * time.Millisecond) // 模拟耗时
			hasDigit := false
			for _, r := range text {
				if r >= '0' && r <= '9' {
					hasDigit = true
					break
				}
			}
			result := fmt.Sprintf("包含数字: %v", hasDigit)
			fmt.Printf("  [并行节点 has_digit] 完成\n")
			return result, nil
		},
	}

	// 执行并行节点
	fmt.Println("并行执行 4 个节点...")
	start := time.Now()

	var wg sync.WaitGroup
	results := make(chan ParallelResult, len(parallelNodes))
	input := "Hello123"

	for name, fn := range parallelNodes {
		wg.Add(1)
		go func(name string, fn NodeFunc) {
			defer wg.Done()
			output, err := fn(ctx, input)
			results <- ParallelResult{Key: name, Output: output, Error: err}
		}(name, fn)
	}

	// 等待所有节点完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	parallelResults := make(map[string]any)
	for r := range results {
		if r.Error != nil {
			fmt.Printf("  节点 %s 执行失败: %v\n", r.Key, r.Error)
		} else {
			parallelResults[r.Key] = r.Output
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("\n并行执行完成，耗时: %v\n", elapsed)
	fmt.Println()

	// 汇总结果
	fmt.Println("--- 汇总结果 ---")
	for key, value := range parallelResults {
		fmt.Printf("  %s: %v\n", key, value)
	}
	fmt.Println()

	// 计算理论上的串行耗时
	serialTime := 100*time.Millisecond + 150*time.Millisecond + 80*time.Millisecond + 120*time.Millisecond
	fmt.Printf("如果串行执行，预计耗时: %v\n", serialTime)
	fmt.Printf("并行执行实际耗时: %v\n", elapsed)
	fmt.Printf("加速比: %.2fx\n", float64(serialTime)/float64(elapsed))
	fmt.Println()
}

// demoGraphAsTool Graph 封装为 Tool 示例
func demoGraphAsTool() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  Graph 封装为 Tool 示例")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.Background()

	// 模拟 ToolInfo（在真实 Eino 中是 schema.ToolInfo）
	type ToolInfo struct {
		Name        string
		Description string
		Parameters  map[string]string
	}

	// 模拟 InvokableTool 接口
	type InvokableTool interface {
		Info() ToolInfo
		Run(ctx context.Context, args string) (string, error)
	}

	// 创建一个文本处理 Graph
	graph := NewGraph()

	// 节点 1：去除空格
	graph.AddNode("trim", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		return strings.TrimSpace(text), nil
	})

	// 节点 2：转大写
	graph.AddNode("to_upper", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		return strings.ToUpper(text), nil
	})

	// 节点 3：添加前缀
	graph.AddNode("add_prefix", func(ctx context.Context, input any) (any, error) {
		text := input.(string)
		return "PROCESSED: " + text, nil
	})

	// 连接边
	graph.AddEdge("START", "trim")
	graph.AddEdge("trim", "to_upper")
	graph.AddEdge("to_upper", "add_prefix")
	graph.AddEdge("add_prefix", "END")

	// 将 Graph 封装为 Tool
	type TextProcessorTool struct {
		graph *Graph
	}

	tool := &TextProcessorTool{graph: graph}

	// 实现 InvokableTool 接口
	toolInfo := ToolInfo{
		Name:        "text_processor",
		Description: "处理文本：去除空格、转大写、添加前缀",
		Parameters: map[string]string{
			"text": "要处理的文本",
		},
	}

	fmt.Printf("工具名称: %s\n", toolInfo.Name)
	fmt.Printf("工具描述: %s\n", toolInfo.Description)
	fmt.Printf("工具参数: %v\n", toolInfo.Parameters)
	fmt.Println()

	// 模拟 LLM 调用 Tool
	fmt.Println("--- 模拟 LLM 调用 Tool ---")
	fmt.Println()

	// LLM 决定调用 text_processor 工具
	fmt.Println("LLM: 我需要调用 text_processor 工具")
	fmt.Println("LLM: 参数: {\"text\": \"  hello world  \"}")
	fmt.Println()

	// 执行工具（内部调用 Graph）
	input := "  hello world  "
	fmt.Printf("工具执行: 输入 %q\n", input)
	result := tool.graph.Execute(ctx, input)
	if result.Error != nil {
		fmt.Printf("工具执行失败: %v\n", result.Error)
	} else {
		fmt.Printf("工具返回: %q\n", result.Output)
	}

	fmt.Println()
	fmt.Println("LLM: 工具返回了结果，我可以继续回答用户")
	fmt.Println()
}

// demoSimpleChain Chain 链式编排示例
func demoSimpleChain() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  Chain 链式编排示例")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	ctx := context.Background()

	// 模拟 Chain（在真实 Eino 中使用 compose.NewChain）
	// Chain 是 Graph 的简化版，适合线性处理

	// 定义链式处理函数
	type ChainStep struct {
		Name string
		Fn   NodeFunc
	}

	// 创建链
	steps := []ChainStep{
		{
			Name: "trim",
			Fn: func(ctx context.Context, input any) (any, error) {
				text := input.(string)
				result := strings.TrimSpace(text)
				fmt.Printf("  [步骤 1: trim] %q → %q\n", text, result)
				return result, nil
			},
		},
		{
			Name: "to_lower",
			Fn: func(ctx context.Context, input any) (any, error) {
				text := input.(string)
				result := strings.ToLower(text)
				fmt.Printf("  [步骤 2: to_lower] %q → %q\n", text, result)
				return result, nil
			},
		},
		{
			Name: "add_prefix",
			Fn: func(ctx context.Context, input any) (any, error) {
				text := input.(string)
				result := "RESULT: " + text
				fmt.Printf("  [步骤 3: add_prefix] %q → %q\n", text, result)
				return result, nil
			},
		},
	}

	// 执行链
	fmt.Println("执行链: trim → to_lower → add_prefix")
	fmt.Println()

	current := any("  HELLO World  ")
	for i, step := range steps {
		fmt.Printf("执行步骤 %d/%d: %s\n", i+1, len(steps), step.Name)
		output, err := step.Fn(ctx, current)
		if err != nil {
			fmt.Printf("步骤 %s 执行失败: %v\n", step.Name, err)
			return
		}
		current = output
	}

	fmt.Printf("\n最终结果: %q\n", current)
	fmt.Println()

	// 对比 Chain 和 Graph
	fmt.Println("--- Chain vs Graph 对比 ---")
	fmt.Println()
	fmt.Println("Chain（链式）:")
	fmt.Println("  输入 → trim → to_lower → add_prefix → 输出")
	fmt.Println("  特点: 简单、线性、易于理解")
	fmt.Println()
	fmt.Println("Graph（图）:")
	fmt.Println("        ┌──→ handle_number ──┐")
	fmt.Println("  输入 → analyze               → 输出")
	fmt.Println("        └──→ handle_text   ──┘")
	fmt.Println("  特点: 灵活、支持分支和并行")
	fmt.Println()
}

// demoFullDemo 完整演示
func demoFullDemo() {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  Eino Graph 编排系统完整演示")
	fmt.Println("  模拟一个 AI Agent 使用 Graph 构建复杂工作流")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	// 运行所有演示
	demoBasicGraph()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	demoChainGraph()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	demoBranchGraph()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	demoParallelGraph()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	demoGraphAsTool()
	fmt.Println(strings.Repeat("-", 70))
	fmt.Println()

	demoSimpleChain()
}

// --------------------------------------------------------------------------
// 1.5 真实 Eino Graph 用法（需要 API Key）
// --------------------------------------------------------------------------

// 以下是真实 Eino Graph 的代码示例。
// 由于需要 API Key 和完整的 Eino 依赖，这里只展示代码结构，
// 不会在 demo 模式中运行。
//
// 如果你想运行真实示例，请：
//   1. 设置 OPENAI_API_KEY 环境变量
//   2. 运行 go run main.go eino

/*
// === 真实 Eino Graph 示例 ===
// 以下代码展示如何在真实 Eino 中使用 Graph

import (
    "context"
    "fmt"
    "os"
    "strconv"
    "strings"

    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/components/tool/utils"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

// --- 示例 1: 基础 Graph ---
func basicGraphExample(ctx context.Context) {
    // 创建图：输入 string → 转大写 → 输出 string
    graph := compose.NewGraph[string, string]()

    // 添加 Lambda 节点
    graph.AddLambdaNode("to_upper",
        compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
            return strings.ToUpper(input), nil
        }),
    )

    // 连接边
    graph.AddEdge(compose.START, "to_upper")
    graph.AddEdge("to_upper", compose.END)

    // 编译并执行
    runnable, err := graph.Compile(ctx)
    if err != nil {
        fmt.Printf("编译失败: %v\n", err)
        return
    }

    result, err := runnable.Invoke(ctx, "hello world")
    if err != nil {
        fmt.Printf("执行失败: %v\n", err)
        return
    }

    fmt.Printf("结果: %s\n", result)
}

// --- 示例 2: 条件分支 ---
func branchExample(ctx context.Context) {
    graph := compose.NewGraph[string, string]()

    // 分析节点
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

    // 添加条件分支
    graph.AddBranch("analyze", compose.NewGraphBranch(
        func(ctx context.Context, input string) (string, error) {
            return input, nil
        },
        map[string]bool{
            "handle_number": true,
            "handle_text":   true,
        },
    ))

    graph.AddEdge("handle_number", compose.END)
    graph.AddEdge("handle_text", compose.END)

    // 编译并执行
    runnable, _ := graph.Compile(ctx)
    result, _ := runnable.Invoke(ctx, "42")
    fmt.Printf("结果: %s\n", result)
}

// --- 示例 3: Graph 封装为 Tool ---
func graphAsToolExample(ctx context.Context) {
    // 定义输入输出结构
    type TextInput struct {
        Text string `json:"text" description:"要处理的文本"`
    }

    type TextOutput struct {
        Result string `json:"result" description:"处理结果"`
    }

    // 创建 Graph
    graph := compose.NewGraph[TextInput, TextOutput]()

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
    textTool, _ := utils.InferTool("text_processor", "处理文本：去除空格并转为大写",
        func(ctx context.Context, input TextInput) (TextOutput, error) {
            return runnable.Invoke(ctx, input)
        },
    )

    // 将 Tool 添加到 Agent
    fmt.Printf("工具创建成功: %s\n", textTool.Info.Name)
}

// --- 示例 4: Chain 链式编排 ---
func chainExample(ctx context.Context) {
    // 创建 Chain
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
    fmt.Printf("结果: %s\n", result)
}

// --- 示例 5: 使用 ToolsNode ---
func toolsNodeExample(ctx context.Context) {
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

    graph.AddToolsNode("tools", toolsNode)
    graph.AddEdge(compose.START, "tools")
    graph.AddEdge("tools", compose.END)

    // 编译并执行
    runnable, _ := graph.Compile(ctx)

    msg := &schema.Message{
        Role: schema.Assistant,
        ToolCalls: []schema.ToolCall{
            {
                ID:        "call_1",
                Name:      "get_weather",
                Arguments: `{"city": "北京"}`,
            },
        },
    }

    results, _ := runnable.Invoke(ctx, msg)
    fmt.Printf("结果: %s\n", results[0].Content)
}

// --- 运行真实 Eino 示例 ---
func runRealEinoExample() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        fmt.Println("此示例需要 OPENAI_API_KEY 环境变量。")
        return
    }

    ctx := context.Background()

    fmt.Println("运行真实 Eino Graph 示例...")
    fmt.Println()

    basicGraphExample(ctx)
    branchExample(ctx)
    graphAsToolExample(ctx)
    chainExample(ctx)
    toolsNodeExample(ctx)
}
*/

// ============================================================================
// main 函数
// ============================================================================

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "demo":
		demoFullDemo()
	case "basic":
		demoBasicGraph()
	case "chain":
		demoChainGraph()
	case "branch":
		demoBranchGraph()
	case "parallel":
		demoParallelGraph()
	case "graphtool":
		demoGraphAsTool()
	case "simplechain":
		demoSimpleChain()
	case "eino":
		demoEinoRealGraph()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino Graph 编排系统学习示例")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run main.go demo          - 运行完整演示（推荐！）")
	fmt.Println("  go run main.go basic         - 基础 Graph 构建示例")
	fmt.Println("  go run main.go chain         - 多节点链式处理示例")
	fmt.Println("  go run main.go branch        - 条件分支示例")
	fmt.Println("  go run main.go parallel      - 并行执行示例")
	fmt.Println("  go run main.go graphtool     - Graph 封装为 Tool 示例")
	fmt.Println("  go run main.go simplechain   - Chain 链式编排示例")
	fmt.Println("  go run main.go eino          - 真实 Eino Graph 示例（需要 API Key）")
	fmt.Println()
	fmt.Println("建议学习顺序:")
	fmt.Println("  1. go run main.go demo       ← 先看完整效果")
	fmt.Println("  2. go run main.go basic      ← 理解基础 Graph")
	fmt.Println("  3. go run main.go chain      ← 理解链式处理")
	fmt.Println("  4. go run main.go branch     ← 理解条件分支")
	fmt.Println("  5. go run main.go parallel   ← 理解并行执行")
	fmt.Println("  6. go run main.go graphtool  ← 理解 Graph 作为 Tool")
	fmt.Println("  7. go run main.go simplechain ← 理解 Chain 编排")
}

// demoEinoRealGraph 展示真实 Eino Graph 的用法
func demoEinoRealGraph() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("此示例需要 OPENAI_API_KEY 环境变量。")
		fmt.Println()
		fmt.Println("请按以下步骤操作：")
		fmt.Println("  1. 设置环境变量: export OPENAI_API_KEY=\"your-key\"")
		fmt.Println("  2. 初始化模块: go mod init chapter08-graph-tool")
		fmt.Println("  3. 安装依赖:   go get github.com/cloudwego/eino github.com/cloudwego/eino-ext/components/model/openai")
		fmt.Println("  4. 取消 main.go 中真实 Eino 代码的注释")
		fmt.Println("  5. 重新运行:   go run main.go eino")
		fmt.Println()
		fmt.Println("或者运行模拟演示: go run main.go demo")
		return
	}

	fmt.Println("真实 Eino Graph 示例")
	fmt.Println("请参考 main.go 中的注释代码，取消注释后运行。")
	fmt.Println("当前 API Key 已配置，长度:", len(apiKey))
}
