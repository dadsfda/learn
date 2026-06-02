package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/pkg/adk"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "simple":
			simpleAgent()
		case "tools":
			agentWithTools()
		case "interactive":
			interactiveAgent()
		default:
			printUsage()
		}
	} else {
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino ChatModelAgent 示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go simple       - 运行基础 Agent 示例")
	fmt.Println("  go run main.go tools        - 运行带工具的 Agent 示例")
	fmt.Println("  go run main.go interactive  - 运行交互式 Agent 对话")
	fmt.Println("")
	fmt.Println("环境变量:")
	fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥（必需）")
}

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

// simpleAgent 演示基础 Agent
func simpleAgent() {
	ctx := context.Background()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		os.Exit(1)
	}

	// 创建 ChatModel
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o",
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 ChatModelAgent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: chatModel,
	})
	if err != nil {
		fmt.Printf("创建 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	// 执行查询
	fmt.Println("正在查询 AI...")
	iter := runner.Query(ctx, "你好，请介绍一下你自己。")

	// 处理事件流
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

// agentWithTools 演示带工具的 Agent
func agentWithTools() {
	ctx := context.Background()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		os.Exit(1)
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o",
		APIKey: apiKey,
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

// interactiveAgent 演示交互式 Agent 对话
func interactiveAgent() {
	ctx := context.Background()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		os.Exit(1)
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o",
		APIKey: apiKey,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

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
