package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

func loadConfig() (*Config, error) {
	// 从项目根目录的 config.yaml 读取
	data, err := os.ReadFile("../config.yaml")
	if err != nil {
		return nil, fmt.Errorf("读取 config.yaml 失败: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 config.yaml 失败: %w", err)
	}
	return &cfg, nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "simple":
			simpleChat()
		case "interactive":
			interactiveChat()
		case "stream":
			streamChat()
		default:
			printUsage()
		}
	} else {
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino ChatModel 示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go simple       - 运行简单对话示例")
	fmt.Println("  go run main.go interactive  - 运行交互式多轮对话")
	fmt.Println("  go run main.go stream       - 运行流式输出示例")
	fmt.Println("")
	fmt.Println("配置:")
	fmt.Println("  在项目根目录的 config.yaml 中配置 api_key、model、base_url")
}

// simpleChat 演示最简单的对话
func simpleChat() {
	ctx := context.Background()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 构建消息列表
	messages := []*schema.Message{
		schema.SystemMessage("你是一个 helpful 的助手，请用简洁明了的语言回答问题。"),
		schema.UserMessage("请用一句话介绍 Go 语言的优势。"),
	}

	// 调用模型生成响应
	fmt.Println("正在调用 AI 模型...")
	resp, err := chatModel.Generate(ctx, messages)
	if err != nil {
		fmt.Printf("调用模型失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	fmt.Println("\n=== AI 回复 ===")
	fmt.Println(resp.Content)
}

// interactiveChat 演示交互式多轮对话
func interactiveChat() {
	ctx := context.Background()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 维护对话历史
	messages := []*schema.Message{
		schema.SystemMessage("你是一个 helpful 的助手。"),
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("=== 欢迎使用 AI 对话系统 ===")
	fmt.Println("命令: 'clear' 清空历史, 'quit' 退出")
	fmt.Println()

	for {
		fmt.Print("你: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// 处理命令
		switch userInput {
		case "quit":
			fmt.Println("再见！")
			return
		case "clear":
			messages = []*schema.Message{
				schema.SystemMessage("你是一个 helpful 的助手。"),
			}
			fmt.Println("✓ 对话历史已清空")
			continue
		case "":
			continue
		}

		// 添加用户消息到历史
		messages = append(messages, schema.UserMessage(userInput))

		// 调用模型
		fmt.Print("AI: ")
		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			fmt.Printf("调用模型失败: %v\n", err)
			continue
		}

		// 添加助手回复到历史
		messages = append(messages, schema.AssistantMessage(resp.Content, nil))

		// 输出回复
		fmt.Println(resp.Content)
		fmt.Println()
	}
}

// streamChat 演示流式输出
func streamChat() {
	ctx := context.Background()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一个 helpful 的助手。"),
		schema.UserMessage("请写一首关于编程的短诗。"),
	}

	// 使用 Stream 方法获取流式响应
	fmt.Println("正在生成流式响应...")
	stream, err := chatModel.Stream(ctx, messages)
	if err != nil {
		fmt.Printf("创建流失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== AI 回复（流式输出）===")
	for {
		chunk, err := stream.Recv()
		if err != nil {
			break // 流结束
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println("\n")
}
