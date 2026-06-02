// ============================================================================
// 第 9 章：Skill Middleware -- 技能中间件
// ============================================================================
//
// 本文件演示 Eino 框架的 Skill 中间件和 ToolSearch 中间件。
//
// 由于 Eino 的 Skill/ToolSearch 中间件需要完整的 Agent 运行环境（需要 API Key 等），
// 本示例采用"先理解原理，再看实际用法"的方式：
//
//   Part 1: 用纯 Go 模拟 Skill 的渐进式披露和 Backend 模式（不需要 API Key）
//   Part 2: 用纯 Go 模拟 ToolSearch 的动态工具发现（不需要 API Key）
//   Part 3: 展示如何在真实 Eino Agent 中使用这些中间件（需要 API Key）
//
// 运行方式：
//   go run main.go demo          - 运行完整演示（模拟，不需要 API Key）
//   go run main.go skill         - 运行 Skill 渐进式披露演示
//   go run main.go toolsearch    - 运行 ToolSearch 动态发现演示
//   go run main.go backend       - 运行自定义 Backend 演示
//   go run main.go combined      - 运行组合演示
//   go run main.go eino          - 运行真实 Eino Agent 示例（需要 API Key）
//
// ============================================================================

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// Part 1: 模拟 Skill 系统（帮助理解渐进式披露）
// ============================================================================
//
// 这部分用纯 Go 代码模拟 Eino 的 Skill 中间件。
// 不需要任何外部依赖，帮助你理解 Skill 的核心思想。
//
// ============================================================================

// --------------------------------------------------------------------------
// 1.1 核心类型定义
// --------------------------------------------------------------------------

// FrontMatter 技能的元数据
// 在真实 Eino 中，对应 skill.FrontMatter 结构体
// 这些元数据在"发现"阶段加载，用于判断是否需要使用该技能
type FrontMatter struct {
	Name        string // 技能的唯一标识符，如 "pdf-processing"
	Description string // 技能描述，用于 Agent 判断是否使用
	Context     string // 上下文模式：inline、fork、fork_with_context
	Agent       string // 指定使用的 Agent 名称（可选）
	Model       string // 指定使用的模型名称（可选）
}

// Skill 完整的技能结构
// 在真实 Eino 中，对应 skill.Skill 结构体
// 包含元数据和完整的执行指令
type Skill struct {
	FrontMatter               // 嵌入元数据
	Content       string      // SKILL.md 的正文内容（执行指令）
	BaseDirectory string      // 技能目录的绝对路径
}

// Backend 技能存储后端的接口
// 在真实 Eino 中，对应 skill.Backend 接口
// 这个接口解耦了技能的存储和使用，支持多种实现
type Backend interface {
	// List 列出所有可用技能的元数据（用于发现阶段）
	// 只返回 FrontMatter，不加载完整内容，节省上下文空间
	List(ctx context.Context) ([]FrontMatter, error)

	// Get 获取指定技能的完整内容（用于激活阶段）
	// 返回完整的 Skill，包含执行指令
	Get(ctx context.Context, name string) (Skill, error)
}

// --------------------------------------------------------------------------
// 1.2 内存 Backend 实现
// --------------------------------------------------------------------------

// MemoryBackend 基于内存的技能后端
// 适用于测试场景或动态生成技能的场景
// 在生产环境中，通常使用文件系统后端或远程服务后端
type MemoryBackend struct {
	mu     sync.RWMutex       // 读写锁，保证并发安全
	skills map[string]Skill   // 技能存储，key 是技能名称
}

// NewMemoryBackend 创建一个新的内存后端
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		skills: make(map[string]Skill),
	}
}

// Register 注册一个技能到后端
// 在实际应用中，这可能在启动时从文件系统加载
func (b *MemoryBackend) Register(s Skill) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.skills[s.Name] = s
	fmt.Printf("  [Backend] 注册技能: %s - %s\n", s.Name, s.Description)
}

// List 列出所有技能的元数据
// 这是发现阶段的核心方法，只返回轻量级的元数据
func (b *MemoryBackend) List(ctx context.Context) ([]FrontMatter, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]FrontMatter, 0, len(b.skills))
	for _, s := range b.skills {
		result = append(result, s.FrontMatter)
	}
	return result, nil
}

// Get 获取指定技能的完整内容
// 这是激活阶段的核心方法，返回完整的 Skill
func (b *MemoryBackend) Get(ctx context.Context, name string) (Skill, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	s, ok := b.skills[name]
	if !ok {
		return Skill{}, fmt.Errorf("skill not found: %s", name)
	}
	return s, nil
}

// --------------------------------------------------------------------------
// 1.3 文件系统 Backend 模拟
// --------------------------------------------------------------------------

// FilesystemBackend 模拟文件系统后端
// 在真实 Eino 中，使用 skill.NewBackendFromFilesystem 创建
type FilesystemBackend struct {
	baseDir string // 技能目录的根路径
}

// NewFilesystemBackend 创建文件系统后端
func NewFilesystemBackend(baseDir string) *FilesystemBackend {
	return &FilesystemBackend{baseDir: baseDir}
}

// List 扫描目录，列出所有技能的元数据
// 在真实实现中，会扫描子目录中的 SKILL.md 文件
func (b *FilesystemBackend) List(ctx context.Context) ([]FrontMatter, error) {
	// 模拟扫描目录
	// 真实实现会使用 os.ReadDir 读取子目录
	fmt.Printf("  [FilesystemBackend] 扫描目录: %s\n", b.baseDir)

	// 返回模拟的技能列表
	return []FrontMatter{
		{Name: "pdf-processing", Description: "处理 PDF 文件，包括文本提取、页面分割、合并等操作", Context: "inline"},
		{Name: "data-analysis", Description: "数据分析和可视化，支持 CSV、Excel 等格式", Context: "fork_with_context"},
		{Name: "code-review", Description: "代码审查，检测常见问题和最佳实践", Context: "fork"},
	}, nil
}

// Get 从文件系统读取技能内容
// 在真实实现中，会读取 SKILL.md 文件并解析 YAML 前置元数据
func (b *FilesystemBackend) Get(ctx context.Context, name string) (Skill, error) {
	fmt.Printf("  [FilesystemBackend] 读取技能: %s/SKILL.md\n", name)

	// 模拟读取文件
	// 真实实现会使用 os.ReadFile 读取 SKILL.md
	skills := map[string]Skill{
		"pdf-processing": {
			FrontMatter: FrontMatter{
				Name:        "pdf-processing",
				Description: "处理 PDF 文件，包括文本提取、页面分割、合并等操作",
				Context:     "inline",
			},
			Content: `# PDF 处理技能

## 功能说明
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
- scripts/merge_pdfs.py - 合并文件`,
			BaseDirectory: b.baseDir + "/pdf-processing",
		},
		"data-analysis": {
			FrontMatter: FrontMatter{
				Name:        "data-analysis",
				Description: "数据分析和可视化，支持 CSV、Excel 等格式",
				Context:     "fork_with_context",
			},
			Content: `# 数据分析技能

## 功能说明
本技能提供数据分析和可视化能力，包括：
- 读取和解析 CSV、Excel 文件
- 数据清洗和转换
- 统计分析和聚合
- 生成图表和可视化

## 使用方法
1. 使用 load_data 脚本加载数据
2. 使用 analyze 脚本进行分析
3. 使用 visualize 脚本生成图表`,
			BaseDirectory: b.baseDir + "/data-analysis",
		},
		"code-review": {
			FrontMatter: FrontMatter{
				Name:        "code-review",
				Description: "代码审查，检测常见问题和最佳实践",
				Context:     "fork",
			},
			Content: `# 代码审查技能

## 功能说明
本技能提供代码审查能力，包括：
- 检测未处理的错误
- 发现硬编码的值
- 检查代码风格
- 识别潜在的安全问题

## 审查规则
1. 错误处理：所有 error 都必须被处理
2. 常量管理：避免硬编码，使用常量或配置
3. 并发安全：检查共享资源的访问
4. 输入验证：所有外部输入都需要验证`,
			BaseDirectory: b.baseDir + "/code-review",
		},
	}

	skill, ok := skills[name]
	if !ok {
		return Skill{}, fmt.Errorf("skill not found: %s", name)
	}
	return skill, nil
}

// --------------------------------------------------------------------------
// 1.4 Skill 中间件模拟
// --------------------------------------------------------------------------

// SkillMiddleware 模拟 Eino 的 Skill 中间件
// 在真实 Eino 中，使用 skill.NewMiddleware 创建
type SkillMiddleware struct {
	backend     Backend     // 技能存储后端
	toolName    string      // 技能工具的名称（默认为 "skill"）
	loadedSkills map[string]Skill // 已加载的技能缓存
}

// NewSkillMiddleware 创建 Skill 中间件
func NewSkillMiddleware(backend Backend) *SkillMiddleware {
	return &SkillMiddleware{
		backend:      backend,
		toolName:     "skill",
		loadedSkills: make(map[string]Skill),
	}
}

// DiscoverSkills 发现阶段：加载所有技能的元数据
// 这是渐进式披露的第一步，只加载轻量级的元数据
func (m *SkillMiddleware) DiscoverSkills(ctx context.Context) ([]FrontMatter, error) {
	fmt.Println("\n[Skill 中间件] === 阶段 1: 发现（Discovery）===")
	fmt.Println("[Skill 中间件] 加载技能元数据...")

	skills, err := m.backend.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("列出技能失败: %w", err)
	}

	fmt.Printf("[Skill 中间件] 发现 %d 个可用技能:\n", len(skills))
	for _, s := range skills {
		fmt.Printf("  - %s: %s\n", s.Name, s.Description)
	}

	return skills, nil
}

// ActivateSkill 激活阶段：加载指定技能的完整内容
// 这是渐进式披露的第二步，只在需要时加载完整内容
func (m *SkillMiddleware) ActivateSkill(ctx context.Context, name string) (Skill, error) {
	fmt.Printf("\n[Skill 中间件] === 阶段 2: 激活（Activation）===\n")
	fmt.Printf("[Skill 中间件] 激活技能: %s\n", name)

	// 检查是否已经加载
	if s, ok := m.loadedSkills[name]; ok {
		fmt.Println("[Skill 中间件] 技能已缓存，直接返回")
		return s, nil
	}

	// 从后端加载完整内容
	s, err := m.backend.Get(ctx, name)
	if err != nil {
		return Skill{}, fmt.Errorf("获取技能失败: %w", err)
	}

	// 缓存已加载的技能
	m.loadedSkills[name] = s

	fmt.Printf("[Skill 中间件] 技能已激活，内容长度: %d 字符\n", len(s.Content))
	return s, nil
}

// ExecuteSkill 执行阶段：按照技能的指令执行任务
// 这是渐进式披露的第三步，执行实际的任务
func (m *SkillMiddleware) ExecuteSkill(ctx context.Context, skill Skill, args string) (string, error) {
	fmt.Printf("\n[Skill 中间件] === 阶段 3: 执行（Execution）===\n")
	fmt.Printf("[Skill 中间件] 执行技能: %s\n", skill.Name)
	fmt.Printf("[Skill 中间件] 执行模式: %s\n", skill.Context)
	fmt.Printf("[Skill 中间件] 输入参数: %s\n", args)

	// 根据上下文模式选择执行方式
	switch skill.Context {
	case "inline":
		fmt.Println("[Skill 中间件] 在当前上下文中执行（Inline 模式）")
		// 模拟执行
		return fmt.Sprintf("[技能 %s 执行结果] 处理完成，输入: %s", skill.Name, args), nil

	case "fork":
		fmt.Println("[Skill 中间件] 创建新 Agent 执行（Fork 模式，无历史）")
		// 模拟创建新 Agent
		return fmt.Sprintf("[技能 %s 执行结果] 独立执行完成，输入: %s", skill.Name, args), nil

	case "fork_with_context":
		fmt.Println("[Skill 中间件] 创建新 Agent 执行（ForkWithContext 模式，复制历史）")
		// 模拟创建新 Agent 并复制历史
		return fmt.Sprintf("[技能 %s 执行结果] 带上下文执行完成，输入: %s", skill.Name, args), nil

	default:
		return "", fmt.Errorf("未知的上下文模式: %s", skill.Context)
	}
}

// --------------------------------------------------------------------------
// 1.5 模拟 Agent 使用 Skill
// --------------------------------------------------------------------------

// SimulateAgentWithSkill 模拟一个使用 Skill 中间件的 Agent
func SimulateAgentWithSkill() {
	fmt.Println("=== Skill 渐进式披露演示 ===")
	fmt.Println("本演示展示 Skill 如何分三个阶段加载和执行\n")

	ctx := context.Background()

	// 1. 创建 Backend（模拟从文件系统加载）
	fmt.Println("--- 初始化 ---")
	backend := NewFilesystemBackend("./skills")

	// 2. 创建 Skill 中间件
	middleware := NewSkillMiddleware(backend)

	// 3. 发现阶段：加载所有技能的元数据
	availableSkills, err := middleware.DiscoverSkills(ctx)
	if err != nil {
		fmt.Printf("发现技能失败: %v\n", err)
		return
	}

	// 4. 模拟用户请求
	fmt.Println("\n--- 用户请求 ---")
	fmt.Println("用户: 帮我分析这个 CSV 文件的数据")

	// 5. Agent 决定使用 data-analysis 技能
	fmt.Println("\n--- Agent 决策 ---")
	fmt.Println("Agent: 我需要使用 data-analysis 技能来处理这个请求")

	// 6. 激活阶段：加载 data-analysis 技能的完整内容
	var targetSkill Skill
	for _, s := range availableSkills {
		if s.Name == "data-analysis" {
			targetSkill, err = middleware.ActivateSkill(ctx, s.Name)
			if err != nil {
				fmt.Printf("激活技能失败: %v\n", err)
				return
			}
			break
		}
	}

	// 7. 执行阶段：按照技能的指令执行任务
	result, err := middleware.ExecuteSkill(ctx, targetSkill, "sales_data.csv")
	if err != nil {
		fmt.Printf("执行技能失败: %v\n", err)
		return
	}

	// 8. 返回结果
	fmt.Println("\n--- 执行结果 ---")
	fmt.Printf("结果: %s\n", result)
}

// ============================================================================
// Part 2: 模拟 ToolSearch 系统（帮助理解动态工具发现）
// ============================================================================
//
// 这部分用纯 Go 代码模拟 Eino 的 ToolSearch 中间件。
// 不需要任何外部依赖，帮助你理解动态工具发现的核心思想。
//
// ============================================================================

// --------------------------------------------------------------------------
// 2.1 核心类型定义
// --------------------------------------------------------------------------

// ToolInfo 工具的元信息
// 在真实 Eino 中，对应 schema.ToolInfo
type ToolInfo struct {
	Name        string            // 工具名称
	Description string            // 工具描述
	Parameters  map[string]string // 参数定义
}

// DynamicTool 动态工具
// 在真实 Eino 中，实现 tool.BaseTool 接口
type DynamicTool struct {
	Info    ToolInfo                    // 工具元信息
	Execute func(args string) string   // 工具执行函数
}

// ToolSearchResult 工具搜索结果
type ToolSearchResult struct {
	Tools []ToolInfo // 匹配的工具列表
	Score int        // 相关性分数
}

// --------------------------------------------------------------------------
// 2.2 ToolSearch 中间件模拟
// --------------------------------------------------------------------------

// ToolSearchMiddleware 模拟 Eino 的 ToolSearch 中间件
// 在真实 Eino 中，使用 toolsearch.New 创建
type ToolSearchMiddleware struct {
	dynamicTools []DynamicTool     // 动态工具列表
	visibleTools map[string]bool   // 当前可见的工具
	toolSearchFn func(query string) []ToolSearchResult // 搜索函数
}

// NewToolSearchMiddleware 创建 ToolSearch 中间件
func NewToolSearchMiddleware(tools []DynamicTool) *ToolSearchMiddleware {
	m := &ToolSearchMiddleware{
		dynamicTools: tools,
		visibleTools: make(map[string]bool),
	}

	// 初始化时，所有动态工具都不可见
	// 只有通过 tool_search 搜索到的工具才会变为可见
	for _, t := range tools {
		m.visibleTools[t.Info.Name] = false
	}

	// 设置搜索函数
	m.toolSearchFn = m.searchTools

	return m
}

// searchTools 搜索工具
// 实现关键词匹配和评分算法
func (m *ToolSearchMiddleware) searchTools(query string) []ToolSearchResult {
	fmt.Printf("\n[ToolSearch] 执行搜索: %q\n", query)

	var results []ToolSearchResult

	// 解析查询类型
	if strings.HasPrefix(query, "select:") {
		// 直接选择模式
		names := strings.Split(strings.TrimPrefix(query, "select:"), ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			for _, t := range m.dynamicTools {
				if t.Info.Name == name {
					results = append(results, ToolSearchResult{
						Tools: []ToolInfo{t.Info},
						Score: 100, // 直接选择，最高分
					})
				}
			}
		}
	} else {
		// 关键词搜索模式
		keywords := strings.Fields(strings.ToLower(query))

		for _, t := range m.dynamicTools {
			score := calculateScore(t.Info, keywords)
			if score > 0 {
				results = append(results, ToolSearchResult{
					Tools: []ToolInfo{t.Info},
					Score: score,
				})
			}
		}
	}

	// 按分数排序
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// calculateScore 计算工具与关键词的匹配分数
// 实现 Eino 的评分算法
func calculateScore(info ToolInfo, keywords []string) int {
	totalScore := 0
	nameParts := splitToolName(info.Name)
	descLower := strings.ToLower(info.Description)

	for _, keyword := range keywords {
		keywordScore := 0

		// 规则 1: 工具名拆分部分精确匹配关键词 → 10 分
		for _, part := range nameParts {
			if part == keyword {
				keywordScore = max(keywordScore, 10)
			}
		}

		// 规则 2: 工具名拆分部分包含关键词 → 5 分
		for _, part := range nameParts {
			if strings.Contains(part, keyword) {
				keywordScore = max(keywordScore, 5)
			}
		}

		// 规则 3: 完整工具名包含关键词 → 3 分
		if strings.Contains(strings.ToLower(info.Name), keyword) {
			keywordScore = max(keywordScore, 3)
		}

		// 规则 4: 工具描述包含关键词 → 2 分
		if strings.Contains(descLower, keyword) {
			keywordScore = max(keywordScore, 2)
		}

		totalScore += keywordScore
	}

	return totalScore
}

// splitToolName 拆分工具名称
// 按下划线、双下划线和驼峰边界拆分
func splitToolName(name string) []string {
	// 先按双下划线拆分（MCP 分隔符）
	parts := strings.Split(name, "__")

	var result []string
	for _, part := range parts {
		// 再按下划线拆分
		subParts := strings.Split(part, "_")
		result = append(result, subParts...)
	}

	// 转换为小写
	for i := range result {
		result[i] = strings.ToLower(result[i])
	}

	return result
}

// GetVisibleTools 获取当前可见的工具列表
func (m *ToolSearchMiddleware) GetVisibleTools() []ToolInfo {
	var tools []ToolInfo
	for _, t := range m.dynamicTools {
		if m.visibleTools[t.Info.Name] {
			tools = append(tools, t.Info)
		}
	}
	return tools
}

// CallToolSearch 调用 tool_search 工具
// 模拟 Agent 调用 tool_search 的过程
func (m *ToolSearchMiddleware) CallToolSearch(query string, maxResults int) []ToolInfo {
	fmt.Printf("\n[ToolSearch] Agent 调用 tool_search(query=%q, max_results=%d)\n", query, maxResults)

	// 执行搜索
	results := m.toolSearchFn(query)

	// 收集匹配的工具
	var matchedTools []ToolInfo
	for _, r := range results {
		matchedTools = append(matchedTools, r.Tools...)
		if len(matchedTools) >= maxResults {
			break
		}
	}

	// 将匹配的工具标记为可见
	for _, t := range matchedTools {
		m.visibleTools[t.Name] = true
		fmt.Printf("[ToolSearch] 工具已变为可见: %s\n", t.Name)
	}

	return matchedTools


	return matchedTools
}

// --------------------------------------------------------------------------
// 2.3 模拟 Agent 使用 ToolSearch
// --------------------------------------------------------------------------

// SimulateAgentWithToolSearch 模拟一个使用 ToolSearch 中间件的 Agent
func SimulateAgentWithToolSearch() {
	fmt.Println("=== ToolSearch 动态工具发现演示 ===")
	fmt.Println("本演示展示 ToolSearch 如何让 Agent 动态发现工具\n")

	// 1. 创建动态工具（模拟 100+ 个工具的场景）
	dynamicTools := []DynamicTool{
		{
			Info: ToolInfo{
				Name:        "weather_forecast",
				Description: "查询指定城市的天气预报，包括温度、湿度、风力等信息",
				Parameters:  map[string]string{"city": "城市名称"},
			},
			Execute: func(args string) string {
				return "北京今天晴，25°C，微风"
			},
		},
		{
			Info: ToolInfo{
				Name:        "weather_alert",
				Description: "查询天气预警信息，包括暴雨、高温、台风等预警",
				Parameters:  map[string]string{"city": "城市名称"},
			},
			Execute: func(args string) string {
				return "无预警信息"
			},
		},
		{
			Info: ToolInfo{
				Name:        "stock_price",
				Description: "查询股票实时价格，支持 A 股、港股、美股",
				Parameters:  map[string]string{"symbol": "股票代码"},
			},
			Execute: func(args string) string {
				return "AAPL: $150.25 (+1.5%)"
			},
		},
		{
			Info: ToolInfo{
				Name:        "currency_convert",
				Description: "货币汇率转换，支持全球主要货币",
				Parameters:  map[string]string{"from": "源货币", "to": "目标货币", "amount": "金额"},
			},
			Execute: func(args string) string {
				return "100 USD = 720 CNY"
			},
		},
		{
			Info: ToolInfo{
				Name:        "mcp__slack__send_message",
				Description: "Send a message to a Slack channel",
				Parameters:  map[string]string{"channel": "频道名称", "message": "消息内容"},
			},
			Execute: func(args string) string {
				return "消息已发送"
			},
		},
		{
			Info: ToolInfo{
				Name:        "mcp__github__create_issue",
				Description: "Create a new issue on GitHub repository",
				Parameters:  map[string]string{"repo": "仓库名称", "title": "标题", "body": "内容"},
			},
			Execute: func(args string) string {
				return "Issue #123 已创建"
			},
		},
	}

	// 2. 创建 ToolSearch 中间件
	middleware := NewToolSearchMiddleware(dynamicTools)

	// 3. 初始状态：所有动态工具都不可见
	fmt.Println("--- 初始状态 ---")
	fmt.Println("Agent 可见的动态工具: 无")
	fmt.Println("(所有动态工具都被隐藏，等待通过 tool_search 发现)")

	// 4. 模拟用户请求
	fmt.Println("\n--- 用户请求 ---")
	fmt.Println("用户: 北京今天天气怎么样？")

	// 5. Agent 决策：需要搜索天气相关工具
	fmt.Println("\n--- Agent 决策 ---")
	fmt.Println("Agent: 我需要搜索天气相关的工具")

	// 6. 调用 tool_search
	matchedTools := middleware.CallToolSearch("weather forecast", 5)

	// 7. 显示搜索结果
	fmt.Println("\n--- 搜索结果 ---")
	fmt.Printf("找到 %d 个匹配的工具:\n", len(matchedTools))
	for _, t := range matchedTools {
		fmt.Printf("  - %s: %s\n", t.Name, t.Description)
	}

	// 8. 显示当前可见的工具
	fmt.Println("\n--- 当前可见的动态工具 ---")
	visibleTools := middleware.GetVisibleTools()
	for _, t := range visibleTools {
		fmt.Printf("  - %s\n", t.Name)
	}

	// 9. Agent 使用发现的工具
	fmt.Println("\n--- Agent 执行 ---")
	fmt.Println("Agent: 调用 weather_forecast(city=北京)")
	for _, t := range dynamicTools {
		if t.Info.Name == "weather_forecast" {
			result := t.Execute("北京")
			fmt.Printf("结果: %s\n", result)
		}
	}

	// 10. 第二轮对话：用户问另一个问题
	fmt.Println("\n--- 第二轮对话 ---")
	fmt.Println("用户: 帮我查一下 AAPL 的股价")

	// 11. Agent 再次搜索
	fmt.Println("\n--- Agent 决策 ---")
	fmt.Println("Agent: 我需要搜索股票相关的工具")
	matchedTools = middleware.CallToolSearch("stock price", 5)

	fmt.Println("\n--- 搜索结果 ---")
	fmt.Printf("找到 %d 个匹配的工具:\n", len(matchedTools))
	for _, t := range matchedTools {
		fmt.Printf("  - %s: %s\n", t.Name, t.Description)
	}

	// 12. 显示最终可见的工具
	fmt.Println("\n--- 最终可见的动态工具 ---")
	visibleTools = middleware.GetVisibleTools()
	for _, t := range visibleTools {
		fmt.Printf("  - %s\n", t.Name)
	}
}

// ============================================================================
// Part 3: 自定义 Backend 演示
// ============================================================================

// --------------------------------------------------------------------------
// 3.1 带缓存的 Backend
// --------------------------------------------------------------------------

// CachedBackend 带缓存的技能后端
// 在第一次调用后缓存结果，避免重复加载
type CachedBackend struct {
	inner       Backend       // 内部后端
	cache       *SkillCache   // 缓存
	cacheTTL    time.Duration // 缓存过期时间
}

// SkillCache 技能缓存
type SkillCache struct {
	mu        sync.RWMutex
	skills    map[string]Skill
	frontMatters []FrontMatter
	lastUpdate  time.Time
	ttl         time.Duration
}

// NewSkillCache 创建技能缓存
func NewSkillCache(ttl time.Duration) *SkillCache {
	return &SkillCache{
		skills: make(map[string]Skill),
		ttl:    ttl,
	}
}

// IsExpired 检查缓存是否过期
func (c *SkillCache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Since(c.lastUpdate) > c.ttl
}

// SetFrontMatters 设置元数据缓存
func (c *SkillCache) SetFrontMatters(fms []FrontMatter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frontMatters = fms
	c.lastUpdate = time.Now()
}

// GetFrontMatters 获取元数据缓存
func (c *SkillCache) GetFrontMatters() []FrontMatter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frontMatters
}

// SetSkill 设置技能缓存
func (c *SkillCache) SetSkill(name string, s Skill) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skills[name] = s
}

// GetSkill 获取技能缓存
func (c *SkillCache) GetSkill(name string) (Skill, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.skills[name]
	return s, ok
}

// NewCachedBackend 创建带缓存的后端
func NewCachedBackend(inner Backend, ttl time.Duration) *CachedBackend {
	return &CachedBackend{
		inner:    inner,
		cache:    NewSkillCache(ttl),
		cacheTTL: ttl,
	}
}

// List 列出所有技能（带缓存）
func (b *CachedBackend) List(ctx context.Context) ([]FrontMatter, error) {
	// 检查缓存是否有效
	if !b.cache.IsExpired() {
		fmt.Println("  [CachedBackend] 使用缓存的元数据")
		return b.cache.GetFrontMatters(), nil
	}

	// 缓存过期，从内部后端加载
	fmt.Println("  [CachedBackend] 缓存过期，重新加载")
	fms, err := b.inner.List(ctx)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	b.cache.SetFrontMatters(fms)
	return fms, nil
}

// Get 获取技能（带缓存）
func (b *CachedBackend) Get(ctx context.Context, name string) (Skill, error) {
	// 检查缓存
	if s, ok := b.cache.GetSkill(name); ok {
		fmt.Printf("  [CachedBackend] 使用缓存的技能: %s\n", name)
		return s, nil
	}

	// 缓存未命中，从内部后端加载
	fmt.Printf("  [CachedBackend] 缓存未命中，加载技能: %s\n", name)
	s, err := b.inner.Get(ctx, name)
	if err != nil {
		return Skill{}, err
	}

	// 更新缓存
	b.cache.SetSkill(name, s)
	return s, nil
}

// --------------------------------------------------------------------------
// 3.2 演示缓存 Backend
// --------------------------------------------------------------------------

// SimulateCachedBackend 演示带缓存的 Backend
func SimulateCachedBackend() {
	fmt.Println("=== 自定义 Backend（带缓存）演示 ===")
	fmt.Println("本演示展示如何实现带缓存的 Backend\n")

	ctx := context.Background()

	// 1. 创建内部后端
	innerBackend := NewMemoryBackend()

	// 2. 注册一些技能
	fmt.Println("--- 注册技能 ---")
	innerBackend.Register(Skill{
		FrontMatter: FrontMatter{
			Name:        "greeting",
			Description: "生成个性化问候语",
			Context:     "inline",
		},
		Content: "根据用户需求生成个性化的问候语。",
	})

	innerBackend.Register(Skill{
		FrontMatter: FrontMatter{
			Name:        "translation",
			Description: "多语言翻译服务",
			Context:     "inline",
		},
		Content: "提供高质量的多语言翻译。",
	})

	// 3. 创建带缓存的后端
	fmt.Println("\n--- 创建缓存 Backend ---")
	cachedBackend := NewCachedBackend(innerBackend, 5*time.Second)

	// 4. 第一次调用（缓存未命中）
	fmt.Println("\n--- 第一次调用（缓存未命中）---")
	fms, _ := cachedBackend.List(ctx)
	fmt.Printf("技能数量: %d\n", len(fms))

	// 5. 第二次调用（缓存命中）
	fmt.Println("\n--- 第二次调用（缓存命中）---")
	fms, _ = cachedBackend.List(ctx)
	fmt.Printf("技能数量: %d\n", len(fms))

	// 6. 获取技能（缓存未命中）
	fmt.Println("\n--- 获取技能（缓存未命中）---")
	s, _ := cachedBackend.Get(ctx, "greeting")
	fmt.Printf("技能: %s\n", s.Name)

	// 7. 再次获取（缓存命中）
	fmt.Println("\n--- 再次获取（缓存命中）---")
	s, _ = cachedBackend.Get(ctx, "greeting")
	fmt.Printf("技能: %s\n", s.Name)

	// 8. 等待缓存过期
	fmt.Println("\n--- 等待缓存过期（5秒）---")
	time.Sleep(5 * time.Second)

	// 9. 缓存过期后的调用
	fmt.Println("\n--- 缓存过期后的调用 ---")
	fms, _ = cachedBackend.List(ctx)
	fmt.Printf("技能数量: %d\n", len(fms))
}

// ============================================================================
// Part 4: 组合演示
// ============================================================================

// SimulateCombined 演示 Skill 和 ToolSearch 的组合使用
func SimulateCombined() {
	fmt.Println("=== Skill + ToolSearch 组合演示 ===")
	fmt.Println("本演示展示如何同时使用 Skill 和 ToolSearch 中间件\n")

	ctx := context.Background()

	// 1. 创建 Skill 系统
	fmt.Println("--- 初始化 Skill 系统 ---")
	skillBackend := NewMemoryBackend()
	skillBackend.Register(Skill{
		FrontMatter: FrontMatter{
			Name:        "data-analysis",
			Description: "数据分析和可视化",
			Context:     "fork_with_context",
		},
		Content: "提供数据分析和可视化能力。",
	})
	skillBackend.Register(Skill{
		FrontMatter: FrontMatter{
			Name:        "report-generation",
			Description: "生成分析报告",
			Context:     "inline",
		},
		Content: "根据分析结果生成报告。",
	})
	skillMiddleware := NewSkillMiddleware(skillBackend)

	// 2. 创建 ToolSearch 系统
	fmt.Println("\n--- 初始化 ToolSearch 系统 ---")
	dynamicTools := []DynamicTool{
		{
			Info: ToolInfo{
				Name:        "weather_forecast",
				Description: "查询天气预报",
				Parameters:  map[string]string{"city": "城市名称"},
			},
			Execute: func(args string) string { return "晴，25°C" },
		},
		{
			Info: ToolInfo{
				Name:        "stock_price",
				Description: "查询股票价格",
				Parameters:  map[string]string{"symbol": "股票代码"},
			},
			Execute: func(args string) string { return "$150.25" },
		},
		{
			Info: ToolInfo{
				Name:        "currency_convert",
				Description: "货币转换",
				Parameters:  map[string]string{"from": "源货币", "to": "目标货币", "amount": "金额"},
			},
			Execute: func(args string) string { return "720 CNY" },
		},
	}
	toolSearchMiddleware := NewToolSearchMiddleware(dynamicTools)

	// 3. 发现所有 Skill
	fmt.Println("\n--- 发现阶段 ---")
	skills, _ := skillMiddleware.DiscoverSkills(ctx)

	// 4. 模拟多轮对话
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("第 1 轮对话")
	fmt.Println(strings.Repeat("=", 60))

	// 用户请求天气
	fmt.Println("\n用户: 北京天气怎么样？")
	fmt.Println("\nAgent 决策: 需要搜索天气工具")
	toolSearchMiddleware.CallToolSearch("weather", 5)

	// Agent 使用发现的工具
	fmt.Println("\nAgent: 调用 weather_forecast")
	for _, t := range dynamicTools {
		if t.Info.Name == "weather_forecast" {
			fmt.Printf("结果: %s\n", t.Execute("北京"))
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("第 2 轮对话")
	fmt.Println(strings.Repeat("=", 60))

	// 用户请求数据分析
	fmt.Println("\n用户: 帮我分析销售数据")
	fmt.Println("\nAgent 决策: 需要使用 data-analysis Skill")

	// 激活 Skill
	var dataAnalysisSkill Skill
	for _, s := range skills {
		if s.Name == "data-analysis" {
			dataAnalysisSkill, _ = skillMiddleware.ActivateSkill(ctx, s.Name)
			break
		}
	}

	// 执行 Skill
	result, _ := skillMiddleware.ExecuteSkill(ctx, dataAnalysisSkill, "sales_data.csv")
	fmt.Printf("\n结果: %s\n", result)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("第 3 轮对话")
	fmt.Println(strings.Repeat("=", 60))

	// 用户请求生成报告
	fmt.Println("\n用户: 生成分析报告")
	fmt.Println("\nAgent 决策: 需要使用 report-generation Skill")

	// 激活 Skill
	var reportSkill Skill
	for _, s := range skills {
		if s.Name == "report-generation" {
			reportSkill, _ = skillMiddleware.ActivateSkill(ctx, s.Name)
			break
		}
	}

	// 执行 Skill
	result, _ = skillMiddleware.ExecuteSkill(ctx, reportSkill, "销售数据分析报告")
	fmt.Printf("\n结果: %s\n", result)

	// 显示最终状态
	fmt.Println("\n--- 最终状态 ---")
	fmt.Println("已激活的 Skill:")
	for name := range skillMiddleware.loadedSkills {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println("可见的动态工具:")
	for _, t := range toolSearchMiddleware.GetVisibleTools() {
		fmt.Printf("  - %s\n", t.Name)
	}
}

// ============================================================================
// Part 5: 真实 Eino Agent 示例（需要 API Key）
// ============================================================================

// ShowEinoExample 展示如何在真实 Eino Agent 中使用 Skill 和 ToolSearch
// 注意：这个函数只是展示代码，不会实际运行（需要 API Key）
func ShowEinoExample() {
	fmt.Println("=== 真实 Eino Agent 示例 ===")
	fmt.Println("以下代码展示如何在真实 Eino Agent 中使用 Skill 和 ToolSearch 中间件")
	fmt.Println("注意：运行这些代码需要设置 OPENAI_API_KEY 环境变量\n")

	fmt.Println(`// 示例 1: 使用 Skill 中间件
//
// import (
//     "github.com/cloudwego/eino/adk"
//     "github.com/cloudwego/eino/adk/middlewares/dynamictool/skill"
//     "github.com/cloudwego/eino/components/model/openai"
// )
//
// func main() {
//     ctx := context.Background()
//
//     // 1. 创建文件系统 Backend
//     // 注意：需要先创建 filesystem.Backend
//     fsBackend := createFilesystemBackend()
//     backend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
//         Backend: fsBackend,
//         BaseDir: "./skills",
//     })
//
//     // 2. 创建 Skill 中间件
//     handler, err := skill.NewMiddleware(ctx, &skill.Config{
//         Backend: backend,
//     })
//
//     // 3. 创建 Agent 并注册中间件
//     chatModel, _ := openai.NewChatModel(ctx, &openai.Config{
//         APIKey: os.Getenv("OPENAI_API_KEY"),
//         Model:  "gpt-4",
//     })
//     agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
//         Model:    chatModel,
//         Handlers: []adk.ChatModelAgentMiddleware{handler},
//     })
// }`)

	fmt.Println("\n" + strings.Repeat("-", 60))

	fmt.Println(`// 示例 2: 使用 ToolSearch 中间件
//
// import (
//     "github.com/cloudwego/eino/adk"
//     "github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
// )
//
// func main() {
//     ctx := context.Background()
//
//     // 1. 创建动态工具
//     dynamicTools := []tool.BaseTool{
//         &weatherTool{},
//         &stockTool{},
//         &currencyTool{},
//     }
//
//     // 2. 创建 ToolSearch 中间件
//     middleware, err := toolsearch.New(ctx, &toolsearch.Config{
//         DynamicTools:       dynamicTools,
//         UseModelToolSearch: false, // 默认模式
//     })
//
//     // 3. 创建 Agent 并注册中间件
//     agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
//         Model:    chatModel,
//         Handlers: []adk.ChatModelAgentMiddleware{middleware},
//     })
// }`)

	fmt.Println("\n" + strings.Repeat("-", 60))

	fmt.Println(`// 示例 3: 组合使用 Skill 和 ToolSearch
//
// func main() {
//     ctx := context.Background()
//
//     // 1. 创建 Skill 中间件
//     skillBackend := createSkillBackend()
//     skillMiddleware, _ := skill.NewMiddleware(ctx, &skill.Config{
//         Backend: skillBackend,
//     })
//
//     // 2. 创建 ToolSearch 中间件
//     dynamicTools := createDynamicTools()
//     toolSearchMiddleware, _ := toolsearch.New(ctx, &toolsearch.Config{
//         DynamicTools: dynamicTools,
//     })
//
//     // 3. 创建 Agent，组合多个中间件
//     agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
//         Model: chatModel,
//         Handlers: []adk.ChatModelAgentMiddleware{
//             skillMiddleware,       // Skill 中间件
//             toolSearchMiddleware,  // ToolSearch 中间件
//         },
//     })
// }`)
}

// ============================================================================
// 工具函数
// ============================================================================

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	// 根据命令行参数选择运行模式
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "demo":
		// 运行完整演示
		SimulateAgentWithSkill()
		fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
		SimulateAgentWithToolSearch()
		fmt.Println("\n" + strings.Repeat("=", 60) + "\n")
		SimulateCombined()

	case "skill":
		SimulateAgentWithSkill()

	case "toolsearch":
		SimulateAgentWithToolSearch()

	case "backend":
		SimulateCachedBackend()

	case "combined":
		SimulateCombined()

	case "eino":
		ShowEinoExample()

	default:
		fmt.Printf("未知命令: %s\n", os.Args[1])
		printUsage()
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("第 9 章：Skill Middleware -- 技能中间件")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run main.go <command>")
	fmt.Println()
	fmt.Println("可用命令:")
	fmt.Println("  demo        - 运行完整演示（模拟，不需要 API Key）")
	fmt.Println("  skill       - 运行 Skill 渐进式披露演示")
	fmt.Println("  toolsearch  - 运行 ToolSearch 动态发现演示")
	fmt.Println("  backend     - 运行自定义 Backend 演示")
	fmt.Println("  combined    - 运行组合演示")
	fmt.Println("  eino        - 展示真实 Eino Agent 示例（需要 API Key）")
}
