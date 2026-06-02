package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// 第 4 章：Tools 和文件系统访问
// =============================================================================
//
// 本章演示 Eino 框架的工具系统，包括：
// 1. 手动实现 InvokableTool 接口
// 2. 使用 InferTool 泛型函数快速创建工具
// 3. 使用 InMemoryBackend 实现文件系统工具
// 4. 将工具集成到 Agent 中
//
// 运行方式：
//   go run main.go basic         - 基础工具实现演示（无需 API Key）
//   go run main.go infer         - InferTool 泛型工具演示（无需 API Key）
//   go run main.go filesystem    - 文件系统工具演示（无需 API Key）
//   go run main.go agent         - 带工具的 Agent 示例（需要 API Key）
//   go run main.go interactive   - 交互式 Agent 对话（需要 API Key）
// =============================================================================

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "basic":
			basicToolDemo()
		case "infer":
			inferToolDemo()
		case "filesystem":
			filesystemDemo()
		case "agent":
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
	fmt.Println("Eino Tools 和文件系统访问示例程序")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  go run main.go basic         - 基础工具实现演示（无需 API Key）")
	fmt.Println("  go run main.go infer         - InferTool 泛型工具演示（无需 API Key）")
	fmt.Println("  go run main.go filesystem    - 文件系统工具演示（无需 API Key）")
	fmt.Println("  go run main.go agent         - 带工具的 Agent 示例（需要 API Key）")
	fmt.Println("  go run main.go interactive   - 交互式 Agent 对话（需要 API Key）")
	fmt.Println("")
	fmt.Println("环境变量:")
	fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥（agent 和 interactive 模式需要）")
}

// =============================================================================
// 示例 1：手动实现 InvokableTool 接口
// =============================================================================
//
// 这是最基础的工具实现方式：直接实现 InvokableTool 接口。
// 适合需要完全控制参数解析和执行逻辑的场景。

// WeatherTool 天气查询工具
// 实现了 InvokableTool 接口，可以被 Agent 调用
type WeatherTool struct{}

// Info 返回工具的元数据
// 这个方法告诉 LLM：这个工具叫什么、做什么、需要什么参数
func (w *WeatherTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_weather",
		Desc: "查询指定城市的当前天气信息，返回温度、天气状况和湿度",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"city": {
					Type:     schema.String,     // 参数类型：字符串
					Desc:     "城市名称，如 '北京'、'上海'、'广州'", // 参数描述
					Required: true,               // 必需参数
				},
			},
		),
	}, nil
}

// InvokableRun 执行工具
// argumentsInJSON 是 LLM 生成的 JSON 格式参数
// 返回值是工具执行结果的字符串
func (w *WeatherTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析 JSON 参数
	var params struct {
		City string `json:"city"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 模拟天气数据（实际应用中会调用天气 API）
	weatherData := map[string]map[string]string{
		"北京": {"temperature": "22°C", "weather": "晴", "humidity": "45%"},
		"上海": {"temperature": "25°C", "weather": "多云", "humidity": "65%"},
		"广州": {"temperature": "28°C", "weather": "阵雨", "humidity": "80%"},
	}

	data, exists := weatherData[params.City]
	if !exists {
		return fmt.Sprintf(`{"error": "未找到城市 '%s' 的天气数据"}`, params.City), nil
	}

	// 返回 JSON 格式的结果
	result := map[string]string{
		"city":        params.City,
		"temperature": data["temperature"],
		"weather":     data["weather"],
		"humidity":    data["humidity"],
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// CalculatorTool 计算器工具
// 支持基本的数学运算
type CalculatorTool struct{}

func (c *CalculatorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "calculator",
		Desc: "执行基本的数学计算（加减乘除）",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"a": {
					Type:     schema.Number,
					Desc:     "第一个操作数",
					Required: true,
				},
				"b": {
					Type:     schema.Number,
					Desc:     "第二个操作数",
					Required: true,
				},
				"operator": {
					Type:     schema.String,
					Desc:     "运算符：+、-、*、/",
					Required: true,
					Enum:     []string{"+", "-", "*", "/"},
				},
			},
		),
	}, nil
}

func (c *CalculatorTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var params struct {
		A        float64 `json:"a"`
		B        float64 `json:"b"`
		Operator string  `json:"operator"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	var result float64
	switch params.Operator {
	case "+":
		result = params.A + params.B
	case "-":
		result = params.A - params.B
	case "*":
		result = params.A * params.B
	case "/":
		if params.B == 0 {
			return `{"error": "除数不能为零"}`, nil
		}
		result = params.A / params.B
	default:
		return fmt.Sprintf(`{"error": "不支持的运算符: %s"}`, params.Operator), nil
	}

	return fmt.Sprintf(`{"expression": "%.2f %s %.2f", "result": %.2f}`, params.A, params.Operator, params.B, result), nil
}

// basicToolDemo 演示手动实现的工具
func basicToolDemo() {
	fmt.Println("=== 示例 1：手动实现 InvokableTool 接口 ===")
	fmt.Println("这个示例展示如何手动创建工具，无需 API Key。")
	fmt.Println()

	ctx := context.Background()

	// 创建工具实例
	weatherTool := &WeatherTool{}
	calculatorTool := &CalculatorTool{}

	// 打印工具信息
	printToolInfo(ctx, weatherTool)
	printToolInfo(ctx, calculatorTool)

	// 直接调用工具（模拟 LLM 的调用）
	fmt.Println("--- 模拟工具调用 ---")
	fmt.Println()

	// 调用天气工具
	fmt.Println("调用 get_weather 工具，参数: {\"city\": \"北京\"}")
	result, err := weatherTool.InvokableRun(ctx, `{"city": "北京"}`)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("结果: %s\n", result)
	}
	fmt.Println()

	// 调用计算器工具
	fmt.Println("调用 calculator 工具，参数: {\"a\": 10, \"b\": 3, \"operator\": \"+\"}")
	result, err = calculatorTool.InvokableRun(ctx, `{"a": 10, "b": 3, "operator": "+"}`)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("结果: %s\n", result)
	}
	fmt.Println()

	fmt.Println("调用 calculator 工具，参数: {\"a\": 10, \"b\": 0, \"operator\": \"/\"}")
	result, err = calculatorTool.InvokableRun(ctx, `{"a": 10, "b": 0, "operator": "/"}`)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("结果: %s\n", result)
	}
}

// printToolInfo 打印工具的元数据信息
func printToolInfo(ctx context.Context, t tool.InvokableTool) {
	info, err := t.Info(ctx)
	if err != nil {
		fmt.Printf("获取工具信息失败: %v\n", err)
		return
	}

	fmt.Printf("工具名称: %s\n", info.Name)
	fmt.Printf("工具描述: %s\n", info.Desc)
	fmt.Println("参数列表:")
	if info.ParamsOneOf != nil {
		schema, err := info.ParamsOneOf.ToJSONSchema()
		if err == nil && schema != nil {
			schemaJSON, _ := json.MarshalIndent(schema, "  ", "  ")
			fmt.Printf("  %s\n", string(schemaJSON))
		}
	}
	fmt.Println()
}

// =============================================================================
// 示例 2：使用 InferTool 快速创建工具
// =============================================================================
//
// InferTool 是 Eino 提供的泛型工具函数，可以：
// 1. 从 Go 结构体自动推导 JSON Schema
// 2. 自动处理 JSON 反序列化
// 3. 大大减少样板代码
//
// 推荐大多数场景使用这种方式。

// ReadFileInput 读取文件的参数结构体
// 结构体的 json tag 会成为参数名，description tag 会成为参数描述
type ReadFileInput struct {
	FilePath string `json:"file_path" description:"要读取的文件路径"`
	Limit    int    `json:"limit" description:"最多读取的行数，0 表示全部"`
}

// ListDirInput 列出目录的参数结构体
type ListDirInput struct {
	DirPath string `json:"dir_path" description:"要列出的目录路径"`
}

// readFileFunc 读取文件的函数
// 这个函数会被 InferTool 包装成 InvokableTool
func readFileFunc(ctx context.Context, input ReadFileInput) (string, error) {
	// 读取文件内容
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	text := string(content)

	// 如果指定了行数限制
	if input.Limit > 0 {
		lines := strings.Split(text, "\n")
		if len(lines) > input.Limit {
			lines = lines[:input.Limit]
			text = strings.Join(lines, "\n") + "\n... (已截断)"
		}
	}

	return text, nil
}

// listDirFunc 列出目录的函数
func listDirFunc(ctx context.Context, input ListDirInput) (string, error) {
	entries, err := os.ReadDir(input.DirPath)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("目录 %s 下的内容：\n", input.DirPath))
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("  [目录] %s\n", entry.Name()))
		} else {
			info, _ := entry.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			result.WriteString(fmt.Sprintf("  [文件] %s (%d bytes)\n", entry.Name(), size))
		}
	}

	return result.String(), nil
}

// inferToolDemo 演示使用 InferTool 创建工具
func inferToolDemo() {
	fmt.Println("=== 示例 2：使用 InferTool 快速创建工具 ===")
	fmt.Println("这个示例展示如何用泛型函数快速创建工具，无需 API Key。")
	fmt.Println()

	ctx := context.Background()

	// 使用 InferTool 创建工具（一行代码！）
	readFileTool, err := utils.InferTool(
		"read_file",           // 工具名称
		"读取指定路径的文件内容", // 工具描述
		readFileFunc,          // 工具函数
	)
	if err != nil {
		fmt.Printf("创建 read_file 工具失败: %v\n", err)
		return
	}

	listDirTool, err := utils.InferTool(
		"list_dir",
		"列出指定目录下的文件和子目录",
		listDirFunc,
	)
	if err != nil {
		fmt.Printf("创建 list_dir 工具失败: %v\n", err)
		return
	}

	// 打印工具信息
	printToolInfo(ctx, readFileTool)
	printToolInfo(ctx, listDirTool)

	// 创建一个临时文件用于测试
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "eino_test.txt")
	os.WriteFile(tmpFile, []byte("Hello, Eino!\n这是第 2 行\n这是第 3 行\n这是第 4 行\n"), 0644)
	defer os.Remove(tmpFile)

	// 调用工具
	fmt.Println("--- 模拟工具调用 ---")
	fmt.Println()

	// 调用 read_file 工具
	fmt.Printf("调用 read_file 工具，参数: {\"file_path\": \"%s\", \"limit\": 2}\n", tmpFile)
	result, err := readFileTool.InvokableRun(ctx, fmt.Sprintf(`{"file_path": "%s", "limit": 2}`, tmpFile))
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("结果:\n%s\n", result)
	}
	fmt.Println()

	// 调用 list_dir 工具
	fmt.Printf("调用 list_dir 工具，参数: {\"dir_path\": \"%s\"}\n", tmpDir)
	result, err = listDirTool.InvokableRun(ctx, fmt.Sprintf(`{"dir_path": "%s"}`, tmpDir))
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Printf("结果:\n%s\n", result)
	}
}

// =============================================================================
// 示例 3：使用 InMemoryBackend 实现文件系统工具
// =============================================================================
//
// Eino 提供了文件系统抽象层 filesystem.Backend，内置了 InMemoryBackend 实现。
// 这个示例展示如何使用 InMemoryBackend 创建一个虚拟文件系统。

// filesystemDemo 演示文件系统工具
func filesystemDemo() {
	fmt.Println("=== 示例 3：文件系统工具演示 ===")
	fmt.Println("这个示例展示如何使用 Eino 的文件系统后端，无需 API Key。")
	fmt.Println()

	ctx := context.Background()

	// 创建内存文件系统
	backend := filesystem.NewInMemoryBackend()

	// 写入一些测试文件
	fmt.Println("--- 初始化虚拟文件系统 ---")

	testFiles := map[string]string{
		"/project/main.go": `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}`,
		"/project/README.md": `# My Project

这是一个示例项目。

## 功能
- 功能 1
- 功能 2
- 功能 3`,
		"/project/config.json": `{
    "name": "my-project",
    "version": "1.0.0",
    "debug": true
}`,
		"/project/src/utils.go": `package src

// Add 两数相加
func Add(a, b int) int {
    return a + b
}

// Multiply 两数相乘
func Multiply(a, b int) int {
    return a * b
}`,
	}

	for path, content := range testFiles {
		err := backend.Write(ctx, &filesystem.WriteRequest{
			FilePath: path,
			Content:  content,
		})
		if err != nil {
			fmt.Printf("写入文件 %s 失败: %v\n", path, err)
		} else {
			fmt.Printf("  已创建: %s\n", path)
		}
	}
	fmt.Println()

	// 1. 列出目录
	fmt.Println("--- 1. 列出目录 /project ---")
	files, err := backend.LsInfo(ctx, &filesystem.LsInfoRequest{Path: "/project"})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		for _, f := range files {
			if f.IsDir {
				fmt.Printf("  [目录] %s\n", f.Path)
			} else {
				fmt.Printf("  [文件] %s (%d bytes)\n", f.Path, f.Size)
			}
		}
	}
	fmt.Println()

	// 2. 读取文件
	fmt.Println("--- 2. 读取文件 /project/main.go ---")
	content, err := backend.Read(ctx, &filesystem.ReadRequest{
		FilePath: "/project/main.go",
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println(content.Content)
	}

	// 3. 读取文件的前 3 行
	fmt.Println("--- 3. 读取文件 /project/README.md 的前 3 行 ---")
	content, err = backend.Read(ctx, &filesystem.ReadRequest{
		FilePath: "/project/README.md",
		Offset:   1,
		Limit:    3,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println(content.Content)
	}
	fmt.Println()

	// 4. 搜索文件内容
	fmt.Println("--- 4. 搜索包含 'func' 的代码 ---")
	matches, err := backend.GrepRaw(ctx, &filesystem.GrepRequest{
		Pattern: "func ",
		Path:    "/project",
		Glob:    "*.go",
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		for _, match := range matches {
			fmt.Printf("  %s:%d: %s\n", match.Path, match.Line, match.Content)
		}
	}
	fmt.Println()

	// 5. Glob 模式匹配
	fmt.Println("--- 5. 查找所有 .go 文件 ---")
	goFiles, err := backend.GlobInfo(ctx, &filesystem.GlobInfoRequest{
		Pattern: "**/*.go",
		Path:    "/project",
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		for _, f := range goFiles {
			fmt.Printf("  %s (%d bytes)\n", f.Path, f.Size)
		}
	}
	fmt.Println()

	// 6. 编辑文件
	fmt.Println("--- 6. 编辑文件 /project/main.go ---")
	err = backend.Edit(ctx, &filesystem.EditRequest{
		FilePath:  "/project/main.go",
		OldString: `"Hello, World!"`,
		NewString: `"Hello, Eino!"`,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("  已将 'Hello, World!' 替换为 'Hello, Eino!'")

		// 验证修改
		content, _ := backend.Read(ctx, &filesystem.ReadRequest{FilePath: "/project/main.go"})
		fmt.Println("  修改后的内容:")
		fmt.Println(content.Content)
	}

	// 7. 创建文件系统工具
	fmt.Println("--- 7. 将文件系统后端封装为工具 ---")
	fmt.Println("  可以将 InMemoryBackend 封装为 InvokableTool，让 Agent 使用虚拟文件系统。")
	fmt.Println("  具体实现请参考 main.go 中的 FsReadTool 和 FsWriteTool。")
}

// =============================================================================
// 示例 4：将工具集成到 Agent
// =============================================================================
//
// 这个示例展示如何将自定义工具注册到 ChatModelAgent，
// 让 LLM 自动决定何时调用哪个工具。
//
// 注意：这个示例需要设置 OPENAI_API_KEY 环境变量。

// agentWithTools 演示带工具的 Agent
func agentWithTools() {
	ctx := context.Background()

	// 检查 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("错误: 请设置 OPENAI_API_KEY 环境变量")
		fmt.Println("  export OPENAI_API_KEY=\"your-api-key-here\"")
		os.Exit(1)
	}

	fmt.Println("=== 示例 4：带工具的 Agent ===")
	fmt.Println("这个示例展示如何将工具集成到 Agent 中。")
	fmt.Println()

	// 1. 创建 ChatModel
	chatModel, err := createChatModel(ctx, apiKey)
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 创建工具列表
	tools := []tool.BaseTool{
		&WeatherTool{},
		&CalculatorTool{},
	}

	// 3. 创建带工具的 Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "tool_agent",
		Description: "一个拥有天气查询和计算器工具的智能助手",
		Instruction: `你是一个智能助手，拥有以下工具：
1. get_weather - 查询城市天气
2. calculator - 执行数学计算

请根据用户的问题，自动选择合适的工具来回答。如果不需要工具，直接回答即可。`,
		Model: chatModel.(model.BaseChatModel),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		fmt.Printf("创建 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 4. 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	queries := []string{
		"北京今天天气怎么样？",
		"请计算 123 * 456 + 789",
		"你好，请介绍一下你自己。",
	}

	for _, query := range queries {
		fmt.Printf("=== 查询: %s ===\n", query)
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
		fmt.Println("  export OPENAI_API_KEY=\"your-api-key-here\"")
		os.Exit(1)
	}

	fmt.Println("=== 示例 5：交互式 Agent 对话 ===")
	fmt.Println("这个示例展示如何与带工具的 Agent 进行交互式对话。")
	fmt.Println()

	// 创建 ChatModel
	chatModel, err := createChatModel(ctx, apiKey)
	if err != nil {
		fmt.Printf("创建 ChatModel 失败: %v\n", err)
		os.Exit(1)
	}

	// 创建工具列表
	tools := []tool.BaseTool{
		&WeatherTool{},
		&CalculatorTool{},
	}

	// 创建 Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "interactive_agent",
		Description: "一个拥有天气查询和计算器工具的智能助手",
		Instruction: `你是一个智能助手，拥有以下工具：
1. get_weather - 查询城市天气
2. calculator - 执行数学计算

请根据用户的问题，自动选择合适的工具来回答。如果不需要工具，直接回答即可。
请用简洁明了的语言回答。`,
		Model: chatModel.(model.BaseChatModel),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		fmt.Printf("创建 Agent 失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	// 交互式对话
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("=== AI Agent 对话系统 ===")
	fmt.Println("命令: 'quit' 退出, 'clear' 清空历史")
	fmt.Println("示例: '北京天气怎么样？', '计算 100 * 200'")
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
			runner = adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
			fmt.Println("✓ 对话历史已清空")
			continue
		case "":
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

// =============================================================================
// 辅助函数
// =============================================================================

// createChatModel 创建 ChatModel 实例
// 这里使用 OpenAI 作为示例，你可以替换为其他 LLM 提供商
func createChatModel(ctx context.Context, apiKey string) (model.BaseChatModel, error) {
	// 注意：这里使用 eino-ext 中的 OpenAI 实现
	// 如果你使用其他 LLM，需要导入对应的包
	//
	// 例如使用 Anthropic Claude：
	// import "github.com/cloudwego/eino-ext/components/model/claude"
	// return claude.NewChatModel(ctx, &claude.ChatModelConfig{
	//     Model:  "claude-3-opus",
	//     APIKey: os.Getenv("ANTHROPIC_API_KEY"),
	// })

	// 由于 eino-ext 的 OpenAI 实现可能有不同的 API，
	// 这里返回一个模拟的 ChatModel 用于演示
	return &mockChatModel{}, nil
}

// mockChatModel 模拟的 ChatModel，用于不需要真实 API 的演示
type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// 模拟 LLM 响应
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "这是一个模拟响应。在实际应用中，这里会是真实的 LLM 回复。",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("stream not implemented in mock")
}

// =============================================================================
// 工具使用技巧和最佳实践
// =============================================================================
//
// 1. 工具命名规范：
//    - 使用 snake_case：如 read_file、get_weather
//    - 动词开头：如 get_、set_、create_、delete_
//    - 简洁明了：让 LLM 一眼就能理解用途
//
// 2. 参数描述：
//    - 详细说明每个参数的含义
//    - 提供示例值
//    - 标明是否必需
//
// 3. 错误处理：
//    - 返回有意义的错误信息
//    - LLM 可以根据错误信息决定重试或告知用户
//    - 避免 panic，优雅处理错误
//
// 4. 安全性：
//    - 验证输入参数
//    - 限制文件访问范围
//    - 设置执行超时
//    - 记录工具调用日志
//
// 5. 性能：
//    - 耗时操作使用 context 控制超时
//    - 大文件使用流式读取
//    - 缓存频繁访问的数据

// init 初始化函数，用于设置日志等
func init() {
	// 设置时间格式
	time.Local = time.FixedZone("CST", 8*3600)
}
